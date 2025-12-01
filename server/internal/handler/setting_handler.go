package handler

import (
	"gallary/server/internal/model"

	"github.com/gin-gonic/gin"

	"gallary/server/internal/service"
	"gallary/server/internal/utils"
)

// SettingHandler 设置处理器
type SettingHandler struct {
	settingService service.SettingService
}

// NewSettingHandler 创建设置处理器实例
func NewSettingHandler(settingService service.SettingService) *SettingHandler {
	return &SettingHandler{settingService: settingService}
}

// GetByCategory 按分类获取设置
//
//	@Summary		按分类获取设置
//	@Description	根据分类获取设置项
//	@Tags			设置
//	@Produce		json
//	@Param			category	path		string			true	"分类名称 (auth, storage, cleanup)"
//	@Success		200			{object}	utils.Response	"设置列表"
//	@Failure		500			{object}	utils.Response	"服务器错误"
//	@Router			/api/settings/{category} [get]
func (h *SettingHandler) GetByCategory(c *gin.Context) {
	category := c.Param("category")
	if category == "" {
		utils.BadRequest(c, "分类名称不能为空")
		return
	}

	settings, err := h.settingService.GetSettingsByCategory(c.Request.Context(), category)
	if err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	utils.Success(c, settings)
}

// UpdatePassword 更新密码
//
//	@Summary		更新管理员密码
//	@Description	更新系统管理员密码
//	@Tags			设置
//	@Accept			json
//	@Produce		json
//	@Param			request	body		service.PasswordUpdateDTO	true	"密码更新信息"
//	@Success		200		{object}	utils.Response				"更新成功"
//	@Failure		400		{object}	utils.Response				"请求参数错误"
//	@Failure		500		{object}	utils.Response				"服务器错误"
//	@Router			/api/settings/password [put]
func (h *SettingHandler) UpdatePassword(c *gin.Context) {
	var req service.PasswordUpdateDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if err := h.settingService.UpdatePassword(c.Request.Context(), &req); err != nil {
		utils.Error(c, 400, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "密码更新成功"})
}

// UpdateStorage 更新存储配置
//
//	@Summary		更新存储配置
//	@Description	更新系统存储配置，如果路径变化会自动触发迁移
//	@Tags			设置
//	@Accept			json
//	@Produce		json
//	@Param			request	body		service.StorageConfigDTO	true	"存储配置"
//	@Success		200		{object}	utils.Response{data=service.StorageUpdateResult}	"更新结果"
//	@Failure		400		{object}	utils.Response				"请求参数错误"
//	@Failure		423		{object}	utils.Response				"迁移进行中，配置被锁定"
//	@Failure		500		{object}	utils.Response				"服务器错误"
//	@Router			/api/settings/storage [put]
func (h *SettingHandler) UpdateStorage(c *gin.Context) {
	var req model.StorageConfigDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	// 检查是否有迁移正在进行
	if h.settingService.IsMigrationRunning(c.Request.Context()) {
		utils.Error(c, 423, "迁移正在进行中，请等待完成后再修改配置")
		return
	}

	result, err := h.settingService.UpdateStorageConfig(c.Request.Context(), &req)
	if err != nil {
		utils.Error(c, 400, err.Error())
		return
	}

	utils.Success(c, result)
}

// UpdateCleanup 更新清理配置
//
//	@Summary		更新清理策略配置
//	@Description	更新系统清理策略配置
//	@Tags			设置
//	@Accept			json
//	@Produce		json
//	@Param			request	body		service.CleanupConfigDTO	true	"清理配置"
//	@Success		200		{object}	utils.Response				"更新成功"
//	@Failure		400		{object}	utils.Response				"请求参数错误"
//	@Failure		500		{object}	utils.Response				"服务器错误"
//	@Router			/api/settings/cleanup [put]
func (h *SettingHandler) UpdateCleanup(c *gin.Context) {
	var req service.CleanupConfigDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if err := h.settingService.UpdateCleanupConfig(c.Request.Context(), &req); err != nil {
		utils.Error(c, 400, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "清理配置更新成功"})
}

// GetPasswordStatus 获取密码状态
//
//	@Summary		获取密码设置状态
//	@Description	检查是否已设置管理员密码
//	@Tags			设置
//	@Produce		json
//	@Success		200	{object}	utils.Response	"密码状态"
//	@Failure		500	{object}	utils.Response	"服务器错误"
//	@Router			/api/settings/password/status [get]
func (h *SettingHandler) GetPasswordStatus(c *gin.Context) {
	isSet, err := h.settingService.IsPasswordSet(c.Request.Context())
	if err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	utils.Success(c, gin.H{"is_set": isSet})
}
