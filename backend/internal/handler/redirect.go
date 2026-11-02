package handler

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/benzi/slink/backend/internal/middleware"
	"github.com/benzi/slink/backend/internal/service"
	"github.com/benzi/slink/backend/pkg/errcode"
)

var shortURIRe = regexp.MustCompile(`^[0-9a-zA-Z_-]{1,16}$`)

// RedirectHandler 跳转热路径
type RedirectHandler struct {
	redirect *service.RedirectService
	events   *service.EventProducer
}

func NewRedirectHandler(redirect *service.RedirectService, events *service.EventProducer) *RedirectHandler {
	return &RedirectHandler{redirect: redirect, events: events}
}

// Handle NoRoute 第一环：疑似短码则跳转，否则交由 SPA 兜底（c.Next）
func (h *RedirectHandler) Handle(c *gin.Context) {
	code := strings.Trim(c.Request.URL.Path, "/")
	// 根路径 / 含路径分隔符或扩展名的（静态资源）→ SPA 处理
	if code == "" || strings.ContainsAny(code, "/.") {
		c.Next()
		return
	}
	// 前端路由保留字 → SPA 处理
	if _, reserved := service.ReservedPaths[code]; reserved {
		c.Next()
		return
	}
	if !shortURIRe.MatchString(code) {
		c.Next() // 不合法短码形态，也交给 SPA（最终 404 兜底）
		return
	}
	res := h.redirect.Resolve(c.Request.Context(), code)
	if res.Err != nil {
		switch res.Err {
		case errcode.LinkNotFound:
			errorPage(c, http.StatusNotFound, "短链接不存在", "该短链接不存在或已被删除")
		case errcode.LinkExpired:
			errorPage(c, http.StatusGone, "短链接已过期", "该短链接的有效期已结束")
		case errcode.LinkDisabled:
			errorPage(c, http.StatusForbidden, "短链接已停用", "该短链接已被管理员停用")
		default:
			errorPage(c, http.StatusInternalServerError, "服务暂时不可用", "请稍后重试")
		}
		c.Abort()
		return
	}
	// 非阻塞发布访问事件（Redis Stream），统计链路异步消费
	h.events.Publish(c.Request.Context(), service.StatsEvent{
		ShortURI: code,
		IP:       middleware.ClientIP(c),
		UA:       c.Request.UserAgent(),
		Referer:  c.Request.Referer(),
		At:       time.Now(),
	})
	c.Abort()
	c.Redirect(http.StatusFound, res.OriginURL) // 302，不缓存以便统计
}

func errorPage(c *gin.Context, status int, title, desc string) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(status, errorTmpl, title, status, title, desc)
}

const errorTmpl = `<!DOCTYPE html>
<html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>%s</title>
<style>
body{margin:0;font-family:system-ui,-apple-system,"PingFang SC","Microsoft YaHei",sans-serif;background:#f8fafc;color:#0f172a;display:flex;min-height:100vh;align-items:center;justify-content:center}
.card{text-align:center;padding:48px 32px;border-radius:16px;background:#fff;box-shadow:0 1px 3px rgba(0,0,0,.08);max-width:400px;margin:24px}
.code{font-size:56px;font-weight:700;line-height:1;margin-bottom:16px;color:#64748b}
h1{font-size:20px;margin:0 0 8px}
p{color:#64748b;margin:0}
</style></head>
<body><div class="card"><div class="code">%d</div><h1>%s</h1><p>%s</p></div></body></html>`
