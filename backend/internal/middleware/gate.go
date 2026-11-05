package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/benzi/slink/backend/internal/config"
)

// IsShortDomain 请求 Host 是否命中配置的短链（公开）域名。
// 比较双方都归一化为「去端口的小写 Host」；短链域未配置时恒为 false（单域模式）。
func IsShortDomain(c *gin.Context, domain string) bool {
	if domain == "" {
		return false
	}
	host := c.Request.Host
	if i := strings.LastIndex(host, ":"); i >= 0 && !strings.Contains(host, "]") {
		host = host[:i]
	}
	return strings.EqualFold(host, domain)
}

// APIGate 管理面 API 守卫：短链域下所有 /api 请求按「路由不存在」处理，
// 与真实 404 不可区分，避免暴露管理面存在性。挂载在 /api/v1 组上。
func APIGate(cfg *config.Config) gin.HandlerFunc {
	domain := cfg.ShortLink.NormalizedDomain()
	return func(c *gin.Context) {
		if IsShortDomain(c, domain) {
			c.AbortWithStatusJSON(404, gin.H{"code": "B0002", "message": "接口不存在", "success": false})
			return
		}
		c.Next()
	}
}
