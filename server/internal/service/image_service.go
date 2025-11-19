package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gallary/server/config"
	"gallary/server/internal/model"
	"gallary/server/internal/repository"
	"gallary/server/internal/storage"
	"gallary/server/internal/utils"
	"gallary/server/pkg/logger"

	"go.uber.org/zap"
)

// ImageService 图片服务接口
type ImageService interface {
	Upload(ctx context.Context, file *multipart.FileHeader) (*model.Image, error)
	GetByID(ctx context.Context, id int64) (*model.Image, error)
	List(ctx context.Context, page, pageSize int) ([]*model.Image, int64, error)
	Delete(ctx context.Context, id int64) error
	Search(ctx context.Context, params *repository.SearchParams) ([]*model.Image, int64, error)
	Download(ctx context.Context, id int64) (io.ReadCloser, string, error)
}

type imageService struct {
	repo    repository.ImageRepository
	storage storage.Storage
	cfg     *config.Config
}

// NewImageService 创建图片服务实例
func NewImageService(repo repository.ImageRepository, storage storage.Storage, cfg *config.Config) ImageService {
	return &imageService{
		repo:    repo,
		storage: storage,
		cfg:     cfg,
	}
}

// Upload 上传图片（包含去重逻辑）
func (s *imageService) Upload(ctx context.Context, fileHeader *multipart.FileHeader) (*model.Image, error) {
	// 1. 验证文件类型
	if !s.cfg.Image.IsAllowedType(fileHeader.Header.Get("Content-Type")) {
		return nil, fmt.Errorf("不支持的文件类型: %s", fileHeader.Header.Get("Content-Type"))
	}

	// 2. 验证文件大小
	if fileHeader.Size > s.cfg.Image.MaxSize {
		return nil, fmt.Errorf("文件大小超过限制: %d bytes", s.cfg.Image.MaxSize)
	}

	// 3. 打开上传的文件
	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("打开上传文件失败: %w", err)
	}
	defer src.Close()

	// 4. 保存到临时文件用于计算hash和提取EXIF
	tempFile, err := os.CreateTemp("", "upload-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// 复制数据到临时文件
	if _, err := io.Copy(tempFile, src); err != nil {
		return nil, fmt.Errorf("写入临时文件失败: %w", err)
	}

	// 5. 计算文件hash（用于去重）
	fileHash, err := utils.CalculateFileHash(tempFile.Name())
	if err != nil {
		return nil, fmt.Errorf("计算文件hash失败: %w", err)
	}

	// 6. 检查是否已存在相同hash的文件（去重）
	existingImage, err := s.repo.FindByHash(ctx, fileHash)
	if err != nil {
		return nil, fmt.Errorf("检查文件hash失败: %w", err)
	}
	if existingImage != nil {
		logger.Info("检测到重复图片，返回已存在的图片",
			zap.String("hash", fileHash),
			zap.Int64("existing_id", existingImage.ID))
		return existingImage, nil
	}

	// 8. 生成存储路径（按日期分目录）
	storagePath := strings.Join(
		[]string{
			"origin",
			time.Now().Format("2006_01_02"),
			fmt.Sprintf("%s.%s", fileHash, filepath.Ext(fileHeader.Filename)),
		},
		"/",
	)

	// 9. 上传到存储系统
	tempFile.Seek(0, 0) // 重置文件指针
	finalPath, err := s.storage.Upload(ctx, tempFile, storagePath)
	if err != nil {
		return nil, fmt.Errorf("上传文件到存储失败: %w", err)
	}

	// 10. 提取图片尺寸
	width, height, err := utils.GetImageDimensions(tempFile.Name())
	if err != nil {
		return nil, err
	}

	image := &model.Image{
		OriginalName: fileHeader.Filename,
		StoragePath:  finalPath,
		StorageType:  s.cfg.Storage.Default,
		FileSize:     fileHeader.Size,
		FileHash:     fileHash,
		MimeType:     fileHeader.Header.Get("Content-Type"),
		Width:        width,
		Height:       height,
	}

	// 12. 创建缩略图
	err = s.thumbImages(ctx, tempFile, image, fileHash)
	if err != nil {
		return nil, err
	}

	// 12. 提取EXIF信息
	exifData, err := utils.ExtractExif(tempFile.Name())
	if err != nil {
		logger.Warn("提取EXIF失败", zap.Error(err))
		exifData = &utils.ExifData{}
	}

	// 设置EXIF数据
	image.TakenAt = exifData.TakenAt
	image.Latitude = exifData.Latitude
	image.Longitude = exifData.Longitude
	image.CameraModel = exifData.CameraModel
	image.CameraMake = exifData.CameraMake
	image.Aperture = exifData.Aperture
	image.ShutterSpeed = exifData.ShutterSpeed
	image.ISO = exifData.ISO
	image.FocalLength = exifData.FocalLength

	// 13. 保存到数据库
	if err := s.repo.Create(ctx, image); err != nil {
		// 保存失败，删除已上传的文件
		s.storage.Delete(ctx, finalPath)
		return nil, fmt.Errorf("保存图片记录失败: %w", err)
	}

	logger.Info("图片上传成功",
		zap.String("hash", image.FileHash),
		zap.String("original_name", image.OriginalName))

	return image, nil
}

