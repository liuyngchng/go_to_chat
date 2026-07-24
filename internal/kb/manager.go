package kb

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go_to_chat/internal/embedding"
	"go_to_chat/internal/model"
	"go_to_chat/internal/store"
	"go_to_chat/internal/vdb"
)

const (
	VdbPrefix       = "./vdb/vdb_idx_"
	UploadDir       = "./upload_doc"
	FilePollInterval = 5 * time.Second
)

// Manager 知识库管理器
type Manager struct {
	cfg       *model.Config
	store     *store.SQLiteStore
	embClient *embedding.Client
	stopCh    chan struct{}
	mu        sync.RWMutex
	stores    map[string]vdb.VectorStore // key: vdbPath -> store
}

// NewManager 创建知识库管理器
func NewManager(cfg *model.Config, metaStore *store.SQLiteStore) *Manager {
	embClient := embedding.New(
		cfg.API.EmbeddingAPIURI,
		cfg.API.EmbeddingAPIKey,
		cfg.API.EmbeddingModelName,
	)

	return &Manager{
		cfg:       cfg,
		store:     metaStore,
		embClient: embClient,
		stopCh:    make(chan struct{}),
		stores:    make(map[string]vdb.VectorStore),
	}
}

// ============================================================
// 知识库 CRUD
// ============================================================

// CreateKB 创建知识库
func (m *Manager) CreateKB(name, uid string, isPublic bool) (int64, error) {
	exists, err := m.store.CheckVdbNameExists(name, uid)
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, fmt.Errorf("知识库名称已存在: %s", name)
	}

	id, err := m.store.CreateVdb(name, uid, isPublic)
	if err != nil {
		return 0, err
	}

	// 初始化向量存储
	vdbPath := m.getVdbPath(uid, id)
	vs, err := m.getOrCreateStore(vdbPath)
	if err != nil {
		return 0, fmt.Errorf("初始化向量存储失败: %w", err)
	}

	// 探测 embedding 维度并初始化 collection
	dim, err := m.embClient.Dimension()
	if err != nil {
		return 0, fmt.Errorf("探测 embedding 维度失败: %w", err)
	}
	if err := vs.EnsureCollection(dim); err != nil {
		return 0, fmt.Errorf("创建向量 collection 失败: %w", err)
	}

	return id, nil
}

// DeleteKB 删除知识库
func (m *Manager) DeleteKB(id int64, uid string) error {
	vdbInfo, err := m.store.GetVdbByID(id)
	if err != nil {
		return err
	}
	if vdbInfo == nil || vdbInfo.UID != uid {
		return fmt.Errorf("无权删除该知识库")
	}

	// 删除向量数据
	vdbPath := m.getVdbPath(uid, id)
	m.mu.Lock()
	if vs, ok := m.stores[vdbPath]; ok {
		vs.Close()
		delete(m.stores, vdbPath)
	}
	m.mu.Unlock()

	// 删除目录
	os.RemoveAll(vdbPath)

	// 删除文件记录中的文件
	files, _ := m.store.GetFilesByVdbID(id)
	for _, f := range files {
		os.Remove(f.FilePath)
	}

	return m.store.DeleteVdb(id)
}

// GetUserKBs 获取用户的所有知识库
func (m *Manager) GetUserKBs(uid string) ([]model.VdbInfo, error) {
	return m.store.GetUserVdbs(uid)
}

// GetPublicKBs 获取公开知识库
func (m *Manager) GetPublicKBs(uid string) ([]model.VdbInfo, error) {
	return m.store.GetPublicVdbs(uid)
}

// SetDefaultKB 设置默认知识库
func (m *Manager) SetDefaultKB(id int64, uid string) error {
	return m.store.SetDefaultVdb(id, uid)
}

// ============================================================
// 文件管理
// ============================================================

// UploadFile 上传文件到知识库
func (m *Manager) UploadFile(vdbID int64, uid, fileName string, reader io.Reader) (*model.VdbFileInfo, error) {
	// 检查知识库是否存在且属于该用户
	vdbInfo, err := m.store.GetVdbByID(vdbID)
	if err != nil {
		return nil, err
	}
	if vdbInfo == nil || vdbInfo.UID != uid {
		return nil, fmt.Errorf("知识库不存在")
	}

	// 创建上传目录
	if err := os.MkdirAll(UploadDir, 0755); err != nil {
		return nil, err
	}

	taskID := fmt.Sprintf("%d", time.Now().UnixNano())
	savedName := fmt.Sprintf("%s_%s", taskID, fileName)
	savedPath := filepath.Join(UploadDir, savedName)

	// 保存文件
	f, err := os.Create(savedPath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	// 计算 MD5
	hash := md5.New()
	tee := io.TeeReader(reader, hash)
	if _, err := io.Copy(f, tee); err != nil {
		return nil, fmt.Errorf("保存文件失败: %w", err)
	}
	fileMD5 := fmt.Sprintf("%x", hash.Sum(nil))

	// 检查重复
	existing, err := m.store.CheckFileMD5Exists(vdbID, fileMD5)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// 删除旧文件
		m.DeleteFile(existing.ID, uid)
	}

	// 创建数据库记录
	finfo := &model.VdbFileInfo{
		Name:        fileName,
		UID:         uid,
		VdbID:       vdbID,
		TaskID:      taskID,
		FilePath:    savedPath,
		Percent:     0,
		ProcessInfo: "文件已上传，等待处理",
		FileMD5:     fileMD5,
	}

	id, err := m.store.CreateFileInfo(finfo)
	if err != nil {
		os.Remove(savedPath)
		return nil, err
	}
	finfo.ID = id

	return finfo, nil
}

