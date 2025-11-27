package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Admin    AdminConfig    `mapstructure:"admin"`
	Database DatabaseConfig `mapstructure:"database"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Image    ImageConfig    `mapstructure:"image"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Logger   LoggerConfig   `mapstructure:"logger"`
	CORS     CORSConfig     `mapstructure:"cors"`
	Share    ShareConfig    `mapstructure:"share"`
	Trash    TrashConfig    `mapstructure:"trash"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type AdminConfig struct {
	Password string `mapstructure:"password"`
}

type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	SSLMode      string `mapstructure:"sslmode"`
	Timezone     string `mapstructure:"timezone"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	LogLevel     string `mapstructure:"log_level"`
}

type StorageConfig struct {
	Default string             `mapstructure:"default"`
	Local   LocalStorageConfig `mapstructure:"local"`
	OSS     OSSStorageConfig   `mapstructure:"oss"`
	S3      S3StorageConfig    `mapstructure:"s3"`
	MinIO   MinIOStorageConfig `mapstructure:"minio"`
}

type LocalStorageConfig struct {
	BasePath  string `mapstructure:"base_path"`
	URLPrefix string `mapstructure:"url_prefix"`
}

type OSSStorageConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	Bucket          string `mapstructure:"bucket"`
	URLPrefix       string `mapstructure:"url_prefix"`
}

type S3StorageConfig struct {
	Region          string `mapstructure:"region"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	Bucket          string `mapstructure:"bucket"`
	URLPrefix       string `mapstructure:"url_prefix"`
}

type MinIOStorageConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	Bucket          string `mapstructure:"bucket"`
	UseSSL          bool   `mapstructure:"use_ssl"`
	URLPrefix       string `mapstructure:"url_prefix"`
}

type ImageConfig struct {
	AllowedTypes []string        `mapstructure:"allowed_types"`
	MaxSize      int64           `mapstructure:"max_size"`
	Thumbnail    ThumbnailConfig `mapstructure:"thumbnail"`
}

type ThumbnailConfig struct {
	Width  int `mapstructure:"width"`
	Height int `mapstructure:"height"`
	//Quality int `mapstructure:"quality"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type LoggerConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}

type CORSConfig struct {
	AllowOrigins     []string `mapstructure:"allow_origins"`
	AllowMethods     []string `mapstructure:"allow_methods"`
	AllowHeaders     []string `mapstructure:"allow_headers"`
	ExposeHeaders    []string `mapstructure:"expose_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"`
}

type ShareConfig struct {
	DefaultExpireHours int `mapstructure:"default_expire_hours"`
	CodeLength         int `mapstructure:"code_length"`
}

type TrashConfig struct {
	AutoDeleteDays int `mapstructure:"auto_delete_days"`
}

var GlobalConfig *Config

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 设置默认值
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	GlobalConfig = &config
	return &config, nil
}

// setDefaults 设置默认配置
func setDefaults() {
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")

	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.timezone", "Asia/Shanghai")
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.max_open_conns", 100)
	viper.SetDefault("database.log_level", "info")

	viper.SetDefault("storage.default", "local")
	viper.SetDefault("storage.local.base_path", "./storage/images")
	viper.SetDefault("storage.local.thumbnail_path", "./storage/thumbnails")
	viper.SetDefault("storage.local.url_prefix", "/static/images")

	viper.SetDefault("image.max_size", 52428800) // 50MB
	viper.SetDefault("image.thumbnail.width", 300)
	viper.SetDefault("image.thumbnail.height", 300)
	viper.SetDefault("image.thumbnail.quality", 85)

	viper.SetDefault("jwt.expire_hours", 168) // 7天

	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.format", "console")
	viper.SetDefault("logger.output", "./logs/app.log")
	viper.SetDefault("logger.max_size", 100)
	viper.SetDefault("logger.max_backups", 5)
	viper.SetDefault("logger.max_age", 30)
	viper.SetDefault("logger.compress", true)

	viper.SetDefault("share.default_expire_hours", 168)
	viper.SetDefault("share.code_length", 8)

	viper.SetDefault("trash.auto_delete_days", 30)
}

// GetAddr 获取服务器地址
func (c *ServerConfig) GetAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		c.Host, c.Port, c.Username, c.Password, c.Database, c.SSLMode, c.Timezone)
}

// IsAuthEnabled 检查是否启用认证
func (c *AdminConfig) IsAuthEnabled() bool {
	return c.Password != ""
}

// GetExpireDuration 获取 JWT 过期时间
func (c *JWTConfig) GetExpireDuration() time.Duration {
	return time.Duration(c.ExpireHours) * time.Hour
}

// IsAllowedType 检查文件类型是否允许
func (c *ImageConfig) IsAllowedType(mimeType string) bool {
	for _, t := range c.AllowedTypes {
		if t == mimeType {
			return true
		}
	}
	return false
}
