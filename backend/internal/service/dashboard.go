package service

import (
	"context"
	"time"

	"github.com/benzi/slink/backend/internal/repo"
)

type DashboardService struct {
	link  *repo.LinkRepo
	stats *repo.StatsRepo
}

func NewDashboardService(link *repo.LinkRepo, stats *repo.StatsRepo) *DashboardService {
	return &DashboardService{link: link, stats: stats}
}

// Overview 仪表盘汇总：短链总数、今日 PV/UV/UIP、总点击、近30天趋势、Top10
func (s *DashboardService) Overview(ctx context.Context) (map[string]any, error) {
	today := time.Now().Format("2006-01-02")
	start30 := time.Now().AddDate(0, 0, -29).Format("2006-01-02")

	linkTotal, err := s.link.CountAll()
	if err != nil {
		return nil, err
	}
	todaySummary, err := s.stats.GlobalTodaySummary(today)
	if err != nil {
		return nil, err
	}
	daily, err := s.stats.GlobalDaily(start30, today)
	if err != nil {
		return nil, err
	}
	top, err := s.stats.TopLinks(start30, today, 10)
	if err != nil {
		return nil, err
	}
	totalClick, err := s.link.SumTotalPV()
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"linkTotal":   linkTotal,
		"today":       todaySummary,
		"totalClick":  totalClick,
		"daily":       fillDaily(daily, start30, today),
		"topLinks":    top,
	}, nil
}