// GetFiles 获取知识库下的所有文件
func (m *Manager) GetFiles(vdbID int64) ([]model.VdbFileInfo, error) {
	return m.store.GetFilesByVdbID(vdbID)
}

// DeleteFile 删除文件
func (m *Manager) DeleteFile(fileID int64, uid string) error {
	finfo, err := m.store.GetFileByID(fileID)
	if err != nil {
		return err
	}
	if finfo == nil || finfo.UID != uid {
		return fmt.Errorf("文件不存在")
	}

	// 从向量库删除
	vdbInfo, err := m.store.GetVdbByID(finfo.VdbID)
	if err == nil && vdbInfo != nil {
		vdbPath := m.getVdbPath(uid, finfo.VdbID)
		absPath, _ := filepath.Abs(finfo.FilePath)
		m.deleteVectorsBySource(vdbPath, absPath)
	}

	// 删除文件
	os.Remove(finfo.FilePath)

	return m.store.DeleteFile(fileID)
}

// ============================================================
// 检索
// ============================================================

// SearchInKB 在单个知识库中检索
func (m *Manager) SearchInKB(query string, vdbID int64, uid string, topK int, scoreThreshold float64) (string, error) {
	vdbPath := m.getVdbPath(uid, vdbID)
	vs, err := m.getOrCreateStore(vdbPath)
	if err != nil {
		return "", err
	}

	// 计算 query 向量
	queryVec, err := m.embClient.EmbedSingle(query)
	if err != nil {
		return "", fmt.Errorf("query embedding 失败: %w", err)
	}

	results, err := vs.Search(queryVec, topK, scoreThreshold)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, r := range results {
		content := strings.ReplaceAll(r.Content, "\n", "")
		if strings.Contains(content, ".......................") {
			continue
		}
		sb.WriteString(content)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// SearchAllKBs 在用户所有知识库中检索
func (m *Manager) SearchAllKBs(query string, uid string, topK int, scoreThreshold float64) string {
	kbList, err := m.store.GetUserVdbs(uid)
	if err != nil {
		log.Printf("获取知识库列表失败: %v", err)
		return ""
	}

	var allContext strings.Builder
	for _, kb := range kbList {
		ctx, err := m.SearchInKB(query, kb.ID, uid, topK, scoreThreshold)
		if err != nil {
			log.Printf("搜索知识库 %s 失败: %v", kb.Name, err)
			continue
		}
		if ctx != "" {
			allContext.WriteString(fmt.Sprintf("[%s]\n", kb.Name))
			allContext.WriteString(ctx)
		}
	}

	return allContext.String()
}

// ============================================================
// 文档处理 Worker
// ============================================================

// StartFileWorker 启动后台文件处理
func (m *Manager) StartFileWorker() {
	log.Println("文件处理 worker 已启动")
	ticker := time.NewTicker(FilePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			log.Println("文件处理 worker 已停止")
			return
		case <-ticker.C:
			m.processPendingFiles()
		}
	}
}

// StopFileWorker 停止 worker
func (m *Manager) StopFileWorker() {
	close(m.stopCh)
}

// processPendingFiles 处理待处理的文件
func (m *Manager) processPendingFiles() {
	files, err := m.store.GetUnprocessedFiles()
	if err != nil {
		log.Printf("获取待处理文件失败: %v", err)
		return
	}

	for _, f := range files {
		if err := m.processFile(&f); err != nil {
			log.Printf("处理文件 %s 失败: %v", f.Name, err)
			m.store.UpdateFileProgress(f.ID, 0, fmt.Sprintf("处理失败: %v", err))
		}
	}
}

// processFile 处理单个文件（向量化入库）
func (m *Manager) processFile(finfo *model.VdbFileInfo) error {
	log.Printf("开始处理文件: %s (id=%d)", finfo.Name, finfo.ID)
	m.store.UpdateFileProgress(finfo.ID, 1, "开始处理文档")

	// 读取文件内容
	content, err := os.ReadFile(finfo.FilePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	text := string(content)
	if strings.TrimSpace(text) == "" {
		m.store.UpdateFileProgress(finfo.ID, 100, "文件内容为空")
		return nil
	}

	// 文本切分
	chunks := splitText(text, 300, 80)
	if len(chunks) == 0 {
		m.store.UpdateFileProgress(finfo.ID, 100, "无可切分的文本内容")
		return nil
	}

	log.Printf("文件 %s 切分为 %d 个文本块", finfo.Name, len(chunks))
	m.store.UpdateFileProgress(finfo.ID, 5, fmt.Sprintf("已切分为 %d 个文本块，开始向量化", len(chunks)))

	// 初始化向量存储
	vdbPath := m.getVdbPath(finfo.UID, finfo.VdbID)
	vs, err := m.getOrCreateStore(vdbPath)
	if err != nil {
		return fmt.Errorf("获取向量存储失败: %w", err)
	}

	dim, err := m.embClient.Dimension()
	if err != nil {
		return fmt.Errorf("探测 embedding 维度失败: %w", err)
	}
	if err := vs.EnsureCollection(dim); err != nil {
		return fmt.Errorf("初始化 collection 失败: %w", err)
	}

	// 批量向量化并插入
	batchSize := 10
	fileName := filepath.Base(finfo.FilePath)
	absPath, _ := filepath.Abs(finfo.FilePath)
	totalChunks := len(chunks)

	for i := 0; i < totalChunks; i += batchSize {
		end := i + batchSize
		if end > totalChunks {
			end = totalChunks
		}

		batch := chunks[i:end]
		batchTexts := make([]string, len(batch))
		for j, c := range batch {
			batchTexts[j] = c
		}

		// 批量 embedding
		embeddings, err := m.embClient.Embed(batchTexts)
		if err != nil {
			return fmt.Errorf("embedding 失败 (batch %d-%d): %w", i, end, err)
		}

		// 构建记录
		records := make([]model.VectorRecord, len(batch))
		for j := range batch {
			records[j] = model.VectorRecord{
				ID:      fmt.Sprintf("%s_chunk_%d", fileName, i+j),
				Vector:  embeddings[j],
				Content: batchTexts[j],
				Meta:    map[string]string{"source": absPath},
			}
		}

		// 插入向量存储
		if err := vs.Insert(records); err != nil {
			return fmt.Errorf("插入向量失败: %w", err)
		}

		// 更新进度
		percent := float64(end) / float64(totalChunks) * 100
		if percent > 99 {
			percent = 99
		}
		m.store.UpdateFileProgress(finfo.ID, percent,
			fmt.Sprintf("已处理 %d/%d 个文本块", end, totalChunks))
	}

	m.store.UpdateFileProgress(finfo.ID, 100, fmt.Sprintf("处理完成，共 %d 个文本块", totalChunks))
	log.Printf("文件处理完成: %s", finfo.Name)
	return nil
}

// ============================================================
// 内部方法
// ============================================================

func (m *Manager) getVdbPath(uid string, vdbID int64) string {
	return fmt.Sprintf("%s%s_%d", VdbPrefix, uid, vdbID)
}

func (m *Manager) getOrCreateStore(vdbPath string) (vdb.VectorStore, error) {
	m.mu.RLock()
	vs, ok := m.stores[vdbPath]
	m.mu.RUnlock()

	if ok {
		return vs, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查
	if vs, ok = m.stores[vdbPath]; ok {
		return vs, nil
	}

	// 确保目录存在
	os.MkdirAll(vdbPath, 0755)

	storeFile := filepath.Join(vdbPath, "vectors.json")
	milvusURI := m.cfg.Milvus.URI
	milvusToken := m.cfg.Milvus.Token

	vs, err := vdb.New(milvusURI, milvusToken, storeFile)
	if err != nil {
		return nil, err
	}

	m.stores[vdbPath] = vs
	return vs, nil
}

func (m *Manager) deleteVectorsBySource(vdbPath, source string) {
	m.mu.RLock()
	vs, ok := m.stores[vdbPath]
	m.mu.RUnlock()

	if ok {
		vs.DeleteBySource(source)
	}
}

// ============================================================
// 文本切分
// ============================================================

// splitText 简单文本切分
func splitText(text string, chunkSize, chunkOverlap int) []string {
	if chunkSize <= 0 {
		chunkSize = 300
	}

	// 按段落分隔
	paragraphs := strings.Split(text, "\n")
	var chunks []string
	var current string

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		runes := []rune(para)
		if len(runes) <= chunkSize {
			if current == "" {
				current = para
			} else {
				combined := current + "\n" + para
				if len([]rune(combined)) <= chunkSize {
					current = combined
				} else {
					chunks = append(chunks, current)
					current = para
				}
			}
		} else {
			// 长段落需要切分
			if current != "" {
				chunks = append(chunks, current)
				current = ""
			}
			for i := 0; i < len(runes); i += chunkSize - chunkOverlap {
				end := i + chunkSize
				if end > len(runes) {
					end = len(runes)
				}
				chunks = append(chunks, string(runes[i:end]))
				if end == len(runes) {
					break
				}
			}
		}
	}

	if current != "" {
		chunks = append(chunks, current)
	}

	return chunks
}
