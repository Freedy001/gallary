package storage

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"gallary/server/internal/model"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"gallary/server/pkg/logger"

	"github.com/google/uuid"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan_web"
	"go.uber.org/zap"
)

// AliyunPanStorage 阿里云盘存储实现
type AliyunPanStorage struct {
	client     *aliyunpan_web.WebPanClient
	httpClient *http.Client // 复用 HTTP Client 以利用连接池
	basePath   string       // 基础存储路径
	driveId    string       // 网盘 ID
	mu         sync.RWMutex
	userInfo   *aliyunpan.UserInfo

	// Token 管理
	refreshToken string
	webToken     aliyunpan_web.WebLoginToken
	appConfig    aliyunpan_web.AppConfig

	// 下载配置
	downloadChunkSize   int64 // 下载分片大小 (字节)
	downloadConcurrency int   // 下载并发数
}

// NewAliyunPanStorage 创建阿里云盘存储实例
func NewAliyunPanStorage(cfg *model.AliyunPanStorageConfig) (*AliyunPanStorage, error) {
	if cfg.RefreshToken == "" {
		return nil, fmt.Errorf("阿里云盘 refresh_token 不能为空")
	}

	// 1. 使用 RefreshToken 获取 AccessToken
	webToken, err := aliyunpan_web.GetAccessTokenFromRefreshToken(cfg.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("获取阿里云盘 AccessToken 失败: %w", err)
	}

	// 2. 生成设备 ID
	deviceId := generateDeviceId()

	// 3. 创建 AppConfig
	appConfig := aliyunpan_web.AppConfig{
		AppId:    "25dzX3vbYqktVxyX",
		DeviceId: deviceId,
		UserId:   "",
		Nonce:    0,
	}

	// 4. 创建 WebPanClient
	client := aliyunpan_web.NewWebPanClient(
		*webToken,
		aliyunpan_web.AppLoginToken{},
		appConfig,
		aliyunpan_web.SessionConfig{
			DeviceName: "GalleryServer",
			ModelName:  "Linux服务器",
		},
	)

	// 5. 获取用户信息
	userInfo, apiErr := client.GetUserInfo()
	if apiErr != nil {
		return nil, fmt.Errorf("获取阿里云盘用户信息失败: %s", apiErr.Error())
	}

	// 6. 更新 userId 并创建会话
	appConfig.UserId = userInfo.UserId
	client.UpdateAppConfig(appConfig)

	_, apiErr = client.CreateSession(nil)
	if apiErr != nil {
		return nil, fmt.Errorf("创建阿里云盘会话失败: %s", apiErr.Error())
	}

	// 7. 根据 DriveType 选择网盘 ID
	driveId := userInfo.FileDriveId // 默认使用备份盘
	switch cfg.DriveType {
	case "album":
		driveId = userInfo.AlbumDriveId
	case "resource":
		driveId = userInfo.ResourceDriveId
	case "file", "":
		driveId = userInfo.FileDriveId
	}

	if driveId == "" {
		return nil, fmt.Errorf("无法获取网盘 ID，请检查 drive_type 配置")
	}

	// 8. 确保基础目录存在
	basePath := cfg.BasePath
	if basePath == "" {
		basePath = "/gallery/images"
	}
	basePath = formatPath(basePath)

	_, apiErr = client.MkdirByFullPath(driveId, basePath)
	if apiErr != nil {
		// 目录可能已存在，忽略错误
	}

	// 初始化复用的 HTTP Client
	sharedHttpClient := &http.Client{
		Timeout: 30 * time.Minute, // 默认较长超时，具体超时由 Context 控制
		Transport: &http.Transport{
			MaxIdleConns:        100,              // 最大空闲连接数
			MaxIdleConnsPerHost: 20,               // 每个 Host 的最大空闲连接数
			IdleConnTimeout:     90 * time.Second, // 空闲超时
		},
	}

	// 处理下载配置，使用默认值
	downloadChunkSize := int64(512 * 1024) // 默认 512KB
	if cfg.DownloadChunkSize > 0 {
		downloadChunkSize = int64(cfg.DownloadChunkSize) * 1024 // 配置以 KB 为单位
	}
	downloadConcurrency := 8 // 默认 8
	if cfg.DownloadConcurrency > 0 {
		downloadConcurrency = cfg.DownloadConcurrency
	}

	storage := &AliyunPanStorage{
		client:              client,
		httpClient:          sharedHttpClient,
		basePath:            basePath,
		driveId:             driveId,
		userInfo:            userInfo,
		refreshToken:        cfg.RefreshToken,
		webToken:            *webToken,
		appConfig:           appConfig,
		downloadChunkSize:   downloadChunkSize,
		downloadConcurrency: downloadConcurrency,
	}

	return storage, nil
}