func (s *imageService) thumbImages(ctx context.Context, tempFile *os.File, image *model.Image, imageHash string) error {
	thumbnailTempFile, err := os.CreateTemp("", "thumbnail-*.jpg")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}

	defer os.Remove(thumbnailTempFile.Name())
	defer thumbnailTempFile.Close()

	maxWith := uint(0)
	maxHeight := uint(0)

	if image.Width > image.Height {
		maxWith = uint(s.cfg.Image.Thumbnail.Width)
	} else {
		maxHeight = uint(s.cfg.Image.Thumbnail.Height)
	}

	thumbnailWidth, thumbnailHeight, err := utils.GenerateThumbnail(
		tempFile.Name(),
		thumbnailTempFile.Name(),
		maxWith,
		maxHeight,
	)

	if err != nil {
		return fmt.Errorf("生成缩略图失败: %w", err)
	}
	// 生成缩略图存储路径
	thumbStoragePath := strings.Join(
		[]string{
			"thumbnails",
			time.Now().Format("2006_01_02"),
			fmt.Sprintf("%s_thumb.jpg", imageHash),
		},
		"/",
	)

	// 上传缩略图到存储系统
	_, _ = thumbnailTempFile.Seek(0, 0)
	thumbnailPath, err := s.storage.Upload(ctx, thumbnailTempFile, thumbStoragePath)
	if err != nil {
		return fmt.Errorf("上传缩略图失败: %w", err)
	}

	logger.Info("缩略图生成成功",
		zap.String("path", thumbnailPath),
		zap.Int("width", thumbnailWidth),
		zap.Int("height", thumbnailHeight))

	image.ThumbnailPath = thumbnailPath
	image.ThumbnailWidth = &thumbnailWidth
	image.ThumbnailHeight = &thumbnailHeight

	return nil
}

// GetByID 根据ID获取图片
func (s *imageService) GetByID(ctx context.Context, id int64) (*model.Image, error) {
	return s.repo.FindByID(ctx, id)
}

// List 获取图片列表
func (s *imageService) List(ctx context.Context, page, pageSize int) ([]*model.Image, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.List(ctx, page, pageSize)
}

// Delete 删除图片
func (s *imageService) Delete(ctx context.Context, id int64) error {
	// 获取图片信息
	image, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// 从存储删除原图文件
	if err := s.storage.Delete(ctx, image.StoragePath); err != nil {
		logger.Warn("删除存储文件失败", zap.Error(err))
	}

	if err := s.storage.Delete(ctx, image.ThumbnailPath); err != nil {
		logger.Warn("删除缩略图文件失败", zap.Error(err))
	}

	// 从数据库删除记录
	return s.repo.Delete(ctx, id)
}

// Search 搜索图片
func (s *imageService) Search(ctx context.Context, params *repository.SearchParams) ([]*model.Image, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	return s.repo.Search(ctx, params)
}

// Download 下载图片
func (s *imageService) Download(ctx context.Context, id int64) (io.ReadCloser, string, error) {
	// 获取图片信息
	image, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, "", err
	}

	// 从存储获取文件
	reader, err := s.storage.Download(ctx, image.StoragePath)
	if err != nil {
		return nil, "", fmt.Errorf("下载文件失败: %w", err)
	}

	return reader, image.OriginalName, nil
}
