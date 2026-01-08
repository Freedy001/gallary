package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"gallary/server/internal/model"
	"gallary/server/pkg/database"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

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
//	@Description	上传单个图片文件，支持去重，可选添加到相册
//	@Tags			图片管理
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file		formData	file								true	"图片文件"
//	@Param			album_id	formData	int									false	"相册ID（可选，上传后自动添加到该相册）"
//	@Success		200			{object}	utils.Response{data=model.Image}	"上传成功"
//	@Failure		400			{object}	utils.Response						"请选择要上传的文件"
//	@Failure		500			{object}	utils.Response						"上传失败"
//	@Router			/api/images/upload [post]
func (h *ImageHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		utils.BadRequest(c, "请选择要上传的文件")
		return
	}

	// 获取可选的相册ID
	var albumID *int64
	if albumIDStr := c.PostForm("album_id"); albumIDStr != "" {
		id, err := strconv.ParseInt(albumIDStr, 10, 64)
		if err == nil && id > 0 {
			albumID = &id
		}
	}

	image, err := database.Transaction1[*model.ImageVO](c, func(ctx context.Context) (*model.ImageVO, error) {
		return h.service.Upload(c.Request.Context(), file, albumID)
	})
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

// GetByIDs 批量获取图片
//
//	@Summary		批量获取图片
//	@Description	根据ID列表获取图片信息
//	@Tags			图片管理
//	@Accept			json
//	@Produce		json
//	@Param			request	body		object{ids=[]int64}	true	"图片ID列表"
//	@Success		200		{object}	utils.Response{data=[]model.ImageVO}	"获取成功"
//	@Failure		400		{object}	utils.Response	"请求参数错误"
//	@Failure		500		{object}	utils.Response	"获取失败"
//	@Router			/api/images/batch [post]
func (h *ImageHandler) GetByIDs(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的请求参数")
		return
	}

	images, err := h.service.GetByIDs(c.Request.Context(), req.IDs)
	if err != nil {
		logger.Error("批量获取图片失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.Success(c, images)
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

// BatchDelete 批量删除图片
//
//	@Summary		批量删除图片
//	@Description	根据ID列表批量删除图片
//	@Tags			图片管理
//	@Accept			json
//	@Produce		json
//	@Param			ids	body		[]int64			true	"图片ID列表"
//	@Success		200	{object}	utils.Response	"删除成功"
//	@Failure		400	{object}	utils.Response	"无效的参数"
//	@Failure		500	{object}	utils.Response	"删除失败"
//	@Router			/api/images/batch-delete [post]
func (h *ImageHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的参数")
		return
	}

	if len(req.IDs) == 0 {
		utils.BadRequest(c, "请选择要删除的图片")
		return
	}

	if err := h.service.DeleteBatch(c.Request.Context(), req.IDs); err != nil {
		logger.Error("批量删除图片失败", zap.Error(err))
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
	h.doDownload(c, id, true)
}

// ProxyFile 代理获取图片文件（用于阿里云盘等需要后端代理的存储）
//
//	@Summary		代理获取图片
//	@Description	通过后端代理获取图片文件内容，用于无法直接访问的存储类型
//	@Tags			图片管理
//	@Produce		image/*
//	@Param			id	path		int				true	"图片ID"
//	@Success		200	{file}		binary			"图片文件"
//	@Failure		400	{object}	utils.Response	"无效的图片ID"
//	@Failure		404	{object}	utils.Response	"图片不存在"
//	@Router			/resouse/:hash/file
func (h *ImageHandler) ProxyFile(c *gin.Context) {
	hash := c.Param("hash")
	if hash == "" {
		utils.BadRequest(c, "未提供文件信息")
		return
	}
	h.doDownload(c, hash, true)
}

func (h *ImageHandler) doDownload(c *gin.Context, idOrHash any, downloadHeader bool) {
	var image *model.Image
	var err error
	if hash, ok := idOrHash.(string); ok {
		image, err = h.service.Repo().FindByHash(c, hash)
	} else if id, ok := idOrHash.(int64); ok {
		image, err = h.service.Repo().FindByID(c, id)
	}

	if image == nil {
		if err != nil {
			utils.NotFound(c, err.Error())
		} else {
			utils.NotFound(c, "无法找到图片")
		}
		return
	}

	// 使用文件哈希作为 ETag
	etag := fmt.Sprintf(`"%s"`, image.FileHash)

	// 检查客户端缓存是否有效
	if match := c.GetHeader("If-None-Match"); match == etag {
		c.Status(304)
		return
	}

	reader, err := h.service.Download(c.Request.Context(), image)
	if err != nil {
		logger.Error("代理获取图片失败", zap.Error(err))
		utils.NotFound(c, err.Error())
		return
	}
	defer reader.Close()

	// 设置完整的缓存头
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("ETag", etag)
	c.Header("Content-Length", fmt.Sprintf("%d", image.FileSize))

	if downloadHeader {
		c.Header("Content-Disposition", "attachment; filename="+image.OriginalName)
		c.Header("Content-Type", "application/octet-stream")
	} else {
		c.Header("Content-Type", image.MimeType)
	}

	// 手动写入响应，确保流式传输正常工作
	c.Status(200)

	// 使用 io.Copy 手动流式写入
	written, err := io.Copy(c.Writer, reader)
	if err != nil {
		logger.Error("流式写入响应失败",
			zap.Error(err),
			zap.Int64("written", written),
			zap.Int64("expected", image.FileSize))
	}
}

// BatchDownload 批量下载图片
//
//	@Summary		批量下载图片
//	@Description	根据ID列表批量下载图片，打包为ZIP（流式传输）
//	@Tags			图片管理
//	@Accept			json
//	@Produce		application/zip
//	@Param			ids	body		[]int64			true	"图片ID列表"
//	@Success		200	{file}		binary			"ZIP文件"
//	@Failure		400	{object}	utils.Response	"无效的参数"
//	@Failure		500	{object}	utils.Response	"下载失败"
//	@Router			/api/images/batch-download [post]
func (h *ImageHandler) BatchDownload(c *gin.Context) {
	var ids []int64

	// 支持 JSON 和表单两种方式
	contentType := c.ContentType()
	if contentType == "application/json" {
		var req struct {
			IDs []int64 `json:"ids" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			utils.BadRequest(c, "无效的参数")
			return
		}
		ids = req.IDs
	} else {
		// 表单方式：ids 是 JSON 字符串
		idsStr := c.PostForm("ids")
		if idsStr == "" {
			utils.BadRequest(c, "无效的参数")
			return
		}
		if err := json.Unmarshal([]byte(idsStr), &ids); err != nil {
			utils.BadRequest(c, "无效的参数格式")
			return
		}
	}

	if len(ids) == 0 {
		utils.BadRequest(c, "请选择要下载的图片")
		return
	}

	// 生成文件名
	filename := fmt.Sprintf("images_%s.zip", time.Now().Format("20060102_150405"))

	// 设置响应头，开始流式传输
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Transfer-Encoding", "chunked")

	// 直接写入到响应流
	_, err := h.service.DownloadZipped(c.Request.Context(), ids, c.Writer)
	if err != nil {
		logger.Error("批量下载图片失败", zap.Error(err))
		return
	}
}

// Search 搜索图片
//
//	@Summary		搜索图片
//	@Description	根据多种条件搜索图片
//	@Tags			图片管理
//	@Accept			json
//	@Produce		json
//	@Param			request	body		repository.SearchParams	true	"搜索参数"
//	@Success		200		{object}	utils.Response{data=utils.PageData{list=[]model.Image}}	"搜索结果"
//	@Failure		500		{object}	utils.Response											"搜索失败"
//	@Router			/api/search [post]
func (h *ImageHandler) Search(c *gin.Context) {
	var params model.SearchParams

	// 检查 Content-Type 判断是 JSON 还是 multipart
	contentType := c.ContentType()

	if strings.Contains(contentType, "multipart/form-data") {
		// multipart 模式：支持图片上传
		if err := c.ShouldBind(&params); err != nil {
			utils.BadRequest(c, "无效的参数: "+err.Error())
			return
		}
		// 处理上传的图片文件
		file, err := c.FormFile("file")
		if err == nil && file != nil {
			f, openErr := file.Open()
			if openErr == nil {
				defer f.Close()
				params.ImageData, _ = io.ReadAll(f)
			}
		}
	} else {
		// JSON 模式：保持向后兼容
		if err := c.ShouldBindJSON(&params); err != nil {
			utils.BadRequest(c, "无效的参数: "+err.Error())
			return
		}
	}

	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}

	images, total, err := h.service.Search(c.Request.Context(), &params)
	if err != nil {
		logger.Error("搜索图片失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.PageResponse(c, images, total, params.Page, params.PageSize)
}

// BatchUpdateMetadata 批量更新图片元数据
//
//	@Summary		批量更新图片元数据
//	@Description	批量更新多个图片的名称、地理位置、自定义元数据和标签
//	@Tags			图片管理
//	@Accept			json
//	@Produce		json
//	@Param			request	body		albumService.UpdateMetadataRequest	true	"批量元数据更新请求"
//	@Success		200		{object}	utils.Response{data=[]int64}	"更新成功"
//	@Failure		400		{object}	utils.Response					"无效的参数"
//	@Failure		500		{object}	utils.Response					"更新失败"
//	@Router			/api/images/metadata [put]
func (h *ImageHandler) BatchUpdateMetadata(c *gin.Context) {
	var req service.UpdateMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的参数: "+err.Error())
		return
	}

	if len(req.ImageIDs) == 0 {
		utils.BadRequest(c, "请选择要更新的图片")
		return
	}

	imageIds, err := database.Transaction1(c.Request.Context(), func(ctx context.Context) ([]int64, error) {
		return h.service.BatchUpdateMetadata(ctx, &req)
	})
	if err != nil {
		logger.Error("批量更新图片元数据失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "批量更新成功", imageIds)
}

// GetClusters 获取图片聚合数据
//
//	@Summary		获取图片聚合数据
//	@Description	根据视窗范围和缩放级别获取图片聚合数据
//	@Tags			图片管理
//	@Produce		json
//	@Param			min_lat	query		number										true	"最小纬度"
//	@Param			max_lat	query		number										true	"最大纬度"
//	@Param			min_lng	query		number										true	"最小经度"
//	@Param			max_lng	query		number										true	"最大经度"
//	@Param			zoom	query		int											true	"缩放级别"
//	@Success		200		{object}	utils.Response{data=[]model.ClusterResult}	"聚合列表"
//	@Failure		500		{object}	utils.Response								"获取失败"
//	@Router			/api/images/clusters [get]
func (h *ImageHandler) GetClusters(c *gin.Context) {
	minLat, _ := strconv.ParseFloat(c.Query("min_lat"), 64)
	maxLat, _ := strconv.ParseFloat(c.Query("max_lat"), 64)
	minLng, _ := strconv.ParseFloat(c.Query("min_lng"), 64)
	maxLng, _ := strconv.ParseFloat(c.Query("max_lng"), 64)
	zoom, _ := strconv.Atoi(c.Query("zoom"))

	// 简单验证参数
	if minLat == 0 && maxLat == 0 && minLng == 0 && maxLng == 0 {
		// 如果没有传参数，默认返回全球范围
		minLat, maxLat = -90, 90
		minLng, maxLng = -180, 180
	}

	clusters, err := h.service.GetClusters(c.Request.Context(), minLat, maxLat, minLng, maxLng, zoom)
	if err != nil {
		logger.Error("获取聚合数据失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.Success(c, clusters)
}

// GetClusterImages 获取指定聚合组内的图片（分页）
//
//	@Summary		获取聚合组图片
//	@Description	获取指定经纬度范围内的图片列表（分页）
//	@Tags			images
//	@Accept			json
//	@Produce		json
//	@Param			min_lat		query		number	true	"最小纬度"
//	@Param			max_lat		query		number	true	"最大纬度"
//	@Param			min_lng		query		number	true	"最小经度"
//	@Param			max_lng		query		number	true	"最大经度"
//	@Param			page		query		int		false	"页码"	default(1)
//	@Param			page_size	query		int		false	"每页数量"	default(20)
//	@Success		200			{object}	utils.Response{data=object{items=[]model.Image,total=int64,page=int,page_size=int}}
//	@Failure		400			{object}	utils.Response
//	@Failure		500			{object}	utils.Response
//	@Router			/api/images/clusters/images [get]
func (h *ImageHandler) GetClusterImages(c *gin.Context) {
	// 解析参数
	minLat, err := strconv.ParseFloat(c.Query("min_lat"), 64)
	if err != nil {
		utils.Error(c, 400, "无效的最小纬度")
		return
	}

	maxLat, err := strconv.ParseFloat(c.Query("max_lat"), 64)
	if err != nil {
		utils.Error(c, 400, "无效的最大纬度")
		return
	}

	minLng, err := strconv.ParseFloat(c.Query("min_lng"), 64)
	if err != nil {
		utils.Error(c, 400, "无效的最小经度")
		return
	}

	maxLng, err := strconv.ParseFloat(c.Query("max_lng"), 64)
	if err != nil {
		utils.Error(c, 400, "无效的最大经度")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 调用服务层
	images, total, err := h.service.GetClusterImages(c.Request.Context(), minLat, maxLat, minLng, maxLng, page, pageSize)
	if err != nil {
		logger.Error("获取聚合组图片失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.PageResponse(c, images, total, page, pageSize)
}

// GetGeoBounds 获取所有带坐标图片的地理边界
//
//	@Summary		获取图片地理边界
//	@Description	获取所有带有地理坐标的图片的边界范围，用于地图初始化
//	@Tags			图片管理
//	@Produce		json
//	@Success		200	{object}	utils.Response{data=model.GeoBounds}	"地理边界"
//	@Failure		500	{object}	utils.Response							"获取失败"
//	@Router			/api/images/geo-bounds [get]
func (h *ImageHandler) GetGeoBounds(c *gin.Context) {
	bounds, err := h.service.GetGeoBounds(c.Request.Context())
	if err != nil {
		logger.Error("获取地理边界失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.Success(c, bounds)
}

// ListDeleted 获取已删除的图片列表
//
//	@Summary		获取已删除图片列表
//	@Description	分页获取回收站中的图片列表
//	@Tags			回收站
//	@Produce		json
//	@Param			page		query		int														false	"页码"	default(1)
//	@Param			page_size	query		int														false	"每页数量"	default(20)
//	@Success		200			{object}	utils.Response{data=utils.PageData{list=[]model.Image}}	"已删除图片列表"
//	@Failure		500			{object}	utils.Response											"获取失败"
//	@Router			/api/images/trash [get]
func (h *ImageHandler) ListDeleted(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	images, total, err := h.service.ListDeleted(c.Request.Context(), page, pageSize)
	if err != nil {
		logger.Error("获取已删除图片列表失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.PageResponse(c, images, total, page, pageSize)
}

// RestoreImages 恢复已删除的图片
//
//	@Summary		恢复图片
//	@Description	从回收站恢复已删除的图片
//	@Tags			回收站
//	@Accept			json
//	@Produce		json
//	@Param			ids	body		[]int64			true	"图片ID列表"
//	@Success		200	{object}	utils.Response	"恢复成功"
//	@Failure		400	{object}	utils.Response	"无效的参数"
//	@Failure		500	{object}	utils.Response	"恢复失败"
//	@Router			/api/images/trash/restore [post]
func (h *ImageHandler) RestoreImages(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的参数")
		return
	}

	if len(req.IDs) == 0 {
		utils.BadRequest(c, "请选择要恢复的图片")
		return
	}

	if err := h.service.RestoreImages(c.Request.Context(), req.IDs); err != nil {
		logger.Error("恢复图片失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "恢复成功", nil)
}

// PermanentlyDelete 彻底删除图片
//
//	@Summary		彻底删除图片
//	@Description	从回收站彻底删除图片（包括物理文件）
//	@Tags			回收站
//	@Accept			json
//	@Produce		json
//	@Param			ids	body		[]int64			true	"图片ID列表"
//	@Success		200	{object}	utils.Response	"删除成功"
//	@Failure		400	{object}	utils.Response	"无效的参数"
//	@Failure		500	{object}	utils.Response	"删除失败"
//	@Router			/api/images/trash/delete [post]
func (h *ImageHandler) PermanentlyDelete(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的参数")
		return
	}

	if len(req.IDs) == 0 {
		utils.BadRequest(c, "请选择要删除的图片")
		return
	}

	if err := h.service.PermanentlyDelete(c.Request.Context(), req.IDs); err != nil {
		logger.Error("彻底删除图片失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "彻底删除成功", nil)
}

// GetTags 获取标签列表
//
//	@Summary		获取标签列表
//	@Description	获取标签列表，支持关键字搜索。无关键字时返回热门标签（默认10个）
//	@Tags			标签管理
//	@Produce		json
//	@Param			keyword	query		string						false	"搜索关键字"
//	@Param			limit	query		int							false	"返回数量限制"	default(10)
//	@Success		200		{object}	utils.Response{data=[]model.Tag}	"标签列表"
//	@Failure		500		{object}	utils.Response						"获取失败"
//	@Router			/api/tags [get]
func (h *ImageHandler) GetTags(c *gin.Context) {
	keyword := c.Query("keyword")
	limit := 10 // 默认返回10个标签

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	var tags []*model.Tag
	var err error

	if keyword != "" {
		// 有关键字时进行搜索
		tags, err = h.service.SearchTags(c.Request.Context(), keyword, limit)
	} else {
		// 无关键字时返回热门标签
		tags, err = h.service.GetPopularTags(c.Request.Context(), limit)
	}

	if err != nil {
		logger.Error("获取标签列表失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.Success(c, tags)
}
