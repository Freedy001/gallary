package model

// ImageVO 图片视图对象，用于API返回
type ImageVO struct {
	Image
	// URL 相关（新增）
	URL          string `json:"url"`                     // 原图访问URL
	ThumbnailURL string `json:"thumbnail_url,omitempty"` // 缩略图访问URL
	// 经纬度字段（从 location 解析）
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
}

// ToVO 将 Image 转换为 ImageVO
func (img *Image) ToVO(url, thumbnailURL string) *ImageVO {
	lat, lng := img.GetLatLng()
	return &ImageVO{
		Image:        *img,
		URL:          url,
		ThumbnailURL: thumbnailURL,
		Latitude:     lat,
		Longitude:    lng,
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

// ImageSource 图片来源（支持二进制数据或 URL）
type ImageSource struct {
	Data []byte // 图片二进制数据
	URL  string // 图片 URL（用于远程存储，避免二次传输）
}
