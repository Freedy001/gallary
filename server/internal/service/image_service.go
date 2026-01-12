package service

import (
	"archive/zip"
	"context"
	"fmt"
	"gallary/server/internal/repository"
	"gallary/server/internal/websocket"
	"gallary/server/pkg/database"
	"io"
	"math"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gallary/server/config"
	"gallary/server/internal/model"
	"gallary/server/internal/storage"
	"gallary/server/internal/utils"
	"gallary/server/pkg/logger"

	"go.uber.org/zap"
)

// UpdateMetadataRequest 元数据更新请求
type UpdateMetadataRequest struct {
	// 要更新的图片ID列表
	ImageIDs []int64 `json:"image_ids" binding:"required"`
	// 基本信息
	OriginalName *string `json:"original_name,omitempty"`

	// 拍摄时间
	TakenAt *time.Time `json:"taken_at,omitempty"`

	// 地理位置信息
	LocationName *string  `json:"location_name,omitempty"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`

	// 自定义元数据
	Metadata []MetadataUpdate `json:"metadata,omitempty"`

	// 标签信息
	Tags []string `json:"tags,omitempty"`
}

// MetadataUpdate 元数据更新项
type MetadataUpdate struct {
	Key       string  `json:"key" binding:"required"`
	Value     *string `json:"value,omitempty"`
	ValueType string  `json:"value_type,omitempty"`
}

// ImageService 图片服务接口
type ImageService interface {
	Upload(ctx context.Context, file *multipart.FileHeader, albumID *int64) (*model.ImageVO, error)
	GetByID(ctx context.Context, id int64) (*model.ImageVO, error)
	GetByIDs(ctx context.Context, ids []int64) ([]*model.ImageVO, error)
	List(ctx context.Context, page, pageSize int, sortBy string) ([]*model.ImageVO, int64, error)
	Delete(ctx context.Context, id int64) error
	DeleteBatch(ctx context.Context, ids []int64) error
	Search(ctx context.Context, params *model.SearchParams) ([]*model.ImageVO, int64, error)
	Download(ctx context.Context, image *model.Image) (io.ReadCloser, error)
	DownloadZipped(ctx context.Context, ids []int64, writer io.Writer) (string, error)
	BatchUpdateMetadata(ctx context.Context, req *UpdateMetadataRequest) ([]int64, error)
	GetImagesWithLocation(ctx context.Context) ([]*model.ImageVO, error)
	GetClusters(ctx context.Context, minLat, maxLat, minLng, maxLng float64, zoom int) ([]*model.ClusterResultVO, error)
	GetClusterImages(ctx context.Context, minLat, maxLat, minLng, maxLng float64, page, pageSize int) ([]*model.ImageVO, int64, error)
	GetGeoBounds(ctx context.Context) (*model.GeoBounds, error)

	// 回收站相关方法
	ListDeleted(ctx context.Context, page, pageSize int) ([]*model.ImageVO, int64, error)
	RestoreImages(ctx context.Context, ids []int64) error
	PermanentlyDelete(ctx context.Context, ids []int64) error
	CleanupExpiredTrash(ctx context.Context) (int, error)
	Repo() repository.ImageRepository

	// 标签相关方法
	SearchTags(ctx context.Context, keyword string, limit int) ([]*model.Tag, error)
	GetPopularTags(ctx context.Context, limit int) ([]*model.Tag, error)
}

type imageService struct {
	repo          repository.ImageRepository
	albumRepo     repository.AlbumRepository
	embeddingRepo repository.ImageEmbeddingRepository
	storage       *storage.StorageManager
	cfg           *config.Config
	notifier      websocket.Notifier
	aiService     AIService
}

// NewImageService 创建图片服务实例
func NewImageService(repo repository.ImageRepository, albumRepo repository.AlbumRepository, embeddingRepo repository.ImageEmbeddingRepository, storage *storage.StorageManager, cfg *config.Config, notifier websocket.Notifier, aiService AIService) ImageService {
	return &imageService{
		repo:          repo,
		albumRepo:     albumRepo,
		embeddingRepo: embeddingRepo,
		storage:       storage,
		cfg:           cfg,
		notifier:      notifier,
		aiService:     aiService,
	}
}

