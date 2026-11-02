package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/benzi/slink/backend/pkg/errcode"
	"github.com/benzi/slink/backend/pkg/result"
)

const (
	TokenKey    = "auth:token:"
	CtxUsername = "username"
)

// RequestID 为每个请求生成 requestId 并写入响应头
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-Id")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("requestId", id)
		c.Header("X-Request-Id", id)
		c.Next()
	}
}

// TokenSaver 抽象 token 的校验（由 service 提供，避免循环依赖）
type TokenSaver interface {
	Verify(ctx context.Context, token string) (username string, ok bool)
}

// Auth 登录态校验中间件：Authorization: Bearer <token>
func Auth(s TokenSaver) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c)
		if token == "" {
			result.Fail(c, errcode.Unauthorized)
			c.Abort()
			return
		}
		username, ok := s.Verify(c.Request.Context(), token)
		if !ok {
			result.Fail(c, errcode.Unauthorized)
			c.Abort()
			return
		}
		c.Set(CtxUsername, username)
		c.Next()
	}
}

func bearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return c.GetHeader("token") // 兼容裸 token 头
}

// ClientIP 提取客户端真实 IP（优先代理头）
func ClientIP(c *gin.Context) string {
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		if i := strings.IndexByte(ip, ','); i > 0 {
			return strings.TrimSpace(ip[:i])
		}
		return strings.TrimSpace(ip)
	}
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}
	return c.ClientIP()
}
