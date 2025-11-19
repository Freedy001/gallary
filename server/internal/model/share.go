package model

import (
	"time"
)

// Share 分享模型
type Share struct {
	ID            int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	ShareCode     string     `gorm:"type:varchar(32);uniqueIndex;not null" json:"share_code"`
	Title         *string    `gorm:"type:varchar(255)" json:"title,omitempty"`
	Description   *string    `gorm:"type:text" json:"description,omitempty"`
	Password      *string    `gorm:"type:varchar(64)" json:"password,omitempty"`
	ExpireAt      *time.Time `gorm:"type:timestamp" json:"expire_at,omitempty"`
	ViewCount     int        `gorm:"default:0" json:"view_count"`
	DownloadCount int        `gorm:"default:0" json:"download_count"`
	IsActive      bool       `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// 关联
	Images []Image `gorm:"many2many:share_images" json:"images,omitempty"`
}

// TableName 指定表名
func (Share) TableName() string {
	return "shares"
}

// IsExpired 检查分享是否过期
func (s *Share) IsExpired() bool {
	if s.ExpireAt == nil {
		return false
	}
	return time.Now().After(*s.ExpireAt)
}

// ShareImage 分享图片关联模型
type ShareImage struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ShareID   int64     `gorm:"not null;index" json:"share_id"`
	ImageID   int64     `gorm:"not null;index" json:"image_id"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

// TableName 指定表名
func (ShareImage) TableName() string {
	return "share_images"
}
