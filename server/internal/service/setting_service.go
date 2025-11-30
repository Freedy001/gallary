package service

import (
	"context"
	"fmt"
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

// StorageConfigDTO 存储配置 DTO
type StorageConfigDTO struct {
	DefaultType string `json:"default_type" binding:"required,oneof=local aliyunpan oss s3 minio"`

	// 本地存储
	LocalBasePath  string `json:"local_base_path,omitempty"`
	LocalURLPrefix string `json:"local_url_prefix,omitempty"`

	// 阿里云盘
	AliyunPanRefreshToken string `json:"aliyunpan_refresh_token,omitempty"`
	AliyunPanBasePath     string `json:"aliyunpan_base_path,omitempty"`
	AliyunPanDriveType    string `json:"aliyunpan_drive_type,omitempty"`

	// OSS
	OSSEndpoint        string `json:"oss_endpoint,omitempty"`
	OSSAccessKeyID     string `json:"oss_access_key_id,omitempty"`
	OSSAccessKeySecret string `json:"oss_access_key_secret,omitempty"`
	OSSBucket          string `json:"oss_bucket,omitempty"`
	OSSURLPrefix       string `json:"oss_url_prefix,omitempty"`

	// S3
	S3Region          string `json:"s3_region,omitempty"`
	S3AccessKeyID     string `json:"s3_access_key_id,omitempty"`
	S3SecretAccessKey string `json:"s3_secret_access_key,omitempty"`
	S3Bucket          string `json:"s3_bucket,omitempty"`
	S3URLPrefix       string `json:"s3_url_prefix,omitempty"`

	// MinIO
	MinIOEndpoint        string `json:"minio_endpoint,omitempty"`
	MinIOAccessKeyID     string `json:"minio_access_key_id,omitempty"`
	MinIOSecretAccessKey string `json:"minio_secret_access_key,omitempty"`
	MinIOBucket          string `json:"minio_bucket,omitempty"`
	MinIOUseSSL          bool   `json:"minio_use_ssl,omitempty"`
	MinIOURLPrefix       string `json:"minio_url_prefix,omitempty"`
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
	// 获取设置
	GetAllSettings(ctx context.Context) (map[string]interface{}, error)
	GetSettingsByCategory(ctx context.Context, category string) (map[string]interface{}, error)
	GetSettingValue(ctx context.Context, key string) (string, error)

	// 更新设置
	UpdatePassword(ctx context.Context, dto *PasswordUpdateDTO) error
	UpdateStorageConfig(ctx context.Context, dto *StorageConfigDTO) error
	UpdateCleanupConfig(ctx context.Context, dto *CleanupConfigDTO) error

	// 应用设置到运行时
	ResetStorage(ctx context.Context) (*storage.StorageManager, error)

	// 初始化默认设置（从 config.yaml 迁移）
	InitializeDefaults(ctx context.Context) error

	// 检查密码是否已设置
	IsPasswordSet(ctx context.Context) (bool, error)

	// 获取密码版本号
	GetPasswordVersion(ctx context.Context) (int64, error)
}

type settingService struct {
	repo           repository.SettingRepository
	cfg            *config.Config
	storageManager *storage.StorageManager
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
	settings := []model.Setting{
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
func (s *settingService) UpdateStorageConfig(ctx context.Context, dto *StorageConfigDTO) error {
	now := time.Now()
	settings := []model.Setting{
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyStorageDefaultType, Value: dto.DefaultType, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyLocalBasePath, Value: dto.LocalBasePath, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyLocalURLPrefix, Value: dto.LocalURLPrefix, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		// 阿里云盘配置
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyAliyunPanBasePath, Value: dto.AliyunPanBasePath, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyAliyunPanDriveType, Value: dto.AliyunPanDriveType, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyAliyunPanRefreshToken, Value: dto.AliyunPanRefreshToken, ValueType: model.SettingValueTypeString, UpdatedAt: now},
	}

	if err := s.repo.BatchUpsert(ctx, settings); err != nil {
		return fmt.Errorf("保存存储配置失败: %w", err)
	}

	// 应用设置到运行时配置
	if _, err := s.ResetStorage(ctx); err != nil {
		logger.Warn("应用存储配置失败", zap.Error(err))
	}

	logger.Info("存储配置已更新", zap.String("type", dto.DefaultType))
	return nil
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

	now := time.Now()
	// 使用代码默认值，不从 config.yaml 读取
	// 这样界面上的设置不会受配置文件的影响
	settings := []model.Setting{
		// 认证设置 - 默认为空（不启用认证）
		{Category: model.SettingCategoryAuth, Key: model.SettingKeyAdminPassword, Value: "", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryAuth, Key: model.SettingKeyPasswordVersion, Value: "0", ValueType: model.SettingValueTypeInt, CreatedAt: now, UpdatedAt: now},

		// 存储设置 - 使用代码默认值
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyStorageDefaultType, Value: "local", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyLocalBasePath, Value: "./storage/images", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyLocalURLPrefix, Value: "/static/images", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		// 阿里云盘默认设置
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyAliyunPanRefreshToken, Value: "", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyAliyunPanBasePath, Value: "/gallery/images", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyAliyunPanDriveType, Value: "file", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		// OSS 默认设置
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyOSSEndpoint, Value: "", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyOSSAccessKeyID, Value: "", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyOSSAccessKeySecret, Value: "", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyOSSBucket, Value: "", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyOSSURLPrefix, Value: "", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyS3Region, Value: "", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyS3AccessKeyID, Value: "", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyS3SecretAccessKey, Value: "", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyS3Bucket, Value: "", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyS3URLPrefix, Value: "", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOEndpoint, Value: "", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOAccessKeyID, Value: "", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOSecretAccessKey, Value: "", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOBucket, Value: "", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOUseSSL, Value: "false", ValueType: model.SettingValueTypeBool, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOURLPrefix, Value: "", ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},

		// 清理设置 - 默认30天
		{Category: model.SettingCategoryCleanup, Key: model.SettingKeyTrashAutoDeleteDays, Value: "30", ValueType: model.SettingValueTypeInt, CreatedAt: now, UpdatedAt: now},
	}

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

func (s *settingService) isSensitiveKey(key string) bool {
	sensitiveKeys := []string{
		model.SettingKeyAliyunPanRefreshToken,
		model.SettingKeyOSSAccessKeySecret,
		model.SettingKeyS3SecretAccessKey,
		model.SettingKeyMinIOSecretAccessKey,
	}
	for _, k := range sensitiveKeys {
		if k == key {
			return true
		}
	}
	return false
}

func (s *settingService) ResetStorage(ctx context.Context) (*storage.StorageManager, error) {
	// 重新加载存储相关设置到运行时配置
	settings, err := s.repo.GetByCategory(ctx, model.SettingCategoryStorage)
	if err != nil {
		return nil, err
	}

	cfg := config.StorageConfig{}
	for _, setting := range settings {
		switch setting.Key {
		case model.SettingKeyStorageDefaultType:
			cfg.Default = config.StorageType(setting.Value)
		case model.SettingKeyLocalBasePath:
			cfg.Local.BasePath = setting.Value
		case model.SettingKeyLocalURLPrefix:
			cfg.Local.URLPrefix = setting.Value
		// 阿里云盘配置
		case model.SettingKeyAliyunPanRefreshToken:
			cfg.AliyunPan.RefreshToken = setting.Value
		case model.SettingKeyAliyunPanBasePath:
			cfg.AliyunPan.BasePath = setting.Value
		case model.SettingKeyAliyunPanDriveType:
			cfg.AliyunPan.DriveType = setting.Value
			// OSS 配置
			//case model.SettingKeyOSSEndpoint:
			//	cfg.OSS.Endpoint = setting.Value
			//case model.SettingKeyOSSAccessKeyID:
			//	cfg.OSS.AccessKeyID = setting.Value
			//case model.SettingKeyOSSAccessKeySecret:
			//	cfg.OSS.AccessKeySecret = setting.Value
			//case model.SettingKeyOSSBucket:
			//	cfg.OSS.Bucket = setting.Value
			//case model.SettingKeyOSSURLPrefix:
			//	cfg.OSS.URLPrefix = setting.Value
			//case model.SettingKeyS3Region:
			//	cfg.S3.Region = setting.Value
			//case model.SettingKeyS3AccessKeyID:
			//	cfg.S3.AccessKeyID = setting.Value
			//case model.SettingKeyS3SecretAccessKey:
			//	cfg.S3.SecretAccessKey = setting.Value
			//case model.SettingKeyS3Bucket:
			//	cfg.S3.Bucket = setting.Value
			//case model.SettingKeyS3URLPrefix:
			//	cfg.S3.URLPrefix = setting.Value
			//case model.SettingKeyMinIOEndpoint:
			//	cfg.MinIO.Endpoint = setting.Value
			//case model.SettingKeyMinIOAccessKeyID:
			//	cfg.MinIO.AccessKeyID = setting.Value
			//case model.SettingKeyMinIOSecretAccessKey:
			//	cfg.MinIO.SecretAccessKey = setting.Value
			//case model.SettingKeyMinIOBucket:
			//	cfg.MinIO.Bucket = setting.Value
			//case model.SettingKeyMinIOUseSSL:
			//	cfg.MinIO.UseSSL = setting.Value == "true"
			//case model.SettingKeyMinIOURLPrefix:
			//	cfg.MinIO.URLPrefix = setting.Value
		}
	}

	// 切换存储管理器
	if s.storageManager == nil {
		manager, err := storage.NewStorageManager(&cfg)
		if err != nil {
			return nil, err
		}
		s.storageManager = manager
		return manager, nil
	}

	if err := s.storageManager.SwitchStorage(&cfg); err != nil {
		logger.Error("切换存储失败", zap.Error(err))
		return nil, fmt.Errorf("切换存储失败: %w", err)
	}

	return s.storageManager, nil
}
