package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gallary/server/internal/service"
	"gallary/server/internal/utils"
	"gallary/server/pkg/logger"
)

// AlbumHandler 相册处理器
type AlbumHandler struct {
	albumService service.AlbumService
	aiService    service.AIService
}

// NewAlbumHandler 创建相册处理器实例
func NewAlbumHandler(albumService service.AlbumService, aiService service.AIService) *AlbumHandler {
	return &AlbumHandler{
		albumService: albumService,
		aiService:    aiService,
	}
}

// Create 创建相册
//
//	@Summary		创建相册
//	@Description	创建新相册
//	@Tags			相册管理
//	@Accept			json
//	@Produce		json
//	@Param			request	body		albumService.CreateAlbumRequest				true	"创建相册请求"
//	@Success		200		{object}	utils.Response{data=model.AlbumVO}		"创建成功"
//	@Failure		400		{object}	utils.Response							"无效的参数"
//	@Failure		500		{object}	utils.Response							"创建失败"
//	@Router			/api/albums [post]
func (h *AlbumHandler) Create(c *gin.Context) {
	var req service.CreateAlbumRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的参数: "+err.Error())
		return
	}

	album, err := h.albumService.Create(c.Request.Context(), &req)
	if err != nil {
		logger.Error("创建相册失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "创建成功", album)
}

// List 获取相册列表
//
//	@Summary		获取相册列表
//	@Description	分页获取相册列表
//	@Tags			相册管理
//	@Produce		json
//	@Param			page		query		int															false	"页码"	default(1)
//	@Param			page_size	query		int															false	"每页数量"	default(20)
//	@Param			is_smart	query		bool														false	"是否智能相册（true-只返回智能相册，false-只返回普通相册，不传-返回全部）"
//	@Success		200			{object}	utils.Response{data=utils.PageData{list=model.AlbumVO}}	"相册列表"
//	@Failure		500			{object}	utils.Response												"获取失败"
//	@Router			/api/albums [get]
func (h *AlbumHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 解析 is_smart 查询参数
	var isSmart *bool
	if isSmartStr := c.Query("is_smart"); isSmartStr != "" {
		val := isSmartStr == "true"
		isSmart = &val
	}

	albums, total, err := h.albumService.List(c.Request.Context(), page, pageSize, isSmart)
	if err != nil {
		logger.Error("获取相册列表失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.PageResponse(c, albums, total, page, pageSize)
}

// Update 更新相册
//
//	@Summary		更新相册
//	@Description	更新相册信息
//	@Tags			相册管理
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int										true	"相册ID"
//	@Param			request	body		albumService.UpdateAlbumRequest				true	"更新相册请求"
//	@Success		200		{object}	utils.Response{data=model.AlbumVO}		"更新成功"
//	@Failure		400		{object}	utils.Response							"无效的参数"
//	@Failure		404		{object}	utils.Response							"相册不存在"
//	@Failure		500		{object}	utils.Response							"更新失败"
//	@Router			/api/albums/{id} [put]
func (h *AlbumHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的相册ID")
		return
	}

	var req service.UpdateAlbumRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的参数: "+err.Error())
		return
	}

	album, err := h.albumService.Update(c.Request.Context(), id, &req)
	if err != nil {
		logger.Error("更新相册失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "更新成功", album)
}

// BatchDelete 批量删除相册
//
//	@Summary		批量删除相册
//	@Description	批量删除多个相册
//	@Tags			相册管理
//	@Accept			json
//	@Produce		json
//	@Param			request	body		object{ids=[]int64}	true	"相册ID列表"
//	@Success		200		{object}	utils.Response		"删除成功"
//	@Failure		400		{object}	utils.Response		"无效的参数"
//	@Failure		500		{object}	utils.Response		"删除失败"
//	@Router			/api/albums/batch-delete [post]
func (h *AlbumHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的参数")
		return
	}

	if len(req.IDs) == 0 {
		utils.BadRequest(c, "请选择要删除的相册")
		return
	}

	if err := h.albumService.BatchDelete(c.Request.Context(), req.IDs); err != nil {
		logger.Error("批量删除相册失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "删除成功", nil)
}

// BatchCopy 批量复制相册
//
//	@Summary		批量复制相册
//	@Description	批量复制多个相册（包括相册内的所有图片关联）
//	@Tags			相册管理
//	@Accept			json
//	@Produce		json
//	@Param			request	body		object{ids=[]int64}						true	"相册ID列表"
//	@Success		200		{object}	utils.Response{data=[]model.AlbumVO}	"复制成功"
//	@Failure		400		{object}	utils.Response							"无效的参数"
//	@Failure		500		{object}	utils.Response							"复制失败"
//	@Router			/api/albums/batch-copy [post]
func (h *AlbumHandler) BatchCopy(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的参数")
		return
	}

	if len(req.IDs) == 0 {
		utils.BadRequest(c, "请选择要复制的相册")
		return
	}

	albums, err := h.albumService.BatchCopy(c.Request.Context(), req.IDs)
	if err != nil {
		logger.Error("批量复制相册失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "复制成功", albums)
}

// AINaming AI 命名相册
//
//	@Summary		AI 命名相册
//	@Description	将相册命名任务添加到 AI 处理队列，使用视觉模型异步生成相册名称
//	@Tags			相册管理
//	@Accept			json
//	@Produce		json
//	@Param			request	body		object{ids=[]int64}						true	"相册ID列表"
//	@Success		200		{object}	utils.Response{data=object{added=int}}	"任务已添加到队列"
//	@Failure		400		{object}	utils.Response						"无效的参数"
//	@Failure		500		{object}	utils.Response						"添加失败"
//	@Router			/api/albums/ai-naming [post]
func (h *AlbumHandler) AINaming(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的参数")
		return
	}

	if len(req.IDs) == 0 {
		utils.BadRequest(c, "请选择要命名的相册")
		return
	}

	added, err := h.aiService.QueueAlbumNaming(c.Request.Context(), req.IDs)
	if err != nil {
		logger.Error("AI 命名相册失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "命名任务已添加到队列", gin.H{"added": added})
}

// GetImages 获取相册内图片
//
//	@Summary		获取相册内图片
//	@Description	分页获取相册内的图片列表
//	@Tags			相册管理
//	@Produce		json
//	@Param			id			path		int															true	"相册ID"
//	@Param			page		query		int															false	"页码"	default(1)
//	@Param			page_size	query		int															false	"每页数量"	default(20)
//	@Success		200			{object}	utils.Response{data=utils.PageData{list=model.ImageVO}}	"图片列表"
//	@Failure		400			{object}	utils.Response												"无效的相册ID"
//	@Failure		500			{object}	utils.Response												"获取失败"
//	@Router			/api/albums/{id}/images [get]
func (h *AlbumHandler) GetImages(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的相册ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	images, total, err := h.albumService.GetImages(c.Request.Context(), id, page, pageSize)
	if err != nil {
		logger.Error("获取相册图片失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.PageResponse(c, images, total, page, pageSize)
}

// AddImages 添加图片到相册
//
//	@Summary		添加图片到相册
//	@Description	将图片添加到指定相册
//	@Tags			相册管理
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int											true	"相册ID"
//	@Param			request	body		object{image_ids=[]int64}					true	"图片ID列表"
//	@Success		200		{object}	utils.Response								"添加成功"
//	@Failure		400		{object}	utils.Response								"无效的参数"
//	@Failure		500		{object}	utils.Response								"添加失败"
//	@Router			/api/albums/{id}/images [post]
func (h *AlbumHandler) AddImages(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的相册ID")
		return
	}

	var req struct {
		ImageIDs []int64 `json:"image_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的参数")
		return
	}

	if err := h.albumService.AddImages(c.Request.Context(), id, req.ImageIDs); err != nil {
		logger.Error("添加图片到相册失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "添加成功", nil)
}

// RemoveImages 从相册移除图片
//
//	@Summary		从相册移除图片
//	@Description	从指定相册移除图片
//	@Tags			相册管理
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int											true	"相册ID"
//	@Param			request	body		object{image_ids=[]int64}					true	"图片ID列表"
//	@Success		200		{object}	utils.Response								"移除成功"
//	@Failure		400		{object}	utils.Response								"无效的参数"
//	@Failure		500		{object}	utils.Response								"移除失败"
//	@Router			/api/albums/{id}/images [delete]
func (h *AlbumHandler) RemoveImages(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的相册ID")
		return
	}

	var req struct {
		ImageIDs []int64 `json:"image_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的参数")
		return
	}

	if err := h.albumService.RemoveImages(c.Request.Context(), id, req.ImageIDs); err != nil {
		logger.Error("从相册移除图片失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "移除成功", nil)
}

// SetCover 设置相册封面
//
//	@Summary		设置相册封面
//	@Description	设置相册的封面图片
//	@Tags			相册管理
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int											true	"相册ID"
//	@Param			request	body		object{image_id=int64}						true	"封面图片ID"
//	@Success		200		{object}	utils.Response								"设置成功"
//	@Failure		400		{object}	utils.Response								"无效的参数"
//	@Failure		500		{object}	utils.Response								"设置失败"
//	@Router			/api/albums/{id}/cover [put]
func (h *AlbumHandler) SetCover(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的相册ID")
		return
	}

	var req struct {
		ImageID int64 `json:"image_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的参数")
		return
	}

	if err := h.albumService.SetCover(c.Request.Context(), id, req.ImageID); err != nil {
		logger.Error("设置相册封面失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "设置成功", nil)
}

// RemoveCover 移除相册封面
//
//	@Summary		移除相册封面
//	@Description	移除相册的自定义封面，恢复使用美学评分最高的图片作为封面
//	@Tags			相册管理
//	@Produce		json
//	@Param			id	path		int				true	"相册ID"
//	@Success		200	{object}	utils.Response	"移除成功"
//	@Failure		400	{object}	utils.Response	"无效的相册ID"
//	@Failure		500	{object}	utils.Response	"移除失败"
//	@Router			/api/albums/{id}/cover [delete]
func (h *AlbumHandler) RemoveCover(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的相册ID")
		return
	}

	if err := h.albumService.RemoveCover(c.Request.Context(), id); err != nil {
		logger.Error("移除相册封面失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "移除成功", nil)
}

// SetAverageCover 设置平均向量封面
//
//	@Summary		设置平均向量封面
//	@Description	计算相册中所有图片的平均向量，选择最接近平均向量的图片作为封面
//	@Tags			相册管理
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int								true	"相册ID"
//	@Param			request	body		object{model_name=string}		true	"模型名称"
//	@Success		200		{object}	utils.Response					"设置成功"
//	@Failure		400		{object}	utils.Response					"无效的参数"
//	@Failure		500		{object}	utils.Response					"设置失败"
//	@Router			/api/albums/{id}/cover/average [put]
func (h *AlbumHandler) SetAverageCover(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的相册ID")
		return
	}

	var req struct {
		ModelName string `json:"model_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的参数: "+err.Error())
		return
	}

	if err := h.albumService.SetAverageCover(c.Request.Context(), id, req.ModelName); err != nil {
		logger.Error("设置平均向量封面失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "设置成功", nil)
}
