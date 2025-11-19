package model

import (
	"time"
)

// Tag 标签模型
type Tag struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"name"`
	Color     *string   `gorm:"type:varchar(7)" json:"color,omitempty"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// 关联
	Images []Image `gorm:"many2many:image_tags" json:"-"`
}

// TableName 指定表名
func (Tag) TableName() string {
	return "tags"
}

// ImageTag 图片标签关联模型
type ImageTag struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ImageID   int64     `gorm:"not null;index" json:"image_id"`
	TagID     int64     `gorm:"not null;index" json:"tag_id"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

// TableName 指定表名
func (ImageTag) TableName() string {
	return "image_tags"
}
