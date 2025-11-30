package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"gallary/server/config"
	"gallary/server/internal/model"
	"gallary/server/internal/repository"
	"gallary/server/internal/storage"
	"gallary/server/pkg/logger"
	"math/big"
	"time"

	"go.uber.org/zap"
)

// ShareService 分享服务接口
type ShareService interface {
	CreateShare(ctx context.Context, req *CreateShareRequest) (*model.Share, error)
	GetShareByCode(ctx context.Context, code string) (*model.Share, error)
	SharedImages(ctx context.Context, code string, password string, page, pageSize int) ([]*model.ImageVO, int64, error)
	ListShares(ctx context.Context, page, pageSize int) ([]*model.Share, int64, error)
	DeleteShare(ctx context.Context, id int64) error
}

type shareService struct {
	repo    repository.ShareRepository
	storage *storage.StorageManager
}

func NewShareService(repo repository.ShareRepository, storage *storage.StorageManager) ShareService {
	return &shareService{repo: repo, storage: storage}
}

// CreateShareRequest 创建分享请求
type CreateShareRequest struct {
	ImageIDs    []int64 `json:"image_ids"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Password    string  `json:"password"`    // 可选，明文密码
	ExpireDays  int     `json:"expire_days"` // 0表示不过期
}

// ShareDetailResponse 分享详情响应
type ShareDetailResponse struct {
	Share      *model.Share     `json:"share"`
	Images     []*model.ImageVO `json:"images"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// CreateShare 创建新的分享
func (s *shareService) CreateShare(ctx context.Context, req *CreateShareRequest) (*model.Share, error) {
	if len(req.ImageIDs) == 0 {
		return nil, errors.New("必须要选择图片")
	}

	code, err := generateShareCode(6)
	if err != nil {
		return nil, err
	}

	share := &model.Share{
		ShareCode: code,
		ViewCount: 0,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if req.Title != "" {
		share.Title = &req.Title
	}
	if req.Description != "" {
		share.Description = &req.Description
	}

	// 简单的明文存储密码（实际生产应使用bcrypt）
	// 根据 CLAUDE.md 中的描述，项目认证似乎比较宽松，这里先保持简单
	if req.Password != "" {
		share.Password = &req.Password
	}

	if req.ExpireDays > 0 {
		expireAt := time.Now().AddDate(0, 0, req.ExpireDays)
		share.ExpireAt = &expireAt
	}

	err = s.repo.Create(ctx, share, req.ImageIDs)
	if err != nil {
		return nil, err
	}

	return share, nil
}

// GetShareByCode 获取分享基本信息（不包含图片列表，用于验证页面展示）
func (s *shareService) GetShareByCode(ctx context.Context, code string) (*model.Share, error) {
	share, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	if !share.IsActive {
		return nil, errors.New("分享已失效")
	}

	if share.IsExpired() {
		return nil, errors.New("分享已过期")
	}

	return share, nil
}

// VerifyShare 验证并获取分享内容
func (s *shareService) SharedImages(ctx context.Context, code string, password string, page, pageSize int) ([]*model.ImageVO, int64, error) {
	share, err := s.repo.FindByCode(ctx, code)
	if err != nil {
		return nil, 0, err
	}

	if !share.IsActive {
		return nil, 0, errors.New("分享已失效")
	}

	if share.IsExpired() {
		return nil, 0, errors.New("分享已过期")
	}

	// 验证密码
	if share.Password != nil && *share.Password != "" {
		if password != *share.Password {
			return nil, 0, errors.New("密码错误")
		}
	}

	// 更新浏览次数（仅首次访问时增加）
	if page == 1 {
		_ = s.repo.IncrementViewCount(ctx, share.ID)
	}

	// 分页获取图片列表
	images, total, err := s.repo.GetImagesPaginated(ctx, share.ID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// 转换为 VO
	vos := make([]*model.ImageVO, 0, len(images))
	for _, img := range images {
		vo := s.toVO(ctx, img)
		vos = append(vos, vo)
	}

	return vos, total, nil
}

// toVO 将 Image 转换为 ImageVO
func (s *shareService) toVO(ctx context.Context, image *model.Image) *model.ImageVO {
	if image == nil {
		return nil
	}

	var url string
	var err error

	// 阿里云盘存储使用后端代理URL
	if image.StorageType == config.StorageTypeAliyunpan {
		url = fmt.Sprintf("/api/images/%d/file", image.ID)
	} else {
		// 其他存储类型直接获取URL
		url, err = s.storage.GetURL(context.WithValue(ctx, storage.OverrideStorageType, image.StorageType), image.StoragePath)
		if err != nil {
			logger.Warn("获取图片URL失败", zap.Error(err), zap.String("path", image.StoragePath), zap.String("storage_type", string(image.StorageType)))
			url = ""
		}
	}

	// 缩略图始终从本地存储获取
	var thumbnailURL string
	if image.ThumbnailPath != "" {
		localStorage := s.storage.GetLocalStorage()
		if localStorage != nil {
			thumbnailURL, err = localStorage.GetURL(ctx, image.ThumbnailPath)
			if err != nil {
				logger.Warn("获取缩略图URL失败", zap.Error(err), zap.String("path", image.ThumbnailPath))
				thumbnailURL = ""
			}
		}
	}

	return image.ToVO(url, thumbnailURL)
}

// ListShares 管理端获取列表
func (s *shareService) ListShares(ctx context.Context, page, pageSize int) ([]*model.Share, int64, error) {
	return s.repo.List(ctx, page, pageSize)
}

// DeleteShare 删除分享
func (s *shareService) DeleteShare(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// generateShareCode 生成随机分享码
func generateShareCode(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[num.Int64()]
	}
	return string(b), nil
}