// ensureValidToken 确保 Token 有效
func (s *AliyunPanStorage) ensureValidToken() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.webToken.IsAccessTokenExpired() {
		newToken, apiErr := aliyunpan_web.GetAccessTokenFromRefreshToken(s.refreshToken)
		if apiErr != nil {
			return fmt.Errorf("刷新 Token 失败: %s", apiErr.Error())
		}
		s.webToken = *newToken
		s.client.UpdateToken(*newToken)

		// 重新创建会话
		_, apiErr = s.client.CreateSession(nil)
		if apiErr != nil {
			return fmt.Errorf("重新创建会话失败: %s", apiErr.Error())
		}
	}
	return nil
}

func (s *AliyunPanStorage) GetType(ctx context.Context) model.StorageId {
	return model.AliyunpanStorageId(s.userInfo.UserId)
}

// Upload 上传文件到阿里云盘
func (s *AliyunPanStorage) Upload(ctx context.Context, file io.Reader, filePath string) (string, error) {
	if err := s.ensureValidToken(); err != nil {
		return "", err
	}

	// 构建完整路径
	fullPath := path.Join(s.basePath, filePath)
	dirPath := path.Dir(fullPath)
	fileName := path.Base(fullPath)

	// 确保目录存在
	dirResult, apiErr := s.client.MkdirByFullPath(s.driveId, dirPath)
	if apiErr != nil {
		return "", fmt.Errorf("创建目录失败: %s", apiErr.Error())
	}
	parentFileId := dirResult.FileId
	if parentFileId == "" {
		parentFileId = aliyunpan.DefaultRootParentFileId
	}

	// 读取文件内容到内存
	fileData, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("读取文件内容失败: %w", err)
	}

	fileSize := int64(len(fileData))

	// 计算 SHA1 哈希
	contentHash := ""
	if fileSize > 0 {
		hash := sha1.New()
		hash.Write(fileData)
		contentHash = strings.ToUpper(hex.EncodeToString(hash.Sum(nil)))
	} else {
		contentHash = aliyunpan.DefaultZeroSizeFileContentHash
	}

	// 尝试创建上传任务
	var createResult *aliyunpan.CreateFileUploadResult

	// 首先尝试正常上传
	createParam := &aliyunpan.CreateFileUploadParam{
		Name:            fileName,
		DriveId:         s.driveId,
		ParentFileId:    parentFileId,
		Size:            fileSize,
		ContentHash:     contentHash,
		ContentHashName: "sha1",
		CheckNameMode:   "auto_rename",
		BlockSize:       aliyunpan.DefaultChunkSize,
	}

	createResult, apiErr = s.client.CreateUploadFile(createParam)

	// 如果遇到 Rapid proof 错误，尝试使用不同的哈希或强制完整上传
	if apiErr != nil && strings.Contains(apiErr.Error(), "Rapid proof needed") {
		logger.Warn("遇到 Rapid proof 错误，尝试强制完整上传", zap.String("file", fileName))

		// 尝试使用空哈希强制完整上传
		createParam.ContentHash = ""
		createParam.ContentHashName = ""

		createResult, apiErr = s.client.CreateUploadFile(createParam)
		if apiErr != nil {
			return "", fmt.Errorf("重新创建上传任务失败: %s", apiErr.Error())
		}
	} else if apiErr != nil {
		return "", fmt.Errorf("创建上传任务失败: %s", apiErr.Error())
	}

	// 检查是否秒传成功
	if createResult.RapidUpload {
		return filePath, nil
	}

	// 分片上传
	if fileSize > 0 {
		chunkSize := aliyunpan.DefaultChunkSize
		offset := int64(0)

		for i, partInfo := range createResult.PartInfoList {
			// 计算当前分片大小
			currentChunkSize := chunkSize
			if offset+currentChunkSize > fileSize {
				currentChunkSize = fileSize - offset
			}

			// 获取分片数据
			chunkData := fileData[offset : offset+currentChunkSize]

			// 上传分片
			uploadData := &aliyunpan.FileUploadChunkData{
				Reader:    bytes.NewReader(chunkData),
				ChunkSize: currentChunkSize,
			}

			apiErr = s.client.UploadDataChunk(partInfo.UploadURL, uploadData)
			if apiErr != nil {
				return "", fmt.Errorf("上传分片 %d 失败: %s", i+1, apiErr.Error())
			}

			offset += currentChunkSize
		}
	}

	// 完成上传
	completeParam := &aliyunpan.CompleteUploadFileParam{
		DriveId:  s.driveId,
		FileId:   createResult.FileId,
		UploadId: createResult.UploadId,
	}

	_, apiErr = s.client.CompleteUploadFile(completeParam)
	if apiErr != nil {
		return "", fmt.Errorf("完成上传失败: %s", apiErr.Error())
	}

	return filePath, nil
}

