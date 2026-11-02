package repo

import (
	"gorm.io/gorm"

	"github.com/benzi/slink/backend/internal/model"
)

type StatsRepo struct{ db *gorm.DB }

func NewStatsRepo(db *gorm.DB) *StatsRepo { return &StatsRepo{db: db} }

// UpsertAccessStats 按 (short_uri, date, hour) 聚合累加
func (r *StatsRepo) UpsertAccessStats(s *model.AccessStats) error {
	return r.db.Exec(`
		INSERT INTO t_link_access_stats (short_uri, date, hour, weekday, pv, uv, uip)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (short_uri, date, hour) DO UPDATE SET
			pv = pv + excluded.pv,
			uv = uv + excluded.uv,
			uip = uip + excluded.uip
	`, s.ShortURI, s.Date, s.Hour, s.Weekday, s.PV, s.UV, s.UIP).Error
}

// UpsertDimStats 维度计数累加
func (r *StatsRepo) UpsertDimStats(s *model.DimStats) error {
	return r.db.Exec(`
		INSERT INTO t_link_dim_stats (short_uri, date, dim, value, cnt)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (short_uri, date, dim, value) DO UPDATE SET
			cnt = cnt + excluded.cnt
	`, s.ShortURI, s.Date, s.Dim, s.Value, s.Cnt).Error
}

// CreateAccessLog 插入访问明细
func (r *StatsRepo) CreateAccessLog(l *model.AccessLog) error {
	return r.db.Create(l).Error
}

// StatsSummary 时间范围内汇总
// 注意：UIP 这类三个连续大写字母的字段 GORM 会转成 ui_p，必须显式声明列名
type StatsSummary struct {
	PV  int64 `json:"pv" gorm:"column:pv"`
	UV  int64 `json:"uv" gorm:"column:uv"`
	UIP int64 `json:"uip" gorm:"column:uip"`
}

func (r *StatsRepo) Summary(shortURI, start, end string) (StatsSummary, error) {
	var s StatsSummary
	err := r.db.Table("t_link_access_stats").
		Select("COALESCE(SUM(pv),0) pv, COALESCE(SUM(uv),0) uv, COALESCE(SUM(uip),0) uip").
		Where("short_uri = ? AND date >= ? AND date <= ?", shortURI, start, end).
		Scan(&s).Error
	return s, err
}

// DailyTrend 按天趋势
type DailyPoint struct {
	Date string `json:"date"`
	PV   int64  `json:"pv" gorm:"column:pv"`
	UV   int64  `json:"uv" gorm:"column:uv"`
	UIP  int64  `json:"uip" gorm:"column:uip"`
}

func (r *StatsRepo) DailyTrend(shortURI, start, end string) ([]DailyPoint, error) {
	var rows []DailyPoint
	err := r.db.Table("t_link_access_stats").
		Select("date, SUM(pv) pv, SUM(uv) uv, SUM(uip) uip").
		Where("short_uri = ? AND date >= ? AND date <= ?", shortURI, start, end).
		Group("date").Order("date ASC").
		Scan(&rows).Error
	return rows, err
}

// HourlyDist 小时分布（范围求和）
func (r *StatsRepo) HourlyDist(shortURI, start, end string) ([]struct {
	Hour int   `json:"hour"`
	PV   int64 `json:"pv"`
}, error) {
	var rows []struct {
		Hour int   `json:"hour"`
		PV   int64 `json:"pv"`
	}
	err := r.db.Table("t_link_access_stats").
		Select("hour, SUM(pv) pv").
		Where("short_uri = ? AND date >= ? AND date <= ? AND hour >= 0", shortURI, start, end).
		Group("hour").Order("hour ASC").
		Scan(&rows).Error
	return rows, err
}

// WeekdayDist 星期分布
func (r *StatsRepo) WeekdayDist(shortURI, start, end string) ([]struct {
	Weekday int   `json:"weekday"`
	PV      int64 `json:"pv"`
}, error) {
	var rows []struct {
		Weekday int   `json:"weekday"`
		PV      int64 `json:"pv"`
	}
	err := r.db.Table("t_link_access_stats").
		Select("weekday, SUM(pv) pv").
		Where("short_uri = ? AND date >= ? AND date <= ?", shortURI, start, end).
		Group("weekday").Order("weekday ASC").
		Scan(&rows).Error
	return rows, err
}

// DimDist 维度分布（TopN）
func (r *StatsRepo) DimDist(shortURI, dim, start, end string, topN int) ([]struct {
	Value string `json:"value"`
	Cnt   int64  `json:"cnt"`
}, error) {
	var rows []struct {
		Value string `json:"value"`
		Cnt   int64  `json:"cnt"`
	}
	err := r.db.Table("t_link_dim_stats").
		Select("value, SUM(cnt) cnt").
		Where("short_uri = ? AND dim = ? AND date >= ? AND date <= ?", shortURI, dim, start, end).
		Group("value").Order("cnt DESC").Limit(topN).
		Scan(&rows).Error
	return rows, err
}

// PageLogs 访问明细分页
func (r *StatsRepo) PageLogs(shortURI string, page, size int64) ([]model.AccessLog, int64, error) {
	var total int64
	tx := r.db.Model(&model.AccessLog{}).Where("short_uri = ?", shortURI)
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var logs []model.AccessLog
	err := tx.Order("id DESC").Offset(int((page - 1) * size)).Limit(int(size)).Find(&logs).Error
	return logs, total, err
}

// GlobalDaily 全局按天趋势（仪表盘）
func (r *StatsRepo) GlobalDaily(start, end string) ([]DailyPoint, error) {
	var rows []DailyPoint
	err := r.db.Table("t_link_access_stats").
		Select("date, SUM(pv) pv, SUM(uv) uv, SUM(uip) uip").
		Where("date >= ? AND date <= ?", start, end).
		Group("date").Order("date ASC").
		Scan(&rows).Error
	return rows, err
}

// GlobalTodaySummary 今日全局汇总
func (r *StatsRepo) GlobalTodaySummary(today string) (StatsSummary, error) {
	var s StatsSummary
	err := r.db.Table("t_link_access_stats").
		Select("COALESCE(SUM(pv),0) pv, COALESCE(SUM(uv),0) uv, COALESCE(SUM(uip),0) uip").
		Where("date = ?", today).
		Scan(&s).Error
	return s, err
}

// TopLinks 点击 TopN
func (r *StatsRepo) TopLinks(start, end string, n int) ([]model.Link, error) {
	var links []model.Link
	err := r.db.Table("t_link").
		Select("t_link.*, COALESCE(SUM(s.pv), 0) AS rank_pv").
		Joins("LEFT JOIN t_link_access_stats s ON s.short_uri = t_link.short_uri AND s.date >= ? AND s.date <= ?", start, end).
		Where("t_link.deleted_at = 0").
		Group("t_link.id").
		Order("rank_pv DESC, t_link.id DESC").
		Limit(n).
		Find(&links).Error
	return links, err
}
