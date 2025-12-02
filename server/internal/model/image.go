package model

import (
	"time"

	"gorm.io/gorm"
)

// Image 图片模型
type Image struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	OriginalName string    `gorm:"type:varchar(255);not null" json:"original_name"`
	StoragePath  string    `gorm:"type:varchar(500);not null" json:"storage_path"`
	StorageId    StorageId `gorm:"type:varchar(64);not null;default:local" json:"storage_type"`
	FileSize     int64     `gorm:"not null" json:"file_size"`
	FileHash     string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"file_hash"`
	MimeType     string    `gorm:"type:varchar(50);not null" json:"mime_type"`
	Width        int       `gorm:"type:int;not null" json:"width,omitempty"`
	Height       int       `gorm:"type:int;not null" json:"height,omitempty"`

	// 缩略图相关
	ThumbnailPath   string `gorm:"type:varchar(500);not null" json:"thumbnail_path,omitempty"`
	ThumbnailWidth  *int   `gorm:"type:int" json:"thumbnail_width,omitempty"`
	ThumbnailHeight *int   `gorm:"type:int" json:"thumbnail_height,omitempty"`

	// EXIF 元数据
	TakenAt      *time.Time `gorm:"type:timestamp" json:"taken_at,omitempty"`
	Latitude     *float64   `gorm:"type:decimal(10,8)" json:"latitude,omitempty"`
	Longitude    *float64   `gorm:"type:decimal(11,8)" json:"longitude,omitempty"`
	LocationName *string    `gorm:"type:varchar(255)" json:"location_name,omitempty"`
	CameraModel  *string    `gorm:"type:varchar(100)" json:"camera_model,omitempty"`
	CameraMake   *string    `gorm:"type:varchar(100)" json:"camera_make,omitempty"`
	Aperture     *string    `gorm:"type:varchar(20)" json:"aperture,omitempty"`
	ShutterSpeed *string    `gorm:"type:varchar(20)" json:"shutter_speed,omitempty"`
	ISO          *int       `gorm:"type:int" json:"iso,omitempty"`
	FocalLength  *string    `gorm:"type:varchar(20)" json:"focal_length,omitempty"`

	// 关联
	Tags     []Tag           `gorm:"many2many:image_tags" json:"tags,omitempty"`
	Metadata []ImageMetadata `gorm:"foreignKey:ImageID" json:"metadata,omitempty"`

	// 迁移状态
	MigrationStatus *string `gorm:"type:varchar(20)" json:"migration_status,omitempty"`
	MigrationTaskID *int64  `gorm:"index" json:"migration_task_id,omitempty"`

	// 系统字段
	CreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty" swaggertype:"primitive,string"`
}

// ClusterResult 聚合结果
type ClusterResult struct {
	MinLat     float64 `json:"min_lat"`     // 最小纬度
	MaxLat     float64 `json:"max_lat"`     // 最大纬度
	MinLng     float64 `json:"min_lng"`     // 最小经度
	MaxLng     float64 `json:"max_lng"`     // 最大经度
	Latitude   float64 `json:"latitude"`    // 聚合中心纬度
	Longitude  float64 `json:"longitude"`   // 聚合中心经度
	Count      int64   `json:"count"`       // 图片数量
	CoverImage *Image  `json:"cover_image"` // 封面图片
}

// GeoBounds 地理边界
type GeoBounds struct {
	MinLat float64 `json:"min_lat"` // 最小纬度
	MaxLat float64 `json:"max_lat"` // 最大纬度
	MinLng float64 `json:"min_lng"` // 最小经度
	MaxLng float64 `json:"max_lng"` // 最大经度
	Count  int64   `json:"count"`   // 带有坐标的图片数量
}

// TableName 指定表名
func (*Image) TableName() string {
	return "images"
}