// 分片下载配置 - 预缓冲分片数量
const downloadBufferCount = 16

// streamingChunkReader 流式分片下载的 ReadCloser 实现
// 使用生产者-消费者模式，边下载边输出，减少内存占用
type streamingChunkReader struct {
	ctx        context.Context
	cancel     context.CancelFunc
	chunkChan  chan chunkData // 有序分片通道
	currentBuf []byte         // 当前正在读取的分片
	bufOffset  int            // 当前分片读取偏移
	err        error          // 下载过程中的错误
	closed     bool
	mu         sync.Mutex
}

type chunkData struct {
	index int
	data  []byte
	err   error
}

func (r *streamingChunkReader) Read(p []byte) (n int, err error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		logger.Debug("streamingChunkReader: 已关闭，返回 EOF")
		return 0, io.EOF
	}
	if r.err != nil {
		err := r.err
		r.mu.Unlock()
		logger.Debug("streamingChunkReader: 存在错误", zap.Error(err))
		return 0, err
	}

	// 如果当前缓冲区还有数据，直接读取
	if r.bufOffset < len(r.currentBuf) {
		n = copy(p, r.currentBuf[r.bufOffset:])
		r.bufOffset += n
		r.mu.Unlock()
		return n, nil
	}
	r.mu.Unlock()

	// 当前缓冲区读完，获取下一个分片（不持有锁）
	select {
	case chunk, ok := <-r.chunkChan:
		if !ok {
			// 通道关闭，所有数据已读取完毕
			logger.Debug("streamingChunkReader: 通道关闭，返回 EOF")
			return 0, io.EOF
		}
		if chunk.err != nil {
			r.mu.Lock()
			r.err = chunk.err
			r.mu.Unlock()
			logger.Error("streamingChunkReader: 分片下载错误", zap.Error(chunk.err))
			return 0, chunk.err
		}
		// logger.Debug("streamingChunkReader: 收到分片", zap.Int("index", chunk.index), zap.Int("size", len(chunk.data)))
		r.mu.Lock()
		r.currentBuf = chunk.data
		r.bufOffset = 0
		n = copy(p, r.currentBuf[r.bufOffset:])
		r.bufOffset += n
		r.mu.Unlock()
		return n, nil
	case <-r.ctx.Done():
		r.mu.Lock()
		r.err = r.ctx.Err()
		err := r.err
		r.mu.Unlock()
		logger.Debug("streamingChunkReader: 上下文取消", zap.Error(err))
		return 0, err
	}
}

func (r *streamingChunkReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}
	r.closed = true
	r.cancel()

	// 排空通道
	go func() {
		for range r.chunkChan {
		}
	}()

	return nil
}

// Download 从阿里云盘下载文件（支持流式分片多线程下载）
func (s *AliyunPanStorage) Download(ctx context.Context, filePath string) (io.ReadCloser, error) {
	if err := s.ensureValidToken(); err != nil {
		return nil, err
	}

	fullPath := path.Join(s.basePath, filePath)

	// 获取文件信息
	fileInfo, apiErr := s.client.FileInfoByPath(s.driveId, fullPath)
	if apiErr != nil {
		return nil, fmt.Errorf("获取文件信息失败: %s", apiErr.Error())
	}

	if fileInfo.IsFolder() {
		return nil, fmt.Errorf("路径是文件夹，不是文件: %s", filePath)
	}

	// 获取下载 URL
	downloadResult, apiErr := s.client.GetFileDownloadUrl(&aliyunpan.GetFileDownloadUrlParam{
		DriveId: s.driveId,
		FileId:  fileInfo.FileId,
	})
	if apiErr != nil {
		return nil, fmt.Errorf("获取下载链接失败: %s", apiErr.Error())
	}

	if downloadResult.Url == "" {
		return nil, fmt.Errorf("下载链接为空")
	}

	// 检查是否为非法资源
	if strings.HasPrefix(downloadResult.Url, aliyunpan.IllegalDownloadUrlPrefix) {
		return nil, fmt.Errorf("资源已被屏蔽")
	}

	fileSize := fileInfo.FileSize

	// 小文件使用单线程下载
	if fileSize <= s.downloadChunkSize {
		return s.downloadSingle(ctx, downloadResult.Url)
	}

	// 大文件使用流式分片多线程下载
	return s.downloadStreamingMultiThread(ctx, downloadResult.Url, fileSize)
}

