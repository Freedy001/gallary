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

const MainCategoryMarker = "__main_category__" // 主分类向量的特殊标记

// HDBSCANParams HDBSCAN 聚类参数
type HDBSCANParams struct {
	MinClusterSize          int     `json:"min_cluster_size"`
	MinSamples              *int    `json:"min_samples,omitempty"`
	ClusterSelectionEpsilon float64 `json:"cluster_selection_epsilon"`
	ClusterSelectionMethod  string  `json:"cluster_selection_method"`
	Metric                  string  `json:"metric"`
	UMAPEnabled             bool    `json:"umap_enabled"`
	UMAPComponents          int     `json:"umap_n_components,omitempty"`
	UMAPNeighbors           int     `json:"umap_n_neighbors,omitempty"`
}

// SmartAlbumConfig 智能相册配置
type SmartAlbumConfig struct {
	ModelName     string         `json:"model_name"`               // 使用的嵌入模型
	Algorithm     string         `json:"algorithm"`                // 算法名称 (hdbscan)
	ClusterID     int            `json:"cluster_id"`               // 原始聚类 ID
	GeneratedAt   time.Time      `json:"generated_at"`             // 生成时间
	HDBSCANParams *HDBSCANParams `json:"hdbscan_params,omitempty"` // HDBSCAN 参数
	ImageCount    int            `json:"image_count"`              // 生成时的图片数量
}

// AlbumMetadata 相册元数据
type AlbumMetadata struct {
	CoverImageID          *int64  `json:"cover_image_id,omitempty"` // 封面图片ID
	Description           *string `json:"description,omitempty"`    // 相册描述
	SortOrder             int     `json:"sort_order"`               // 排序顺序
	IsSmartAlbum          bool    `json:"is_smart_album,omitempty"` // 是否智能相册
	HDBSCANAvgProbability float32 `json:"hdbscan_avg_probability"`  // 聚类平均概率 (0-1)
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
	if m.CoverImageID == nil && m.Description == nil && m.SortOrder == 0 && !m.IsSmartAlbum {
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
