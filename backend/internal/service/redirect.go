package service

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"

	"github.com/benzi/slink/backend/internal/model"
	"github.com/benzi/slink/backend/internal/repo"
	"github.com/benzi/slink/backend/pkg/bloom"
	"github.com/benzi/slink/backend/pkg/errcode"
)

// RedirectResult 跳转解析结果
type RedirectResult struct {
	OriginURL string
	Err       *errcode.Error // 404/410/403 对应的业务错误
}

// RedirectService 跳转热路径：布隆过滤器 → 缓存 → 空值缓存 → singleflight 合并回源
type RedirectService struct {
	link  *repo.LinkRepo
	bloom *bloom.Filter
	cache *LinkCache
	sf    singleflight.Group
}

func NewRedirectService(link *repo.LinkRepo, bf *bloom.Filter, cache *LinkCache) *RedirectService {
	return &RedirectService{link: link, bloom: bf, cache: cache, sf: singleflight.Group{}}
}

// Resolve 解析短码。非阻塞统计通过 events 回调发出（调用方接 Redis Stream）。
func (s *RedirectService) Resolve(ctx context.Context, shortURI string) RedirectResult {
	// 1. 布隆过滤器：一定不存在 → 直接 404，不打缓存不打库
	exists, err := s.bloom.Exists(ctx, shortURI)
	if err == nil && !exists {
		return RedirectResult{Err: errcode.LinkNotFound}
	}

	// 2. 缓存
	if entry, ok, isNull := s.cache.Get(ctx, shortURI); ok {
		return judge(entry, time.Now())
	} else if isNull {
		return RedirectResult{Err: errcode.LinkNotFound}
	}

	// 3. singleflight 合并并发回源（多实例部署可替换为 Redis SET NX 分布式锁）
	v, err2, _ := s.sf.Do(shortURI, func() (any, error) {
		l, err := s.link.GetByShortURI(shortURI)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.cache.SetNull(ctx, shortURI)
			return RedirectResult{Err: errcode.LinkNotFound}, nil
		}
		if err != nil {
			return RedirectResult{}, err
		}
		// 回填缓存后按有效期裁决
		s.cache.Warm(ctx, l)
		return judgeFromLink(l, time.Now()), nil
	})
	if err2 != nil {
		return RedirectResult{Err: errcode.ServerError}
	}
	return v.(RedirectResult)
}

// judge 按启用状态与有效期裁决缓存条目
func judge(e cacheEntry, now time.Time) RedirectResult {
	if e.EnableStatus != 0 {
		return RedirectResult{Err: errcode.LinkDisabled}
	}
	if e.ValidDateType == 1 && e.ValidDateUnix > 0 && now.Unix() >= e.ValidDateUnix {
		return RedirectResult{Err: errcode.LinkExpired}
	}
	return RedirectResult{OriginURL: e.Origin}
}

func judgeFromLink(l *model.Link, now time.Time) RedirectResult {
	if l.EnableStatus != 0 {
		return RedirectResult{Err: errcode.LinkDisabled}
	}
	if l.ValidDateType == 1 && l.ValidDate != nil && now.After(*l.ValidDate) {
		return RedirectResult{Err: errcode.LinkExpired}
	}
	return RedirectResult{OriginURL: l.OriginURL}
}

// StatsEvent 访问事件（发布到 Redis Stream）
type StatsEvent struct {
	ShortURI string
	IP       string
	UA       string
	Referer  string
	At       time.Time
}

// EventProducer 统计事件发布器
type EventProducer struct {
	rdb    *redis.Client
	stream string
}

// NewEventProducer 创建事件发布器
func NewEventProducer(rdb *redis.Client, stream string) *EventProducer {
	return &EventProducer{rdb: rdb, stream: stream}
}

// Publish 非阻塞发布；Stream 满或 Redis 异常时丢弃事件（统计允许少量丢失，不阻塞跳转）。
// 注意必须脱离请求 context：handler 返回后请求 ctx 会被取消，异步 XAdd 会随之失败。
func (p *EventProducer) Publish(ctx context.Context, e StatsEvent) {
	ctx = context.WithoutCancel(ctx)
	go func() {
		_ = p.rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: p.stream,
			MaxLen: 100000,
			Approx: true,
			Values: map[string]any{
				"short_uri": e.ShortURI,
				"ip":        e.IP,
				"ua":        e.UA,
				"referer":   e.Referer,
				"ts":        e.At.UnixMilli(),
			},
		}).Err()
	}()
}
