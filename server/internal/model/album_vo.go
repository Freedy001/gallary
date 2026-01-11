package model

// AlbumVO 相册视图对象，用于API返回
type AlbumVO struct {
	ID                    int64    `json:"id"`
	Name                  string   `json:"name"`
	Description           *string  `json:"description,omitempty"`
	CoverImage            *ImageVO `json:"cover_image,omitempty"`    // 封面图片
	CoverImageID          *int64   `json:"cover_image_id,omitempty"` // 自定义封面ID
	ImageCount            int64    `json:"image_count"`              // 图片数量
	SortOrder             int      `json:"sort_order"`
	IsSmartAlbum          bool     `json:"is_smart_album"`                    // 是否智能相册
	HDBSCANAvgProbability float32  `json:"hdbscan_avg_probability,omitempty"` // 聚类平均概率 (0-1)
	CreatedAt             string   `json:"created_at"`
	UpdatedAt             string   `json:"updated_at"`
}

// ToAlbumVO 将 Tag(相册类型) 转换为 AlbumVO
func (t *Tag) ToAlbumVO(coverImage *ImageVO, imageCount int64) *AlbumVO {
	vo := &AlbumVO{
		ID:         t.ID,
		Name:       t.Name,
		ImageCount: imageCount,
		CreatedAt:  t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  t.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if t.Metadata != nil {
		vo.Description = t.Metadata.Description
		vo.SortOrder = t.Metadata.SortOrder
		vo.IsSmartAlbum = t.Metadata.IsSmartAlbum
		vo.HDBSCANAvgProbability = t.Metadata.HDBSCANAvgProbability
		vo.CoverImageID = t.Metadata.CoverImageID // 包含自定义封面ID
	}

	if coverImage != nil {
		vo.CoverImage = coverImage
	}

	return vo
}
