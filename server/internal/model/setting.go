package model

import (
	"time"
)

// Setting 系统设置模型
type Setting struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Category  string    `gorm:"type:varchar(50);not null;index" json:"category"`   // auth, storage, cleanup
	Key       string    `gorm:"type:varchar(100);not null;uniqueIndex" json:"key"` // 设置键名
	Value     string    `gorm:"type:text" json:"value"`                            // 设置值
	ValueType string    `gorm:"type:varchar(20);default:string" json:"value_type"` // string, int, bool, json
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName 指定表名
func (Setting) TableName() string {
	return "settings"
}

// 设置分类常量
const (
	SettingCategoryAuth    = "auth"
	SettingCategoryStorage = "storage"
	SettingCategoryCleanup = "cleanup"
)

// 设置键名常量
const (
	// 认证相关
	SettingKeyAdminPassword   = "admin_password"
	SettingKeyPasswordVersion = "password_version" // 密码版本号，用于使旧token失效

	// 存储相关
	SettingKeyStorageDefaultType = "storage_default_type"
	SettingKeyLocalBasePath      = "local_base_path"
	SettingKeyLocalURLPrefix     = "local_url_prefix"

	// OSS 相关
	SettingKeyOSSEndpoint        = "oss_endpoint"
	SettingKeyOSSAccessKeyID     = "oss_access_key_id"
	SettingKeyOSSAccessKeySecret = "oss_access_key_secret"
	SettingKeyOSSBucket          = "oss_bucket"
	SettingKeyOSSURLPrefix       = "oss_url_prefix"

	// S3 相关
	SettingKeyS3Region          = "s3_region"
	SettingKeyS3AccessKeyID     = "s3_access_key_id"
	SettingKeyS3SecretAccessKey = "s3_secret_access_key"
	SettingKeyS3Bucket          = "s3_bucket"
	SettingKeyS3URLPrefix       = "s3_url_prefix"

	// MinIO 相关
	SettingKeyMinIOEndpoint        = "minio_endpoint"
	SettingKeyMinIOAccessKeyID     = "minio_access_key_id"
	SettingKeyMinIOSecretAccessKey = "minio_secret_access_key"
	SettingKeyMinIOBucket          = "minio_bucket"
	SettingKeyMinIOUseSSL          = "minio_use_ssl"
	SettingKeyMinIOURLPrefix       = "minio_url_prefix"

	// 阿里云盘相关
	SettingKeyAliyunPanRefreshToken = "aliyunpan_refresh_token"
	SettingKeyAliyunPanBasePath     = "aliyunpan_base_path"
	SettingKeyAliyunPanDriveType    = "aliyunpan_drive_type"

	// 清理相关
	SettingKeyTrashAutoDeleteDays = "trash_auto_delete_days"
)

// 值类型常量
const (
	SettingValueTypeString = "string"
	SettingValueTypeInt    = "int"
	SettingValueTypeBool   = "bool"
	SettingValueTypeJSON   = "json"
)
