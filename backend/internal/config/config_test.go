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
