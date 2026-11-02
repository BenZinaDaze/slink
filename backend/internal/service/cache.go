package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/benzi/slink/backend/internal/config"
	"github.com/benzi/slink/backend/internal/model"
)

// cache:link:{uri} -> JSON{origin,enable_status,valid_date_type,valid_date_unix}
// cache:null:{uri} -> "" 空值缓存，短 TTL 防穿透

type cacheEntry struct {
	Origin        string `json:"origin"`
	EnableStatus  int    `json:"enableStatus"`
	ValidDateType int    `json:"validDateType"`
	ValidDateUnix int64  `json:"validDateUnix"` // 0 表示永久
}

type LinkCache struct {
	rdb *redis.Client
	cfg *config.Config
}

func NewLinkCache(rdb *redis.Client, cfg *config.Config) *LinkCache {
	return &LinkCache{rdb: rdb, cfg: cfg}
}

func linkKey(uri string) string { return "cache:link:" + uri }

func nullKey(uri string) string { return "cache:null:" + uri }

// ttl 计算缓存过期时间：自定义有效期取剩余时长，永久链接取默认 TTL
func (c *LinkCache) ttl(l *model.Link) time.Duration {
	if l.ValidDateType == 1 && l.ValidDate != nil {
		d := time.Until(*l.ValidDate)
		if d <= 0 {
			return time.Second // 已过期，几乎立即失效
		}
		return d
	}
	return c.cfg.ShortLink.DefaultCacheTTL
}

// Warm 缓存预热（创建/恢复短链时调用）
func (c *LinkCache) Warm(ctx context.Context, l *model.Link) {
	var vdt int64
	if l.ValidDate != nil {
		vdt = l.ValidDate.Unix()
	}
	entry := cacheEntry{Origin: l.OriginURL, EnableStatus: l.EnableStatus, ValidDateType: l.ValidDateType, ValidDateUnix: vdt}
	b, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_ = c.rdb.Set(ctx, linkKey(l.ShortURI), b, c.ttl(l)).Err()
}

// Evict 删除缓存（修改/软删短链时调用）
func (c *LinkCache) Evict(ctx context.Context, uri string) {
	_ = c.rdb.Del(ctx, linkKey(uri), nullKey(uri)).Err()
}

// Get 命中返回缓存条目；isNull 表示命中的是空值缓存（短链不存在）
func (c *LinkCache) Get(ctx context.Context, uri string) (entry cacheEntry, ok, isNull bool) {
	if n, _ := c.rdb.Exists(ctx, nullKey(uri)).Result(); n > 0 {
		return cacheEntry{}, false, true
	}
	b, err := c.rdb.Get(ctx, linkKey(uri)).Bytes()
	if err != nil {
		return cacheEntry{}, false, false
	}
	if err := json.Unmarshal(b, &entry); err != nil {
		return cacheEntry{}, false, false
	}
	return entry, true, false
}

// SetNull 写入空值缓存
func (c *LinkCache) SetNull(ctx context.Context, uri string) {
	_ = c.rdb.Set(ctx, nullKey(uri), "", c.cfg.ShortLink.NullCacheTTL).Err()
}
