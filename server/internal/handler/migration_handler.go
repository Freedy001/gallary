package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gallary/server/internal/service"
	"gallary/server/internal/utils"
	"gallary/server/pkg/logger"
)

// MigrationHandler 迁移处理器
type MigrationHandler struct {
	migrationService service.MigrationService
}

// NewMigrationHandler 创建迁移处理器实例
func NewMigrationHandler(migrationService service.MigrationService) *MigrationHandler {
	return &MigrationHandler{migrationService: migrationService}
}

// StartMigrationRequest 开始迁移请求
type StartMigrationRequest struct {
	NewBasePath  string `json:"new_base_path" binding:"required"`
	NewURLPrefix string `json:"new_url_prefix" binding:"required"`
}

// GetActive 获取当前活跃的迁移任务
//
//	@Summary		获取活跃迁移任务
//	@Description	获取当前正在进行的迁移任务
//	@Tags			存储管理
//	@Produce		json
//	@Success		200	{object}	utils.Response{data=model.MigrationTask}	"迁移任务信息（可能为空）"
//	@Failure		500	{object}	utils.Response								"获取失败"
//	@Router			/api/storage/migration/active [get]
func (h *MigrationHandler) GetActive(c *gin.Context) {
	task, err := h.migrationService.GetActiveMigration(c.Request.Context())
	if err != nil {
		logger.Error("获取活跃迁移任务失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	utils.Success(c, task)
}

// GetByID 获取迁移任务详情
//
//	@Summary		获取迁移任务详情
//	@Description	根据ID获取迁移任务详情
//	@Tags			存储管理
//	@Produce		json
//	@Param			id	path		int									true	"任务ID"
//	@Success		200	{object}	utils.Response{data=model.MigrationTask}	"迁移任务信息"
//	@Failure		400	{object}	utils.Response							"参数错误"
//	@Failure		404	{object}	utils.Response							"任务不存在"
//	@Failure		500	{object}	utils.Response							"获取失败"
//	@Router			/api/storage/migration/{id} [get]
func (h *MigrationHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Error(c, 400, "无效的任务ID")
		return
	}

	task, err := h.migrationService.GetMigrationStatus(c.Request.Context(), id)
	if err != nil {
		logger.Error("获取迁移任务失败", zap.Error(err))
		utils.Error(c, 500, err.Error())
		return
	}

	if task == nil {
		utils.Error(c, 404, "迁移任务不存在")
		return
	}

	utils.Success(c, task)
}