// downloadSingle 单线程下载（适用于小文件）
func (s *AliyunPanStorage) downloadSingle(ctx context.Context, url string) (io.ReadCloser, error) {
	// 复用 shared client
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建下载请求失败: %w", err)
	}
	req.Header.Set("Referer", "https://www.aliyundrive.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载文件失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		resp.Body.Close()
		return nil, fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// downloadStreamingMultiThread 流式多线程分片下载
// 使用有序输出通道，边下载边写回，减少内存占用
func (s *AliyunPanStorage) downloadStreamingMultiThread(ctx context.Context, url string, fileSize int64) (io.ReadCloser, error) {
	// 创建带取消的上下文
	downloadCtx, cancel := context.WithCancel(ctx)

	// 有序输出通道（缓冲一些分片以平滑输出）
	outputChan := make(chan chunkData, downloadBufferCount)

	reader := &streamingChunkReader{
		ctx:       downloadCtx,
		cancel:    cancel,
		chunkChan: outputChan,
	}

	// 启动下载协调器
	go s.coordinateDownload(downloadCtx, cancel, url, fileSize, outputChan)

	return reader, nil
}

// coordinateDownload 协调多线程下载并有序输出
// 优化：移除了冗余的锁，利用单 goroutine 优势进行排序
func (s *AliyunPanStorage) coordinateDownload(
	ctx context.Context,
	cancel context.CancelFunc,
	url string,
	fileSize int64,
	outputChan chan<- chunkData,
) {
	defer close(outputChan)
	// 注意：不能在这里 defer cancel()！
	// 关闭 outputChan 会让 reader.Read() 返回 EOF，io.Copy 会正常结束
	// 如果这里 cancel，会导致 reader 在还没读完所有数据时就收到 context canceled 错误

	chunkCount := (fileSize + s.downloadChunkSize - 1) / s.downloadChunkSize

	logger.Info("开始流式多线程下载",
		zap.Int64("fileSize", fileSize),
		zap.Int64("chunkCount", chunkCount),
		zap.Int("concurrency", s.downloadConcurrency))

	// 分片结果缓存（用于乱序下载后有序输出）
	// 注意：此 Map 仅在当前 Goroutine 访问，无需加锁
	resultCache := make(map[int][]byte)
	nextOutput := 0 // 下一个要输出的分片索引

	// 下载任务通道
	taskChan := make(chan int64, chunkCount)
	resultChan := make(chan chunkData, s.downloadConcurrency*2)

	// 启动下载工作池
	var wg sync.WaitGroup
	for i := 0; i < s.downloadConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case chunkIndex, ok := <-taskChan:
					if !ok {
						return
					}
					// 计算分片范围
					start := chunkIndex * s.downloadChunkSize
					end := start + s.downloadChunkSize - 1
					if end >= fileSize {
						end = fileSize - 1
					}

					// 下载分片
					data, err := s.downloadChunkData(ctx, url, start, end)
					select {
					case resultChan <- chunkData{index: int(chunkIndex), data: data, err: err}:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	// 发送所有下载任务
	go func() {
		for i := int64(0); i < chunkCount; i++ {
			select {
			case taskChan <- i:
			case <-ctx.Done():
				close(taskChan)
				return
			}
		}
		close(taskChan)
	}()

	// 等待所有下载完成后关闭结果通道
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果并有序输出
	for result := range resultChan {
		if result.err != nil {
			// 发送错误，取消其他下载任务并退出
			cancel() // 只在出错时 cancel
			select {
			case outputChan <- result:
			case <-ctx.Done():
			}
			return
		}

		// 存入缓存
		resultCache[result.index] = result.data

		// 尝试有序输出已准备好的连续分片
		for {
			if data, ok := resultCache[nextOutput]; ok {
				// 找到了下一个需要的数据
				delete(resultCache, nextOutput) // 用完即删，释放 map 引用

				select {
				case outputChan <- chunkData{index: nextOutput, data: data}:
					nextOutput++
				case <-ctx.Done():
					return
				}
			} else {
				// 缺少中间分片，停止输出，等待更多结果
				break
			}
		}
	}

	logger.Info("流式多线程下载完成", zap.Int64("totalChunks", chunkCount))
}

// downloadChunkData 下载指定范围的数据
func (s *AliyunPanStorage) downloadChunkData(ctx context.Context, url string, start, end int64) ([]byte, error) {
	// 使用带超时的 Context 控制单次请求
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Referer", "https://www.aliyundrive.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	// 复用 shared client
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	return data, nil
}

// Delete 删除阿里云盘文件（移至回收站）
func (s *AliyunPanStorage) Delete(ctx context.Context, filePath string) error {
	if err := s.ensureValidToken(); err != nil {
		return err
	}

	fullPath := path.Join(s.basePath, filePath)

	// 获取文件信息
	fileInfo, apiErr := s.client.FileInfoByPath(s.driveId, fullPath)
	if apiErr != nil {
		// 如果文件不存在，视为删除成功
		if strings.Contains(apiErr.Error(), "文件不存在") {
			return nil
		}
		return fmt.Errorf("获取文件信息失败: %s", apiErr.Error())
	}

	// 删除文件（移至回收站）
	deleteParam := []*aliyunpan.FileBatchActionParam{
		{
			DriveId: s.driveId,
			FileId:  fileInfo.FileId,
		},
	}

	results, apiErr := s.client.FileDelete(deleteParam)
	if apiErr != nil {
		return fmt.Errorf("删除文件失败: %s", apiErr.Error())
	}

	if len(results) > 0 && !results[0].Success {
		return fmt.Errorf("删除文件失败")
	}

	return nil
}

// DeleteBatch 批量删除阿里云盘文件（并发10线程）
func (s *AliyunPanStorage) DeleteBatch(ctx context.Context, paths []string) []DeleteResult {
	if len(paths) == 0 {
		return []DeleteResult{}
	}

	const maxConcurrency = 10
	results := make([]DeleteResult, len(paths))
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i, filePath := range paths {
		wg.Add(1)
		go func(index int, p string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[index] = DeleteResult{Path: p, Error: ctx.Err()}
				return
			}

			results[index] = DeleteResult{
				Path:  p,
				Error: s.Delete(ctx, p),
			}
		}(i, filePath)
	}

	wg.Wait()
	return results
}

// GetURL 获取文件访问 URL
func (s *AliyunPanStorage) GetURL(ctx context.Context, filePath string) (string, error) {
	if err := s.ensureValidToken(); err != nil {
		return "", err
	}

	fullPath := path.Join(s.basePath, filePath)

	// 获取文件信息
	fileInfo, apiErr := s.client.FileInfoByPath(s.driveId, fullPath)
	if apiErr != nil {
		return "", fmt.Errorf("获取文件信息失败: %s", apiErr.Error())
	}

	if fileInfo.IsFolder() {
		return "", fmt.Errorf("路径是文件夹，不是文件: %s", filePath)
	}

	// 获取下载 URL（有效期约 4 小时）
	downloadResult, apiErr := s.client.GetFileDownloadUrl(&aliyunpan.GetFileDownloadUrlParam{
		DriveId:   s.driveId,
		FileId:    fileInfo.FileId,
		ExpireSec: 14400, // 4 小时
	})
	if apiErr != nil {
		return "", fmt.Errorf("获取下载链接失败: %s", apiErr.Error())
	}

	return downloadResult.Url, nil
}

// GetURLBatch 批量获取文件访问 URL（并发10线程）
func (s *AliyunPanStorage) GetURLBatch(ctx context.Context, paths []string) []URLResult {
	if len(paths) == 0 {
		return []URLResult{}
	}

	const maxConcurrency = 10
	results := make([]URLResult, len(paths))
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for i, filePath := range paths {
		wg.Add(1)
		go func(index int, p string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				results[index] = URLResult{Path: p, Error: ctx.Err()}
				return
			}

			url, err := s.GetURL(ctx, p)
			results[index] = URLResult{
				Path:  p,
				URL:   url,
				Error: err,
			}
		}(i, filePath)
	}

	wg.Wait()
	return results
}

