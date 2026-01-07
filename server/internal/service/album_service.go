package service

import (
	"context"
	"fmt"
	"gallary/server/pkg/database"

	"gallary/server/internal/model"
	"gallary/server/internal/repository"
	"gallary/server/internal/storage"
	"gallary/server/pkg/logger"

	"go.uber.org/zap"
)

// AlbumService 相册服务接口
type AlbumService interface {
	Create(ctx context.Context, req *CreateAlbumRequest) (*model.AlbumVO, error)
	List(ctx context.Context, page, pageSize int, isSmart *bool) ([]*model.AlbumVO, int64, error)
	Update(ctx context.Context, id int64, req *UpdateAlbumRequest) (*model.AlbumVO, error)
	Delete(ctx context.Context, id int64) error
	Copy(ctx context.Context, id int64) (*model.AlbumVO, error)

	// 图片管理
	GetImages(ctx context.Context, albumID int64, page, pageSize int) ([]*model.ImageVO, int64, error)
	AddImages(ctx context.Context, albumID int64, imageIDs []int64) error
	RemoveImages(ctx context.Context, albumID int64, imageIDs []int64) error
	SetCover(ctx context.Context, albumID int64, imageID int64) error
	RemoveCover(ctx context.Context, albumID int64) error
	SetAverageCover(ctx context.Context, albumID int64, modelName string) error
}

// CreateAlbumRequest 创建相册请求
type CreateAlbumRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
}

// UpdateAlbumRequest 更新相册请求
type UpdateAlbumRequest struct {
	Name         *string `json:"name,omitempty"`
	CoverImageID *int64  `json:"cover_image_id,omitempty"`
	Description  *string `json:"description,omitempty"`
	SortOrder    *int    `json:"sort_order,omitempty"`
}

type albumService struct {
	repo    repository.AlbumRepository
	storage *storage.StorageManager
}

// NewAlbumService 创建相册服务实例
func NewAlbumService(repo repository.AlbumRepository, storage *storage.StorageManager) AlbumService {
	return &albumService{
		repo:    repo,
		storage: storage,
	}
}

// Create 创建相册
func (s *albumService) Create(ctx context.Context, req *CreateAlbumRequest) (*model.AlbumVO, error) {
	album := &model.Tag{
		Name: req.Name,
		Type: model.TagTypeAlbum,
	}

	if req.Description != nil {
		album.Metadata = &model.AlbumMetadata{
			Description: req.Description,
		}
	}

	if err := s.repo.Create(ctx, album); err != nil {
		return nil, fmt.Errorf("创建相册失败: %w", err)
	}

	logger.Info("创建相册成功", zap.Int64("id", album.ID), zap.String("name", album.Name))
	return album.ToAlbumVO(nil, 0), nil
}

// List 获取相册列表
// isSmart: nil-不过滤, true-只返回智能相册, false-只返回普通相册
func (s *albumService) List(ctx context.Context, page, pageSize int, isSmart *bool) ([]*model.AlbumVO, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	albums, total, err := s.repo.List(ctx, page, pageSize, isSmart)
	if err != nil {
		return nil, 0, err
	}

	// 批量获取图片数量和封面
	albumIDs := make([]int64, 0, len(albums))
	for _, album := range albums {
		albumIDs = append(albumIDs, album.ID)
	}

	// 批量获取图片数量
	countMap, err := s.repo.GetImageCounts(ctx, albumIDs)
	if err != nil {
		logger.Warn("批量获取图片数量失败", zap.Error(err))
		countMap = make(map[int64]int64)
	}

	// 批量获取第一张图片作为默认封面
	firstImagesMap, err := s.repo.GetFirstImages(ctx, albumIDs)
	if err != nil {
		logger.Warn("批量获取默认封面失败", zap.Error(err))
		firstImagesMap = make(map[int64]*model.Image)
	}

	vos := make([]*model.AlbumVO, 0, len(albums))
	for _, album := range albums {
		count := countMap[album.ID]

		var coverImage *model.ImageVO

		// 优先使用设置的封面
		if album.Metadata != nil && album.Metadata.CoverImageID != nil {
			img, err := s.repo.GetCoverImage(ctx, *album.Metadata.CoverImageID)
			if err == nil && img != nil {
				coverImage = s.storage.ToVO(img)
			}
		}

		// 如果没有设置封面，使用第一张图片
		if coverImage == nil {
			if img, ok := firstImagesMap[album.ID]; ok {
				coverImage = s.storage.ToVO(img)
			}
		}

		vos = append(vos, album.ToAlbumVO(coverImage, count))
	}

	return vos, total, nil
}

