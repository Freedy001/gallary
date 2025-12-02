package service

import (
	"context"
	"fmt"
	"gallary/server/internal"
	"gallary/server/internal/model"
	"gallary/server/internal/repository"
	"gallary/server/internal/storage"
	"gallary/server/pkg/logger"
	"slices"
	"strconv"

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
	UserId     string `json:"user_id"`
	NickName   string `json:"nick_name,omitempty"`
	Avatar     string `json:"avatar,omitempty"`
}

// PasswordUpdateDTO 密码更新 DTO
type PasswordUpdateDTO struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// SettingService 设置服务接口
type SettingService interface {
	GetSettingsByCategory(ctx context.Context, category string) (model.SettingPO, error)

	// 更新设置
	UpdatePassword(ctx context.Context, dto *PasswordUpdateDTO) error
	UpdateStorageConfig(ctx context.Context, dto model.StorageItem) (*StorageUpdateResult, error)
	UpdateCleanupConfig(ctx context.Context, dto *model.CleanupPO) error

	// 存储配置 CRUD
	AddStorageConfig(ctx context.Context, storageItem model.StorageItem) (*StorageUpdateResult, error)
	DeleteStorageConfig(ctx context.Context, storageId model.StorageId) error
	SetDefaultStorage(ctx context.Context, storageId model.StorageId) error
	UpdateGlobalConfig(ctx context.Context, globalConfig *model.AliyunPanGlobalConfig) error

	// 应用设置到运行时
	ResetStorage(ctx context.Context) (*storage.StorageManager, error)

	GetStorageManager() *storage.StorageManager

	// 初始化默认设置（从 config.yaml 迁移）
	InitializeDefaults(ctx context.Context) error

	// 检查密码是否已设置
	GetPassword(ctx context.Context) (string, error)
	GetPasswordVersion(ctx context.Context) (int64, error)

	// 获取阿里云盘用户信息
	GetAliyunPanUserInfo(ctx context.Context) []*AliyunPanUserInfo

	// 设置迁移服务
	SetMigrationService(migrationSvc MigrationService)

	// 检查是否有迁移正在进行
	IsMigrationRunning(ctx context.Context) bool
}

type settingService struct {
	repo             repository.SettingRepository
	cfg              *internal.PlatformConfig
	storageManager   *storage.StorageManager
	migrationService MigrationService
}

// NewSettingService 创建设置服务实例
func NewSettingService(repo repository.SettingRepository, cfg *internal.PlatformConfig) SettingService {
	return &settingService{
		repo: repo,
		cfg:  cfg,
	}
}

func (s *settingService) GetStorageManager() *storage.StorageManager {
	return s.storageManager
}

type StorageConfigDTO struct {
	model.StorageConfigPO
	AliyunpanUser []*AliyunPanUserInfo `json:"aliyunpan_user"`
}

// GetSettingsByCategory 按分类获取设置
func (s *settingService) GetSettingsByCategory(ctx context.Context, category string) (model.SettingPO, error) {
	if category == "auth" {
		return nil, fmt.Errorf("禁止获取认证类设置")
	}
	settings, err := s.repo.GetByCategory(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("获取设置失败: %w", err)
	}

	switch category {
	case model.SettingCategoryStorage:
		return StorageConfigDTO{
			StorageConfigPO: model.ToSettingPO[model.StorageConfigPO](settings),
			AliyunpanUser:   s.GetAliyunPanUserInfo(ctx),
		}, nil
	case model.SettingCategoryCleanup:
		return model.ToSettingPO[model.CleanupPO](settings), nil
	default:
		return nil, fmt.Errorf("未知的设置分类: %s", category)
	}
}

