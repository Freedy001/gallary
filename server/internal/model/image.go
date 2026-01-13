package model

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Image 图片模型
type Image struct {
	ID           int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	OriginalName string `gorm:"type:varchar(255);not null" json:"original_name"`
	FileSize     int64  `gorm:"not null" json:"file_size"`
	FileHash     string `gorm:"type:varchar(64);uniqueIndex;not null" json:"file_hash"`
	MimeType     string `gorm:"type:varchar(50);not null" json:"mime_type"`
	Width        int    `gorm:"type:int;not null" json:"width,omitempty"`
	Height       int    `gorm:"type:int;not null" json:"height,omitempty"`
	//存储相关
	StoragePath string    `gorm:"type:varchar(500);not null" json:"storage_path"`
	StorageId   StorageId `gorm:"type:varchar(64);not null;default:local" json:"storage_type"`

	// 缩略图相关
	ThumbnailPath      string    `gorm:"type:varchar(500);not null" json:"thumbnail_path,omitempty"`
	ThumbnailStorageId StorageId `gorm:"type:varchar(64);not null;default:local" json:"thumbnail_storage_id"`
	ThumbnailWidth     *int      `gorm:"type:int" json:"thumbnail_width,omitempty"`
	ThumbnailHeight    *int      `gorm:"type:int" json:"thumbnail_height,omitempty"`

	// EXIF 元数据
	TakenAt      *time.Time `gorm:"type:timestamp" json:"taken_at,omitempty"`
	Location     *string    `gorm:"type:geometry(Point,4326)" json:"-"` // PostGIS GEOMETRY 类型，不直接序列化到 JSON
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

	// AI 相关字段
	AIScore *float64 `gorm:"type:decimal(3,2)" json:"ai_score,omitempty"`

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

// SetLocation 根据经纬度设置 Location 字段（用于写入数据库）
func (img *Image) SetLocation(latitude, longitude *float64) {
	if latitude != nil && longitude != nil {
		loc := fmt.Sprintf("SRID=4326;POINT(%f %f)", *longitude, *latitude)
		img.Location = &loc
	} else {
		img.Location = nil
	}
}

// GetLatLng 从 Location 字段解析经纬度（用于 API 返回）
// 返回 latitude, longitude
func (img *Image) GetLatLng() (*float64, *float64) {
	if img.Location == nil || *img.Location == "" {
		return nil, nil
	}

	// Location 格式可能是 "SRID=4326;POINT(lng lat)" 或 "POINT(lng lat)" 或 "0101000020E6100000..." (WKB hex)
	loc := *img.Location

	// 尝试解析 WKT 格式: POINT(lng lat) 或 SRID=4326;POINT(lng lat)
	var lng, lat float64
	if n, _ := fmt.Sscanf(loc, "SRID=4326;POINT(%f %f)", &lng, &lat); n == 2 {
		return &lat, &lng
	}
	if n, _ := fmt.Sscanf(loc, "POINT(%f %f)", &lng, &lat); n == 2 {
		return &lat, &lng
	}

	return nil, nil
}

// SearchParams 搜索参数
type SearchParams struct {
	Keyword   string         `json:"keyword" form:"keyword"`
	StartDate *string        `json:"start_date" form:"start_date"`
	EndDate   *string        `json:"end_date" form:"end_date"`
	Tags      []int64        `json:"tags" form:"tags"`
	Page      int            `json:"page" form:"page"`
	PageSize  int            `json:"page_size" form:"page_size"`
	ModelId   CopositModelId `json:"model_id" form:"model_id"`   // 使用的模型名称
	Latitude  *float64       `json:"latitude" form:"latitude"`   // 中心纬度
	Longitude *float64       `json:"longitude" form:"longitude"` // 中心经度
	Radius    *float64       `json:"radius" form:"radius"`       // 搜索半径（公里），默认 10km
	ImageData []byte         `json:"-" form:"-"`                 // 图片搜索数据（由 handler 处理文件上传）
}
