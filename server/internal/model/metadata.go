package model

import (
	"time"
)

// ImageMetadata 图片自定义元数据模型
type ImageMetadata struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ImageID   int64     `gorm:"not null;index" json:"image_id"`
	MetaKey   string    `gorm:"type:varchar(100);not null" json:"meta_key"`
	MetaValue *string   `gorm:"type:text" json:"meta_value,omitempty"`
	ValueType string    `gorm:"type:varchar(20);default:string" json:"value_type"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// 关联
	Image *Image `gorm:"foreignKey:ImageID" json:"-"`
}

// TableName 指定表名
func (ImageMetadata) TableName() string {
	return "image_metadata"
}
