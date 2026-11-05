package config

import (
	"os"
	"path/filepath"
	"testing"
)

const testConfig = `
server:
  port: 8080
  mode: debug
auth:
  username: admin
  password: admin123
  token_ttl: 720h
database:
  path: /tmp/test-slink.db
redis:
  addr: 127.0.0.1:6379
  password: ""
  db: 0
shortlink:
  domain: ""
  default_cache_ttl: 720h
  null_cache_ttl: 30s
  whitelist: []
`

// TestEnvOverride 环境变量（SLINK_*）应覆盖 yaml 同名配置，
// docker-compose 依赖此行为注入 Redis 密码与初始账号
func TestEnvOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(testConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SLINK_REDIS_ADDR", "redis:6379")
	t.Setenv("SLINK_REDIS_PASSWORD", "secret99")
	t.Setenv("SLINK_SERVER_MODE", "release")
	t.Setenv("SLINK_AUTH_USERNAME", "bob")

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Redis.Addr != "redis:6379" {
		t.Fatalf("redis.addr = %q, 期望被环境变量覆盖为 redis:6379", cfg.Redis.Addr)
	}
	if cfg.Redis.Password != "secret99" {
		t.Fatalf("redis.password = %q, 期望被环境变量覆盖为 secret99", cfg.Redis.Password)
	}
	if cfg.Server.Mode != "release" {
		t.Fatalf("server.mode = %q, 期望 release", cfg.Server.Mode)
	}
	if cfg.Auth.Username != "bob" {
		t.Fatalf("auth.username = %q, 期望 bob", cfg.Auth.Username)
	}
	// 未覆盖项保持 yaml 值
	if cfg.Auth.Password != "admin123" {
		t.Fatalf("auth.password = %q, 期望保持 yaml 值 admin123", cfg.Auth.Password)
	}
}

// TestNormalizedDomain 短链域名 Host 归一化：域分离守卫按小写 Host（去 scheme/端口）匹配
func TestNormalizedDomain(t *testing.T) {
	cfg := &ShortLinkConfig{}
	cases := []struct{ raw, want string }{
		{"", ""},
		{"   ", ""},
		{"s.example.com", "s.example.com"},
		{"https://s.example.com", "s.example.com"},
		{"http://S.Example.COM:8443/", "s.example.com"},
		{"https://s.example.com/path?x=1", "s.example.com"},
		{"[::1]:8080", "::1"},
	}
	for _, tc := range cases {
		cfg.Domain = tc.raw
		if got := cfg.NormalizedDomain(); got != tc.want {
			t.Errorf("NormalizedDomain(%q) = %q, 期望 %q", tc.raw, got, tc.want)
		}
	}
}

// TestAdminPathValidate 管理入口路径校验：合法放行，非法/保留路径拒绝；前缀形态带 /
func TestAdminPathValidate(t *testing.T) {
	adm := &AdminConfig{}
	for _, ok := range []string{"my-panel", "PanEl_9", "ab"} {
		adm.Path = ok
		if err := adm.validate(); err != nil {
			t.Errorf("validate(%q) 不应报错: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "a", "admin123admin12345678901234567890", "含中文", "pa th", "api", "health"} {
		adm.Path = bad
		if err := adm.validate(); err == nil && bad != "" {
			t.Errorf("validate(%q) 应报错", bad)
		}
	}
	adm.Path = "my-panel"
	if got := adm.AdminPrefix(); got != "/my-panel" {
		t.Errorf("AdminPrefix() = %q, 期望 /my-panel", got)
	}
	adm.Path = ""
	if got := adm.AdminPrefix(); got != "" {
		t.Errorf("未配置时 AdminPrefix() 应为空, 实际 %q", got)
	}
	if adm.IsWeak() {
		t.Error("空路径不应判为弱路径")
	}
	adm.Path = "admin"
	if !adm.IsWeak() {
		t.Error("admin 应判为弱路径")
	}
}
