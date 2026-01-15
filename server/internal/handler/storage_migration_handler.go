package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"gallary/server/internal/service"
	"gallary/server/internal/utils"
)

// StorageMigrationHandler 存储迁移处理器
type StorageMigrationHandler struct {
	migrationService service.StorageMigrationService
}

// NewStorageMigrationHandler 创建存储迁移处理器实例
func NewStorageMigrationHandler(migrationService service.StorageMigrationService) *StorageMigrationHandler {
	return &StorageMigrationHandler{migrationService: migrationService}
}

// CreateMigration 创建迁移任务
//
//	@Summary		创建存储迁移任务
//	@Description	创建一个新的存储迁移任务
//	@Tags			存储迁移
//	@Accept			json
//	@Produce		json
//	@Param			request	body		service.CreateMigrationRequest	true	"迁移请求"
//	@Success		200		{object}	utils.Response{data=model.StorageMigrationTask}	"迁移任务"
//	@Failure		400		{object}	utils.Response	"参数错误"
//	@Failure		500		{object}	utils.Response	"创建失败"
//	@Router			/api/storage/migration [post]
func (h *StorageMigrationHandler) CreateMigration(c *gin.Context) {
	var req service.CreateMigrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	task, err := h.migrationService.CreateMigration(c.Request.Context(), &req)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}

	utils.Success(c, task)
}

// PreviewMigration 预览迁移
//
//	@Summary		预览存储迁移
//	@Description	统计待迁移文件数量，不实际执行迁移
//	@Tags			存储迁移
//	@Accept			json
//	@Produce		json
//	@Param			request	body		service.CreateMigrationRequest	true	"迁移请求"
//	@Success		200		{object}	utils.Response{data=service.MigrationPreview}	"预览结果"
//	@Failure		400		{object}	utils.Response	"参数错误"
//	@Failure		500		{object}	utils.Response	"预览失败"
//	@Router			/api/storage/migration/preview [post]
func (h *StorageMigrationHandler) PreviewMigration(c *gin.Context) {
	var req service.CreateMigrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	preview, err := h.migrationService.PreviewMigration(c.Request.Context(), &req)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}

	utils.Success(c, preview)
}

// ListActiveMigrations 获取所有活跃迁移任务
//
//	@Summary		获取所有活跃迁移任务
//	@Description	获取所有 pending/running/paused 状态的迁移任务列表
//	@Tags			存储迁移
//	@Produce		json
//	@Success		200	{object}	utils.Response{data=model.MigrationStatusVO}	"活跃任务列表"
//	@Router			/api/storage/migration/list/active [get]
func (h *StorageMigrationHandler) ListActiveMigrations(c *gin.Context) {
	statusVO, err := h.migrationService.GetMigrationStatusVO(c.Request.Context())
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}

	utils.Success(c, statusVO)
}

// PauseMigration 暂停迁移任务
//
//	@Summary		暂停迁移任务
//	@Description	暂停正在执行的迁移任务，可稍后恢复
//	@Tags			存储迁移
//	@Produce		json
//	@Param			id	path		int	true	"任务ID"
//	@Success		200	{object}	utils.Response	"暂停成功"
//	@Failure		400	{object}	utils.Response	"参数错误"
//	@Failure		500	{object}	utils.Response	"暂停失败"
//	@Router			/api/storage/migration/{id}/pause [post]
func (h *StorageMigrationHandler) PauseMigration(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Error(c, 400, "无效的任务ID")
		return
	}

	if err := h.migrationService.PauseMigration(c.Request.Context(), id); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "暂停成功"})
}

// ResumeMigration 恢复迁移任务
//
//	@Summary		恢复迁移任务
//	@Description	恢复暂停的迁移任务
//	@Tags			存储迁移
//	@Produce		json
//	@Param			id	path		int	true	"任务ID"
//	@Success		200	{object}	utils.Response	"恢复成功"
//	@Failure		400	{object}	utils.Response	"参数错误"
//	@Failure		500	{object}	utils.Response	"恢复失败"
//	@Router			/api/storage/migration/{id}/resume [post]
func (h *StorageMigrationHandler) ResumeMigration(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Error(c, 400, "无效的任务ID")
		return
	}

	if err := h.migrationService.ResumeMigration(c.Request.Context(), id); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "恢复成功"})
}

// GetFailedFileRecords 获取失败文件记录
//
//	@Summary		获取失败文件记录
//	@Description	分页获取迁移任务中失败的文件记录
//	@Tags			存储迁移
//	@Produce		json
//	@Param			id			path		int	true	"任务ID"
//	@Param			page		query		int	false	"页码"	default(1)
//	@Param			page_size	query		int	false	"每页数量"	default(20)
//	@Success		200			{object}	utils.Response{data=[]model.MigrationFileRecordVO}	"失败文件列表"
//	@Failure		400			{object}	utils.Response	"参数错误"
//	@Failure		500			{object}	utils.Response	"查询失败"
//	@Router			/api/storage/migration/{id}/failed [get]
func (h *StorageMigrationHandler) GetFailedFileRecords(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Error(c, 400, "无效的任务ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	records, total, err := h.migrationService.GetFailedFileRecords(c.Request.Context(), id, page, pageSize)
	if err != nil {
		utils.Error(c, 500, err.Error())
		return
	}

	utils.Success(c, gin.H{
		"items": records,
		"total": total,
		"page":  page,
	})
}

// RetryFailedFiles 重试失败文件
//
//	@Summary		重试失败文件
//	@Description	重试迁移任务中所有失败的文件
//	@Tags			存储迁移
//	@Produce		json
//	@Param			id	path		int	true	"任务ID"
//	@Success		200	{object}	utils.Response	"重试已启动"
//	@Failure		400	{object}	utils.Response	"参数错误"
//	@Failure		500	{object}	utils.Response	"重试失败"
//	@Router			/api/storage/migration/{id}/retry [post]
func (h *StorageMigrationHandler) RetryFailedFiles(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Error(c, 400, "无效的任务ID")
		return
	}

	if err := h.migrationService.RetryFailedFiles(c.Request.Context(), id); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "重试已启动"})
}

// DismissFailedFiles 忽略失败文件
//
//	@Summary		忽略失败文件
//	@Description	清除迁移任务中的失败文件记录，不再显示该任务
//	@Tags			存储迁移
//	@Produce		json
//	@Param			id	path		int	true	"任务ID"
//	@Success		200	{object}	utils.Response	"已忽略"
//	@Failure		400	{object}	utils.Response	"参数错误"
//	@Failure		500	{object}	utils.Response	"操作失败"
//	@Router			/api/storage/migration/{id}/dismiss [post]
func (h *StorageMigrationHandler) DismissFailedFiles(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Error(c, 400, "无效的任务ID")
		return
	}

	if err := h.migrationService.DismissMigration(c.Request.Context(), id); err != nil {
		utils.Error(c, 500, err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "已忽略失败文件"})
}
