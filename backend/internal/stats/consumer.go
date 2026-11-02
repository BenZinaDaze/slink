package stats

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/benzi/slink/backend/internal/config"
	"github.com/benzi/slink/backend/internal/model"
	"github.com/benzi/slink/backend/internal/repo"
)

// Consumer Redis Stream 消费者：拉取访问事件 → 批量聚合 → 事务落库 → Ack。
// 启动后先重投本消费者未 Ack 的 pending 消息，再消费新消息。
type Consumer struct {
	cfg     *config.Config
	rdb     *redis.Client
	stats   *repo.StatsRepo
	link    *repo.LinkRepo
	locale  *LocaleResolver
	uvTTL   time.Duration
}

func NewConsumer(cfg *config.Config, rdb *redis.Client, stats *repo.StatsRepo, link *repo.LinkRepo, locale *LocaleResolver) *Consumer {
	return &Consumer{cfg: cfg, rdb: rdb, stats: stats, link: link, locale: locale, uvTTL: 7 * 24 * time.Hour}
}

// Run 阻塞运行直至 ctx 取消
func (c *Consumer) Run(ctx context.Context) error {
	// 创建消费者组（已存在则忽略）
	if err := c.rdb.XGroupCreateMkStream(ctx, c.cfg.Stats.Stream, c.cfg.Stats.Group, "0").Err(); err != nil {
		if !isBusyGroup(err) {
			return err
		}
	}
	log.Printf("[stats] 消费者组就绪 stream=%s group=%s consumer=%s", c.cfg.Stats.Stream, c.cfg.Stats.Group, c.cfg.Stats.Consumer)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		// 先处理本消费者 pending（重启恢复），再拉新消息
		if n, err := c.drainPending(ctx); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("[stats] 消费 pending 异常: %v", err)
			time.Sleep(time.Second)
			continue
		} else if n == 0 {
			if err := c.consumeNew(ctx); err != nil && ctx.Err() == nil {
				log.Printf("[stats] 消费异常: %v", err)
				time.Sleep(time.Second)
			}
		}
	}
}

func isBusyGroup(err error) bool {
	return err != nil && err.Error() == "BUSYGROUP Consumer Group name already exists"
}

// drainPending 处理历史遗留的未 Ack 消息，返回处理条数
func (c *Consumer) drainPending(ctx context.Context) (int, error) {
	total := 0
	for {
		streams, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.cfg.Stats.Group,
			Consumer: c.cfg.Stats.Consumer,
			Streams:  []string{c.cfg.Stats.Stream, "0"}, // "0" = 本消费者 pending
			Count:    c.cfg.Stats.BatchSize,
		}).Result()
		if err != nil || len(streams) == 0 {
			return total, err
		}
		n, err := c.handleMessages(ctx, streams[0].Messages)
		total += n
		if err != nil {
			return total, err
		}
		if len(streams[0].Messages) < int(c.cfg.Stats.BatchSize) {
			return total, nil
		}
	}
}

// consumeNew 阻塞拉取并处理新消息
func (c *Consumer) consumeNew(ctx context.Context) error {
	streams, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    c.cfg.Stats.Group,
		Consumer: c.cfg.Stats.Consumer,
		Streams:  []string{c.cfg.Stats.Stream, ">"}, // ">" = 仅新消息
		Count:    c.cfg.Stats.BatchSize,
		Block:    time.Duration(c.cfg.Stats.BlockMS) * time.Millisecond,
	}).Result()
	if err != nil {
		if err == redis.Nil || ctx.Err() != nil { // Block 超时，正常循环
			return nil
		}
		return err
	}
	if len(streams) == 0 {
		return nil
	}
	_, err = c.handleMessages(ctx, streams[0].Messages)
	return err
}

// handleMessages 聚合一批消息并落库，成功后 Ack
func (c *Consumer) handleMessages(ctx context.Context, msgs []redis.XMessage) (int, error) {
	if len(msgs) == 0 {
		return 0, nil
	}
	batch := newAggregate()
	for _, m := range msgs {
		e, ok := parseEvent(m)
		if !ok {
			continue // 非法消息直接 Ack 丢弃
		}
		c.aggregate(ctx, batch, e)
	}
	if err := batch.flush(ctx, c.stats, c.link); err != nil {
		return 0, err
	}
	ids := make([]string, 0, len(msgs))
	for _, m := range msgs {
		ids = append(ids, m.ID)
	}
	if err := c.rdb.XAck(ctx, c.cfg.Stats.Stream, c.cfg.Stats.Group, ids...).Err(); err != nil {
		log.Printf("[stats] XAck 失败（消息将重投，聚合按累加幂等重算 UV/UIP 有偏差风险）: %v", err)
		return len(msgs), err
	}
	return len(msgs), nil
}

