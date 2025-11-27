package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gallary/server/config"
	"gallary/server/internal/model"
	"gallary/server/internal/repository"
	"gallary/server/pkg/logger"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// StorageConfigDTO 存储配置 DTO
type StorageConfigDTO struct {
	DefaultType string `json:"default_type" binding:"required,oneof=local oss s3 minio"`

	// 本地存储
	LocalBasePath  string `json:"local_base_path,omitempty"`
	LocalURLPrefix string `json:"local_url_prefix,omitempty"`

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
	ApplySettings(ctx context.Context) error

	// 初始化默认设置（从 config.yaml 迁移）
	InitializeDefaults(ctx context.Context) error

	// 检查密码是否已设置
	IsPasswordSet(ctx context.Context) (bool, error)
}

type settingService struct {
	repo repository.SettingRepository
	cfg  *config.Config
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
	settings, err := s.repo.GetByCategory(ctx, category)
	if err != nil {
		return nil, fmt.Errorf("获取设置失败: %w", err)
	}

	result := make(map[string]interface{})
	for _, setting := range settings {
		// 不返回密码
		if setting.Key == model.SettingKeyAdminPassword {
			continue
		}
		// 不返回敏感的密钥信息（只返回是否已设置）
		if s.isSensitiveKey(setting.Key) {
			if setting.Value != "" {
				result[setting.Key] = "******"
			} else {
				result[setting.Key] = ""
			}
			continue
		}
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

	// 保存新密码
	setting := &model.Setting{
		Category:  model.SettingCategoryAuth,
		Key:       model.SettingKeyAdminPassword,
		Value:     string(hashedPassword),
		ValueType: model.SettingValueTypeString,
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Upsert(ctx, setting); err != nil {
		return fmt.Errorf("保存密码失败: %w", err)
	}

	// 更新运行时配置
	s.cfg.Admin.Password = string(hashedPassword)

	logger.Info("密码已更新")
	return nil
}

// UpdateStorageConfig 更新存储配置
func (s *settingService) UpdateStorageConfig(ctx context.Context, dto *StorageConfigDTO) error {
	now := time.Now()
	settings := []model.Setting{
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyStorageDefaultType, Value: dto.DefaultType, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyLocalBasePath, Value: dto.LocalBasePath, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyLocalURLPrefix, Value: dto.LocalURLPrefix, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyOSSEndpoint, Value: dto.OSSEndpoint, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyOSSAccessKeyID, Value: dto.OSSAccessKeyID, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyOSSBucket, Value: dto.OSSBucket, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyOSSURLPrefix, Value: dto.OSSURLPrefix, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyS3Region, Value: dto.S3Region, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyS3AccessKeyID, Value: dto.S3AccessKeyID, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyS3Bucket, Value: dto.S3Bucket, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyS3URLPrefix, Value: dto.S3URLPrefix, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOEndpoint, Value: dto.MinIOEndpoint, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOAccessKeyID, Value: dto.MinIOAccessKeyID, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOBucket, Value: dto.MinIOBucket, ValueType: model.SettingValueTypeString, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOUseSSL, Value: strconv.FormatBool(dto.MinIOUseSSL), ValueType: model.SettingValueTypeBool, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOURLPrefix, Value: dto.MinIOURLPrefix, ValueType: model.SettingValueTypeString, UpdatedAt: now},
	}

	// 处理敏感字段（只有在非空时才更新）
	if dto.OSSAccessKeySecret != "" && dto.OSSAccessKeySecret != "******" {
		settings = append(settings, model.Setting{
			Category: model.SettingCategoryStorage, Key: model.SettingKeyOSSAccessKeySecret,
			Value: dto.OSSAccessKeySecret, ValueType: model.SettingValueTypeString, UpdatedAt: now,
		})
	}
	if dto.S3SecretAccessKey != "" && dto.S3SecretAccessKey != "******" {
		settings = append(settings, model.Setting{
			Category: model.SettingCategoryStorage, Key: model.SettingKeyS3SecretAccessKey,
			Value: dto.S3SecretAccessKey, ValueType: model.SettingValueTypeString, UpdatedAt: now,
		})
	}
	if dto.MinIOSecretAccessKey != "" && dto.MinIOSecretAccessKey != "******" {
		settings = append(settings, model.Setting{
			Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOSecretAccessKey,
			Value: dto.MinIOSecretAccessKey, ValueType: model.SettingValueTypeString, UpdatedAt: now,
		})
	}

	if err := s.repo.BatchUpsert(ctx, settings); err != nil {
		return fmt.Errorf("保存存储配置失败: %w", err)
	}

	// 应用设置到运行时
	if err := s.applyStorageSettings(ctx); err != nil {
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
func (s *settingService) ApplySettings(ctx context.Context) error {
	settings, err := s.repo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("获取设置失败: %w", err)
	}

	for _, setting := range settings {
		switch setting.Key {
		case model.SettingKeyAdminPassword:
			s.cfg.Admin.Password = setting.Value
		case model.SettingKeyStorageDefaultType:
			s.cfg.Storage.Default = setting.Value
		case model.SettingKeyLocalBasePath:
			if setting.Value != "" {
				s.cfg.Storage.Local.BasePath = setting.Value
			}
		case model.SettingKeyLocalURLPrefix:
			if setting.Value != "" {
				s.cfg.Storage.Local.URLPrefix = setting.Value
			}
		case model.SettingKeyOSSEndpoint:
			s.cfg.Storage.OSS.Endpoint = setting.Value
		case model.SettingKeyOSSAccessKeyID:
			s.cfg.Storage.OSS.AccessKeyID = setting.Value
		case model.SettingKeyOSSAccessKeySecret:
			s.cfg.Storage.OSS.AccessKeySecret = setting.Value
		case model.SettingKeyOSSBucket:
			s.cfg.Storage.OSS.Bucket = setting.Value
		case model.SettingKeyOSSURLPrefix:
			s.cfg.Storage.OSS.URLPrefix = setting.Value
		case model.SettingKeyS3Region:
			s.cfg.Storage.S3.Region = setting.Value
		case model.SettingKeyS3AccessKeyID:
			s.cfg.Storage.S3.AccessKeyID = setting.Value
		case model.SettingKeyS3SecretAccessKey:
			s.cfg.Storage.S3.SecretAccessKey = setting.Value
		case model.SettingKeyS3Bucket:
			s.cfg.Storage.S3.Bucket = setting.Value
		case model.SettingKeyS3URLPrefix:
			s.cfg.Storage.S3.URLPrefix = setting.Value
		case model.SettingKeyMinIOEndpoint:
			s.cfg.Storage.MinIO.Endpoint = setting.Value
		case model.SettingKeyMinIOAccessKeyID:
			s.cfg.Storage.MinIO.AccessKeyID = setting.Value
		case model.SettingKeyMinIOSecretAccessKey:
			s.cfg.Storage.MinIO.SecretAccessKey = setting.Value
		case model.SettingKeyMinIOBucket:
			s.cfg.Storage.MinIO.Bucket = setting.Value
		case model.SettingKeyMinIOUseSSL:
			s.cfg.Storage.MinIO.UseSSL = setting.Value == "true"
		case model.SettingKeyMinIOURLPrefix:
			s.cfg.Storage.MinIO.URLPrefix = setting.Value
		case model.SettingKeyTrashAutoDeleteDays:
			if days, err := strconv.Atoi(setting.Value); err == nil {
				s.cfg.Trash.AutoDeleteDays = days
			}
		}
	}

	logger.Info("设置已应用到运行时配置")
	return nil
}

// InitializeDefaults 从 config.yaml 初始化默认设置到数据库
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
	settings := []model.Setting{
		// 认证设置
		{Category: model.SettingCategoryAuth, Key: model.SettingKeyAdminPassword, Value: s.cfg.Admin.Password, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},

		// 存储设置
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyStorageDefaultType, Value: s.cfg.Storage.Default, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyLocalBasePath, Value: s.cfg.Storage.Local.BasePath, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyLocalURLPrefix, Value: s.cfg.Storage.Local.URLPrefix, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyOSSEndpoint, Value: s.cfg.Storage.OSS.Endpoint, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyOSSAccessKeyID, Value: s.cfg.Storage.OSS.AccessKeyID, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyOSSAccessKeySecret, Value: s.cfg.Storage.OSS.AccessKeySecret, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyOSSBucket, Value: s.cfg.Storage.OSS.Bucket, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyOSSURLPrefix, Value: s.cfg.Storage.OSS.URLPrefix, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyS3Region, Value: s.cfg.Storage.S3.Region, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyS3AccessKeyID, Value: s.cfg.Storage.S3.AccessKeyID, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyS3SecretAccessKey, Value: s.cfg.Storage.S3.SecretAccessKey, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyS3Bucket, Value: s.cfg.Storage.S3.Bucket, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyS3URLPrefix, Value: s.cfg.Storage.S3.URLPrefix, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOEndpoint, Value: s.cfg.Storage.MinIO.Endpoint, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOAccessKeyID, Value: s.cfg.Storage.MinIO.AccessKeyID, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOSecretAccessKey, Value: s.cfg.Storage.MinIO.SecretAccessKey, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOBucket, Value: s.cfg.Storage.MinIO.Bucket, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOUseSSL, Value: strconv.FormatBool(s.cfg.Storage.MinIO.UseSSL), ValueType: model.SettingValueTypeBool, CreatedAt: now, UpdatedAt: now},
		{Category: model.SettingCategoryStorage, Key: model.SettingKeyMinIOURLPrefix, Value: s.cfg.Storage.MinIO.URLPrefix, ValueType: model.SettingValueTypeString, CreatedAt: now, UpdatedAt: now},

		// 清理设置
		{Category: model.SettingCategoryCleanup, Key: model.SettingKeyTrashAutoDeleteDays, Value: strconv.Itoa(s.cfg.Trash.AutoDeleteDays), ValueType: model.SettingValueTypeInt, CreatedAt: now, UpdatedAt: now},
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

func (s *settingService) applyStorageSettings(ctx context.Context) error {
	// 重新加载存储相关设置到运行时配置
	settings, err := s.repo.GetByCategory(ctx, model.SettingCategoryStorage)
	if err != nil {
		return err
	}

	for _, setting := range settings {
		switch setting.Key {
		case model.SettingKeyStorageDefaultType:
			s.cfg.Storage.Default = setting.Value
		case model.SettingKeyLocalBasePath:
			if setting.Value != "" {
				s.cfg.Storage.Local.BasePath = setting.Value
			}
		case model.SettingKeyLocalURLPrefix:
			if setting.Value != "" {
				s.cfg.Storage.Local.URLPrefix = setting.Value
			}
		case model.SettingKeyOSSEndpoint:
			s.cfg.Storage.OSS.Endpoint = setting.Value
		case model.SettingKeyOSSAccessKeyID:
			s.cfg.Storage.OSS.AccessKeyID = setting.Value
		case model.SettingKeyOSSAccessKeySecret:
			s.cfg.Storage.OSS.AccessKeySecret = setting.Value
		case model.SettingKeyOSSBucket:
			s.cfg.Storage.OSS.Bucket = setting.Value
		case model.SettingKeyOSSURLPrefix:
			s.cfg.Storage.OSS.URLPrefix = setting.Value
		case model.SettingKeyS3Region:
			s.cfg.Storage.S3.Region = setting.Value
		case model.SettingKeyS3AccessKeyID:
			s.cfg.Storage.S3.AccessKeyID = setting.Value
		case model.SettingKeyS3SecretAccessKey:
			s.cfg.Storage.S3.SecretAccessKey = setting.Value
		case model.SettingKeyS3Bucket:
			s.cfg.Storage.S3.Bucket = setting.Value
		case model.SettingKeyS3URLPrefix:
			s.cfg.Storage.S3.URLPrefix = setting.Value
		case model.SettingKeyMinIOEndpoint:
			s.cfg.Storage.MinIO.Endpoint = setting.Value
		case model.SettingKeyMinIOAccessKeyID:
			s.cfg.Storage.MinIO.AccessKeyID = setting.Value
		case model.SettingKeyMinIOSecretAccessKey:
			s.cfg.Storage.MinIO.SecretAccessKey = setting.Value
		case model.SettingKeyMinIOBucket:
			s.cfg.Storage.MinIO.Bucket = setting.Value
		case model.SettingKeyMinIOUseSSL:
			s.cfg.Storage.MinIO.UseSSL = setting.Value == "true"
		case model.SettingKeyMinIOURLPrefix:
			s.cfg.Storage.MinIO.URLPrefix = setting.Value
		}
	}

	return nil
}
