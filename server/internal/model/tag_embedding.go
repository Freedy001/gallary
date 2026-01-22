package model

import (
	"time"
)

// TagEmbedding 标签向量嵌入
type TagEmbedding struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TagID     int64     `gorm:"not null;uniqueIndex:idx_tag_embedding_tag_model" json:"tag_id"`              // 外键关联 tags 表
	ModelName string    `gorm:"type:varchar(100);uniqueIndex:idx_tag_embedding_tag_model" json:"model_name"` // 模型名称
	ModelId   string    `gorm:"type:varchar(100)" json:"model_id"`                                           // 模型名称
	Dimension int       `gorm:"not null" json:"dimension"`                                                   // 向量维度
	Embedding Vector    `gorm:"type:vector" json:"-"`                                                        // 向量数据（pgvector）
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// 关联
	Tag *Tag `gorm:"foreignKey:TagID" json:"tag,omitempty"`
}

// TableName 指定表名
func (*TagEmbedding) TableName() string {
	return "tag_embeddings"
}

// SourceTag 解析后的标签结构（用于加载 JSON）
type SourceTag struct {
	Name              string `json:"name"`               // 中文名称
	NameEn            string `json:"name_en"`            // 英文名称
	VectorDescription string `json:"vector_description"` // 向量描述
	CategoryId        string `json:"-"`                  // 所属分类ID（解析时填充，如 portrait_photography）
	SubCategoryId     string `json:"-"`                  // 子分类ID（解析时填充，如 traditional_studio）
}

// TagCategory 标签分类结构（用于解析 index.json）
type TagCategory struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	NameEn            string `json:"name_en"`
	File              string `json:"file"`
	Description       string `json:"description"`
	VectorDescription string `json:"vector_description"` // 主分类的向量描述（用于分类判断）
}

// TagIndex 标签索引文件结构
type TagIndex struct {
	Version     string        `json:"version"`
	Description string        `json:"description"`
	Categories  []TagCategory `json:"categories"`
}