// Update 更新相册
func (s *albumService) Update(ctx context.Context, id int64, req *UpdateAlbumRequest) (*model.AlbumVO, error) {
	album, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		album.Name = *req.Name
	}

	if album.Metadata == nil {
		album.Metadata = &model.AlbumMetadata{}
	}

	if req.CoverImageID != nil {
		album.Metadata.CoverImageID = req.CoverImageID
	}
	if req.Description != nil {
		album.Metadata.Description = req.Description
	}
	if req.SortOrder != nil {
		album.Metadata.SortOrder = *req.SortOrder
	}

	if err := s.repo.Update(ctx, album); err != nil {
		return nil, fmt.Errorf("更新相册失败: %w", err)
	}

	return s.toVO(ctx, album)
}

// Delete 删除相册
func (s *albumService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// Copy 复制相册
func (s *albumService) Copy(ctx context.Context, id int64) (*model.AlbumVO, error) {
	// 获取原相册
	originalAlbum, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取原相册失败: %w", err)
	}

	// 创建新相册
	newAlbum := &model.Tag{
		Name: originalAlbum.Name + " (副本)",
		Type: model.TagTypeAlbum,
	}

	// 复制元数据（不复制封面）
	if originalAlbum.Metadata != nil {
		newAlbum.Metadata = &model.AlbumMetadata{
			Description: originalAlbum.Metadata.Description,
		}
	}

	if err := s.repo.Create(ctx, newAlbum); err != nil {
		return nil, fmt.Errorf("创建相册副本失败: %w", err)
	}

	// 复制相册内的图片关联
	if err := s.repo.CopyImages(ctx, id, newAlbum.ID); err != nil {
		logger.Warn("复制相册图片关联失败", zap.Error(err))
	}

	logger.Info("复制相册成功", zap.Int64("original_id", id), zap.Int64("new_id", newAlbum.ID))
	return s.toVO(ctx, newAlbum)
}

// GetImages 获取相册内图片
func (s *albumService) GetImages(ctx context.Context, albumID int64, page, pageSize int) ([]*model.ImageVO, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	images, total, err := s.repo.GetImages(ctx, albumID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	vos := s.storage.ToVOList(images)
	return vos, total, nil
}

// AddImages 添加图片到相册
func (s *albumService) AddImages(ctx context.Context, albumID int64, imageIDs []int64) error {
	// 验证相册存在
	_, err := s.repo.FindByID(ctx, albumID)
	if err != nil {
		return err
	}

	return database.Transaction0(ctx, func(ctx context.Context) error {
		return s.repo.AddImages(ctx, albumID, imageIDs)
	})
}

// RemoveImages 从相册移除图片
func (s *albumService) RemoveImages(ctx context.Context, albumID int64, imageIDs []int64) error {
	return s.repo.RemoveImages(ctx, albumID, imageIDs)
}

// SetCover 设置相册封面
func (s *albumService) SetCover(ctx context.Context, albumID int64, imageID int64) error {
	album, err := s.repo.FindByID(ctx, albumID)
	if err != nil {
		return err
	}

	if album.Metadata == nil {
		album.Metadata = &model.AlbumMetadata{}
	}
	album.Metadata.CoverImageID = &imageID

	return s.repo.Update(ctx, album)
}

// RemoveCover 移除相册自定义封面
func (s *albumService) RemoveCover(ctx context.Context, albumID int64) error {
	album, err := s.repo.FindByID(ctx, albumID)
	if err != nil {
		return err
	}

	if album.Metadata == nil {
		return nil // 没有设置封面，无需移除
	}

	album.Metadata.CoverImageID = nil
	return s.repo.Update(ctx, album)
}

// SetAverageCover 设置平均向量封面
func (s *albumService) SetAverageCover(ctx context.Context, albumID int64, modelName string) error {
	// 获取最适合的封面图片ID
	bestImageID, err := s.repo.FindBestCoverByAverageVector(ctx, albumID, modelName)
	if err != nil {
		return fmt.Errorf("计算平均向量封面失败: %w", err)
	}

	// 设置封面
	return s.SetCover(ctx, albumID, bestImageID)
}

// toVO 转换为 AlbumVO
func (s *albumService) toVO(ctx context.Context, album *model.Tag) (*model.AlbumVO, error) {
	count, err := s.repo.GetImageCount(ctx, album.ID)
	if err != nil {
		return nil, err
	}

	var coverImage *model.ImageVO

	// 优先使用设置的封面
	if album.Metadata != nil && album.Metadata.CoverImageID != nil {
		img, err := s.repo.GetCoverImage(ctx, *album.Metadata.CoverImageID)
		if err == nil && img != nil {
			coverImage = s.storage.ToVO(img)
		}
	}

	// 如果没有设置封面，使用美学评分最高的图片，如果没有评分则使用第一张
	if coverImage == nil && count > 0 {
		firstImagesMap, err := s.repo.GetFirstImages(ctx, []int64{album.ID})
		if err == nil {
			if img, ok := firstImagesMap[album.ID]; ok {
				coverImage = s.storage.ToVO(img)
			}
		}
	}

	return album.ToAlbumVO(coverImage, count), nil
}
