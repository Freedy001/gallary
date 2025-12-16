package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"gallary/server/internal/service"
	"gallary/server/internal/utils"
)

// AIHandler AI 处理器
type AIHandler struct {
	service service.AIService
}

// NewAIHandler 创建 AI 处理器实例
func NewAIHandler(service service.AIService) *AIHandler {
	return &AIHandler{service: service}
}

// TestConnection 测试 AI 服务连接
//
//	@Summary		测试 AI 服务连接
//	@Description	测试嵌入模型或 LLM 服务连接
//	@Tags			AI
//	@Accept			json
//	@Produce		json
//	@Param			request	body		TestConnectionRequest	true	"测试请求"
//	@Success		200		{object}	utils.Response			"连接成功"
//	@Failure		400		{object}	utils.Response			"请求参数错误"
//	@Failure		500		{object}	utils.Response			"连接失败"
//	@Router			/api/ai/test [post]
func (h *AIHandler) TestConnection(c *gin.Context) {
	type Body struct {
		ID string `json:"id"`
	}
	var req Body
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if err := h.service.TestConnection(c.Request.Context(), req.ID); err != nil {
		utils.InternalServerError(c, "连接测试失败: "+err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "连接成功"})
}

// GetQueueStatus 获取 AI 队列状态
//
//	@Summary		获取 AI 队列状态
//	@Description	获取所有 AI 队列的状态汇总
//	@Tags			AI
//	@Produce		json
//	@Success		200	{object}	utils.Response{data=model.AIQueueStatus}	"获取成功"
//	@Failure		500	{object}	utils.Response								"获取失败"
//	@Router			/api/ai/queues [get]
func (h *AIHandler) GetQueueStatus(c *gin.Context) {
	status, err := h.service.GetQueueStatus(c.Request.Context())
	if err != nil {
		utils.InternalServerError(c, "获取队列状态失败: "+err.Error())
		return
	}
	utils.Success(c, status)
}

// GetQueueDetail 获取队列详情
//
//	@Summary		获取队列详情
//	@Description	获取指定队列的详情，包括失败图片列表
//	@Tags			AI
//	@Produce		json
//	@Param			id			path		int									true	"队列 ID"
//	@Param			page		query		int									false	"页码"		default(1)
//	@Param			page_size	query		int									false	"每页数量"	default(20)
//	@Success		200			{object}	utils.Response{data=model.AIQueueDetail}	"获取成功"
//	@Failure		400			{object}	utils.Response							"请求参数错误"
//	@Failure		500			{object}	utils.Response							"获取失败"
//	@Router			/api/ai/queues/{id} [get]
func (h *AIHandler) GetQueueDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的队列 ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	detail, err := h.service.GetQueueDetail(c.Request.Context(), id, page, pageSize)
	if err != nil {
		utils.InternalServerError(c, "获取队列详情失败: "+err.Error())
		return
	}

	utils.Success(c, detail)
}

// RetryQueueFailedImages 重试队列所有失败图片
//
//	@Summary		重试队列所有失败图片
//	@Description	重试指定队列中所有失败的图片
//	@Tags			AI
//	@Produce		json
//	@Param			id	path		int				true	"队列 ID"
//	@Success		200	{object}	utils.Response	"重试成功"
//	@Failure		400	{object}	utils.Response	"请求参数错误"
//	@Failure		500	{object}	utils.Response	"重试失败"
//	@Router			/api/ai/queues/{id}/retry [post]
func (h *AIHandler) RetryQueueFailedImages(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的队列 ID")
		return
	}

	if err := h.service.RetryQueueFailedImages(c.Request.Context(), id); err != nil {
		utils.InternalServerError(c, "重试失败: "+err.Error())
		return
	}

	utils.Success(c, nil)
}

// RetryTaskImage 重试单张图片
//
//	@Summary		重试单张图片
//	@Description	重试指定的失败图片
//	@Tags			AI
//	@Produce		json
//	@Param			id	path		int				true	"任务图片 ID"
//	@Success		200	{object}	utils.Response	"重试成功"
//	@Failure		400	{object}	utils.Response	"请求参数错误"
//	@Failure		500	{object}	utils.Response	"重试失败"
//	@Router			/api/ai/task-images/{id}/retry [post]
func (h *AIHandler) RetryTaskImage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的任务图片 ID")
		return
	}

	if err := h.service.RetryTaskImage(c.Request.Context(), id); err != nil {
		utils.InternalServerError(c, "重试失败: "+err.Error())
		return
	}

	utils.Success(c, nil)
}

// IgnoreTaskImage 忽略单张图片
//
//	@Summary		忽略单张图片
//	@Description	忽略指定的失败图片（从队列中移除）
//	@Tags			AI
//	@Produce		json
//	@Param			id	path		int				true	"任务图片 ID"
//	@Success		200	{object}	utils.Response	"操作成功"
//	@Failure		400	{object}	utils.Response	"请求参数错误"
//	@Failure		500	{object}	utils.Response	"操作失败"
//	@Router			/api/ai/task-images/{id}/ignore [post]
func (h *AIHandler) IgnoreTaskImage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的任务图片 ID")
		return
	}

	if err := h.service.IgnoreTaskImage(c.Request.Context(), id); err != nil {
		utils.InternalServerError(c, "操作失败: "+err.Error())
		return
	}

	utils.Success(c, nil)
}

// SemanticSearchRequest 语义搜索请求
type SemanticSearchRequest struct {
	Query     string `json:"query" binding:"required"` // 搜索查询
	ModelName string `json:"model_name"`               // 使用的模型名称（可选）
	Limit     int    `json:"limit"`                    // 返回结果数量
}

// SemanticSearch 语义搜索
//
//	@Summary		语义搜索
//	@Description	使用向量进行语义搜索
//	@Tags			AI
//	@Accept			json
//	@Produce		json
//	@Param			request	body		SemanticSearchRequest						true	"搜索请求"
//	@Success		200		{object}	utils.Response{data=[]model.Image}	"搜索成功"
//	@Failure		400		{object}	utils.Response								"请求参数错误"
//	@Failure		500		{object}	utils.Response								"搜索失败"
//	@Router			/api/ai/search [post]
func (h *AIHandler) SemanticSearch(c *gin.Context) {
	var req SemanticSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	images, err := h.service.SemanticSearch(c.Request.Context(), req.Query, req.ModelName, limit)
	if err != nil {
		utils.InternalServerError(c, "语义搜索失败: "+err.Error())
		return
	}

	utils.Success(c, images)
}
