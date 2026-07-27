package handler

import (
	"net/http"

	"go_to_chat/internal/model"

	"github.com/gin-gonic/gin"
)

// PageHandler 页面处理器
type PageHandler struct {
	cfg *model.Config
}

// NewPageHandler 创建页面处理器
func NewPageHandler(cfg *model.Config) *PageHandler {
	return &PageHandler{cfg: cfg}
}

// Index 聊天主页面
func (h *PageHandler) Index(c *gin.Context) {
	uid := c.Query("uid")
	if uid == "" {
		uid = "default"
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"sys_name":  h.cfg.Sys.Name,
		"uid":       uid,
		"app_source": "csm",
	})
}

// VdbIndex 知识库管理页面
func (h *PageHandler) VdbIndex(c *gin.Context) {
	uid := c.Query("uid")
	if uid == "" {
		uid = "default"
	}

	c.HTML(http.StatusOK, "vdb.html", gin.H{
		"sys_name":  h.cfg.Sys.Name,
		"uid":       uid,
		"app_source": "csm",
	})
}

// ConfigIndex 系统配置页面
func (h *PageHandler) ConfigIndex(c *gin.Context) {
	uid := c.Query("uid")
	if uid == "" {
		uid = "default"
	}

	c.HTML(http.StatusOK, "config.html", gin.H{
		"sys_name":  h.cfg.Sys.Name,
		"uid":       uid,
		"app_source": "csm",
	})
}
