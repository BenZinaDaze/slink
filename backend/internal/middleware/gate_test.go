package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/benzi/slink/backend/internal/config"
)

func newTestContext(host string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Host = host
	return c, w
}

// TestIsShortDomain 短链域匹配：小写比较、忽略端口、未配置时恒不命中
func TestIsShortDomain(t *testing.T) {
	c, _ := newTestContext("S.Example.COM:8443")
	if !IsShortDomain(c, "s.example.com") {
		t.Error("S.Example.COM:8443 应命中 s.example.com")
	}
	c, _ = newTestContext("admin.example.com")
	if IsShortDomain(c, "s.example.com") {
		t.Error("admin.example.com 不应命中 s.example.com")
	}
	c, _ = newTestContext("s.example.com")
	if IsShortDomain(c, "") {
		t.Error("未配置短链域时不应命中（单域模式）")
	}
}

// TestAPIGate 短链域下管理 API 必须按「路由不存在」处理，与真实 404 不可区分
func TestAPIGate(t *testing.T) {
	cfg := &config.Config{ShortLink: config.ShortLinkConfig{Domain: "s.example.com"}}
	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(APIGate(cfg))
	api.GET("/links", func(c *gin.Context) { c.String(200, "secret") })

	// 命中短链域 → 404 JSON，不泄露接口信息
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/links", nil)
	req.Host = "s.example.com"
	r.ServeHTTP(w, req)
	if w.Code != 404 {
		t.Fatalf("短链域访问 API 应 404，实际 %d", w.Code)
	}

	// 其他域 → 正常处理
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/links", nil)
	req.Host = "admin.example.com"
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("管理域访问 API 应 200，实际 %d", w.Code)
	}
}

// TestSecurityHeaders 全局安全响应头必须存在
func TestSecurityHeaders(t *testing.T) {
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	want := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for k, v := range want {
		if got := w.Header().Get(k); got != v {
			t.Errorf("%s = %q, 期望 %q", k, got, v)
		}
	}
	if w.Header().Get("Content-Security-Policy") == "" {
		t.Error("缺少 Content-Security-Policy")
	}
}
