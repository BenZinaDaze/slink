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
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	ShortLink ShortLinkConfig `mapstructure:"shortlink"`
	Stats     StatsConfig     `mapstructure:"stats"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // debug | release
}

type AuthConfig struct {
	Username  string        `mapstructure:"username"`
	Password  string        `mapstructure:"password"`
	TokenTTL  time.Duration `mapstructure:"token_ttl"`
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
	Domain          string        `mapstructure:"domain"`
	DefaultCacheTTL time.Duration `mapstructure:"default_cache_ttl"`
	NullCacheTTL    time.Duration `mapstructure:"null_cache_ttl"`
	Whitelist       []string      `mapstructure:"whitelist"`
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
