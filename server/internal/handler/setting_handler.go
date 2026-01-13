package handler

import (
	"encoding/json"
	"fmt"
	"gallary/server/internal"
	"gallary/server/internal/model"
	"gallary/server/internal/storage"
	"gallary/server/pkg/logger"
	"reflect"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

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
//	@Success		200			{object}	utils.Response{data=model.SettingPO}	"设置列表"
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
//	@Param			request	body		albumService.PasswordUpdateDTO	true	"密码更新信息"
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
//	@Param			request	body		model.StorageConfigPO	true	"存储配置"
//	@Success		200		{object}	utils.Response{data=albumService.StorageUpdateResult}	"更新结果"
//	@Failure		400		{object}	utils.Response				"请求参数错误"
//	@Failure		423		{object}	utils.Response				"迁移进行中，配置被锁定"
//	@Failure		500		{object}	utils.Response				"服务器错误"
//	@Router			/api/settings/storage/{storageId} [put]
func (h *SettingHandler) UpdateStorage(c *gin.Context) {
	id := c.Param("storageId")
	if id == "" {
		utils.BadRequest(c, "请传入storageId")
		return
	}

	var req = model.CreateStorageItemById(model.StorageId(id))
	if req == nil {
		utils.BadRequest(c, "storageId: "+id+"不合法")
		return
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	// 检查是否有迁移正在进行
	if h.settingService.IsMigrationRunning(c.Request.Context()) {
		utils.Error(c, 423, "迁移正在进行中，请等待完成后再修改配置")
		return
	}

	result, err := h.settingService.UpdateStorageConfig(c.Request.Context(), req)
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
//	@Param			request	body		model.CleanupPO		true	"清理配置"
//	@Success		200		{object}	utils.Response				"更新成功"
//	@Failure		400		{object}	utils.Response				"请求参数错误"
//	@Failure		500		{object}	utils.Response				"服务器错误"
//	@Router			/api/settings/cleanup [put]
func (h *SettingHandler) UpdateCleanup(c *gin.Context) {
	var req model.CleanupPO
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
	password, err := h.settingService.GetPassword(c.Request.Context())
	if err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}
	utils.Success(c, gin.H{"is_set": password != ""})
}

// AddStorageRequest 添加存储配置请求
type AddStorageRequest struct {
	Type   string          `json:"type" binding:"required"` // 存储类型: aliyunpan, s3
	Config json.RawMessage `json:"config" binding:"required"`
}

// AddStorage 添加存储配置
//
//	@Summary		添加存储配置
//	@Description	添加新的存储配置（如阿里云盘账号、S3存储）
//	@Tags			设置
//	@Accept			json
//	@Produce		json
//	@Param			request	body		AddStorageRequest	true	"存储配置"
//	@Success		200		{object}	utils.Response{data=albumService.StorageUpdateResult}	"添加结果"
//	@Failure		400		{object}	utils.Response				"请求参数错误"
//	@Failure		500		{object}	utils.Response				"服务器错误"
//	@Router			/api/settings/storage [post]
func (h *SettingHandler) AddStorage(c *gin.Context) {
	var req AddStorageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	var storageItem model.StorageItem
	var err error

	switch req.Type {
	case "aliyunpan":
		var config model.AliyunPanStorageConfig
		if err = json.Unmarshal(req.Config, &config); err != nil {
			utils.BadRequest(c, "阿里云盘配置解析失败: "+err.Error())
			return
		}
		storageItem = &config
	case "s3":
		var config model.S3StorageConfig
		if err = json.Unmarshal(req.Config, &config); err != nil {
			utils.BadRequest(c, "S3配置解析失败: "+err.Error())
			return
		}
		storageItem = &config
	default:
		utils.BadRequest(c, "不支持的存储类型: "+req.Type)
		return
	}

	result, err := h.settingService.AddStorageConfig(c.Request.Context(), storageItem)
	if err != nil {
		utils.Error(c, 400, err.Error())
		return
	}

	utils.Success(c, result)
}

// DeleteStorage 删除存储配置
//
//	@Summary		删除存储配置
//	@Description	删除指定的存储配置
//	@Tags			设置
//	@Produce		json
//	@Param			storageId	path		string	true	"存储ID"
//	@Success		200			{object}	utils.Response	"删除成功"
//	@Failure		400			{object}	utils.Response	"请求参数错误"
//	@Failure		500			{object}	utils.Response	"服务器错误"
//	@Router			/api/settings/storage/{storageId} [delete]
func (h *SettingHandler) DeleteStorage(c *gin.Context) {
	id := c.Param("storageId")
	if id == "" {
		utils.BadRequest(c, "请传入 storageId")
		return
	}

	if err := h.settingService.DeleteStorageConfig(c.Request.Context(), model.StorageId(id)); err != nil {
		utils.Error(c, 400, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "存储配置删除成功"})
}

// SetDefaultStorageRequest 设置默认存储请求
type SetDefaultStorageRequest struct {
	StorageId string `json:"storageId" binding:"required"`
}

// SetDefaultStorage 设置默认存储
//
//	@Summary		设置默认存储
//	@Description	设置系统默认使用的存储
//	@Tags			设置
//	@Accept			json
//	@Produce		json
//	@Param			request	body		SetDefaultStorageRequest	true	"存储ID"
//	@Success		200		{object}	utils.Response	"设置成功"
//	@Failure		400		{object}	utils.Response	"请求参数错误"
//	@Failure		500		{object}	utils.Response	"服务器错误"
//	@Router			/api/settings/storage/default [put]
func (h *SettingHandler) SetDefaultStorage(c *gin.Context) {
	var req SetDefaultStorageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if err := h.settingService.SetDefaultStorage(c.Request.Context(), model.StorageId(req.StorageId)); err != nil {
		utils.Error(c, 400, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "默认存储设置成功"})
}

// SetThumbnailDefaultStorage 设置缩略图默认存储
//
//	@Summary		设置缩略图默认存储
//	@Description	设置缩略图上传的默认存储终端
//	@Tags			设置
//	@Accept			json
//	@Produce		json
//	@Param			request	body		SetDefaultStorageRequest	true	"存储ID"
//	@Success		200		{object}	utils.Response	"设置成功"
//	@Failure		400		{object}	utils.Response	"请求参数错误"
//	@Failure		500		{object}	utils.Response	"服务器错误"
//	@Router			/api/settings/storage/thumbnail/default [put]
func (h *SettingHandler) SetThumbnailDefaultStorage(c *gin.Context) {
	var req SetDefaultStorageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if err := h.settingService.SetThumbnailDefaultStorage(c.Request.Context(), model.StorageId(req.StorageId)); err != nil {
		utils.Error(c, 400, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "缩略图默认存储设置成功"})
}

// UpdateAliyunpanGlobalConfig 更新阿里云盘全局配置
//
//	@Summary		更新阿里云盘全局配置
//	@Description	更新所有阿里云盘账号共享的配置（下载分片大小、并发数等）
//	@Tags			设置
//	@Accept			json
//	@Produce		json
//	@Param			request	body		model.AliyunPanGlobalConfig	true	"全局配置"
//	@Success		200		{object}	utils.Response	"更新成功"
//	@Failure		400		{object}	utils.Response	"请求参数错误"
//	@Failure		500		{object}	utils.Response	"服务器错误"
//	@Router			/api/settings/storage/alyunpan/global [put]
func (h *SettingHandler) UpdateAliyunpanGlobalConfig(c *gin.Context) {
	var req model.AliyunPanGlobalConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if err := h.settingService.UpdateGlobalConfig(c.Request.Context(), &req); err != nil {
		utils.Error(c, 400, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "全局配置更新成功"})
}

// UpdateAI 更新 AI 配置
//
//	@Summary		更新 AI 配置
//	@Description	更新嵌入模型和 LLM 配置
//	@Tags			设置
//	@Accept			json
//	@Produce		json
//	@Param			request	body		model.AIPo		true	"AI 配置"
//	@Success		200		{object}	utils.Response	"更新成功"
//	@Failure		400		{object}	utils.Response	"请求参数错误"
//	@Failure		500		{object}	utils.Response	"服务器错误"
//	@Router			/api/settings/ai [put]
func (h *SettingHandler) UpdateAI(c *gin.Context) {
	var req model.AIPo
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	if err := h.settingService.UpdateAIConfig(c.Request.Context(), &req); err != nil {
		utils.Error(c, 400, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "AI 配置更新成功"})
}

// HasConfigDefaultModel 获取是否配置默认模型
//
//	@Summary	 	检测是否存在默认配置
//	@Tags			设置
//	@Accept			json
//	@Produce		json
//	@Param			type	path		string		true	"AI 配置"
//	@Success		200		{object}	utils.Response	"更新成功"
//	@Failure		400		{object}	utils.Response	"请求参数错误"
//	@Failure		500		{object}	utils.Response	"服务器错误"
//	@Router			/api/settings/configed-default-model [get]
func (h *SettingHandler) HasConfigDefaultModel(c *gin.Context) {
	defer func() {
		if err := recover(); err != nil {
			var errMsg string
			switch v := err.(type) {
			case error:
				errMsg = v.Error()
			case string:
				errMsg = v
			default:
				errMsg = fmt.Sprintf("Unknown error: %v", v)
			}
			utils.Error(c, 500, errMsg)
		}
	}()

	id := c.Param("type")
	if id == "" {
		utils.BadRequest(c, "请传入 type")
		return
	}

	utils.Success(c, reflect.ValueOf(*internal.PlatConfig.GlobalConfig).FieldByName(id).String() != "")
}

// TestS3Connection 测试 S3 连接
//
//	@Summary		测试 S3 连接
//	@Description	使用提供的配置测试 S3 存储连接是否正常
//	@Tags			设置
//	@Accept			json
//	@Produce		json
//	@Param			request	body		model.S3StorageConfig	true	"S3 配置"
//	@Success		200		{object}	utils.Response	"连接成功"
//	@Failure		400		{object}	utils.Response	"连接失败"
//	@Router			/api/settings/storage/s3/test [post]
func (h *SettingHandler) TestS3Connection(c *gin.Context) {
	var config model.S3StorageConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		utils.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	// 打印收到的配置用于调试
	logger.Info("测试 S3 连接，收到配置",
		zap.String("name", config.Name),
		zap.String("endpoint", config.Endpoint),
		zap.String("region", config.Region),
		zap.String("bucket", config.Bucket),
		zap.Bool("useSSL", config.UseSSL),
		zap.Bool("forcePathStyle", config.ForcePathStyle),
	)

	// 创建临时 S3 存储实例进行测试
	testStorage, err := storage.NewS3Storage(&config)
	if err != nil {
		utils.Error(c, 400, "创建 S3 客户端失败: "+err.Error())
		return
	}

	// 尝试列出存储桶内容来验证连接（限制只获取1个对象）
	ctx := c.Request.Context()
	exists, err := testStorage.TestConnection(ctx)
	if err != nil {
		utils.Error(c, 400, "连接测试失败: "+err.Error())
		return
	}

	if !exists {
		utils.Error(c, 400, "Bucket 不存在或无访问权限")
		return
	}

	utils.Success(c, gin.H{
		"message": "连接成功",
		"bucket":  config.Bucket,
		"region":  config.Region,
	})
}
