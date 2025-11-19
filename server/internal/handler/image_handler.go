package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gallary/server/internal/repository"
	"gallary/server/internal/service"
	"gallary/server/internal/utils"
	"gallary/server/pkg/logger"
)

// ImageHandler 图片处理器
type ImageHandler struct {
	service service.ImageService
}

// NewImageHandler 创建图片处理器实例
func NewImageHandler(service service.ImageService) *ImageHandler {
	return &ImageHandler{service: service}
}

// Upload 上传图片
//
//	@Summary		上传图片
//	@Description	上传单个图片文件，支持去重
//	@Tags			图片管理
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file								true	"图片文件"
//	@Success		200		{object}	utils.Response{data=model.Image}	"上传成功"
//	@Failure		400		{object}	utils.Response						"请选择要上传的文件"
//	@Failure		500		{object}	utils.Response						"上传失败"
//	@Router			/api/images/upload [post]
func (h *ImageHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		utils.BadRequest(c, "请选择要上传的文件")
		return
	}

	image, err := h.service.Upload(c.Request.Context(), file)
	if err != nil {
		logger.Error("上传图片失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "上传成功", image)
}

// List 获取图片列表
//
//	@Summary		获取图片列表
//	@Description	分页获取图片列表
//	@Tags			图片管理
//	@Produce		json
//	@Param			page		query		int														false	"页码"	default(1)
//	@Param			page_size	query		int														false	"每页数量"	default(20)
//	@Success		200			{object}	utils.Response{data=utils.PageData{list=[]model.Image}}	"图片列表"
//	@Failure		500			{object}	utils.Response											"获取失败"
//	@Router			/api/images [get]
func (h *ImageHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	images, total, err := h.service.List(c.Request.Context(), page, pageSize)
	if err != nil {
		logger.Error("获取图片列表失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.PageResponse(c, images, total, page, pageSize)
}

// GetByID 根据ID获取图片详情
//
//	@Summary		获取图片详情
//	@Description	根据图片ID获取详情
//	@Tags			图片管理
//	@Produce		json
//	@Param			id	path		int									true	"图片ID"
//	@Success		200	{object}	utils.Response{data=model.Image}	"图片详情"
//	@Failure		400	{object}	utils.Response						"无效的图片ID"
//	@Failure		404	{object}	utils.Response						"图片不存在"
//	@Router			/api/images/{id} [get]
func (h *ImageHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的图片ID")
		return
	}

	image, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		logger.Error("获取图片详情失败", zap.Error(err))
		utils.NotFound(c, err.Error())
		return
	}

	utils.Success(c, image)
}

// Delete 删除图片
//
//	@Summary		删除图片
//	@Description	根据ID删除图片
//	@Tags			图片管理
//	@Produce		json
//	@Param			id	path		int				true	"图片ID"
//	@Success		200	{object}	utils.Response	"删除成功"
//	@Failure		400	{object}	utils.Response	"无效的图片ID"
//	@Failure		500	{object}	utils.Response	"删除失败"
//	@Router			/api/images/{id} [delete]
func (h *ImageHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的图片ID")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		logger.Error("删除图片失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "删除成功", nil)
}

// Download 下载图片
//
//	@Summary		下载图片
//	@Description	根据ID下载图片原文件
//	@Tags			图片管理
//	@Produce		application/octet-stream
//	@Param			id	path		int				true	"图片ID"
//	@Success		200	{file}		binary			"图片文件"
//	@Failure		400	{object}	utils.Response	"无效的图片ID"
//	@Failure		404	{object}	utils.Response	"图片不存在"
//	@Router			/api/images/{id}/download [get]
func (h *ImageHandler) Download(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的图片ID")
		return
	}

	reader, filename, err := h.service.Download(c.Request.Context(), id)
	if err != nil {
		logger.Error("下载图片失败", zap.Error(err))
		utils.NotFound(c, err.Error())
		return
	}
	defer reader.Close()

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.DataFromReader(200, -1, "application/octet-stream", reader, nil)
}

// Search 搜索图片
//
//	@Summary		搜索图片
//	@Description	根据多种条件搜索图片
//	@Tags			图片管理
//	@Produce		json
//	@Param			keyword			query		string													false	"关键词"
//	@Param			start_date		query		string													false	"开始日期"
//	@Param			end_date		query		string													false	"结束日期"
//	@Param			location		query		string													false	"地点"
//	@Param			camera_model	query		string													false	"相机型号"
//	@Param			tags			query		string													false	"标签ID列表(逗号分隔)"
//	@Param			page			query		int														false	"页码"	default(1)
//	@Param			page_size		query		int														false	"每页数量"	default(20)
//	@Success		200				{object}	utils.Response{data=utils.PageData{list=[]model.Image}}	"搜索结果"
//	@Failure		500				{object}	utils.Response											"搜索失败"
//	@Router			/api/search [get]
func (h *ImageHandler) Search(c *gin.Context) {
	params := &repository.SearchParams{
		Keyword:      c.Query("keyword"),
		LocationName: c.Query("location"),
		CameraModel:  c.Query("camera_model"),
	}

	if startDate := c.Query("start_date"); startDate != "" {
		params.StartDate = &startDate
	}

	if endDate := c.Query("end_date"); endDate != "" {
		params.EndDate = &endDate
	}

	// 解析标签ID
	if tagsStr := c.Query("tags"); tagsStr != "" {
		// 这里简化处理，实际应该解析逗号分隔的ID列表
	}

	params.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	params.PageSize, _ = strconv.Atoi(c.DefaultQuery("page_size", "20"))

	images, total, err := h.service.Search(c.Request.Context(), params)
	if err != nil {
		logger.Error("搜索图片失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.PageResponse(c, images, total, params.Page, params.PageSize)
}
