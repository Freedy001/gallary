package model

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// TagType 标签类型
type TagType string

const (
	TagTypeNormal TagType = "normal" // 普通标签
	TagTypeAlbum  TagType = "album"  // 相册
	TagTypeDevice TagType = "device" // 设备标签
)

// AlbumMetadata 相册元数据
type AlbumMetadata struct {
	CoverImageID *int64  `json:"cover_image_id,omitempty"` // 封面图片ID
	Description  *string `json:"description,omitempty"`    // 相册描述
	SortOrder    int     `json:"sort_order"`               // 排序顺序
}

// Scan 实现 sql.Scanner 接口
func (m *AlbumMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, m)
}

// Value 实现 driver.Valuer 接口
func (m AlbumMetadata) Value() (driver.Value, error) {
	if m.CoverImageID == nil && m.Description == nil && m.SortOrder == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

// Tag 标签模型
type Tag struct {
	ID                int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name              string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"name"`
	NameEn            *string        `gorm:"type:varchar(100)" json:"name_en,omitempty"`            // 英文名称
	VectorDescription *string        `gorm:"type:text" json:"vector_description,omitempty"`         // 用于生成向量的描述
	SourceCategoryId  *string        `gorm:"type:varchar(100)" json:"source_category_id,omitempty"` // 来源分类ID（如 portrait_photography）
	SubCategoryId     *string        `gorm:"type:varchar(100)" json:"sub_category_id,omitempty"`    // 子分类ID（如 traditional_studio）
	Color             *string        `gorm:"type:varchar(7)" json:"color,omitempty"`
	Type              TagType        `gorm:"type:varchar(20);not null;default:normal;index" json:"type"`
	Metadata          *AlbumMetadata `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt         time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// 关联
	Images     []Image        `gorm:"many2many:image_tags" json:"-"`
	Embeddings []TagEmbedding `gorm:"foreignKey:TagID" json:"-"` // 一对多：一个标签可有多个模型的向量
}

// TableName 指定表名
func (*Tag) TableName() string {
	return "tags"
}

// IsAlbum 判断是否是相册
func (t *Tag) IsAlbum() bool {
	return t.Type == TagTypeAlbum
}

// ImageTag 图片标签关联模型
type ImageTag struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ImageID   int64     `gorm:"not null;index;uniqueIndex:idx_image_tag" json:"image_id"`
	TagID     int64     `gorm:"not null;index;uniqueIndex:idx_image_tag" json:"tag_id"`
	SortOrder int       `gorm:"default:0" json:"sort_order"` // 相册内图片排序
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

// TableName 指定表名
func (ImageTag) TableName() string {
	return "image_tags"
}
