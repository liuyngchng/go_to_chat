package vdb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"go_to_chat/internal/model"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

const (
	defaultCollectionName = "knowledge_base"
	defaultShardsNum      = 1
)

// MilvusStore 远程 Milvus 向量存储
type MilvusStore struct {
	cli            client.Client
	ctx            context.Context
	collectionName string
}

// NewMilvusStore 创建 Milvus 远程连接
func NewMilvusStore(uri, token string) (*MilvusStore, error) {
	ctx := context.Background()

	cfg := client.Config{Address: uri}
	if token != "" {
		cfg.APIKey = token
	}

	cli, err := client.NewClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("连接 Milvus 失败: %w", err)
	}

	return &MilvusStore{
		cli:            cli,
		ctx:            ctx,
		collectionName: defaultCollectionName,
	}, nil
}

// EnsureCollection 确保 collection 存在
func (s *MilvusStore) EnsureCollection(dimension int) error {
	has, err := s.cli.HasCollection(s.ctx, s.collectionName)
	if err != nil {
		return fmt.Errorf("检查 collection 失败: %w", err)
	}

	if has {
		slog.Info("Milvus collection 已存在", "name", s.collectionName)
		return s.cli.LoadCollection(s.ctx, s.collectionName, false)
	}

	// 创建 schema
	schema := entity.NewSchema().
		WithName(s.collectionName).
		WithField(entity.NewField().
			WithName("id").
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(512).
			WithIsPrimaryKey(true)).
		WithField(entity.NewField().
			WithName("vector").
			WithDataType(entity.FieldTypeFloatVector).
			WithDim(int64(dimension))).
		WithField(entity.NewField().
			WithName("content").
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(65535)).
		WithField(entity.NewField().
			WithName("source").
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(1024))

	if err := s.cli.CreateCollection(s.ctx, schema, defaultShardsNum); err != nil {
		return fmt.Errorf("创建 collection 失败: %w", err)
	}

	// 创建索引
	idx, err := entity.NewIndexHNSW(entity.COSINE, 16, 200)
	if err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	if err := s.cli.CreateIndex(s.ctx, s.collectionName, "vector", idx, false); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	slog.Info("Milvus collection 已创建", "name", s.collectionName, "dim", dimension)

	return s.cli.LoadCollection(s.ctx, s.collectionName, false)
}

// Insert 批量插入向量
func (s *MilvusStore) Insert(records []model.VectorRecord) error {
	if len(records) == 0 {
		return nil
	}

	ids := make([]string, len(records))
	vectors := make([][]float32, len(records))
	contents := make([]string, len(records))
	sources := make([]string, len(records))

	for i, r := range records {
		ids[i] = r.ID
		// float64 -> float32
		vectors[i] = make([]float32, len(r.Vector))
		for j, v := range r.Vector {
			vectors[i][j] = float32(v)
		}
		contents[i] = r.Content
		source := ""
		if r.Meta != nil {
			source = r.Meta["source"]
		}
		sources[i] = source
	}

	dim := 0
	if len(vectors) > 0 {
		dim = len(vectors[0])
	}

	idCol := entity.NewColumnVarChar("id", ids)
	vecCol := entity.NewColumnFloatVector("vector", dim, vectors)
	contentCol := entity.NewColumnVarChar("content", contents)
	sourceCol := entity.NewColumnVarChar("source", sources)

	_, err := s.cli.Upsert(s.ctx, s.collectionName, "", idCol, vecCol, contentCol, sourceCol)
	if err != nil {
		return fmt.Errorf("插入向量失败: %w", err)
	}

	return nil
}

// Search 向量检索
func (s *MilvusStore) Search(queryVector []float64, topK int, scoreThreshold float64) ([]model.SearchResult, error) {
	if err := s.cli.LoadCollection(s.ctx, s.collectionName, false); err != nil {
		slog.Warn("LoadCollection", "error", err)
	}

	// float64 -> float32
	vec32 := make([]float32, len(queryVector))
	for i, v := range queryVector {
		vec32[i] = float32(v)
	}

	sp, _ := entity.NewIndexHNSWSearchParam(16)
	searchVectors := []entity.Vector{entity.FloatVector(vec32)}

	results, err := s.cli.Search(
		s.ctx, s.collectionName, nil, "",
		[]string{"id", "content", "source"},
		searchVectors, "vector", entity.COSINE, topK, sp,
	)
	if err != nil {
		return nil, fmt.Errorf("向量检索失败: %w", err)
	}

	var searchResults []model.SearchResult
	for _, result := range results {
		// result.Scores is []float32
		idColumn := result.IDs
		fields := result.Fields

		contentCol := fields.GetColumn("content")
		sourceCol := fields.GetColumn("source")

		for i := 0; i < result.ResultCount; i++ {
			score := float64(result.Scores[i])
			if score < scoreThreshold {
				continue
			}

			id, _ := idColumn.GetAsString(i)
			content := getColumnString(contentCol, i)
			source := getColumnString(sourceCol, i)

			searchResults = append(searchResults, model.SearchResult{
				ID:      id,
				Content: content,
				Meta:    map[string]string{"source": source},
				Score:   score,
			})
		}
	}

	return searchResults, nil
}

// DeleteByIDs 根据 ID 删除
func (s *MilvusStore) DeleteByIDs(ids []string) error {
	expr := fmt.Sprintf("id in [%s]", strings.Join(quoteStrings(ids), ", "))
	return s.cli.Delete(s.ctx, s.collectionName, "", expr)
}

// DeleteBySource 根据 source 删除
func (s *MilvusStore) DeleteBySource(source string) error {
	expr := fmt.Sprintf(`source == "%s"`, source)
	return s.cli.Delete(s.ctx, s.collectionName, "", expr)
}

// Purge 清空 collection 数据（通过重建实现）
func (s *MilvusStore) Purge() error {
	has, err := s.cli.HasCollection(s.ctx, s.collectionName)
	if err != nil {
		return err
	}
	if !has {
		return nil
	}
	if err := s.cli.DropCollection(s.ctx, s.collectionName); err != nil {
		return err
	}
	return nil
}

// Close 关闭连接
func (s *MilvusStore) Close() error {
	return s.cli.Close()
}

// ============================================================
// 工厂函数
// ============================================================

// New 根据配置创建向量存储
// milvusURI 为空或非 http 时返回 LocalStore，否则返回 MilvusStore
func New(milvusURI, milvusToken, storePath string, vdbID int64) (VectorStore, error) {
	if milvusURI != "" && (strings.HasPrefix(milvusURI, "http://") || strings.HasPrefix(milvusURI, "https://")) {
		slog.Info("使用远程 Milvus", "uri", milvusURI)
		return NewMilvusStore(milvusURI, milvusToken)
	}

	slog.Info("使用本地向量存储", "path", storePath)
	return NewLocalStore(storePath, vdbID)
}

// ============================================================
// 辅助函数
// ============================================================

func getColumnString(col entity.Column, idx int) string {
	if col == nil {
		return ""
	}
	s, _ := col.GetAsString(idx)
	return s
}

func quoteStrings(strs []string) []string {
	quoted := make([]string, len(strs))
	for i, s := range strs {
		quoted[i] = fmt.Sprintf(`"%s"`, s)
	}
	return quoted
}