// Upload 上传图片（包含去重逻辑）
//
//goland:noinspection GoUnhandledErrorResult
func (s *imageService) Upload(ctx context.Context, fileHeader *multipart.FileHeader, albumID *int64) (*model.ImageVO, error) {
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
		// 检查是否是逻辑删除的记录
		if existingImage.DeletedAt.Valid {
			// 恢复逻辑删除的记录
			logger.Info("检测到已删除的重复图片，恢复记录",
				zap.String("hash", fileHash),
				zap.Int64("existing_id", existingImage.ID))

			if err := s.repo.Restore(ctx, existingImage.ID); err != nil {
				return nil, fmt.Errorf("恢复图片记录失败: %w", err)
			}

			// 如果指定了相册ID，添加到相册
			if albumID != nil {
				if err := s.albumRepo.AddImages(ctx, *albumID, []int64{existingImage.ID}); err != nil {
					return nil, fmt.Errorf("添加图片到相册失败: %w", err)
				}
			}

			// 重新获取完整的记录信息
			restored, err := s.repo.FindByID(ctx, existingImage.ID)
			if err != nil {
				return nil, err
			}
			return s.storage.ToVO(restored), nil
		}

		// 图片已存在，如果指定了相册ID，也添加到相册
		if albumID != nil {
			if err := s.albumRepo.AddImages(ctx, *albumID, []int64{existingImage.ID}); err != nil {
				return nil, fmt.Errorf("添加图片到相册失败: %w", err)
			}
		}

		logger.Info("检测到重复图片，返回已存在的图片",
			zap.String("hash", fileHash),
			zap.Int64("existing_id", existingImage.ID))
		vo := s.storage.ToVO(existingImage)
		return vo, nil
	}

	// 8. 生成存储路径（按日期分目录）
	storagePath := strings.Join(
		[]string{
			"origin",
			time.Now().Format("2006_01_02"),
			fmt.Sprintf("%s%s", fileHash, filepath.Ext(fileHeader.Filename)),
		},
		"/",
	)

	// 9. 上传到存储系统
	tempFile.Seek(0, 0) // 重置文件指针
	finalPath, err := s.storage.UploadToDefaultStorage(ctx, tempFile, storagePath)
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
		StorageId:    s.storage.DefaultType(),
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
	image.SetLocation(exifData.Latitude, exifData.Longitude) // 使用 SetLocation 设置 PostGIS 格式
	image.CameraModel = exifData.CameraModel
	image.CameraMake = exifData.CameraMake
	image.Aperture = exifData.Aperture
	image.ShutterSpeed = exifData.ShutterSpeed
	image.ISO = exifData.ISO
	image.FocalLength = exifData.FocalLength

	// 13. 保存到数据库
	if err := s.repo.Create(ctx, image); err != nil {
		// 保存失败，删除已上传的文件
		_ = s.storage.Delete(ctx, image.StorageId, finalPath)
		return nil, fmt.Errorf("保存图片记录失败: %w", err)
	}

	// 14. 自动添加设备标签
	if err := s.autoAddDeviceTags(ctx, image); err != nil {
		// 添加标签失败，但图片已上传成功，记录错误但不回滚
		logger.Warn("自动添加设备标签失败", zap.Error(err), zap.Int64("image_id", image.ID))
	}

	// 15. 如果指定了相册ID，添加到相册
	if albumID != nil {
		if err := s.albumRepo.AddImages(ctx, *albumID, []int64{image.ID}); err != nil {
			// 添加相册失败，但图片已上传成功，记录错误但不回滚
			logger.Error("添加图片到相册失败", zap.Error(err), zap.Int64("image_id", image.ID), zap.Int64("album_id", *albumID))
		}
	}

	logger.Info("图片上传成功",
		zap.String("hash", image.FileHash),
		zap.String("original_name", image.OriginalName))

	// 广播图片总数更新
	s.notifyCount(ctx)
	return s.storage.ToVO(image), nil
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

	// 缩略图始终上传到本地存储
	localStorage := s.storage.GetLocalStorage()
	if localStorage == nil {
		return fmt.Errorf("本地存储未初始化")
	}

	_, _ = thumbnailTempFile.Seek(0, 0)
	thumbnailPath, err := localStorage.Upload(ctx, thumbnailTempFile, thumbStoragePath)
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
func (s *imageService) GetByID(ctx context.Context, id int64) (*model.ImageVO, error) {
	image, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 检查迁移状态
	if image != nil && image.MigrationStatus != nil && *image.MigrationStatus != "" {
		return nil, fmt.Errorf("图片正在迁移中，暂时不可访问")
	}

	return s.storage.ToVO(image), nil
}

