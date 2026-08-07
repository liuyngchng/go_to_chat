package vdb

import "kb-chat-flow/internal/model"

// VectorsDB 本地向量数据库目录。
// 每个知识库一个独立 sqlite 文件：<VectorsDB>/kb_<vdbID>.db（与 qdrant/milvus 的 kb_<vdbID> collection 对齐）
const VectorsDB = "./vdb"

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

	// ListBySource 根据 source 字段列出所有记录（用于查看文档分块）
	ListBySource(source string) ([]model.SearchResult, error)

	// Close 关闭连接，释放资源
	Close() error

	// Purge 清空当前 store 的所有数据
	Purge() error
}
