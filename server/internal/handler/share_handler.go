package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gallary/server/internal/service"
	"gallary/server/internal/utils"
	"gallary/server/pkg/logger"
)

type ShareHandler struct {
	service service.ShareService
}

func NewShareHandler(service service.ShareService) *ShareHandler {
	return &ShareHandler{service: service}
}

// Create 创建分享
//
//	@Summary		创建分享
//	@Description	创建图片分享链接
//	@Tags			分享管理
//	@Accept			json
//	@Produce		json
//	@Param			request	body		service.CreateShareRequest			true	"创建分享请求"
//	@Success		200		{object}	utils.Response{data=model.Share}	"创建成功"
//	@Failure		400		{object}	utils.Response						"无效的参数"
//	@Failure		500		{object}	utils.Response						"创建失败"
//	@Router			/api/shares [post]
func (h *ShareHandler) Create(c *gin.Context) {
	var req service.CreateShareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "无效的参数: "+err.Error())
		return
	}

	share, err := h.service.CreateShare(c.Request.Context(), &req)
	if err != nil {
		logger.Error("创建分享失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "创建成功", share)
}

// List 获取分享列表
//
//	@Summary		获取分享列表
//	@Description	分页获取分享列表
//	@Tags			分享管理
//	@Produce		json
//	@Param			page		query		int														false	"页码"	default(1)
//	@Param			page_size	query		int														false	"每页数量"	default(20)
//	@Success		200			{object}	utils.Response{data=utils.PageData{list=model.Share}}	"分享列表"
//	@Failure		500			{object}	utils.Response											"获取失败"
//	@Router			/api/shares [get]
func (h *ShareHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	shares, total, err := h.service.ListShares(c.Request.Context(), page, pageSize)
	if err != nil {
		logger.Error("获取分享列表失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.PageResponse(c, shares, total, page, pageSize)
}

// Delete 删除分享
//
//	@Summary		删除分享
//	@Description	根据ID删除分享
//	@Tags			分享管理
//	@Produce		json
//	@Param			id	path		int				true	"分享ID"
//	@Success		200	{object}	utils.Response	"删除成功"
//	@Failure		400	{object}	utils.Response	"无效的分享ID"
//	@Failure		500	{object}	utils.Response	"删除失败"
//	@Router			/api/shares/{id} [delete]
func (h *ShareHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequest(c, "无效的分享ID")
		return
	}

	if err := h.service.DeleteShare(c.Request.Context(), id); err != nil {
		logger.Error("删除分享失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.SuccessWithMessage(c, "删除成功", nil)
}

// GetPublicInfo 获取分享公开信息（不需要密码）
//
//	@Summary		获取分享公开信息
//	@Description	根据分享码获取公开信息，用于显示分享页面
//	@Tags			公开分享
//	@Produce		json
//	@Param			code	path		string			true	"分享码"
//	@Success		200		{object}	utils.Response	"分享信息"
//	@Failure		404		{object}	utils.Response	"分享不存在"
//	@Router			/api/s/{code}/info [get]
func (h *ShareHandler) GetPublicInfo(c *gin.Context) {
	code := c.Param("code")
	share, err := h.service.GetShareByCode(c.Request.Context(), code)
	if err != nil {
		logger.Error("获取分享信息失败", zap.Error(err))
		utils.NotFound(c, err.Error())
		return
	}

	// 返回有限信息，不包含详细统计和密码
	utils.Success(c, map[string]any{
		"title":        share.Title,
		"description":  share.Description,
		"has_password": share.Password != nil && *share.Password != "",
		"expire_at":    share.ExpireAt,
		"created_at":   share.CreatedAt,
		"share_code":   share.ShareCode,
	})
}

// SharedImages 验证密码并获取内容
//
//	@Summary		验证并获取分享内容
//	@Description	验证分享密码并获取图片列表（支持分页）
//	@Tags			公开分享
//	@Accept			json
//	@Produce		json
//	@Param			code		path		string												true	"分享码"
//	@Param			page		query		int													false	"页码"		default(1)
//	@Param			page_size	query		int													false	"每页数量"	default(20)
//	@Param			request		body		object{password=string}								false	"密码（如需要）"
//	@Success		200			{object}	utils.Response{data=utils.PageData{list=model.Image}}	"分享详情"
//	@Failure		403			{object}	utils.Response										"密码错误或分享已失效"
//	@Router			/api/s/{code}/images [post]
func (h *ShareHandler) SharedImages(c *gin.Context) {
	code := c.Param("code")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var req struct {
		Password string `json:"password"`
	}
	// 允许空body（无密码情况）
	_ = c.ShouldBindJSON(&req)

	result, total, err := h.service.SharedImages(c.Request.Context(), code, req.Password, page, pageSize)
	if err != nil {
		logger.Error("验证分享失败", zap.Error(err))
		// 使用 403 而非 401，避免前端 HTTP 拦截器跳转登录
		utils.Forbidden(c, err.Error())
		return
	}
	utils.PageResponse(c, result, total, page, pageSize)
}
