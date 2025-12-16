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
	service service.AlbumService
}

// NewAlbumHandler 创建相册处理器实例
func NewAlbumHandler(service service.AlbumService) *AlbumHandler {
	return &AlbumHandler{service: service}
}

// Create 创建相册
//
//	@Summary		创建相册
//	@Description	创建新相册
//	@Tags			相册管理
//	@Accept			json
//	@Produce		json
//	@Param			request	body		service.CreateAlbumRequest				true	"创建相册请求"
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

	album, err := h.service.Create(c.Request.Context(), &req)
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
//	@Success		200			{object}	utils.Response{data=utils.PageData{list=model.AlbumVO}}	"相册列表"
//	@Failure		500			{object}	utils.Response												"获取失败"
//	@Router			/api/albums [get]
func (h *AlbumHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	albums, total, err := h.service.List(c.Request.Context(), page, pageSize)
	if err != nil {
		logger.Error("获取相册列表失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.PageResponse(c, albums, total, page, pageSize)
}

// GetByID 获取相册详情
//
//	@Summary		获取相册详情
//	@Description	根据ID获取相册详情
//	@Tags			相册管理
//	@Produce		json
//	@Param			id	path		int										true	"相册ID"
//	@Success		200	{object}	utils.Response{data=model.AlbumVO}		"相册详情"
//	@Failure		400	{object}	utils.Response							"无效的相册ID"
//	@Failure		404	{object}	utils.Response							"相册不存在"
//	@Router			/api/albums/{id} [get]
func (h *AlbumHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的相册ID")
		return
	}

	album, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		logger.Error("获取相册详情失败", zap.Error(err))
		utils.NotFound(c, err.Error())
		return
	}

	utils.Success(c, album)
}

// Update 更新相册
//
//	@Summary		更新相册
//	@Description	更新相册信息
//	@Tags			相册管理
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int										true	"相册ID"
//	@Param			request	body		service.UpdateAlbumRequest				true	"更新相册请求"
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

	album, err := h.service.Update(c.Request.Context(), id, &req)
	if err != nil {
		logger.Error("更新相册失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "更新成功", album)
}

// Delete 删除相册
//
//	@Summary		删除相册
//	@Description	根据ID删除相册
//	@Tags			相册管理
//	@Produce		json
//	@Param			id	path		int				true	"相册ID"
//	@Success		200	{object}	utils.Response	"删除成功"
//	@Failure		400	{object}	utils.Response	"无效的相册ID"
//	@Failure		500	{object}	utils.Response	"删除失败"
//	@Router			/api/albums/{id} [delete]
func (h *AlbumHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的相册ID")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		logger.Error("删除相册失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "删除成功", nil)
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

	images, total, err := h.service.GetImages(c.Request.Context(), id, page, pageSize)
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

	if err := h.service.AddImages(c.Request.Context(), id, req.ImageIDs); err != nil {
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

	if err := h.service.RemoveImages(c.Request.Context(), id, req.ImageIDs); err != nil {
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

	if err := h.service.SetCover(c.Request.Context(), id, req.ImageID); err != nil {
		logger.Error("设置相册封面失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "设置成功", nil)
}
