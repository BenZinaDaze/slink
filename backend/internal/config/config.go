package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Admin     AdminConfig     `mapstructure:"admin"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	ShortLink ShortLinkConfig `mapstructure:"shortlink"`
	Stats     StatsConfig     `mapstructure:"stats"`
}

type AdminConfig struct {
	// Path 管理后台入口路径（如 "my-panel" → https://域名/my-panel）。
	// 配置后整个管理界面只在该前缀下提供，根路径与 /login 一律 404；留空则管理面在根路径（旧行为）
	Path string `mapstructure:"path"`
}

// commonAdminPaths 常见易猜路径：仅启动时打日志提醒，不强制
var commonAdminPaths = map[string]struct{}{
	"admin": {}, "login": {}, "manage": {}, "manager": {},
	"panel": {}, "console": {}, "backend": {}, "slink": {}, "dashboard": {},
}

// IsWeak 是否为常见易猜的管理入口路径（仅供启动日志提示）
func (c *AdminConfig) IsWeak() bool {
	_, ok := commonAdminPaths[strings.ToLower(c.Path)]
	return ok
}

// AdminPrefix 管理入口的 URL 前缀形态（"/my-panel"），未配置返回空串。
// 请求路径匹配与前端 basename 都以此为准；路径区分大小写
func (c *AdminConfig) AdminPrefix() string {
	if c.Path == "" {
		return ""
	}
	return "/" + c.Path
}

func (c *AdminConfig) validate() error {
	p := c.Path
	if p == "" {
		return nil
	}
	if len(p) < 2 || len(p) > 32 {
		return fmt.Errorf("admin.path 长度需在 2-32 个字符之间")
	}
	for _, r := range p {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return fmt.Errorf("admin.path 仅允许字母、数字、-、_")
		}
	}
	if p == "api" || p == "health" {
		return fmt.Errorf("admin.path 不能为保留路径 %q", p)
	}
	return nil
}

type ServerConfig struct {
	Port          int      `mapstructure:"port"`
	Mode          string   `mapstructure:"mode"` // debug | release
	TrustedProxies []string `mapstructure:"trusted_proxies"` // 受信反代 CIDR；空 = 不信任任何代理头（客户端 IP 取直连地址）
}

type AuthConfig struct {
	Username              string        `mapstructure:"username"`
	Password              string        `mapstructure:"password"`
	TokenTTL              time.Duration `mapstructure:"token_ttl"`
	CookieSecure          *bool         `mapstructure:"cookie_secure"`          // nil = 自动（HTTPS 请求时启用 Secure）
	AllowDefaultPassword  bool          `mapstructure:"allow_default_password"` // true = 不强制修改初始密码（仅横幅提醒）
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type ShortLinkConfig struct {
	// Domain 短链（公开）域名。配置后该域只提供跳转服务：
	// /api、SPA、静态资源一律 404，管理面必须走其他域名访问；留空则单域模式（行为同旧版）
	Domain          string        `mapstructure:"domain"`
	DefaultCacheTTL time.Duration `mapstructure:"default_cache_ttl"`
	NullCacheTTL    time.Duration `mapstructure:"null_cache_ttl"`
	Whitelist       []string      `mapstructure:"whitelist"`
}

// NormalizedDomain 短链域名的 Host 形态（去 scheme/端口、小写），用于请求 Host 匹配；
// 未配置时返回空串
func (c *ShortLinkConfig) NormalizedDomain() string {
	return normalizeHost(c.Domain)
}

// normalizeHost 从任意形态（https://s.example.com、s.example.com:8443、S.Example.COM/）
// 提取小写 Host，不含端口
func normalizeHost(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return ""
	}
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	if i := strings.LastIndex(s, ":"); i >= 0 && !strings.Contains(s, "]") {
		s = s[:i] // IPv6 裸地址不含端口场景极少，含 ] 时跳过去端口
	}
	if i := strings.Index(s, "]"); i >= 0 { // [::1]:8080 → ::1
		s = s[1:i]
	}
	return s
}

type StatsConfig struct {
	Stream    string `mapstructure:"stream"`
	Group     string `mapstructure:"group"`
	Consumer  string `mapstructure:"consumer"`
	BatchSize int64  `mapstructure:"batch_size"`
	BlockMS   int64  `mapstructure:"block_ms"`
}

// Load 按优先级加载配置：环境变量(SLINK_*) > config.yaml
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetEnvPrefix("SLINK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if err := cfg.Admin.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Server.Port <= 0 {
		return fmt.Errorf("server.port 无效")
	}
	if c.Auth.Username == "" || c.Auth.Password == "" {
		return fmt.Errorf("auth.username / auth.password 不能为空")
	}
	if c.Auth.TokenTTL <= 0 {
		c.Auth.TokenTTL = 720 * time.Hour
	}
	if c.Redis.Addr == "" {
		c.Redis.Addr = "127.0.0.1:6379"
	}
	if c.ShortLink.DefaultCacheTTL <= 0 {
		c.ShortLink.DefaultCacheTTL = 720 * time.Hour
	}
	if c.ShortLink.NullCacheTTL <= 0 {
		c.ShortLink.NullCacheTTL = 30 * time.Second
	}
	if c.Stats.Stream == "" {
		c.Stats.Stream = "slink:stats:events"
	}
	if c.Stats.Group == "" {
		c.Stats.Group = "slink-stats-group"
	}
	if c.Stats.Consumer == "" {
		c.Stats.Consumer = "consumer-1"
	}
	if c.Stats.BatchSize <= 0 {
		c.Stats.BatchSize = 50
	}
	if c.Stats.BlockMS <= 0 {
		c.Stats.BlockMS = 1000
	}
	return nil
}
