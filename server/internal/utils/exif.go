package utils

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/rwcarlsen/goexif/exif"
)

// ExifData EXIF 数据结构
type ExifData struct {
	TakenAt      *time.Time
	Latitude     *float64
	Longitude    *float64
	CameraModel  *string
	CameraMake   *string
	Aperture     *string
	ShutterSpeed *string
	ISO          *int
	FocalLength  *string
}

// ExtractExif 从文件中提取 EXIF 信息
func ExtractExif(filePath string) (*ExifData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	x, err := exif.Decode(file)
	if err != nil {
		// 如果没有 EXIF 数据，返回空结构
		return &ExifData{}, nil
	}

	data := &ExifData{}

	// 提取拍摄时间
	if tm, err := x.DateTime(); err == nil {
		data.TakenAt = &tm
	}

	// 提取GPS信息
	if lat, lon, err := x.LatLong(); err == nil {
		data.Latitude = &lat
		data.Longitude = &lon
	}

	// 提取相机制造商
	if make, err := x.Get(exif.Make); err == nil {
		if makeStr, err := make.StringVal(); err == nil {
			data.CameraMake = &makeStr
		}
	}

	// 提取相机型号
	if model, err := x.Get(exif.Model); err == nil {
		if modelStr, err := model.StringVal(); err == nil {
			data.CameraModel = &modelStr
		}
	}

	// 提取光圈
	if fNumber, err := x.Get(exif.FNumber); err == nil {
		if num, denom, err := fNumber.Rat2(0); err == nil && denom != 0 {
			aperture := fmt.Sprintf("f/%.1f", float64(num)/float64(denom))
			data.Aperture = &aperture
		}
	}

	// 提取快门速度
	if expTime, err := x.Get(exif.ExposureTime); err == nil {
		if num, denom, err := expTime.Rat2(0); err == nil && denom != 0 {
			if num == 1 {
				shutter := fmt.Sprintf("1/%d", denom)
				data.ShutterSpeed = &shutter
			} else {
				shutter := fmt.Sprintf("%.2f", float64(num)/float64(denom))
				data.ShutterSpeed = &shutter
			}
		}
	}

	// 提取ISO
	if isoTag, err := x.Get(exif.ISOSpeedRatings); err == nil {
		if isoInt, err := isoTag.Int(0); err == nil {
			data.ISO = &isoInt
		}
	}

	// 提取焦距
	if focal, err := x.Get(exif.FocalLength); err == nil {
		if num, denom, err := focal.Rat2(0); err == nil && denom != 0 {
			focalStr := fmt.Sprintf("%.1fmm", float64(num)/float64(denom))
			data.FocalLength = &focalStr
		}
	}

	return data, nil
}

// ParseGPSCoordinate 解析GPS坐标
func ParseGPSCoordinate(coord string, ref string) (float64, error) {
	// 简化版本的GPS坐标解析
	// 实际使用时可能需要更复杂的解析逻辑
	val, err := strconv.ParseFloat(coord, 64)
	if err != nil {
		return 0, err
	}

	// 根据参考方向调整符号
	if ref == "S" || ref == "W" {
		val = -val
	}

	return val, nil
}
