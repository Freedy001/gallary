package model

// ImageVO 图片视图对象，用于API返回
type ImageVO struct {
	Image
	// URL 相关（新增）
	URL          string `json:"url"`                     // 原图访问URL
	ThumbnailURL string `json:"thumbnail_url,omitempty"` // 缩略图访问URL
}

// ToVO 将 Image 转换为 ImageVO
func (img *Image) ToVO(url, thumbnailURL string) *ImageVO {
	return &ImageVO{
		Image:        *img,
		URL:          url,
		ThumbnailURL: thumbnailURL,
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
