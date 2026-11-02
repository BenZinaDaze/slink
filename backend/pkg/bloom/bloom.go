// Package bloom 基于 Redis Bitmap 的布隆过滤器，用于跳转链路拦截不存在的短码，
// 防止缓存穿透。误判率约 0.1%（容量 100 万，约 1.8MB，10 个哈希函数）。
// 不支持删除：软删除的短码依然可能命中过滤器，由空值缓存/DB 查询兜底。
package bloom

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	capacity = 1_000_000
	fpr      = 0.001
	k        = 10 // 哈希函数个数
	// m = -n·lnp/(ln2)^2 ≈ 1.44e7 bits
	bits = 14_377_587
)

// Filter Redis 位图布隆过滤器
type Filter struct {
	rdb   *redis.Client
	key   string
	salt  string // 实例盐，避免与同库其他过滤器混用
	cache *localCache
}

// New 创建过滤器；Add/Exists 会同时维护一份进程内位图缓存，
// 命中本地缓存时可省去 Redis 往返（单机部署下跳转链路更快）。
func New(rdb *redis.Client, key string) *Filter {
	return &Filter{
		rdb:   rdb,
		key:   key,
		salt:  uuid.NewString(),
		cache: newLocalCache(),
	}
}

// Add 将元素加入过滤器
func (f *Filter) Add(ctx context.Context, item string) error {
	h1, h2 := f.hash(item)
	pipe := f.rdb.Pipeline()
	for i := 0; i < k; i++ {
		pipe.SetBit(ctx, f.key, f.offset(h1, h2, i), 1)
	}
	f.cache.set(h1, h2)
	_, err := pipe.Exec(ctx)
	return err
}

// Exists 判断元素是否可能存在；false 表示一定不存在
func (f *Filter) Exists(ctx context.Context, item string) (bool, error) {
	h1, h2 := f.hash(item)
	if f.cache.has(h1, h2) {
		return true, nil
	}
	pipe := f.rdb.Pipeline()
	cmds := make([]*redis.IntCmd, k)
	for i := 0; i < k; i++ {
		cmds[i] = pipe.GetBit(ctx, f.key, f.offset(h1, h2, i))
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return true, err // Redis 异常时放行，走缓存/DB 兜底
	}
	for _, cmd := range cmds {
		if cmd.Val() == 0 {
			return false, nil
		}
	}
	return true, nil
}

func (f *Filter) offset(h1, h2 uint64, i int) int64 {
	return int64((h1 + uint64(i)*h2) % bits)
}

// hash 双哈希：FNV-1a 与 djb2，加盐
func (f *Filter) hash(item string) (uint64, uint64) {
	s := f.salt + item
	var h1, h2 uint64 = 14695981039346656037, 5381
	for i := 0; i < len(s); i++ {
		c := s[i]
		h1 ^= uint64(c)
		h1 *= 1099511628211
		h2 = h2*33 + uint64(c)
	}
	return h1, h2
}

// 进程内缓存：记录已加入过滤器的元素指纹（h1,h2），
// 热点短码命中本地缓存时可省去 Redis 往返
type localCache struct {
	m  map[[2]uint64]struct{}
	mu sync.RWMutex
}

func newLocalCache() *localCache { return &localCache{m: make(map[[2]uint64]struct{})} }

func (c *localCache) set(h1, h2 uint64) {
	c.mu.Lock()
	c.m[[2]uint64{h1, h2}] = struct{}{}
	if len(c.m) > 100_000 { // 防止无限膨胀，超过后重建
		c.m = make(map[[2]uint64]struct{})
	}
	c.mu.Unlock()
}

func (c *localCache) has(h1, h2 uint64) bool {
	c.mu.RLock()
	_, ok := c.m[[2]uint64{h1, h2}]
	c.mu.RUnlock()
	return ok
}