type rawEvent struct {
	ShortURI string
	IP       string
	UA       string
	At       time.Time
}

func parseEvent(m redis.XMessage) (rawEvent, bool) {
	get := func(k string) string {
		if v, ok := m.Values[k]; ok {
			switch s := v.(type) {
			case string:
				return s
			default:
				return ""
			}
		}
		return ""
	}
	uri := get("short_uri")
	if uri == "" {
		return rawEvent{}, false
	}
	ts := int64(0)
	if s := get("ts"); s != "" {
		ts, _ = strconv.ParseInt(s, 10, 64)
	}
	if ts <= 0 {
		ts = time.Now().UnixMilli()
	}
	return rawEvent{ShortURI: uri, IP: get("ip"), UA: get("ua"), At: time.UnixMilli(ts)}, true
}

// ---------- 批内聚合 ----------

type accKey struct {
	uri  string
	date string
	hour int
}

type dimKey struct {
	uri  string
	date string
	dim  string
	val  string
}

type aggregate struct {
	access map[accKey]*model.AccessStats
	dims   map[dimKey]*model.DimStats
	logs   []model.AccessLog
	totals map[string][3]int64 // uri -> {pv, uv, uip}
}

func newAggregate() *aggregate {
	return &aggregate{
		access: map[accKey]*model.AccessStats{},
		dims:   map[dimKey]*model.DimStats{},
		totals: map[string][3]int64{},
	}
}

// aggregate 单事件归入批内聚合结构；UV/UIP 通过 Redis SET 判定
func (c *Consumer) aggregate(ctx context.Context, agg *aggregate, e rawEvent) {
	v := ParseUA(e.UA)
	locale := c.locale.Resolve(e.IP)
	local := e.At.Local()
	date := local.Format("2006-01-02")
	hour := local.Hour()
	weekday := int(local.Weekday())
	if weekday == 0 {
		weekday = 7
	}

	// UV/UIP 去重：SADD 返回 1 表示本日首次出现
	uvKey := "stats:uv:" + e.ShortURI + ":" + date
	uipKey := "stats:uip:" + e.ShortURI + ":" + date
	newUV, newUIP := false, false
	if v.UVKey != "" && e.IP != "" {
		pipe := c.rdb.Pipeline()
		uvCmd := pipe.SAdd(ctx, uvKey, v.UVKey+"@"+e.IP)
		uipCmd := pipe.SAdd(ctx, uipKey, e.IP)
		pipe.Expire(ctx, uvKey, c.uvTTL)
		pipe.Expire(ctx, uipKey, c.uvTTL)
		if _, err := pipe.Exec(ctx); err == nil {
			newUV = uvCmd.Val() > 0
			newUIP = uipCmd.Val() > 0
		}
	}

	// 按小时聚合
	k := accKey{e.ShortURI, date, hour}
	row := agg.access[k]
	if row == nil {
		row = &model.AccessStats{ShortURI: e.ShortURI, Date: date, Hour: hour, Weekday: weekday}
		agg.access[k] = row
	}
	row.PV++
	if newUV {
		row.UV++
	}
	if newUIP {
		row.UIP++
	}

	// 维度聚合
	for _, d := range [][2]string{
		{"browser", v.Browser}, {"os", v.OS}, {"device", v.Device}, {"locale", locale},
	} {
		dk := dimKey{e.ShortURI, date, d[0], d[1]}
		if dm := agg.dims[dk]; dm != nil {
			dm.Cnt++
		} else {
			agg.dims[dk] = &model.DimStats{ShortURI: e.ShortURI, Date: date, Dim: d[0], Value: d[1], Cnt: 1}
		}
	}

	// 明细
	agg.logs = append(agg.logs, model.AccessLog{
		ShortURI: e.ShortURI, IP: e.IP, Browser: v.Browser, OS: v.OS,
		Device: v.Device, Locale: locale, CreatedAt: local,
	})

	// 总量
	t := agg.totals[e.ShortURI]
	t[0]++
	if newUV {
		t[1]++
	}
	if newUIP {
		t[2]++
	}
	agg.totals[e.ShortURI] = t
}

// flush 事务落库
func (a *aggregate) flush(ctx context.Context, stats *repo.StatsRepo, link *repo.LinkRepo) error {
	for _, row := range a.access {
		if err := stats.UpsertAccessStats(row); err != nil {
			return err
		}
	}
	for _, d := range a.dims {
		if err := stats.UpsertDimStats(d); err != nil {
			return err
		}
	}
	for i := range a.logs {
		if err := stats.CreateAccessLog(&a.logs[i]); err != nil {
			return err
		}
	}
	for uri, t := range a.totals {
		if err := link.IncTotals(uri, t[0], t[1], t[2]); err != nil {
			return err
		}
	}
	return nil
}
