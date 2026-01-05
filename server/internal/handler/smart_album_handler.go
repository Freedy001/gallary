package handler

import (
	"net/http"

	"gallary/server/internal/model"
	"gallary/server/internal/service"
	"gallary/server/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SmartAlbumHandler 智能相册处理器
type SmartAlbumHandler struct {
	smartAlbumService service.SmartAlbumService
}

// NewSmartAlbumHandler 创建智能相册处理器
func NewSmartAlbumHandler(smartAlbumService service.SmartAlbumService) *SmartAlbumHandler {
	return &SmartAlbumHandler{
		smartAlbumService: smartAlbumService,
	}
}

// ==================== 请求/响应结构 ====================

// SubmitTaskRequest 提交智能相册任务请求
type SubmitTaskRequest struct {
	ModelName     string                  `json:"model_name" binding:"required"`
	Algorithm     string                  `json:"algorithm" binding:"required"`
	HDBSCANParams *model.HDBSCANParamsDTO `json:"hdbscan_params"`
}

// ==================== Handler 方法 ====================

// SubmitTask 提交智能相册任务
// @Summary 提交智能相册生成任务
// @Tags 智能相册
// @Accept json
// @Produce json
// @Param request body SubmitTaskRequest true "任务参数"
// @Success 200 {object} model.SmartAlbumProgressVO
// @Router /api/albums/smart-tasks [post]
func (h *SmartAlbumHandler) SubmitTask(c *gin.Context) {
	var req SubmitTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 转换为服务层请求
	serviceReq := &service.GenerateSmartAlbumsRequest{
		ModelName:     req.ModelName,
		Algorithm:     req.Algorithm,
		HDBSCANParams: req.HDBSCANParams,
	}

	progressVO, err := h.smartAlbumService.SubmitSmartAlbumTask(c.Request.Context(), serviceReq)
	if err != nil {
		logger.Error("Failed to submit smart album task", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, progressVO)
}
