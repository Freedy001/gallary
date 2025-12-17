package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"gallary/server/internal/middleware"
	ws "gallary/server/internal/websocket"
	"gallary/server/pkg/logger"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 生产环境应该验证 Origin
		return true
	},
}

// WebSocketHandler WebSocket 处理器
type WebSocketHandler struct {
	hub      *ws.Hub
	adminCfg *middleware.AdminConfig
}

// NewWebSocketHandler 创建 WebSocket 处理器
func NewWebSocketHandler(hub *ws.Hub, adminCfg *middleware.AdminConfig) *WebSocketHandler {
	return &WebSocketHandler{
		hub:      hub,
		adminCfg: adminCfg,
	}
}

// HandleWebSocket 处理 WebSocket 连接
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	// 1. 认证检查（通过 query 参数获取 token）
	if h.adminCfg.IsAuthEnabled() {
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证token"})
			return
		}

		// 验证 token
		if !h.validateToken(token) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
			return
		}
	}

	// 2. 升级为 WebSocket 连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error("WebSocket 升级失败", zap.Error(err))
		return
	}

	// 3. 创建客户端并注册
	h.hub.Register(conn)
}

// validateToken 验证 JWT Token
func (h *WebSocketHandler) validateToken(tokenString string) bool {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.adminCfg.Secret), nil
	})

	if err != nil || !token.Valid {
		return false
	}

	// 验证密码版本号
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		tokenPV := int64(0)
		if pv, exists := claims["pv"]; exists {
			switch v := pv.(type) {
			case float64:
				tokenPV = int64(v)
			case int64:
				tokenPV = v
			}
		}
		if tokenPV < h.adminCfg.PasswordVersion {
			return false
		}
	}

	return true
}
