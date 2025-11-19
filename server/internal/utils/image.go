package utils

import (
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"os"

	"github.com/nfnt/resize"
)

// GetImageDimensions 获取图片尺寸
func GetImageDimensions(filePath string) (width, height int, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, fmt.Errorf("打开图片失败: %w", err)
	}
	defer file.Close()

	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, fmt.Errorf("解析图片失败: %w", err)
	}

	return img.Width, img.Height, nil
}

// GenerateThumbnail 生成缩略图
// sourcePath: 原图路径
// destPath: 缩略图保存路径
// maxWidth: 缩略图最大宽度
// 返回: 缩略图实际宽度, 高度, 错误
func GenerateThumbnail(sourcePath, destPath string, maxWidth uint, maxHeight uint) (width, height int, err error) {
	// 打开原图
	file, err := os.Open(sourcePath)
	if err != nil {
		return 0, 0, fmt.Errorf("打开原图失败: %w", err)
	}
	defer file.Close()

	// 解码图片
	img, format, err := image.Decode(file)
	if err != nil {
		return 0, 0, fmt.Errorf("解码图片失败: %w", err)
	}

	// 生成缩略图 (保持宽高比)
	thumbnail := resize.Resize(maxWidth, maxHeight, img, resize.Lanczos3)

	// 创建输出文件
	out, err := os.Create(destPath)
	if err != nil {
		return 0, 0, fmt.Errorf("创建缩略图文件失败: %w", err)
	}
	defer out.Close()

	// 根据格式编码输出
	switch format {
	case "jpeg", "jpg":
		err = jpeg.Encode(out, thumbnail, &jpeg.Options{Quality: 85})
	default:
		// 其他格式统一转为JPEG
		err = jpeg.Encode(out, thumbnail, &jpeg.Options{Quality: 85})
	}

	if err != nil {
		return 0, 0, fmt.Errorf("编码缩略图失败: %w", err)
	}

	// 返回缩略图尺寸
	bounds := thumbnail.Bounds()
	return bounds.Dx(), bounds.Dy(), nil
}