// GetByIDs 根据ID列表获取图片
func (s *imageService) GetByIDs(ctx context.Context, ids []int64) ([]*model.ImageVO, error) {
	if len(ids) == 0 {
		return []*model.ImageVO{}, nil
	}
	images, err := s.repo.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return s.storage.ToVOList(images), nil
}

// List 获取图片列表
func (s *imageService) List(ctx context.Context, page, pageSize int, sortBy string) ([]*model.ImageVO, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	images, total, err := s.repo.List(ctx, page, pageSize, sortBy)
	if err != nil {
		return nil, 0, err
	}
	vos := s.storage.ToVOList(images)
	return vos, total, err
}

// Delete 删除图片（逻辑删除，不删除本地文件）
func (s *imageService) Delete(ctx context.Context, id int64) error {
	// 获取图片信息
	image, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	logger.Info("逻辑删除图片，保留本地文件",
		zap.Int64("id", id),
		zap.String("file_hash", image.FileHash),
		zap.String("storage_path", image.StoragePath))

	// 仅从数据库逻辑删除记录，不删除本地文件
	err = s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	// 广播图片总数更新
	s.notifyCount(ctx)
	return nil
}

// DeleteBatch 批量删除图片（逻辑删除，不删除本地文件）
func (s *imageService) DeleteBatch(ctx context.Context, ids []int64) error {
	// 获取所有要删除的图片信息
	images, err := s.repo.FindByIDs(ctx, ids)
	if err != nil {
		return err
	}

	logger.Info("批量逻辑删除图片，保留本地文件",
		zap.Int("count", len(images)),
		zap.Any("ids", ids))

	// 仅从数据库批量逻辑删除记录，不删除本地文件
	err = s.repo.DeleteBatch(ctx, ids)
	if err != nil {
		return err
	}

	// 广播图片总数更新
	s.notifyCount(ctx)
	return nil
}

