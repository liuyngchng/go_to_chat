package vdb

import "go_to_chat/internal/model"

// VectorStore 向量存储接口
type VectorStore interface {
	// EnsureCollection 确保 collection 存在，不存在则创建
	EnsureCollection(dimension int) error

	// Insert 批量插入向量记录
	Insert(records []model.VectorRecord) error

	// Search 向量相似度检索，返回 top_k 结果
	Search(queryVector []float64, topK int, scoreThreshold float64) ([]model.SearchResult, error)

	// DeleteByIDs 根据 ID 列表删除记录
	DeleteByIDs(ids []string) error

	// DeleteBySource 根据 source 字段删除记录
	DeleteBySource(source string) error

	// Close 关闭连接，释放资源
	Close() error

	// Purge 清空当前 store 的所有数据
	Purge() error
}
