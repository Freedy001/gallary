package model

import (
	"gallary/server/config"
	"time"

	"gorm.io/gorm"
)

// ImageVO 图片视图对象，用于API返回
type ImageVO struct {
	ID           int64              `json:"id"`
	OriginalName string             `json:"original_name"`
	StoragePath  string             `json:"storage_path"`
	StorageType  config.StorageType `json:"storage_type"`
	FileSize     int64              `json:"file_size"`
	FileHash     string             `json:"file_hash"`
	MimeType     string             `json:"mime_type"`
	Width        int                `json:"width,omitempty"`
	Height       int                `json:"height,omitempty"`

	// URL 相关（新增）
	URL          string `json:"url"`                     // 原图访问URL
	ThumbnailURL string `json:"thumbnail_url,omitempty"` // 缩略图访问URL

	// 缩略图相关
	ThumbnailPath   string `json:"thumbnail_path,omitempty"`
	ThumbnailWidth  *int   `json:"thumbnail_width,omitempty"`
	ThumbnailHeight *int   `json:"thumbnail_height,omitempty"`

	// EXIF 元数据
	TakenAt      *time.Time `json:"taken_at,omitempty"`
	Latitude     *float64   `json:"latitude,omitempty"`
	Longitude    *float64   `json:"longitude,omitempty"`
	LocationName *string    `json:"location_name,omitempty"`
	CameraModel  *string    `json:"camera_model,omitempty"`
	CameraMake   *string    `json:"camera_make,omitempty"`
	Aperture     *string    `json:"aperture,omitempty"`
	ShutterSpeed *string    `json:"shutter_speed,omitempty"`
	ISO          *int       `json:"iso,omitempty"`
	FocalLength  *string    `json:"focal_length,omitempty"`

	// 关联
	Tags     []Tag           `json:"tags,omitempty"`
	Metadata []ImageMetadata `json:"metadata,omitempty"`

	// 迁移状态
	MigrationStatus *string `json:"migration_status,omitempty"`
	MigrationTaskID *int64  `json:"migration_task_id,omitempty"`

	// 系统字段
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" swaggertype:"primitive,string"`
}

// ToVO 将 Image 转换为 ImageVO
func (img *Image) ToVO(url, thumbnailURL string) *ImageVO {
	return &ImageVO{
		ID:              img.ID,
		OriginalName:    img.OriginalName,
		StoragePath:     img.StoragePath,
		StorageType:     img.StorageType,
		FileSize:        img.FileSize,
		FileHash:        img.FileHash,
		MimeType:        img.MimeType,
		Width:           img.Width,
		Height:          img.Height,
		URL:             url,
		ThumbnailURL:    thumbnailURL,
		ThumbnailPath:   img.ThumbnailPath,
		ThumbnailWidth:  img.ThumbnailWidth,
		ThumbnailHeight: img.ThumbnailHeight,
		TakenAt:         img.TakenAt,
		Latitude:        img.Latitude,
		Longitude:       img.Longitude,
		LocationName:    img.LocationName,
		CameraModel:     img.CameraModel,
		CameraMake:      img.CameraMake,
		Aperture:        img.Aperture,
		ShutterSpeed:    img.ShutterSpeed,
		ISO:             img.ISO,
		FocalLength:     img.FocalLength,
		Tags:            img.Tags,
		Metadata:        img.Metadata,
		MigrationStatus: img.MigrationStatus,
		MigrationTaskID: img.MigrationTaskID,
		CreatedAt:       img.CreatedAt,
		UpdatedAt:       img.UpdatedAt,
		DeletedAt:       img.DeletedAt,
	}
}

// ClusterResultVO 聚合结果视图对象
type ClusterResultVO struct {
	MinLat     float64  `json:"min_lat"`
	MaxLat     float64  `json:"max_lat"`
	MinLng     float64  `json:"min_lng"`
	MaxLng     float64  `json:"max_lng"`
	Latitude   float64  `json:"latitude"`
	Longitude  float64  `json:"longitude"`
	Count      int64    `json:"count"`
	CoverImage *ImageVO `json:"cover_image"`
}
