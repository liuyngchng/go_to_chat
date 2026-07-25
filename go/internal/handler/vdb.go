package handler

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"go_to_chat/internal/kb"
	"go_to_chat/internal/model"
	"go_to_chat/internal/store"

	"github.com/gin-gonic/gin"
)

// VdbHandler 知识库管理 API 处理器
type VdbHandler struct {
	cfg   *model.Config
	kbMgr *kb.Manager
	store *store.SQLiteStore
}

// NewVdbHandler 创建知识库处理器
func NewVdbHandler(cfg *model.Config, kbMgr *kb.Manager, metaStore *store.SQLiteStore) *VdbHandler {
	return &VdbHandler{
		cfg:   cfg,
		kbMgr: kbMgr,
		store: metaStore,
	}
}

// MyList 获取用户的知识库列表
func (h *VdbHandler) MyList(c *gin.Context) {
	uid := getUID(c)
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

// PubList 获取公开知识库列表
func (h *VdbHandler) PubList(c *gin.Context) {
	uid := getUID(c)
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

// FileList 获取知识库文件列表
func (h *VdbHandler) FileList(c *gin.Context) {
	vdbID := getIntParam(c, "vdb_id")
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

// SetDefault 设置默认知识库
func (h *VdbHandler) SetDefault(c *gin.Context) {
	uid := getUID(c)
	vdbID := getIntParam(c, "id")
	if err := h.kbMgr.SetDefaultKB(vdbID, uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Create 创建知识库
func (h *VdbHandler) Create(c *gin.Context) {
	uid := getUID(c)
	name := c.PostForm("name")
	isPublic := c.PostForm("is_public") == "true" || c.PostForm("is_public") == "1"

	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "知识库名称不能为空"})
		return
	}

	id, err := h.kbMgr.CreateKB(name, uid, isPublic)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "id": id})
}

// Delete 删除知识库
func (h *VdbHandler) Delete(c *gin.Context) {
	uid := getUID(c)
	vdbID := getIntParam(c, "id")
	if err := h.kbMgr.DeleteKB(vdbID, uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Upload 上传文件到知识库
func (h *VdbHandler) Upload(c *gin.Context) {
	uid := getUID(c)
	vdbIDStr := c.PostForm("vdb_id")
	vdbID, err := strconv.ParseInt(vdbIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的知识库 ID"})
		return
	}

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

// ProcessInfo 获取文件处理进度
func (h *VdbHandler) ProcessInfo(c *gin.Context) {
	fileID := getIntParam(c, "file_id")
	finfo, err := h.store.GetFileByID(fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": finfo})
}

// Search 在知识库中检索
func (h *VdbHandler) Search(c *gin.Context) {
	uid := getUID(c)
	vdbID := getIntParam(c, "vdb_id")
	query := c.PostForm("query")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入搜索内容"})
		return
	}

	result, err := h.kbMgr.SearchInKB(query, vdbID, uid, 5, 0.1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// FileDelete 删除文件
func (h *VdbHandler) FileDelete(c *gin.Context) {
	uid := getUID(c)
	fileID := getIntParam(c, "file_id")
	if err := h.kbMgr.DeleteFile(fileID, uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ============================================================
// 辅助函数
// ============================================================

func getUID(c *gin.Context) string {
	uid := c.PostForm("uid")
	if uid == "" {
		uid = c.Query("uid")
	}
	if uid == "" {
		uid = "default"
	}
	return uid
}

func getIntParam(c *gin.Context, key string) int64 {
	val := c.PostForm(key)
	if val == "" {
		val = c.Query(key)
	}
	n, _ := strconv.ParseInt(val, 10, 64)
	return n
}
