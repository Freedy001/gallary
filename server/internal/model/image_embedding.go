package model

import (
	"time"
)

// ImageEmbedding 图片向量嵌入（支持多模型）
type ImageEmbedding struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ImageID   int64     `gorm:"not null;uniqueIndex:idx_embedding_image_model" json:"image_id"`
	ModelName string    `gorm:"type:varchar(100);uniqueIndex:idx_embedding_image_model" json:"model_name"` // 模型显示名称
	ModelID   string    `gorm:"type:varchar(100);not null;" json:"model_id"`                               // 模型唯一标识
	Dimension int       `gorm:"not null" json:"dimension"`                                                 // 向量维度
	Embedding Vector    `gorm:"type:vector" json:"-"`                                                      // 向量数据（动态维度）
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// 关联
	Image *Image `gorm:"foreignKey:ImageID" json:"image,omitempty"`
}

// TableName 指定表名
func (*ImageEmbedding) TableName() string {
	return "image_embeddings"
}
