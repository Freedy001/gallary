package handler

import (
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gallary/server/internal/service"
	"gallary/server/internal/storage"
	"gallary/server/internal/utils"
	"gallary/server/pkg/logger"
)

// StorageHandler 存储处理器
type StorageHandler struct {
	storage        storage.Storage
	storageManager *storage.StorageManager
	settingService service.SettingService

	// 阿里云盘登录管理
	aliyunPanLogin *storage.AliyunPanLogin
	loginMu        sync.Mutex
}

// NewStorageHandler 创建存储处理器实例
func NewStorageHandler(storage storage.Storage) *StorageHandler {
	return &StorageHandler{storage: storage}
}

// SetStorageManager 设置存储管理器
func (h *StorageHandler) SetStorageManager(manager *storage.StorageManager) {
	h.storageManager = manager
}

// SetSettingService 设置配置服务
func (h *StorageHandler) SetSettingService(settingService service.SettingService) {
	h.settingService = settingService
}

// GetStats 获取存储统计信息
//
//	@Summary		获取存储统计
//	@Description	获取所有存储提供者的空间使用情况
//	@Tags			存储管理
//	@Produce		json
//	@Success		200	{object}	utils.Response{data=storage.MultiStorageStats}	"存储统计信息"
//	@Failure		500	{object}	utils.Response								"获取失败"
//	@Router			/api/storage/stats [get]
func (h *StorageHandler) GetStats(c *gin.Context) {
	if h.storageManager != nil {
		stats := h.storageManager.GetMultiStorageStats(c.Request.Context())
		utils.Success(c, stats)
		return
	}

	// 回退到单个存储的统计
	stats, err := h.storage.GetStats(c.Request.Context())
	if err != nil {
		logger.Error("获取存储统计失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.Success(c, stats)
}

// AliyunPanQRCodeResponse 阿里云盘二维码响应
type AliyunPanQRCodeResponse struct {
	QRCodeURL string `json:"qr_code_url"` // 二维码内容（用于生成二维码图片）
	Status    string `json:"status"`      // 状态
	Message   string `json:"message"`     // 消息
}

// AliyunPanLoginResponse 阿里云盘登录响应
type AliyunPanLoginResponse struct {
	Status       string `json:"status"`        // 状态
	Message      string `json:"message"`       // 消息
	RefreshToken string `json:"refresh_token"` // 刷新令牌（登录成功后）
	UserName     string `json:"user_name"`     // 用户名
	NickName     string `json:"nick_name"`     // 昵称
	Avatar       string `json:"avatar"`        // 头像URL
}

// GenerateAliyunPanQRCode 生成阿里云盘登录二维码
//
//	@Summary		生成阿里云盘登录二维码
//	@Description	生成用于扫码登录阿里云盘的二维码
//	@Tags			存储管理
//	@Produce		json
//	@Success		200	{object}	utils.Response{data=AliyunPanQRCodeResponse}	"二维码信息"
//	@Failure		500	{object}	utils.Response									"生成失败"
//	@Router			/api/storage/aliyunpan/qrcode [post]
func (h *StorageHandler) GenerateAliyunPanQRCode(c *gin.Context) {
	h.loginMu.Lock()
	defer h.loginMu.Unlock()

	// 创建新的登录管理器
	login, err := storage.NewAliyunPanLogin()
	if err != nil {
		logger.Error("创建阿里云盘登录管理器失败", zap.Error(err))
		utils.Error(c, 500, "创建登录管理器失败")
		return
	}

	// 生成二维码
	result, err := login.GenerateQRCode()
	if err != nil {
		logger.Error("生成阿里云盘二维码失败", zap.Error(err))
		utils.Error(c, 500, "生成二维码失败: "+err.Error())
		return
	}

	// 保存登录管理器
	h.aliyunPanLogin = login

	utils.Success(c, AliyunPanQRCodeResponse{
		QRCodeURL: result.QRCodeURL,
		Status:    string(result.Status),
		Message:   result.Message,
	})
}

// CheckAliyunPanQRCodeStatus 检查阿里云盘二维码扫描状态
//
//	@Summary		检查阿里云盘二维码状态
//	@Description	检查用户是否已扫描并确认登录
//	@Tags			存储管理
//	@Produce		json
//	@Success		200	{object}	utils.Response{data=AliyunPanLoginResponse}	"登录状态"
//	@Failure		400	{object}	utils.Response								"请先生成二维码"
//	@Failure		500	{object}	utils.Response								"检查失败"
//	@Router			/api/storage/aliyunpan/qrcode/status [get]
func (h *StorageHandler) CheckAliyunPanQRCodeStatus(c *gin.Context) {
	h.loginMu.Lock()
	defer h.loginMu.Unlock()

	if h.aliyunPanLogin == nil {
		utils.Error(c, 400, "请先生成二维码")
		return
	}

	result, err := h.aliyunPanLogin.CheckQRCodeStatus()
	if err != nil {
		logger.Error("检查阿里云盘二维码状态失败", zap.Error(err))
		utils.Error(c, 500, "检查状态失败: "+err.Error())
		return
	}

	resp := AliyunPanLoginResponse{
		Status:  string(result.Status),
		Message: result.Message,
	}

	// 如果登录成功，返回 refresh_token
	if result.Status == storage.QRCodeStatusConfirmed {
		resp.RefreshToken = result.RefreshToken
		resp.UserName = result.UserName
		resp.NickName = result.NickName
		resp.Avatar = result.Avatar

		// 清理登录管理器
		h.aliyunPanLogin = nil

		logger.Info("阿里云盘登录成功",
			zap.String("user_name", result.UserName),
			zap.String("nick_name", result.NickName))
	}

	utils.Success(c, resp)
}

// LogoutAliyunPan 退出阿里云盘登录
//
//	@Summary		退出阿里云盘登录
//	@Description	清除阿里云盘 refresh_token 并重新初始化存储
//	@Tags			存储管理
//	@Produce		json
//	@Success		200	{object}	utils.Response	"退出成功"
//	@Failure		500	{object}	utils.Response	"退出失败"
//	@Router			/api/storage/aliyunpan/logout [post]
func (h *StorageHandler) LogoutAliyunPan(c *gin.Context) {
	if h.settingService == nil {
		utils.Error(c, 500, "服务未初始化")
		return
	}

	// 重新加载存储配置（会重新初始化存储，由于 refresh_token 需要在前端清除后保存）
	// 这里只是返回成功，实际的 token 清除由前端调用 updateStorage 接口完成
	utils.Success(c, gin.H{"message": "退出成功"})
}
