package handler

import (
	"net/http"
	"strconv"

	"kb-chat-flow/internal/model"
	"kb-chat-flow/internal/store"

	"github.com/gin-gonic/gin"
)

// WorkflowHandler 工作流管理处理器
type WorkflowHandler struct {
	store store.MetaStore
}

// NewWorkflowHandler 创建工作流处理器
func NewWorkflowHandler(metaStore store.MetaStore) *WorkflowHandler {
	return &WorkflowHandler{store: metaStore}
}

// ListPublic 返回公开的工作流列表（聊天页下拉用，不含节点详情）
func (h *WorkflowHandler) ListPublic(c *gin.Context) {
	workflows, err := h.store.ListWorkflows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取工作流列表失败: " + err.Error()})
		return
	}

	type pubWorkflow struct {
		ID          int64                `json:"id"`
		Name        string               `json:"name"`
		Description string               `json:"description"`
		Classifier  *model.ClassifierDef `json:"classifier,omitempty"`
		Nodes       []model.WorkflowNode `json:"nodes,omitempty"`
	}
	result := make([]pubWorkflow, 0, len(workflows))
	for _, w := range workflows {
		result = append(result, pubWorkflow{
			ID:          w.ID,
			Name:        w.Name,
			Description: w.Description,
			Classifier:  w.Classifier,
			Nodes:       w.Nodes,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// List 管理员获取所有工作流（完整节点）
func (h *WorkflowHandler) List(c *gin.Context) {
	workflows, err := h.store.ListWorkflows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取工作流列表失败: " + err.Error()})
		return
	}

	if workflows == nil {
		workflows = []model.WorkflowDef{}
	}

	c.JSON(http.StatusOK, gin.H{"data": workflows})
}

// Get 获取单个工作流
func (h *WorkflowHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	workflow, err := h.store.GetWorkflow(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取工作流失败: " + err.Error()})
		return
	}
	if workflow == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工作流不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": workflow})
}

// Create 创建工作流
func (h *WorkflowHandler) Create(c *gin.Context) {
	var req model.CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if len(req.Nodes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "工作流至少需要一个节点"})
		return
	}

	// 自动标记最后一个节点为 final
	req.Nodes[len(req.Nodes)-1].IsFinal = true

	workflow := &model.WorkflowDef{
		Name:        req.Name,
		Description: req.Description,
		Classifier:  req.Classifier,
		Nodes:       req.Nodes,
	}

	id, err := h.store.CreateWorkflow(workflow)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建工作流失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "id": id})
}

// Update 更新工作流
func (h *WorkflowHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	existing, err := h.store.GetWorkflow(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取工作流失败: " + err.Error()})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工作流不存在"})
		return
	}

	var req model.CreateWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误: " + err.Error()})
		return
	}

	if len(req.Nodes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "工作流至少需要一个节点"})
		return
	}

	// 自动标记最后一个节点为 final，其余为 false
	for i := range req.Nodes {
		req.Nodes[i].IsFinal = (i == len(req.Nodes)-1)
	}

	workflow := &model.WorkflowDef{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Classifier:  req.Classifier,
		Nodes:       req.Nodes,
	}

	if err := h.store.UpdateWorkflow(workflow); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新工作流失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Delete 删除工作流
func (h *WorkflowHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	if err := h.store.DeleteWorkflow(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除工作流失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