// Exists 检查文件是否存在
func (s *AliyunPanStorage) Exists(ctx context.Context, filePath string) (bool, error) {
	if err := s.ensureValidToken(); err != nil {
		return false, err
	}

	fullPath := path.Join(s.basePath, filePath)

	_, apiErr := s.client.FileInfoByPath(s.driveId, fullPath)
	if apiErr != nil {
		if strings.Contains(apiErr.Error(), "文件不存在") {
			return false, nil
		}
		return false, fmt.Errorf("检查文件失败: %s", apiErr.Error())
	}

	return true, nil
}

// GetStats 获取存储统计信息
func (s *AliyunPanStorage) GetStats(ctx context.Context) (*StorageStats, error) {
	if err := s.ensureValidToken(); err != nil {
		return nil, err
	}

	userInfo, apiErr := s.client.GetUserInfo()
	if apiErr != nil {
		return nil, fmt.Errorf("获取用户信息失败: %s", apiErr.Error())
	}

	return &StorageStats{
		UsedBytes:  userInfo.UsedSize,
		TotalBytes: userInfo.TotalSize,
	}, nil
}

// generateDeviceId 生成唯一设备 ID
func generateDeviceId() string {
	return uuid.New().String()
}

// formatPath 格式化路径
func formatPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if p != "/" {
		p = strings.TrimSuffix(p, "/")
	}
	return p
}

