package vdb

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"kb-chat-flow/internal/model"

	_ "modernc.org/sqlite"
)

// LocalStore 本地 SQLite 向量存储（内存缓存 + 持久化）
type LocalStore struct {
	db     *sql.DB
	dbPath string // 独立 sqlite 文件路径（kb_<vdbID>.db）
	mu     sync.RWMutex
	dim    int

	vdbID int64
	docs  []vectorDoc // 内存缓存，启动时从 DB 加载，写入时同步更新
}

// NewLocalStore 创建本地向量存储。
// 每个知识库一个独立 sqlite 文件：<dbDir>/kb_<vdbID>.db（物理隔离，对齐 qdrant/milvus 的 kb_<vdbID> collection）
func NewLocalStore(dbDir string, vdbID int64) (*LocalStore, error) {
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("创建向量目录失败: %w", err)
	}
	dbPath := filepath.Join(dbDir, fmt.Sprintf("kb_%d.db", vdbID))

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开向量数据库失败: %w", err)
	}

	// WAL 模式：读不阻塞写，写不阻塞读
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("启用 WAL 失败: %w", err)
	}

	// 读多写少场景，允许多个并发读
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)

	store := &LocalStore{db: db, dbPath: dbPath, vdbID: vdbID}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("向量表初始化失败: %w", err)
	}

	// 启动时加载已有向量到内存
	if err := store.loadMem(); err != nil {
		db.Close()
		return nil, fmt.Errorf("加载向量到内存失败: %w", err)
	}

	return store, nil
}

