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
	TokenCookie = "slink_token"
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

// Auth 登录态校验中间件。token 来源优先级：HttpOnly Cookie → Authorization: Bearer → 裸 token 头
// （后两者兼容脚本调用与旧版客户端）
func Auth(s TokenSaver) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := TokenFromRequest(c)
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

// TokenFromRequest 按优先级提取登录 token
func TokenFromRequest(c *gin.Context) string {
	if ck, err := c.Cookie(TokenCookie); err == nil && ck != "" {
		return ck
	}
	h := c.GetHeader("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return c.GetHeader("token") // 兼容裸 token 头
}
