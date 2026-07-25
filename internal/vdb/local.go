package vdb

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"sort"

	"go_to_chat/internal/model"

	_ "modernc.org/sqlite"
)

// LocalStore 本地 SQLite 向量存储
type LocalStore struct {
	db    *sql.DB
	dim   int
	vdbID int64
}

// NewLocalStore 创建本地向量存储
func NewLocalStore(dbPath string, vdbID int64) (*LocalStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开向量数据库失败: %w", err)
	}

	// SQLite 单写
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &LocalStore{db: db, vdbID: vdbID}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("向量表初始化失败: %w", err)
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
	s.dim = dimension
	return nil
}

// Insert 批量插入向量记录
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

	for _, r := range records {
		source := ""
		if r.Meta != nil {
			source = r.Meta["source"]
		}
		if s.dim == 0 && len(r.Vector) > 0 {
			s.dim = len(r.Vector)
		}

		vecBytes := floatsToBytes(r.Vector)
		if _, err := stmt.Exec(r.ID, s.vdbID, r.Content, vecBytes, source); err != nil {
			return fmt.Errorf("插入向量失败: %w", err)
		}
	}

	return tx.Commit()
}

// Search 余弦相似度检索
func (s *LocalStore) Search(queryVector []float64, topK int, scoreThreshold float64) ([]model.SearchResult, error) {
	docs, err := s.loadByVdbID()
	if err != nil {
		return nil, err
	}

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

// DeleteByIDs 根据 ID 删除
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

	return nil
}

// DeleteBySource 根据 source 删除
func (s *LocalStore) DeleteBySource(source string) error {
	_, err := s.db.Exec("DELETE FROM vectors WHERE vdb_id = ? AND source = ?", s.vdbID, source)
	return err
}

// Purge 清空当前知识库的所有向量
func (s *LocalStore) Purge() error {
	_, err := s.db.Exec("DELETE FROM vectors WHERE vdb_id = ?", s.vdbID)
	return err
}

// Close 关闭数据库
func (s *LocalStore) Close() error {
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

func (s *LocalStore) loadByVdbID() ([]vectorDoc, error) {
	rows, err := s.db.Query("SELECT id, content, vector, source FROM vectors WHERE vdb_id = ?", s.vdbID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []vectorDoc
	for rows.Next() {
		var doc vectorDoc
		var vecBytes []byte
		if err := rows.Scan(&doc.ID, &doc.Content, &vecBytes, &doc.Source); err != nil {
			return nil, err
		}
		doc.Vector = bytesToFloats(vecBytes)
		docs = append(docs, doc)
	}

	return docs, rows.Err()
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
