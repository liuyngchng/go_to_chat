package handler

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"kb-chat-flow/internal/kb"
	"kb-chat-flow/internal/model"
	"kb-chat-flow/internal/store"

	"github.com/gin-gonic/gin"
)

// VdbHandler 知识库管理 API 处理器
type VdbHandler struct {
	cfg   *model.Config
	kbMgr *kb.Manager
	store store.MetaStore
}

// NewVdbHandler 创建知识库处理器
func NewVdbHandler(cfg *model.Config, kbMgr *kb.Manager, metaStore store.MetaStore) *VdbHandler {
	return &VdbHandler{
		cfg:   cfg,
		kbMgr: kbMgr,
		store: metaStore,
	}
}

// MyList 获取用户的知识库列表 GET /api/vdb
func (h *VdbHandler) MyList(c *gin.Context) {
	uid := getAuthUID(c)
	list, err := h.kbMgr.GetUserKBs(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []model.VdbInfo{}
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// PubList 获取公开知识库列表 GET /api/vdb/pub
func (h *VdbHandler) PubList(c *gin.Context) {
	uid := getAuthUID(c)
	list, err := h.kbMgr.GetPublicKBs(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []model.VdbInfo{}
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

// FileList 获取知识库文件列表 GET /api/vdb/:id/files
func (h *VdbHandler) FileList(c *gin.Context) {
	vdbID := getPathIntParam(c, "id")
	files, err := h.kbMgr.GetFiles(vdbID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if files == nil {
		files = []model.VdbFileInfo{}
	}
	c.JSON(http.StatusOK, gin.H{"data": files})
}

// SetDefault 设置默认知识库 PUT /api/vdb/:id/default
func (h *VdbHandler) SetDefault(c *gin.Context) {
	uid := getAuthUID(c)
	vdbID := getPathIntParam(c, "id")
	if err := h.kbMgr.SetDefaultKB(vdbID, uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Create 创建知识库 POST /api/vdb
func (h *VdbHandler) Create(c *gin.Context) {
	uid := getAuthUID(c)

	var req model.VdbCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "知识库名称不能为空"})
		return
	}

	id, err := h.kbMgr.CreateKB(req.Name, uid, req.IsPublic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "id": id})
}

// Delete 删除知识库 DELETE /api/vdb/:id
func (h *VdbHandler) Delete(c *gin.Context) {
	uid := getAuthUID(c)
	vdbID := getPathIntParam(c, "id")
	if err := h.kbMgr.DeleteKB(vdbID, uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Upload 上传文件到知识库 POST /api/vdb/:id/upload (multipart/form-data)
func (h *VdbHandler) Upload(c *gin.Context) {
	uid := getAuthUID(c)
	vdbID := getPathIntParam(c, "id")

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择文件"})
		return
	}

	// 检查文件类型
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".txt": true, ".md": true,
		".pdf": true, ".docx": true,
		".xlsx": true,
	}
	if !allowedExts[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的文件格式，支持: txt, md, pdf, docx, xlsx"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "打开文件失败"})
		return
	}
	defer f.Close()

	finfo, err := h.kbMgr.UploadFile(vdbID, uid, file.Filename, f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "file": finfo})
}

// ProcessInfo 获取文件处理进度 GET /api/vdb/file/:id/progress
func (h *VdbHandler) ProcessInfo(c *gin.Context) {
	fileID := getPathIntParam(c, "id")
	finfo, err := h.store.GetFileByID(fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": finfo})
}

// Search 在知识库中检索 POST /api/vdb/search
func (h *VdbHandler) Search(c *gin.Context) {
	uid := getAuthUID(c)

	var req model.VdbSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	result, err := h.kbMgr.SearchInKB(req.Query, req.VdbID, uid, h.cfg.KB.TopK, h.cfg.KB.ScoreThreshold)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// FileDelete 删除文件 DELETE /api/vdb/file/:id
func (h *VdbHandler) FileDelete(c *gin.Context) {
	uid := getAuthUID(c)
	fileID := getPathIntParam(c, "id")
	if err := h.kbMgr.DeleteFile(fileID, uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// getPathIntParam 从 URL 路径参数中解析 int64
func getPathIntParam(c *gin.Context, key string) int64 {
	val := c.Param(key)
	n, _ := strconv.ParseInt(val, 10, 64)
	return n
}
