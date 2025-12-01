package service

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"gallary/server/config"
	"gallary/server/internal/model"
	"gallary/server/internal/repository"
	"gallary/server/internal/storage"
	"gallary/server/pkg/logger"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// StorageUpdateResult 存储配置更新结果
type StorageUpdateResult struct {
	NeedsMigration bool   `json:"needs_migration"`   // 是否触发了迁移
	TaskID         int64  `json:"task_id,omitempty"` // 迁移任务ID（如果触发了迁移）
	Message        string `json:"message"`           // 结果消息
}

// AliyunPanUserInfo 阿里云盘用户信息
type AliyunPanUserInfo struct {
	IsLoggedIn bool   `json:"is_logged_in"`
	NickName   string `json:"nick_name,omitempty"`
	Avatar     string `json:"avatar,omitempty"`
}

// CleanupConfigDTO 清理配置 DTO
type CleanupConfigDTO struct {
	TrashAutoDeleteDays int `json:"trash_auto_delete_days" binding:"min=0,max=365"`
}

// PasswordUpdateDTO 密码更新 DTO
type PasswordUpdateDTO struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// SettingService 设置服务接口
type SettingService interface {
	GetSettingsByCategory(ctx context.Context, category string) (map[string]interface{}, error)

	// 更新设置
	UpdatePassword(ctx context.Context, dto *PasswordUpdateDTO) error
	UpdateStorageConfig(ctx context.Context, dto *model.StorageConfigDTO) (*StorageUpdateResult, error)
	UpdateCleanupConfig(ctx context.Context, dto *CleanupConfigDTO) error

	// 应用设置到运行时
	ResetStorage(ctx context.Context) (*storage.StorageManager, error)

	// 初始化默认设置（从 config.yaml 迁移）
	InitializeDefaults(ctx context.Context) error

	// 检查密码是否已设置
	IsPasswordSet(ctx context.Context) (bool, error)

	// 获取密码版本号
	GetPasswordVersion(ctx context.Context) (int64, error)

	// 获取阿里云盘用户信息
	GetAliyunPanUserInfo(ctx context.Context) *AliyunPanUserInfo

	// 设置存储管理器
	SetStorageManager(manager *storage.StorageManager)

	// 设置迁移服务
	SetMigrationService(migrationSvc MigrationService)

	// 检查是否有迁移正在进行
	IsMigrationRunning(ctx context.Context) bool
}

type settingService struct {
	repo             repository.SettingRepository
	cfg              *config.Config
	storageManager   *storage.StorageManager
	migrationService MigrationService
}

// NewSettingService 创建设置服务实例
func NewSettingService(repo repository.SettingRepository, cfg *config.Config) SettingService {
	return &settingService{
		repo: repo,
		cfg:  cfg,
	}
}

// GetAllSettings 获取所有设置
func (s *settingService) GetAllSettings(ctx context.Context) (map[string]interface{}, error) {
	settings, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取设置失败: %w", err)
	}

	result := make(map[string]interface{})
	for _, setting := range settings {
		// 不返回密码
		if setting.Key == model.SettingKeyAdminPassword {
			continue
		}
		result[setting.Key] = s.convertValue(setting.Value, setting.ValueType)
	}

	return result, nil
}

// GetSettingsByCategory 按分类获取设置
func (s *settingService) GetSettingsByCategory(ctx context.Context, category string) (map[string]interface{}, error) {
	if category == "auth" {
		return nil, fmt.Errorf("禁止获取认证类设置")
	}
	settings, err := s.repo.GetByCategory(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("获取设置失败: %w", err)
	}

	result := make(map[string]interface{})
	for _, setting := range settings {
		result[setting.Key] = s.convertValue(setting.Value, setting.ValueType)
	}

	// 如果是 storage 分类，附带阿里云盘用户信息
	if category == model.SettingCategoryStorage {
		result["aliyunpan_user"] = s.GetAliyunPanUserInfo(ctx)
	}

	return result, nil
}

// GetSettingValue 获取单个设置值
func (s *settingService) GetSettingValue(ctx context.Context, key string) (string, error) {
	setting, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return "", err
	}
	if setting == nil {
		return "", nil
	}
	return setting.Value, nil
}

