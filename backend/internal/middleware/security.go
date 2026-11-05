package middleware

import "github.com/gin-gonic/gin"

// securityCSP 内容安全策略：
// - script 限定自持（vite 产物为外链脚本）；style 需要 unsafe-inline（内联错误页与组件库内联样式）
// - frame-ancestors none 与 X-Frame-Options: DENY 双保险，禁止被嵌入 iframe
const securityCSP = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data:; connect-src 'self'; font-src 'self'; " +
	"frame-ancestors 'none'; base-uri 'self'; form-action 'self'"

// SecurityHeaders 全局安全响应头。HSTS 由 TLS 终结的反代层负责，不在应用侧设置。
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy", securityCSP)
		c.Next()
	}
}
