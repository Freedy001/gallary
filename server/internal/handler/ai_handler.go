package handler

import (
	"gallary/server/pkg/logger"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gallary/server/internal/service"
	"gallary/server/internal/utils"
)

// AIHandler AI 处理器
type AIHandler struct {
	service           service.AIService
	smartAlbumService service.SmartAlbumService
}

// NewAIHandler 创建 AI 处理器实例
func NewAIHandler(service service.AIService, smartAlbumService service.SmartAlbumService) *AIHandler {
	return &AIHandler{service: service, smartAlbumService: smartAlbumService}
}

// TestConnection 测试 AI 服务连接
//
//	@Summary		测试 AI 服务连接
//	@Description	测试嵌入模型或 LLM 服务连接（使用传入的临时配置）
//	@Tags			AI
//	@Accept			json
//	@Produce		json
//	@Param			request	body		TestConnectionRequest	true	"测试请求"
//	@Success		200		{object}	utils.Response			"连接成功"
//	@Failure		400		{object}	utils.Response			"请求参数错误"
//	@Failure		500		{object}	utils.Response			"连接失败"
//	@Router			/api/ai/test-connection [post]
func (h *AIHandler) TestConnection(c *gin.Context) {
	var req service.TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if err := h.service.TestConnection(c.Request.Context(), &req); err != nil {
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

// RetryQueueFailedItems 重试队列所有失败项目
//
//	@Summary		重试队列所有失败项目
//	@Description	重试指定队列中所有失败的项目
//	@Tags			AI
//	@Produce		json
//	@Param			id	path		int				true	"队列 ID"
//	@Success		200	{object}	utils.Response	"重试成功"
//	@Failure		400	{object}	utils.Response	"请求参数错误"
//	@Failure		500	{object}	utils.Response	"重试失败"
//	@Router			/api/ai/queues/{id}/retry [post]
func (h *AIHandler) RetryQueueFailedItems(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的队列 ID")
		return
	}

	if err := h.service.RetryQueueFailedItems(c.Request.Context(), id); err != nil {
		utils.InternalServerError(c, "重试失败: "+err.Error())
		return
	}

	utils.Success(c, nil)
}

// RetryTaskItem 重试单个任务项
//
//	@Summary		重试单个任务项
//	@Description	重试指定的失败任务项
//	@Tags			AI
//	@Produce		json
//	@Param			id	path		int				true	"任务项 ID"
//	@Success		200	{object}	utils.Response	"重试成功"
//	@Failure		400	{object}	utils.Response	"请求参数错误"
//	@Failure		500	{object}	utils.Response	"重试失败"
//	@Router			/api/ai/task-items/{id}/retry [post]
func (h *AIHandler) RetryTaskItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的任务项 ID")
		return
	}

	if err := h.service.RetryTaskItem(c.Request.Context(), id); err != nil {
		utils.InternalServerError(c, "重试失败: "+err.Error())
		return
	}

	utils.Success(c, nil)
}

// IgnoreTaskItem 忽略单个任务项
//
//	@Summary		忽略单个任务项
//	@Description	忽略指定的失败任务项（从队列中移除）
//	@Tags			AI
//	@Produce		json
//	@Param			id	path		int				true	"任务项 ID"
//	@Success		200	{object}	utils.Response	"操作成功"
//	@Failure		400	{object}	utils.Response	"请求参数错误"
//	@Failure		500	{object}	utils.Response	"操作失败"
//	@Router			/api/ai/task-items/{id}/ignore [post]
func (h *AIHandler) IgnoreTaskItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的任务项 ID")
		return
	}

	if err := h.service.IgnoreTaskItem(c.Request.Context(), id); err != nil {
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

// GetEmbeddingModels 获取可用的嵌入模型列表
//
//	@Summary		获取可用的嵌入模型列表
//	@Description	获取所有已启用且支持嵌入功能的模型名称列表
//	@Tags			AI
//	@Produce		json
//	@Success		200	{object}	utils.Response{data=[]string}	"获取成功"
//	@Failure		500	{object}	utils.Response					"获取失败"
//	@Router			/api/ai/embedding-models [get]
func (h *AIHandler) GetEmbeddingModels(c *gin.Context) {
	models, err := h.service.GetEmbeddingModels(c.Request.Context())
	if err != nil {
		utils.InternalServerError(c, "获取嵌入模型列表失败: "+err.Error())
		return
	}
	utils.Success(c, models)
}

// GetChatCompletionModels 获取支持 ChatCompletion 的模型列表
//
//	@Summary		获取支持 ChatCompletion 的模型列表
//	@Description	获取所有已启用且支持 ChatCompletion 功能的模型名称列表
//	@Tags			AI
//	@Produce		json
//	@Success		200	{object}	utils.Response{data=[]string}	"获取成功"
//	@Failure		500	{object}	utils.Response					"获取失败"
//	@Router			/api/ai/chat-completion-models [get]
func (h *AIHandler) GetChatCompletionModels(c *gin.Context) {
	models, err := h.service.GetChatCompletionModels(c.Request.Context())
	if err != nil {
		utils.InternalServerError(c, "获取 ChatCompletion 模型列表失败: "+err.Error())
		return
	}
	utils.Success(c, models)
}

// OptimizePromptRequest 优化提示词请求
type OptimizePromptRequest struct {
	Query string `json:"query" binding:"required"` // 原始查询
}

// OptimizePromptResponse 优化提示词响应
type OptimizePromptResponse struct {
	OriginalQuery   string `json:"original_query"`
	OptimizedPrompt string `json:"optimized_prompt"`
}

// OptimizePrompt 优化搜索提示词
//
//	@Summary		优化搜索提示词
//	@Description	将中文搜索查询优化为适合语义搜索的英文描述
//	@Tags			AI
//	@Accept			json
//	@Produce		json
//	@Param			request	body		OptimizePromptRequest						true	"优化请求"
//	@Success		200		{object}	utils.Response{data=OptimizePromptResponse}	"优化成功"
//	@Failure		400		{object}	utils.Response								"请求参数错误"
//	@Failure		500		{object}	utils.Response								"优化失败"
//	@Router			/api/ai/optimize-prompt [post]
func (h *AIHandler) OptimizePrompt(c *gin.Context) {
	var req OptimizePromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	optimized, err := h.service.OptimizePrompt(c.Request.Context(), req.Query)
	if err != nil {
		utils.InternalServerError(c, "提示词优化失败: "+err.Error())
		return
	}

	utils.Success(c, OptimizePromptResponse{
		OriginalQuery:   req.Query,
		OptimizedPrompt: optimized,
	})
}

// GenerateSmartAlbums 生成智能相册
//
//	@Summary		生成智能相册
//	@Description	使用 HDBSCAN 算法对图片进行聚类，生成智能相册
//	@Tags			智能相册
//	@Accept			json
//	@Produce		json
//	@Param			request	body		albumService.GenerateSmartAlbumsRequest					true	"生成请求"
//	@Success		200		{object}	utils.Response{data=model.SmartAlbumProgressVO}		"生成成功"
//	@Failure		400		{object}	utils.Response										"无效的参数"
//	@Failure		500		{object}	utils.Response										"生成失败"
//	@Router			/api/ai/smart-albums-generate [post]
func (h *AIHandler) GenerateSmartAlbums(c *gin.Context) {
	var req service.GenerateSmartAlbumsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的参数: "+err.Error())
		return
	}

	result, err := h.smartAlbumService.SubmitSmartAlbumTask(c.Request.Context(), &req)
	if err != nil {
		logger.Error("生成智能相册失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "智能相册生成成功", result)
}