func (s *LocalStore) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS vectors (
			id      TEXT NOT NULL,
			vdb_id  INTEGER NOT NULL,
			content TEXT NOT NULL,
			vector  BLOB NOT NULL,
			source  TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (vdb_id, id)
		);
		CREATE INDEX IF NOT EXISTS idx_vectors_vdb_id ON vectors(vdb_id);
		CREATE INDEX IF NOT EXISTS idx_vectors_source ON vectors(vdb_id, source);
	`)
	return err
}

// EnsureCollection 确保已初始化
func (s *LocalStore) EnsureCollection(dimension int) error {
	s.mu.Lock()
	s.dim = dimension
	s.mu.Unlock()
	return nil
}

// Insert 批量插入向量记录（同时更新 DB 和内存）
func (s *LocalStore) Insert(records []model.VectorRecord) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		"INSERT OR REPLACE INTO vectors (id, vdb_id, content, vector, source) VALUES (?, ?, ?, ?, ?)",
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	newDocs := make([]vectorDoc, len(records))
	for i, r := range records {
		source := ""
		if r.Meta != nil {
			source = r.Meta["source"]
		}

		vecBytes := floatsToBytes(r.Vector)
		if _, err := stmt.Exec(r.ID, s.vdbID, r.Content, vecBytes, source); err != nil {
			return fmt.Errorf("插入向量失败: %w", err)
		}

		newDocs[i] = vectorDoc{
			ID:      r.ID,
			Content: r.Content,
			Vector:  r.Vector,
			Source:  source,
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// DB 写入成功 → 更新内存
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dim == 0 && len(newDocs) > 0 && len(newDocs[0].Vector) > 0 {
		s.dim = len(newDocs[0].Vector)
	}

	// 用 map 去重更新
	index := make(map[string]int, len(s.docs))
	for i, d := range s.docs {
		index[d.ID] = i
	}
	for _, nd := range newDocs {
		if pos, ok := index[nd.ID]; ok {
			s.docs[pos] = nd
		} else {
			s.docs = append(s.docs, nd)
			index[nd.ID] = len(s.docs) - 1
		}
	}

	return nil
}

// Search 余弦相似度检索（从内存读取）
func (s *LocalStore) Search(queryVector []float64, topK int, scoreThreshold float64) ([]model.SearchResult, error) {
	s.mu.RLock()
	docs := s.docs
	s.mu.RUnlock()

	if len(docs) == 0 {
		return nil, nil
	}

	type scoredDoc struct {
		doc   vectorDoc
		score float64
	}

	var results []scoredDoc
	for _, doc := range docs {
		if len(doc.Vector) != len(queryVector) {
			continue
		}
		score := cosineSimilarity(queryVector, doc.Vector)
		if score >= scoreThreshold {
			results = append(results, scoredDoc{doc: doc, score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if topK > len(results) {
		topK = len(results)
	}
	results = results[:topK]

	formatted := make([]model.SearchResult, len(results))
	for i, r := range results {
		formatted[i] = model.SearchResult{
			ID:      r.doc.ID,
			Content: r.doc.Content,
			Meta:    map[string]string{"source": r.doc.Source},
			Score:   r.score,
		}
	}

	return formatted, nil
}

// DeleteByIDs 根据 ID 删除（同时更新 DB 和内存）
func (s *LocalStore) DeleteByIDs(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	batchSize := 500
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[i:end]

		placeholders := make([]byte, 0, len(batch)*2)
		args := make([]any, len(batch)+1)
		args[0] = s.vdbID
		for j, id := range batch {
			if j > 0 {
				placeholders = append(placeholders, ',')
			}
			placeholders = append(placeholders, '?')
			args[j+1] = id
		}

		query := "DELETE FROM vectors WHERE vdb_id = ? AND id IN (" + string(placeholders) + ")"
		if _, err := s.db.Exec(query, args...); err != nil {
			return fmt.Errorf("批量删除向量失败: %w", err)
		}
	}

	// 更新内存
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	filtered := make([]vectorDoc, 0, len(s.docs))
	for _, doc := range s.docs {
		if !idSet[doc.ID] {
			filtered = append(filtered, doc)
		}
	}
	s.docs = filtered

	return nil
}

// DeleteBySource 根据 source 删除（同时更新 DB 和内存）
func (s *LocalStore) DeleteBySource(source string) error {
	if _, err := s.db.Exec("DELETE FROM vectors WHERE vdb_id = ? AND source = ?", s.vdbID, source); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]vectorDoc, 0, len(s.docs))
	for _, doc := range s.docs {
		if doc.Source != source {
			filtered = append(filtered, doc)
		}
	}
	s.docs = filtered

	return nil
}

// ListBySource 根据 source 列出所有 chunks（从内存缓存读取，按 ID 排序）
func (s *LocalStore) ListBySource(source string) ([]model.SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []model.SearchResult
	for _, doc := range s.docs {
		if doc.Source == source {
			results = append(results, model.SearchResult{
				ID:      doc.ID,
				Content: doc.Content,
				Meta:    map[string]string{"source": doc.Source},
			})
		}
	}
	return results, nil
}

// Purge 清空当前知识库的所有向量（物理隔离下 = 关闭并删除独立 db 文件）
func (s *LocalStore) Purge() error {
	// 关闭连接，释放文件句柄
	s.mu.Lock()
	s.docs = nil
	s.mu.Unlock()

	if err := s.db.Close(); err != nil {
		return err
	}
	s.db = nil

	// 删除独立的 db 文件
	if err := os.Remove(s.dbPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Close 关闭数据库
func (s *LocalStore) Close() error {
	if s.db == nil {
		return nil // Purge 已关闭并删除文件
	}
	return s.db.Close()
}

// ============================================================
// 内部方法
// ============================================================

type vectorDoc struct {
	ID      string
	Content string
	Vector  []float64
	Source  string
}

func (s *LocalStore) loadMem() error {
	rows, err := s.db.Query("SELECT id, content, vector, source FROM vectors WHERE vdb_id = ?", s.vdbID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var docs []vectorDoc
	for rows.Next() {
		var doc vectorDoc
		var vecBytes []byte
		if err := rows.Scan(&doc.ID, &doc.Content, &vecBytes, &doc.Source); err != nil {
			return err
		}
		doc.Vector = bytesToFloats(vecBytes)
		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	s.docs = docs
	if len(docs) > 0 && s.dim == 0 {
		s.dim = len(docs[0].Vector)
	}
	s.mu.Unlock()

	return nil
}

// floatsToBytes 将 float64 切片转为二进制 BLOB
func floatsToBytes(f []float64) []byte {
	buf := make([]byte, len(f)*8)
	for i, v := range f {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(v))
	}
	return buf
}

// bytesToFloats 将二进制 BLOB 转回 float64 切片
func bytesToFloats(b []byte) []float64 {
	if len(b)%8 != 0 {
		return nil
	}
	f := make([]float64, len(b)/8)
	for i := range f {
		f[i] = math.Float64frombits(binary.LittleEndian.Uint64(b[i*8:]))
	}
	return f
}

// cosineSimilarity 计算余弦相似度 [0, 1]
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