// UpdatePassword 更新密码
func (s *settingService) UpdatePassword(ctx context.Context, dto *PasswordUpdateDTO) error {
	// 检查当前是否有密码
	currentSetting, err := s.repo.GetByKey(ctx, model.SettingKeyAdminPassword)
	if err != nil {
		return fmt.Errorf("获取当前密码失败: %w", err)
	}

	// 如果当前有密码，需要验证旧密码
	if currentSetting != nil && currentSetting.Value != "" {
		if dto.OldPassword == "" {
			return fmt.Errorf("请输入当前密码")
		}
		if err := bcrypt.CompareHashAndPassword([]byte(currentSetting.Value), []byte(dto.OldPassword)); err != nil {
			return fmt.Errorf("当前密码错误")
		}
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(dto.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	// 获取当前密码版本号并递增
	currentVersion, _ := s.GetPasswordVersion(ctx)
	newVersion := currentVersion + 1

	now := time.Now()

	// 保存新密码和新版本号
	settings := []*model.Setting{
		{
			Category:  model.SettingCategoryAuth,
			Key:       model.SettingKeyAdminPassword,
			Value:     string(hashedPassword),
			ValueType: model.SettingValueTypeString,
			UpdatedAt: now,
		},
		{
			Category:  model.SettingCategoryAuth,
			Key:       model.SettingKeyPasswordVersion,
			Value:     strconv.FormatInt(newVersion, 10),
			ValueType: model.SettingValueTypeInt,
			UpdatedAt: now,
		},
	}

	if err := s.repo.BatchUpsert(ctx, settings); err != nil {
		return fmt.Errorf("保存密码失败: %w", err)
	}

	// 更新运行时配置
	s.cfg.Admin.Password = string(hashedPassword)
	s.cfg.Admin.PasswordVersion = newVersion

	logger.Info("密码已更新", zap.Int64("password_version", newVersion))
	return nil
}

// UpdateStorageConfig 更新存储配置
func (s *settingService) UpdateStorageConfig(ctx context.Context, dto *model.StorageConfigDTO) (*StorageUpdateResult, error) {
	// 1. 检查是否有迁移正在进行
	if s.IsMigrationRunning(ctx) {
		return nil, fmt.Errorf("迁移正在进行中，请等待完成后再修改配置")
	}

	// 2. 获取当前配置
	currentConfig, err := s.getCurrentStorageConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取当前配置失败: %w", err)
	}

	// 3. 检测是否需要迁移
	needsMigration, migrationType := s.detectMigrationNeeded(currentConfig, dto)

	// 4. 如果需要迁移，启动异步迁移任务
	if needsMigration && s.migrationService != nil {
		var task *model.MigrationTask
		var err error

		//task, err = s.migrationService.StartSelfMigration(
		//	ctx,
		//	currentConfig.LocalBasePath,
		//	dto.LocalBasePath,
		//	migrationType,
		//	func(err error) {
		//		if err != nil {
		//			return
		//		}
		//
		//		if s.storageManager != nil {
		//			newStorageCfg := &config.StorageConfig{
		//				Default: config.StorageTypeLocal,
		//				Local: config.LocalStorageConfig{
		//					BasePath:  newBasePath,
		//					URLPrefix: newURLPrefix,
		//				},
		//			}
		//			if err := s.storageManager.SwitchStorage(newStorageCfg); err != nil {
		//				logger.Warn("重新初始化存储管理器失败", zap.Error(err))
		//			} else {
		//				logger.Info("存储管理器已更新到新路径", zap.String("path", newBasePath))
		//			}
		//		}
		//	},
		//)

		if err != nil {
			return nil, fmt.Errorf("启动迁移失败: %w", err)
		}

		logger.Info("存储配置更新触发迁移",
			zap.String("type", string(migrationType)),
			zap.Int64("task_id", task.ID))

		return &StorageUpdateResult{
			NeedsMigration: true,
			TaskID:         task.ID,
			Message:        "配置变更需要迁移文件，迁移任务已启动",
		}, nil
	}

	// 5. 无需迁移，直接保存配置
	if err := s.repo.BatchUpsert(ctx, model.ToSettings(model.SettingCategoryStorage, dto)); err != nil {
		return nil, fmt.Errorf("保存存储配置失败: %w", err)
	}

	// 6. 应用设置到运行时配置
	if _, err := s.ResetStorage(ctx); err != nil {
		logger.Warn("应用存储配置失败", zap.Error(err))
	}

	logger.Info("存储配置已更新", zap.String("id", string(dto.DefaultId)))
	return &StorageUpdateResult{
		NeedsMigration: false,
		Message:        "存储配置更新成功",
	}, nil
}

// detectMigrationNeeded 检测是否需要迁移
func (s *settingService) detectMigrationNeeded(current, new *model.StorageConfigDTO) (needsMigration bool, migrationType model.StorageId) {
	// 检查本地存储路径变化
	if current.LocalConfig != nil {
		if current.LocalConfig.BasePath != new.LocalConfig.BasePath && current.LocalConfig.BasePath != "" && new.LocalConfig.BasePath != "" {
			return true, model.StorageTypeLocal
		}
	}

	// 检查阿里云盘路径变化
	//if new.DefaultId == "aliyunpan" || current.DefaultId == "aliyunpan" {
	//	if current.AliyunPanBasePath != new.AliyunPanBasePath && new.AliyunPanBasePath != "" && current.AliyunPanBasePath != "" {
	//		return true, config.StorageTypeAliyunpan
	//	}
	//}

	return false, ""
}

// SetMigrationService 设置迁移服务
func (s *settingService) SetMigrationService(migrationSvc MigrationService) {
	s.migrationService = migrationSvc
}

// IsMigrationRunning 检查是否有迁移正在进行
func (s *settingService) IsMigrationRunning(ctx context.Context) bool {
	if s.migrationService == nil {
		return false
	}
	task, err := s.migrationService.GetActiveMigration(ctx)
	if err != nil {
		return false
	}
	return task != nil && task.IsActive()
}

// UpdateCleanupConfig 更新清理配置
func (s *settingService) UpdateCleanupConfig(ctx context.Context, dto *CleanupConfigDTO) error {
	setting := &model.Setting{
		Category:  model.SettingCategoryCleanup,
		Key:       model.SettingKeyTrashAutoDeleteDays,
		Value:     strconv.Itoa(dto.TrashAutoDeleteDays),
		ValueType: model.SettingValueTypeInt,
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Upsert(ctx, setting); err != nil {
		return fmt.Errorf("保存清理配置失败: %w", err)
	}

	// 更新运行时配置
	s.cfg.Trash.AutoDeleteDays = dto.TrashAutoDeleteDays

	logger.Info("清理配置已更新", zap.Int("trash_auto_delete_days", dto.TrashAutoDeleteDays))
	return nil
}

// ApplySettings 从数据库加载设置并应用到运行时
// 注意：数据库中的设置会完全覆盖配置文件中的值
func (s *settingService) ApplySettings(ctx context.Context) error {
	settings, err := s.repo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("获取设置失败: %w", err)
	}

	// 如果数据库中有设置，则完全使用数据库的值覆盖配置文件的值
	// 这确保界面上的设置不会受配置文件影响
	for _, setting := range settings {
		switch setting.Key {
		case model.SettingKeyAdminPassword:
			s.cfg.Admin.Password = setting.Value
		case model.SettingKeyPasswordVersion:
			if version, err := strconv.ParseInt(setting.Value, 10, 64); err == nil {
				s.cfg.Admin.PasswordVersion = version
			}
		case model.SettingKeyTrashAutoDeleteDays:
			if days, err := strconv.Atoi(setting.Value); err == nil {
				s.cfg.Trash.AutoDeleteDays = days
			}
		}
	}

	logger.Info("设置已应用到运行时配置")
	return nil
}

// InitializeDefaults 初始化默认设置到数据库（使用代码默认值，不从配置文件读取）
func (s *settingService) InitializeDefaults(ctx context.Context) error {
	// 检查是否已有设置
	count, err := s.repo.Count(ctx)
	if err != nil {
		return fmt.Errorf("检查设置表失败: %w", err)
	}

	// 如果已有设置，则跳过初始化
	if count > 0 {
		logger.Info("设置表已有数据，跳过初始化")
		return nil
	}

	logger.Info("初始化默认设置...")

	settings := slices.Concat(
		model.ToSettings(model.SettingCategoryAuth, model.AuthDTO{
			Password:        "",
			PasswordVersion: 0,
		}),
		model.ToSettings(model.SettingCategoryStorage, model.StorageConfigDTO{
			DefaultId: model.StorageTypeLocal,
			LocalConfig: &model.LocalStorageConfig{
				Id:        model.StorageTypeLocal,
				BasePath:  "./storage/images",
				URLPrefix: "/static/images",
			},
		}),
		model.ToSettings(model.SettingCategoryCleanup, model.CleanupDTO{
			TrashAutoDeleteDays: 30,
		}),
	)

	if err := s.repo.BatchUpsert(ctx, settings); err != nil {
		return fmt.Errorf("初始化默认设置失败: %w", err)
	}

	logger.Info("默认设置初始化完成", zap.Int("count", len(settings)))
	return nil
}

// IsPasswordSet 检查密码是否已设置
func (s *settingService) IsPasswordSet(ctx context.Context) (bool, error) {
	setting, err := s.repo.GetByKey(ctx, model.SettingKeyAdminPassword)
	if err != nil {
		return false, err
	}
	return setting != nil && setting.Value != "", nil
}

// GetPasswordVersion 获取密码版本号
func (s *settingService) GetPasswordVersion(ctx context.Context) (int64, error) {
	setting, err := s.repo.GetByKey(ctx, model.SettingKeyPasswordVersion)
	if err != nil {
		return 0, err
	}
	if setting == nil || setting.Value == "" {
		return 0, nil
	}
	version, err := strconv.ParseInt(setting.Value, 10, 64)
	if err != nil {
		return 0, nil
	}
	return version, nil
}

// 辅助方法

func (s *settingService) convertValue(value string, valueType string) interface{} {
	switch valueType {
	case model.SettingValueTypeInt:
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
		return 0
	case model.SettingValueTypeBool:
		return value == "true"
	default:
		return value
	}
}

func (s *settingService) ResetStorage(ctx context.Context) (*storage.StorageManager, error) {
	cfg, err := s.getCurrentStorageConfig(ctx)
	if err != nil {
		return nil, err
	}

	// 切换存储管理器
	if s.storageManager == nil {
		manager, err := storage.NewStorageManager(cfg)
		if err != nil {
			return nil, err
		}
		s.storageManager = manager
		return manager, nil
	}

	if err := s.storageManager.SwitchStorage(cfg); err != nil {
		logger.Error("切换存储失败", zap.Error(err))
		return nil, fmt.Errorf("切换存储失败: %w", err)
	}

	return s.storageManager, nil
}

// GetAliyunPanUserInfo 获取阿里云盘用户信息
func (s *settingService) GetAliyunPanUserInfo(ctx context.Context) *AliyunPanUserInfo {
	if s.storageManager == nil {
		return &AliyunPanUserInfo{IsLoggedIn: false}
	}

	aliyunPan := s.storageManager.GetAliyunPanStorage("")
	if aliyunPan == nil {
		return &AliyunPanUserInfo{IsLoggedIn: false}
	}

	userInfo := aliyunPan.GetUserInfo()
	if userInfo == nil {
		return &AliyunPanUserInfo{IsLoggedIn: false}
	}

	return &AliyunPanUserInfo{
		IsLoggedIn: true,
		NickName:   userInfo.UserName,
		Avatar:     "", // UserInfo 不包含头像信息
	}
}

// SetStorageManager 设置存储管理器
func (s *settingService) SetStorageManager(manager *storage.StorageManager) {
	s.storageManager = manager
}

// getCurrentStorageConfig 从数据库获取当前存储配置
func (s *settingService) getCurrentStorageConfig(ctx context.Context) (*model.StorageConfigDTO, error) {
	// 重新加载存储相关设置到运行时配置
	settings, err := s.repo.GetByCategory(ctx, model.SettingCategoryStorage)
	if err != nil {
		return nil, err
	}

	cfg := model.ToSettingDTO[model.StorageConfigDTO](model.SettingCategoryStorage, settings)
	return &cfg, nil
}
