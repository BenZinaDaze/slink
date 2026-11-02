package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/benzi/slink/backend/internal/dto"
	"github.com/benzi/slink/backend/internal/repo"
	"github.com/benzi/slink/backend/pkg/errcode"
)

type StatsService struct {
	link  *repo.LinkRepo
	stats *repo.StatsRepo
}

func NewStatsService(link *repo.LinkRepo, stats *repo.StatsRepo) *StatsService {
	return &StatsService{link: link, stats: stats}
}

// mustLinkByShortURI 通过 id 找到短链并返回短码
func (s *StatsService) mustLinkByShortURI(id int64) (string, error) {
	l, err := s.link.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", errcode.LinkNotFound
	}
	if err != nil {
		return "", err
	}
	return l.ShortURI, nil
}

func defaultRange(start, end string) (string, string) {
	today := time.Now().Format("2006-01-02")
	if start == "" || end == "" {
		return today, today
	}
	return start, end
}

// Overview 汇总 + 按天趋势（补零）
func (s *StatsService) Overview(ctx context.Context, id int64, start, end string) (map[string]any, error) {
	uri, err := s.mustLinkByShortURI(id)
	if err != nil {
		return nil, err
	}
	start, end = defaultRange(start, end)
	summary, err := s.stats.Summary(uri, start, end)
	if err != nil {
		return nil, err
	}
	daily, err := s.stats.DailyTrend(uri, start, end)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"summary": summary,
		"daily":   fillDaily(daily, start, end),
	}, nil
}

// Hourly 小时分布（补零）
func (s *StatsService) Hourly(ctx context.Context, id int64, start, end string) ([]dto.Point2, error) {
	uri, err := s.mustLinkByShortURI(id)
	if err != nil {
		return nil, err
	}
	start, end = defaultRange(start, end)
	rows, err := s.stats.HourlyDist(uri, start, end)
	if err != nil {
		return nil, err
	}
	m := map[int]int64{}
	for _, r := range rows {
		m[r.Hour] = r.PV
	}
	out := make([]dto.Point2, 0, 24)
	for h := 0; h < 24; h++ {
		out = append(out, dto.Point2{Label: strconv.Itoa(h) + "点", Value: m[h]})
	}
	return out, nil
}

// Weekday 星期分布
func (s *StatsService) Weekday(ctx context.Context, id int64, start, end string) ([]dto.Point2, error) {
	uri, err := s.mustLinkByShortURI(id)
	if err != nil {
		return nil, err
	}
	start, end = defaultRange(start, end)
	rows, err := s.stats.WeekdayDist(uri, start, end)
	if err != nil {
		return nil, err
	}
	m := map[int]int64{}
	for _, r := range rows {
		m[r.Weekday] = r.PV
	}
	names := []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}
	out := make([]dto.Point2, 0, 7)
	for i := 1; i <= 7; i++ {
		out = append(out, dto.Point2{Label: names[i-1], Value: m[i]})
	}
	return out, nil
}

// Dim 维度分布
func (s *StatsService) Dim(ctx context.Context, id int64, dim, start, end string) ([]dto.Point2, error) {
	uri, err := s.mustLinkByShortURI(id)
	if err != nil {
		return nil, err
	}
	start, end = defaultRange(start, end)
	rows, err := s.stats.DimDist(uri, dim, start, end, 10)
	if err != nil {
		return nil, err
	}
	out := make([]dto.Point2, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.Point2{Label: r.Value, Value: r.Cnt})
	}
	return out, nil
}

// Records 访问明细分页
func (s *StatsService) Records(ctx context.Context, id int64, page, size int64) ([]map[string]any, int64, error) {
	uri, err := s.mustLinkByShortURI(id)
	if err != nil {
		return nil, 0, err
	}
	logs, total, err := s.stats.PageLogs(uri, page, size)
	if err != nil {
		return nil, 0, err
	}
	out := make([]map[string]any, 0, len(logs))
	for _, l := range logs {
		out = append(out, map[string]any{
			"id": l.ID, "ip": l.IP, "browser": l.Browser, "os": l.OS,
			"device": l.Device, "locale": l.Locale, "createdAt": l.CreatedAt,
		})
	}
	return out, total, nil
}

// fillDaily 补齐日期空洞，值为 0
func fillDaily(rows []repo.DailyPoint, start, end string) []repo.DailyPoint {
	m := map[string]repo.DailyPoint{}
	for _, r := range rows {
		m[r.Date] = r
	}
	out := make([]repo.DailyPoint, 0)
	s, _ := time.ParseInLocation("2006-01-02", start, time.Local)
	e, _ := time.ParseInLocation("2006-01-02", end, time.Local)
	for d := s; !d.After(e); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		if r, ok := m[key]; ok {
			out = append(out, r)
		} else {
			out = append(out, repo.DailyPoint{Date: key})
		}
	}
	return out
}
