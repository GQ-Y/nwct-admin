package api

import (
	"net/http"
	"nwct/client-nps/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

// corsMiddleware CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// authMiddleware JWT认证中间件
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 未初始化阶段：允许引导页完成网络相关操作（否则用户无法联网完成初始化）
		// 仅放开 /api/v1/network/*，其余仍需鉴权。
		if s != nil && s.config != nil && !s.config.Initialized {
			if strings.HasPrefix(c.Request.URL.Path, "/api/v1/network/") {
				c.Next()
				return
			}
		}

		// 从Header获取Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "未授权",
			})
			c.Abort()
			return
		}

		// 提取Token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Token格式错误",
			})
			c.Abort()
			return
		}

		token := parts[1]

		// 验证JWT Token
		claims, err := utils.VerifyJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "Token无效",
			})
			c.Abort()
			return
		}

		// 将设备ID存储到上下文
		if deviceID, ok := claims["device_id"].(string); ok {
			c.Set("device_id", deviceID)
		}

		c.Next()
	}
}