// UpdatePassword 更新密码
func (s *settingService) UpdatePassword(ctx context.Context, dto *PasswordUpdateDTO) error {
	// 检查当前是否有密码
	settings, err := s.repo.GetByCategory(ctx, model.SettingCategoryAuth)
	if err != nil {
		return fmt.Errorf("获取当前密码失败: %w", err)
	}
	po := model.ToSettingPO[model.AuthPO](settings)

	// 如果当前有密码，需要验证旧密码
	if po.Password != "" {
		if dto.OldPassword == "" {
			return fmt.Errorf("请输入当前密码")
		}
		if err := bcrypt.CompareHashAndPassword([]byte(po.Password), []byte(dto.OldPassword)); err != nil {
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

	po.Password = string(hashedPassword)
	po.PasswordVersion = currentVersion + 1

	if err := s.repo.BatchUpsert(ctx, po.ToSettings()); err != nil {
		return fmt.Errorf("保存密码失败: %w", err)
	}

	// 更新运行时配置
	s.cfg.Password = string(hashedPassword)
	s.cfg.PasswordVersion = newVersion

	logger.Info("密码已更新", zap.Int64("password_version", newVersion))
	return nil
}

// UpdateStorageConfig 更新存储配置
func (s *settingService) UpdateStorageConfig(ctx context.Context, storageItem model.StorageItem) (*StorageUpdateResult, error) {
	// 1. 检查是否有迁移正在进行
	if s.IsMigrationRunning(ctx) {
		return nil, fmt.Errorf("迁移正在进行中，请等待完成后再修改配置")
	}

	// 2. 获取当前配置
	storageConfig, err := s.getCurrentStorageConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取当前配置失败: %w", err)
	}

	oldStorageItem := storageConfig.GetStorageConfigById(storageItem.StorageId())

	var oldPath, newPath, oldUrl, newUrl = oldStorageItem.Path(), storageItem.Path(), "", ""

	if newConfig, ok := storageItem.(*model.LocalStorageConfig); ok {
		if oldConfig, ok := oldStorageItem.(*model.LocalStorageConfig); ok {
			oldUrl = oldConfig.URLPrefix
			newUrl = newConfig.URLPrefix
		}
	}

	// 3. 检测是否需要迁移
	if oldPath != newPath {
		logger.Info("检测到存储路径变更，准备进行迁移",
			zap.String("old_path", oldPath),
			zap.String("new_path", newPath))

		task, err := s.migrationService.StartSelfMigration(
			ctx,
			oldPath,
			newPath,
			storageItem.StorageId(),
			func(err error) {
				if err != nil {
					return
				}

				s.cfg.DynamicStaticConfig.Update(newUrl, newPath)

				if err = s.repo.BatchUpsert(ctx, storageItem.ToSettings()); err != nil {
					logger.Error("保存设置结果失败", zap.Error(err))
					return
				}

				if _, err = s.ResetStorage(ctx); err != nil {
					logger.Error("重置存储后端失败", zap.Error(err))
					return
				}
			},
		)

		if err != nil {
			return nil, fmt.Errorf("启动迁移失败: %w", err)
		}

		logger.Info("存储配置更新触发迁移",
			zap.String("type", string(model.StorageTypeLocal)),
			zap.Int64("task_id", task.ID))

		return &StorageUpdateResult{
			NeedsMigration: true,
			TaskID:         task.ID,
			Message:        "配置变更需要迁移文件，迁移任务已启动",
		}, nil
	} else {
		if newUrl != oldUrl {
			s.cfg.DynamicStaticConfig.Update(newUrl, newPath)
		}

		// 5. 无需迁移，直接保存配置
		if err := s.repo.BatchUpsert(ctx, storageItem.ToSettings()); err != nil {
			return nil, fmt.Errorf("保存存储配置失败: %w", err)
		}

		// 6. 应用设置到运行时配置
		if _, err := s.ResetStorage(ctx); err != nil {
			logger.Warn("应用存储配置失败", zap.Error(err))
		}

		logger.Info("存储配置已更新")
	}

	return &StorageUpdateResult{
		NeedsMigration: false,
		Message:        "存储配置更新成功",
	}, nil
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
func (s *settingService) UpdateCleanupConfig(ctx context.Context, dto *model.CleanupPO) error {
	if err := s.repo.BatchUpsert(ctx, dto.ToSettings()); err != nil {
		return fmt.Errorf("保存清理配置失败: %w", err)
	}

	// 更新运行时配置
	s.cfg.TrashAutoDeleteDays = dto.TrashAutoDeleteDays
	logger.Info("清理配置已更新", zap.Int("trash_auto_delete_days", dto.TrashAutoDeleteDays))
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
		model.AuthPO{
			Password:        "",
			PasswordVersion: 0,
		}.ToSettings(),
		model.StorageConfigPO{
			DefaultId: model.StorageTypeLocal,
			LocalConfig: &model.LocalStorageConfig{
				Id:        model.StorageTypeLocal,
				BasePath:  "./storage/images",
				URLPrefix: "/static/images",
			},
		}.ToSettings(),
		model.CleanupPO{
			TrashAutoDeleteDays: 30,
		}.ToSettings(),
	)

	if err := s.repo.BatchUpsert(ctx, settings); err != nil {
		return fmt.Errorf("初始化默认设置失败: %w", err)
	}

	logger.Info("默认设置初始化完成", zap.Int("count", len(settings)))
	return nil
}

// IsPasswordSet 检查密码是否已设置
func (s *settingService) GetPassword(ctx context.Context) (string, error) {
	setting, err := s.repo.GetByCategoryKey(ctx, model.SettingCategoryAuth, "password")
	if err != nil {
		return "false", err
	}
	return setting.Value, nil
}

// GetPasswordVersion 获取密码版本号
func (s *settingService) GetPasswordVersion(ctx context.Context) (int64, error) {
	setting, err := s.repo.GetByCategoryKey(ctx, model.SettingCategoryAuth, "passwordVersion")
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

func (s *settingService) ResetStorage(ctx context.Context) (*storage.StorageManager, error) {
	cfg, err := s.getCurrentStorageConfig(ctx)
	if err != nil {
		return nil, err
	}

	// 切换存储管理器
	if s.storageManager == nil {
		manager := storage.NewStorageManager(cfg)
		s.storageManager = manager
		return manager, nil
	}

	if err := s.storageManager.InitStorage(cfg); err != nil {
		logger.Error("切换存储失败", zap.Error(err))
		return nil, fmt.Errorf("切换存储失败: %w", err)
	}

	return s.storageManager, nil
}

// GetAliyunPanUserInfo 获取阿里云盘用户信息
func (s *settingService) GetAliyunPanUserInfo(_ context.Context) []*AliyunPanUserInfo {
	if s.storageManager == nil {
		return []*AliyunPanUserInfo{}
	}

	aliyunPan := s.storageManager.GetAliyunPanStorage()
	if aliyunPan == nil {
		return []*AliyunPanUserInfo{}
	}

	var userInfos []*AliyunPanUserInfo

	for _, a := range aliyunPan {
		userInfo := a.GetUserInfo()
		if userInfo == nil {
			userInfos = append(userInfos, &AliyunPanUserInfo{IsLoggedIn: false})
			continue
		}

		userInfos = append(userInfos, &AliyunPanUserInfo{
			IsLoggedIn: true,
			UserId:     userInfo.UserId,
			NickName:   userInfo.UserName,
			Avatar:     "", // UserInfo 不包含头像信息
		})
	}

	return userInfos
}

// getCurrentStorageConfig 从数据库获取当前存储配置
func (s *settingService) getCurrentStorageConfig(ctx context.Context) (*model.StorageConfigPO, error) {
	// 重新加载存储相关设置到运行时配置
	settings, err := s.repo.GetByCategory(ctx, model.SettingCategoryStorage)
	if err != nil {
		return nil, err
	}

	cfg := model.ToSettingPO[model.StorageConfigPO](settings)
	return &cfg, nil
}

// AddStorageConfig 添加存储配置
func (s *settingService) AddStorageConfig(ctx context.Context, storageItem model.StorageItem) (*StorageUpdateResult, error) {
	// 1. 获取当前配置
	storageConfig, err := s.getCurrentStorageConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取当前配置失败: %w", err)
	}

	// 2. 检查是否已存在
	existing := storageConfig.GetStorageConfigById(storageItem.StorageId())
	if existing != nil {
		return nil, fmt.Errorf("存储配置已存在: %s", storageItem.StorageId())
	}

	// 3. 添加新配置到数组
	switch item := storageItem.(type) {
	case *model.AliyunPanStorageConfig:
		storageConfig.AliyunpanConfig = append(storageConfig.AliyunpanConfig, item)
	default:
		return nil, fmt.Errorf("不支持添加此类型的存储配置")
	}

	// 4. 保存配置
	if err := s.repo.BatchUpsert(ctx, storageConfig.ToSettings()); err != nil {
		return nil, fmt.Errorf("保存存储配置失败: %w", err)
	}

	// 5. 重新加载存储
	if _, err := s.ResetStorage(ctx); err != nil {
		logger.Warn("重新加载存储失败", zap.Error(err))
	}

	logger.Info("存储配置已添加", zap.String("storageId", string(storageItem.StorageId())))
	return &StorageUpdateResult{
		NeedsMigration: false,
		Message:        "存储配置添加成功",
	}, nil
}

// DeleteStorageConfig 删除存储配置
func (s *settingService) DeleteStorageConfig(ctx context.Context, storageId model.StorageId) error {
	// 1. 禁止删除本地存储
	if storageId == model.StorageTypeLocal {
		return fmt.Errorf("不能删除本地存储配置")
	}

	// 2. 获取当前配置
	storageConfig, err := s.getCurrentStorageConfig(ctx)
	if err != nil {
		return fmt.Errorf("获取当前配置失败: %w", err)
	}

	// 3. 检查是否是默认存储
	if storageConfig.DefaultId == storageId {
		return fmt.Errorf("不能删除当前默认存储，请先切换到其他存储")
	}

	// 4. 从配置中移除
	found := false
	newAliyunpanConfig := make([]*model.AliyunPanStorageConfig, 0)
	for _, cfg := range storageConfig.AliyunpanConfig {
		if cfg.StorageId() == storageId {
			found = true
			continue
		}
		newAliyunpanConfig = append(newAliyunpanConfig, cfg)
	}

	if !found {
		return fmt.Errorf("存储配置不存在: %s", storageId)
	}

	storageConfig.AliyunpanConfig = newAliyunpanConfig

	// 5. 保存配置
	if err := s.repo.BatchUpsert(ctx, storageConfig.ToSettings()); err != nil {
		return fmt.Errorf("保存存储配置失败: %w", err)
	}

	// 6. 重新加载存储
	if _, err := s.ResetStorage(ctx); err != nil {
		logger.Warn("重新加载存储失败", zap.Error(err))
	}

	logger.Info("存储配置已删除", zap.String("storageId", string(storageId)))
	return nil
}

// SetDefaultStorage 设置默认存储
func (s *settingService) SetDefaultStorage(ctx context.Context, storageId model.StorageId) error {
	// 1. 获取当前配置
	storageConfig, err := s.getCurrentStorageConfig(ctx)
	if err != nil {
		return fmt.Errorf("获取当前配置失败: %w", err)
	}

	// 2. 验证存储配置存在
	existing := storageConfig.GetStorageConfigById(storageId)
	if existing == nil {
		return fmt.Errorf("存储配置不存在: %s", storageId)
	}

	// 3. 更新默认存储
	storageConfig.DefaultId = storageId

	// 4. 保存配置
	if err := s.repo.BatchUpsert(ctx, storageConfig.ToSettings()); err != nil {
		return fmt.Errorf("保存存储配置失败: %w", err)
	}

	// 5. 重新加载存储
	if _, err := s.ResetStorage(ctx); err != nil {
		logger.Warn("重新加载存储失败", zap.Error(err))
	}

	logger.Info("默认存储已设置", zap.String("storageId", string(storageId)))
	return nil
}

// UpdateGlobalConfig 更新阿里云盘全局配置
func (s *settingService) UpdateGlobalConfig(ctx context.Context, globalConfig *model.AliyunPanGlobalConfig) error {
	// 1. 获取当前配置
	storageConfig, err := s.getCurrentStorageConfig(ctx)
	if err != nil {
		return fmt.Errorf("获取当前配置失败: %w", err)
	}

	// 2. 更新全局配置
	storageConfig.AliyunpanGlobal = globalConfig

	// 3. 保存配置
	if err := s.repo.BatchUpsert(ctx, storageConfig.ToSettings()); err != nil {
		return fmt.Errorf("保存存储配置失败: %w", err)
	}

	// 4. 重新加载存储（让新的全局配置生效）
	if _, err := s.ResetStorage(ctx); err != nil {
		logger.Warn("重新加载存储失败", zap.Error(err))
	}

	logger.Info("阿里云盘全局配置已更新",
		zap.Int64("download_chunk_size", globalConfig.DownloadChunkSize),
		zap.Int("download_concurrency", globalConfig.DownloadConcurrency))
	return nil
}