// Search 搜索图片
func (s *imageService) Search(ctx context.Context, params *model.SearchParams) ([]*model.ImageVO, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	// 文本语义搜索
	if params.ModelId != "" {
		if s.aiService == nil {
			return nil, 0, fmt.Errorf("未配置 ai service")
		}
		// 检查是否有传统筛选条件
		hasNormalFilter :=
			params.StartDate != nil ||
				params.EndDate != nil ||
				len(params.Tags) > 0 ||
				params.Latitude != nil ||
				params.Longitude != nil

		if hasNormalFilter {
			// 1. 先执行传统筛选获取候选图片ID列表（限制候选数量防止性能问题）
			candidateIDs, err := s.repo.SearchIDs(ctx, params, 1000)
			if err != nil {
				return nil, 0, fmt.Errorf("筛选候选图片失败: %w", err)
			}

			// 如果没有符合条件的图片，直接返回空结果
			if len(candidateIDs) == 0 {
				return []*model.ImageVO{}, 0, nil
			}

			// 2. 在候选ID范围内进行语义搜索排序
			images, err := s.aiService.SemanticSearchWithinIDs(ctx, params.ImageData, params.Keyword, params.ModelId, candidateIDs, params.PageSize)
			if err != nil {
				return nil, 0, fmt.Errorf("语义搜索失败: %w", err)
			}

			// 转换为 VO 列表，总数为候选图片数量
			vos := s.storage.ToVOList(images)
			return vos, int64(len(candidateIDs)), nil
		}

		// 纯语义搜索（无传统筛选条件）
		images, err := s.aiService.SemanticSearchWithinIDs(ctx, params.ImageData, params.Keyword, params.ModelId, nil, params.PageSize)
		if err != nil {
			return nil, 0, fmt.Errorf("语义搜索失败: %w", err)
		}

		// 转换为 VO 列表
		vos := s.storage.ToVOList(images)
		return vos, int64(len(vos)), nil
	}

	// 普通搜索
	images, total, err := s.repo.Search(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	vos := s.storage.ToVOList(images)
	return vos, total, err
}

func (s *imageService) Download(ctx context.Context, image *model.Image) (io.ReadCloser, error) {
	// 检查迁移状态
	if image.MigrationStatus != nil && *image.MigrationStatus != "" {
		return nil, fmt.Errorf("图片正在迁移中，暂时不可下载")
	}

	// 从存储获取文件
	reader, err := s.storage.Download(ctx, image.StorageId, image.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("下载文件失败: %w", err)
	}

	return reader, nil
}

// DownloadBatch 批量下载图片（打包为 ZIP，流式写入）
func (s *imageService) DownloadZipped(ctx context.Context, ids []int64, writer io.Writer) (string, error) {
	// 获取所有图片信息
	images, err := s.repo.FindByIDs(ctx, ids)
	if err != nil {
		return "", fmt.Errorf("获取图片信息失败: %w", err)
	}

	if len(images) == 0 {
		return "", fmt.Errorf("未找到要下载的图片")
	}

	// 创建 ZIP writer，直接写入到响应流
	zipWriter := zip.NewWriter(writer)
	defer zipWriter.Close()

	// 用于处理重名文件
	nameCount := make(map[string]int)

	// 将每个图片添加到 ZIP 中
	for _, image := range images {
		// 从存储获取文件
		reader, err := s.storage.Download(ctx, image.StorageId, image.StoragePath)
		if err != nil {
			logger.Warn("下载文件失败，跳过", zap.Int64("id", image.ID), zap.Error(err))
			continue
		}

		// 处理重名文件
		filename := image.OriginalName
		if count, exists := nameCount[filename]; exists {
			ext := filepath.Ext(filename)
			base := strings.TrimSuffix(filename, ext)
			filename = fmt.Sprintf("%s_%d%s", base, count+1, ext)
		}
		nameCount[image.OriginalName]++

		// 创建 ZIP 内的文件
		writer, err := zipWriter.Create(filename)
		if err != nil {
			reader.Close()
			logger.Warn("创建 ZIP 条目失败，跳过", zap.String("filename", filename), zap.Error(err))
			continue
		}

		// 写入文件内容
		_, err = io.Copy(writer, reader)
		reader.Close()
		if err != nil {
			logger.Warn("写入 ZIP 内容失败，跳过", zap.String("filename", filename), zap.Error(err))
			continue
		}
	}

	zipFilename := fmt.Sprintf("images_%s.zip", time.Now().Format("20060102_150405"))

	logger.Info("批量下载图片打包完成",
		zap.Int("count", len(images)),
		zap.String("filename", zipFilename))

	return zipFilename, nil
}

// updateImageMetadata 更新图片自定义元数据
func (s *imageService) updateImageMetadata(ctx context.Context, image *model.Image, metadata []MetadataUpdate) error {
	// 获取现有的元数据
	existingMetadata, err := s.repo.GetMetadata(ctx, image.ID)
	if err != nil {
		return fmt.Errorf("获取现有元数据失败: %w", err)
	}

	// 创建现有元数据的映射
	existingMap := make(map[string]*model.ImageMetadata)
	for i := range existingMetadata {
		existingMap[existingMetadata[i].MetaKey] = &existingMetadata[i]
	}

	// 处理每个元数据项
	for _, item := range metadata {
		if item.Value == nil || *item.Value == "" {
			// 如果值为空，删除该元数据
			if existing, exists := existingMap[item.Key]; exists {
				if err := s.repo.DeleteMetadata(ctx, existing.ID); err != nil {
					return fmt.Errorf("删除元数据失败 %s: %w", item.Key, err)
				}
			}
		} else {
			// 如果值不为空，更新或创建元数据
			valueType := item.ValueType
			if valueType == "" {
				valueType = "string"
			}

			if existing, exists := existingMap[item.Key]; exists {
				// 更新现有元数据
				existing.MetaValue = item.Value
				existing.ValueType = valueType
				if err := s.repo.UpdateMetadata(ctx, existing); err != nil {
					return fmt.Errorf("更新元数据失败 %s: %w", item.Key, err)
				}
			} else {
				// 创建新元数据
				newMetadata := &model.ImageMetadata{
					ImageID:   image.ID,
					MetaKey:   item.Key,
					MetaValue: item.Value,
					ValueType: valueType,
				}
				if err := s.repo.CreateMetadata(ctx, newMetadata); err != nil {
					return fmt.Errorf("创建元数据失败 %s: %w", item.Key, err)
				}
			}
		}
	}

	return nil
}

// autoAddDeviceTags 自动添加设备标签
func (s *imageService) autoAddDeviceTags(ctx context.Context, image *model.Image) error {
	var tagNames []string

	// 添加相机品牌标签
	if image.CameraMake != nil && *image.CameraMake != "" {
		tagNames = append(tagNames, *image.CameraMake)
	}

	// 添加相机型号标签
	if image.CameraModel != nil && *image.CameraModel != "" {
		tagNames = append(tagNames, *image.CameraModel)
	}

	// 如果没有标签需要添加，直接返回
	if len(tagNames) == 0 {
		return nil
	}

	// 获取或创建标签，并标记为设备标签
	var tagIDs []int64
	for _, tagName := range tagNames {
		// 查找现有标签
		tag, err := s.repo.FindTagByName(ctx, tagName)
		if err != nil {
			return fmt.Errorf("查找标签失败 %s: %w", tagName, err)
		}

		if tag == nil {
			// 创建新标签，类型设置为设备标签
			tag = &model.Tag{
				Name: tagName,
				Type: model.TagTypeDevice,
			}
			if err := s.repo.CreateTag(ctx, tag); err != nil {
				return fmt.Errorf("创建设备标签失败 %s: %w", tagName, err)
			}
		}

		tagIDs = append(tagIDs, tag.ID)
	}

	// 添加图片标签关联（不覆盖现有标签）
	if err := s.repo.AddImageTags(ctx, image.ID, tagIDs); err != nil {
		return fmt.Errorf("添加图片设备标签关联失败: %w", err)
	}

	logger.Info("自动添加设备标签成功",
		zap.Int64("image_id", image.ID),
		zap.Strings("tags", tagNames))

	return nil
}

// updateImageTags 更新图片标签（只允许使用现有标签）
func (s *imageService) updateImageTags(ctx context.Context, image *model.Image, tagNames []string) error {
	// 获取现有标签
	var tagIDs []int64
	for _, tagName := range tagNames {
		if tagName == "" {
			continue
		}

		// 查找现有标签
		tag, err := s.repo.FindTagByName(ctx, tagName)
		if err != nil {
			return fmt.Errorf("查找标签失败 %s: %w", tagName, err)
		}

		if tag == nil {
			// 标签不存在，跳过（不再创建新标签）
			logger.Warn("标签不存在，跳过", zap.String("tag_name", tagName))
			continue
		}

		tagIDs = append(tagIDs, tag.ID)
	}

	// 更新图片标签关联
	if err := s.repo.UpdateImageTags(ctx, image.ID, tagIDs); err != nil {
		return fmt.Errorf("更新图片标签关联失败: %w", err)
	}

	return nil
}

// BatchUpdateMetadata 批量更新图片元数据
func (s *imageService) BatchUpdateMetadata(ctx context.Context, req *UpdateMetadataRequest) ([]int64, error) {
	var updatedImages []int64

	// 遍历每个图片ID进行更新
	for _, imageID := range req.ImageIDs {
		// 将批量请求转换为单个请求
		// 1. 获取现有图片信息
		image, err := s.repo.FindByID(ctx, imageID)
		if err != nil {
			return nil, err
		}

		// 2. 更新基本信息
		if req.OriginalName != nil {
			image.OriginalName = *req.OriginalName
		}

		// 3. 更新拍摄时间
		if req.TakenAt != nil {
			image.TakenAt = req.TakenAt
		}

		// 4. 更新地理位置信息
		if req.LocationName != nil {
			image.LocationName = req.LocationName
		}
		if req.Latitude != nil || req.Longitude != nil {
			// 获取当前经纬度
			currentLat, currentLng := image.GetLatLng()
			newLat := currentLat
			newLng := currentLng
			if req.Latitude != nil {
				newLat = req.Latitude
			}
			if req.Longitude != nil {
				newLng = req.Longitude
			}
			image.SetLocation(newLat, newLng)
		}

		// 5. 更新自定义元数据
		if req.Metadata != nil {
			if err := s.updateImageMetadata(ctx, image, req.Metadata); err != nil {
				return nil, fmt.Errorf("更新自定义元数据失败: %w", err)
			}
		}

		// 6. 更新标签信息
		if req.Tags != nil {
			if err := s.updateImageTags(ctx, image, req.Tags); err != nil {
				return nil, fmt.Errorf("更新标签失败: %w", err)
			}
		}

		// 7. 保存更新
		if err := s.repo.Update(ctx, image); err != nil {
			return nil, fmt.Errorf("保存图片更新失败: %w", err)
		}

		updatedImages = append(updatedImages, imageID)
	}

	logger.Info("批量更新图片元数据成功",
		zap.Int("count", len(updatedImages)),
		zap.Any("image_ids", req.ImageIDs))

	return updatedImages, nil
}

// GetImagesWithLocation 获取带有地理位置的图片
func (s *imageService) GetImagesWithLocation(ctx context.Context) ([]*model.ImageVO, error) {
	images, err := s.repo.GetImagesWithLocation(ctx)
	if err != nil {
		return nil, err
	}
	return s.storage.ToVOList(images), nil
}

// GetClusters 获取图片聚合数据
func (s *imageService) GetClusters(ctx context.Context, minLat, maxLat, minLng, maxLng float64, zoom int) ([]*model.ClusterResultVO, error) {
	// 根据缩放级别计算网格大小
	// 这是一个启发式策略，可以根据实际效果调整
	gridSize := 0.5 // 默认值

	if zoom >= 18 {
		gridSize = 0.0002 // 约20米
	} else if zoom >= 16 {
		gridSize = 0.001 // 约100米
	} else if zoom >= 14 {
		gridSize = 0.005 // 约500米
	} else if zoom >= 12 {
		gridSize = 0.02 // 约2公里
	} else if zoom >= 10 {
		gridSize = 0.1 // 约10公里
	} else if zoom >= 8 {
		gridSize = 0.5 // 约50公里
	} else if zoom >= 6 {
		gridSize = 2.0 // 约200公里
	} else {
		gridSize = 10.0 // 很大
	}

	// 计算中心纬度，用于调整经度网格大小（解决投影变形问题）
	centerLat := (minLat + maxLat) / 2.0
	// 将纬度转换为弧度
	radLat := centerLat * math.Pi / 180.0
	// 计算经度方向的网格大小
	// gridSizeLng = gridSizeLat / cos(lat)
	// 注意：cos(lat) 可能接近0（极地），需要做边界处理
	cosLat := math.Abs(math.Cos(radLat))
	if cosLat < 0.0001 {
		cosLat = 0.0001
	}
	gridSizeLng := gridSize / cosLat

	clusters, err := s.repo.GetClusters(ctx, minLat, maxLat, minLng, maxLng, gridSize, gridSizeLng)
	if err != nil {
		return nil, err
	}

	// 转换为 VO
	result := make([]*model.ClusterResultVO, 0, len(clusters))
	for _, c := range clusters {
		vo := &model.ClusterResultVO{
			MinLat:    c.MinLat,
			MaxLat:    c.MaxLat,
			MinLng:    c.MinLng,
			MaxLng:    c.MaxLng,
			Latitude:  c.Latitude,
			Longitude: c.Longitude,
			Count:     c.Count,
		}
		if c.CoverImage != nil {
			vo.CoverImage = s.storage.ToVO(c.CoverImage)
		}
		result = append(result, vo)
	}
	return result, nil
}

// GetClusterImages 获取指定聚合组内的图片（分页）
func (s *imageService) GetClusterImages(ctx context.Context, minLat, maxLat, minLng, maxLng float64, page, pageSize int) ([]*model.ImageVO, int64, error) {
	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	images, total, err := s.repo.GetClusterImages(ctx, minLat, maxLat, minLng, maxLng, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	vos := s.storage.ToVOList(images)
	return vos, total, err
}

// GetGeoBounds 获取所有带坐标图片的地理边界
func (s *imageService) GetGeoBounds(ctx context.Context) (*model.GeoBounds, error) {
	return s.repo.GetGeoBounds(ctx)
}

// ListDeleted 获取已删除的图片列表
func (s *imageService) ListDeleted(ctx context.Context, page, pageSize int) ([]*model.ImageVO, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	images, total, err := s.repo.ListDeleted(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	vos := s.storage.ToVOList(images)
	return vos, total, err
}

// RestoreImages 恢复已删除的图片
func (s *imageService) RestoreImages(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	logger.Info("恢复已删除的图片",
		zap.Int("count", len(ids)),
		zap.Any("ids", ids))

	err := s.repo.RestoreBatch(ctx, ids)
	if err != nil {
		return err
	}

	s.notifyCount(ctx)
	return nil
}

func (s *imageService) notifyCount(ctx context.Context) {
	// 广播图片总数更新
	if s.notifier != nil {
		total, err := s.repo.Count(ctx)
		if err == nil {
			s.notifier.NotifyImageCount(total)
		}
	}
}

// PermanentlyDelete 彻底删除图片（包括物理文件）
func (s *imageService) PermanentlyDelete(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	// 获取要删除的图片信息
	images, err := s.repo.FindDeletedByIDs(ctx, ids)
	if err != nil {
		return fmt.Errorf("获取图片信息失败: %w", err)
	}

	// 收集文件路径信息（事务外获取，事务内删除数据库记录）
	pathsByType := make(map[model.StorageId][]string)
	var thumbnailPaths []string

	for _, image := range images {
		pathsByType[image.StorageId] = append(pathsByType[image.StorageId], image.StoragePath)
		if image.ThumbnailPath != "" {
			thumbnailPaths = append(thumbnailPaths, image.ThumbnailPath)
		}
	}

	// 使用事务删除数据库记录
	err = database.Transaction0(ctx, func(txCtx context.Context) error {
		// 1. 删除关联的向量嵌入数据
		if err := s.embeddingRepo.DeleteByImageIDs(txCtx, ids); err != nil {
			return fmt.Errorf("删除向量嵌入数据失败: %w", err)
		}

		// 2. 删除相册关联数据
		if err := s.albumRepo.RemoveImagesFromAllAlbums(txCtx, ids); err != nil {
			return fmt.Errorf("删除相册关联数据失败: %w", err)
		}

		// 3. 从数据库物理删除图片记录
		if err := s.repo.HardDeleteBatch(txCtx, ids); err != nil {
			return fmt.Errorf("物理删除数据库记录失败: %w", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 事务成功后删除物理文件（文件删除失败不影响数据一致性）
	for storageType, paths := range pathsByType {
		s.doDeleteBatch(ctx, storageType, paths)
	}

	if len(thumbnailPaths) > 0 {
		s.doDeleteBatch(ctx, model.StorageTypeLocal, thumbnailPaths)
	}

	logger.Info("彻底删除图片完成",
		zap.Int("count", len(images)),
		zap.Any("ids", ids))

	return nil
}

func (s *imageService) doDeleteBatch(ctx context.Context, storageId model.StorageId, paths []string) {
	results, err := s.storage.DeleteBatch(ctx, storageId, paths)
	if err != nil {
		logger.Warn("删除缩略图文件失败",
			zap.String("storage_type", string(storageId)),
			zap.Error(err))
	} else {
		for _, result := range results {
			if result.Error != nil {
				logger.Warn("删除缩略图文件失败",
					zap.String("storage_type", string(storageId)),
					zap.String("path", result.Path),
					zap.Error(result.Error))
			}
		}
	}
}

// CleanupExpiredTrash 清理过期的已删除图片
func (s *imageService) CleanupExpiredTrash(ctx context.Context) (int, error) {
	days := s.cfg.Trash.AutoDeleteDays
	if days <= 0 {
		return 0, nil // 不自动删除
	}

	// 查找过期的已删除图片
	images, err := s.repo.FindExpiredDeleted(ctx, days)
	if err != nil {
		return 0, fmt.Errorf("查找过期图片失败: %w", err)
	}

	if len(images) == 0 {
		return 0, nil
	}

	// 收集ID
	ids := make([]int64, len(images))
	for i, img := range images {
		ids[i] = img.ID
	}

	// 彻底删除
	if err := s.PermanentlyDelete(ctx, ids); err != nil {
		return 0, err
	}

	logger.Info("自动清理过期已删除图片完成",
		zap.Int("count", len(ids)),
		zap.Int("expire_days", days))

	return len(ids), nil
}

func (s *imageService) Repo() repository.ImageRepository {
	return s.repo
}

// GetAllNormalTags 获取所有普通标签

// SearchTags 搜索标签
func (s *imageService) SearchTags(ctx context.Context, keyword string, limit int) ([]*model.Tag, error) {
	return s.repo.SearchTags(ctx, keyword, limit)
}

// GetPopularTags 获取热门标签
func (s *imageService) GetPopularTags(ctx context.Context, limit int) ([]*model.Tag, error) {
	return s.repo.GetPopularTags(ctx, limit)
}
