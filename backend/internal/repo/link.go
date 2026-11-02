package repo

import (
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/benzi/slink/backend/internal/model"
)

type LinkQuery struct {
	GID     string // 分组筛选，空为全部
	Keyword string // 短码/原始链接/标题/描述 模糊搜索
	Order   string // created_at_desc(默认) | created_at_asc | pv_desc
	Page    int64
	Size    int64
}

type LinkRepo struct{ db *gorm.DB }

func NewLinkRepo(db *gorm.DB) *LinkRepo { return &LinkRepo{db: db} }

// Page 分页查询未删除短链
func (r *LinkRepo) Page(q LinkQuery) ([]model.Link, int64, error) {
	tx := r.db.Model(&model.Link{}).Where("deleted_at = 0")
	if q.GID != "" {
		tx = tx.Where("gid = ?", q.GID)
	}
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("short_uri LIKE ? OR origin_url LIKE ? OR title LIKE ? OR describe LIKE ?", like, like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "id DESC"
	switch q.Order {
	case "created_at_asc":
		order = "id ASC"
	case "pv_desc":
		order = "total_pv DESC, id DESC"
	}
	var links []model.Link
	err := tx.Order(order).Offset(int((q.Page - 1) * q.Size)).Limit(int(q.Size)).Find(&links).Error
	return links, total, err
}

// GetByID 查询单条（未删除）
func (r *LinkRepo) GetByID(id int64) (*model.Link, error) {
	var l model.Link
	if err := r.db.Where("id = ? AND deleted_at = 0", id).First(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// GetByShortURI 按短码查询未删除短链（跳转/创建冲突校验）
func (r *LinkRepo) GetByShortURI(uri string) (*model.Link, error) {
	var l model.Link
	if err := r.db.Where("short_uri = ? AND deleted_at = 0", uri).First(&l).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *LinkRepo) Create(l *model.Link) error {
	return r.db.Create(l).Error
}

func (r *LinkRepo) Update(l *model.Link) error {
	return r.db.Model(l).Select(
		"origin_url", "gid", "describe", "valid_date_type", "valid_date", "enable_status", "title", "updated_at",
	).Updates(l).Error
}

// SoftDelete 软删除：写入删除时间戳，配合唯一键 (short_uri, deleted_at) 允许同码重建
func (r *LinkRepo) SoftDelete(id int64) error {
	return r.db.Model(&model.Link{}).Where("id = ? AND deleted_at = 0", id).
		Updates(map[string]any{
			"deleted_at": time.Now().Unix(),
			"updated_at": time.Now(),
		}).Error
}

// Restore 从回收站恢复。若短码已被新链接占用会触发唯一约束冲突，由调用方转换为业务错误。
func (r *LinkRepo) Restore(id int64) (*model.Link, error) {
	var l model.Link
	if err := r.db.Where("id = ? AND deleted_at > 0", id).First(&l).Error; err != nil {
		return nil, err
	}
	if err := r.db.Model(&l).Updates(map[string]any{
		"deleted_at": 0,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return nil, err
	}
	return &l, nil
}

// PageRecycle 回收站分页
func (r *LinkRepo) PageRecycle(page, size int64, keyword string) ([]model.Link, int64, error) {
	tx := r.db.Model(&model.Link{}).Where("deleted_at > 0")
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("short_uri LIKE ? OR origin_url LIKE ?", like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var links []model.Link
	err := tx.Order("deleted_at DESC").Offset(int((page - 1) * size)).Limit(int(size)).Find(&links).Error
	return links, total, err
}

// AllActiveURIs 全量未删除短码（启动时重建布隆过滤器）
func (r *LinkRepo) AllActiveURIs() ([]string, error) {
	var uris []string
	err := r.db.Model(&model.Link{}).Where("deleted_at = 0").Pluck("short_uri", &uris).Error
	return uris, err
}

// CountAll 未删除短链总数（仪表盘）
func (r *LinkRepo) CountAll() (int64, error) {
	var n int64
	err := r.db.Model(&model.Link{}).Where("deleted_at = 0").Count(&n).Error
	return n, err
}

// SumTotalPV 全量短链历史点击总和（仪表盘）
func (r *LinkRepo) SumTotalPV() (int64, error) {
	var n int64
	err := r.db.Model(&model.Link{}).Select("COALESCE(SUM(total_pv),0)").Where("deleted_at = 0").Scan(&n).Error
	return n, err
}

// IncTotals 事件聚合后累计历史 PV/UV/UIP
func (r *LinkRepo) IncTotals(shortURI string, pv, uv, uip int64) error {
	return r.db.Model(&model.Link{}).
		Where("short_uri = ? AND deleted_at = 0", shortURI).
		Updates(map[string]any{
			"total_pv":  gorm.Expr("total_pv + ?", pv),
			"total_uv":  gorm.Expr("total_uv + ?", uv),
			"total_uip": gorm.Expr("total_uip + ?", uip),
		}).Error
}
