package vdb

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"sync"

	"go_to_chat/internal/model"
)

// localDoc 本地存储的文档
type localDoc struct {
	ID      string            `json:"id"`
	Vector  []float64         `json:"vector"`
	Content string            `json:"content"`
	Source  string            `json:"source"`
}

// localData 持久化的数据结构
type localData struct {
	Dimension int        `json:"dimension"`
	Docs      []localDoc `json:"docs"`
}

// LocalStore 本地嵌入式向量存储
type LocalStore struct {
	mu       sync.RWMutex
	dataPath string
	dim      int
	docs     []localDoc
}

// NewLocalStore 创建本地向量存储
func NewLocalStore(dataPath string) (*LocalStore, error) {
	store := &LocalStore{dataPath: dataPath}

	// 尝试从文件加载已有数据
	if _, err := os.Stat(dataPath); err == nil {
		if err := store.load(); err != nil {
			return nil, fmt.Errorf("加载本地向量数据失败: %w", err)
		}
	}

	return store, nil
}

// EnsureCollection 确保本地存储已初始化
func (s *LocalStore) EnsureCollection(dimension int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.docs == nil {
		s.docs = make([]localDoc, 0)
		s.dim = dimension
	}
	return nil
}

// Insert 批量插入向量记录
func (s *LocalStore) Insert(records []model.VectorRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range records {
		source := ""
		if r.Meta != nil {
			source = r.Meta["source"]
		}
		doc := localDoc{
			ID:      r.ID,
			Vector:  r.Vector,
			Content: r.Content,
			Source:  source,
		}
		// 更新或追加
		found := false
		for i, d := range s.docs {
			if d.ID == doc.ID {
				s.docs[i] = doc
				found = true
				break
			}
		}
		if !found {
			s.docs = append(s.docs, doc)
		}
		if s.dim == 0 && len(doc.Vector) > 0 {
			s.dim = len(doc.Vector)
		}
	}

	return s.save()
}

// Search 余弦相似度检索
func (s *LocalStore) Search(queryVector []float64, topK int, scoreThreshold float64) ([]model.SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.docs) == 0 {
		return nil, nil
	}

	type scoredDoc struct {
		doc   localDoc
		score float64
	}

	var results []scoredDoc
	for _, doc := range s.docs {
		if len(doc.Vector) != len(queryVector) {
			continue
		}
		score := cosineSimilarity(queryVector, doc.Vector)
		if score >= scoreThreshold {
			results = append(results, scoredDoc{doc: doc, score: score})
		}
	}

	// 按分数降序排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// 取 topK
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
	s.mu.Lock()
	defer s.mu.Unlock()

	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}

	filtered := make([]localDoc, 0, len(s.docs))
	for _, doc := range s.docs {
		if !idSet[doc.ID] {
			filtered = append(filtered, doc)
		}
	}
	s.docs = filtered

	return s.save()
}

// DeleteBySource 根据 source 删除
func (s *LocalStore) DeleteBySource(source string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]localDoc, 0, len(s.docs))
	for _, doc := range s.docs {
		if doc.Source != source {
			filtered = append(filtered, doc)
		}
	}
	s.docs = filtered

	return s.save()
}

// Close 关闭存储
func (s *LocalStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save()
}

// ============================================================
// 内部方法
// ============================================================

func (s *LocalStore) load() error {
	data, err := os.ReadFile(s.dataPath)
	if err != nil {
		return err
	}

	var ld localData
	if err := json.Unmarshal(data, &ld); err != nil {
		return err
	}

	s.dim = ld.Dimension
	s.docs = ld.Docs
	return nil
}

func (s *LocalStore) save() error {
	ld := localData{
		Dimension: s.dim,
		Docs:      s.docs,
	}

	data, err := json.Marshal(ld)
	if err != nil {
		return err
	}

	return os.WriteFile(s.dataPath, data, 0644)
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