func (s *AliyunPanStorage) GetClient() *aliyunpan_web.WebPanClient {
	return s.client
}

// GetUserInfo 获取阿里云盘用户信息
func (s *AliyunPanStorage) GetUserInfo() *aliyunpan.UserInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.userInfo
}

// GetDownloadConfig 获取下载配置
func (s *AliyunPanStorage) GetDownloadConfig() (chunkSizeKB int, concurrency int) {
	return int(s.downloadChunkSize / 1024), s.downloadConcurrency
}

// Move 移动文件到新路径（使用阿里云盘原生 FileMove API）
func (s *AliyunPanStorage) Move(ctx context.Context, oldPath, newPath string) error {
	if err := s.ensureValidToken(); err != nil {
		return err
	}

	// 1. 获取源文件信息
	oldFullPath := path.Join(s.basePath, oldPath)
	fileInfo, apiErr := s.client.FileInfoByPath(s.driveId, oldFullPath)
	if apiErr != nil {
		if strings.Contains(apiErr.Error(), "文件不存在") {
			return fmt.Errorf("源文件不存在: %s", oldPath)
		}
		return fmt.Errorf("获取源文件信息失败: %s", apiErr.Error())
	}

	// 2. 确保目标目录存在
	newFullPath := path.Join(s.basePath, newPath)
	newDirPath := path.Dir(newFullPath)
	dirResult, apiErr := s.client.MkdirByFullPath(s.driveId, newDirPath)
	if apiErr != nil {
		return fmt.Errorf("创建目标目录失败: %s", apiErr.Error())
	}
	toParentFileId := dirResult.FileId
	if toParentFileId == "" {
		toParentFileId = aliyunpan.DefaultRootParentFileId
	}

	// 3. 调用 FileMove API
	moveParams := []*aliyunpan.FileMoveParam{
		{
			DriveId:        s.driveId,
			FileId:         fileInfo.FileId,
			ToDriveId:      s.driveId,
			ToParentFileId: toParentFileId,
		},
	}

	results, apiErr := s.client.FileMove(moveParams)
	if apiErr != nil {
		return fmt.Errorf("移动文件失败: %s", apiErr.Error())
	}

	if len(results) > 0 && !results[0].Success {
		return fmt.Errorf("移动文件失败")
	}

	logger.Info("阿里云盘文件移动成功",
		zap.String("from", oldPath),
		zap.String("to", newPath))

	return nil
}

// MoveBatch 批量移动文件
func (s *AliyunPanStorage) MoveBatch(ctx context.Context, moves map[string]string) []MoveResult {
	results := make([]MoveResult, 0, len(moves))

	for oldPath, newPath := range moves {
		err := s.Move(ctx, oldPath, newPath)
		results = append(results, MoveResult{
			OldPath: oldPath,
			NewPath: newPath,
			Error:   err,
		})
	}

	return results
}
