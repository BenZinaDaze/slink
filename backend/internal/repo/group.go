package repo

import (
	"gorm.io/gorm"

	"github.com/benzi/slink/backend/internal/model"
)

type GroupRepo struct{ db *gorm.DB }

func NewGroupRepo(db *gorm.DB) *GroupRepo { return &GroupRepo{db: db} }

func (r *GroupRepo) List() ([]model.Group, error) {
	var groups []model.Group
	err := r.db.Where("del_flag = 0").Order("sort_order ASC, id ASC").Find(&groups).Error
	return groups, err
}

func (r *GroupRepo) GetByGID(gid string) (*model.Group, error) {
	var g model.Group
	err := r.db.Where("gid = ? AND del_flag = 0", gid).First(&g).Error
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *GroupRepo) GetByName(name string) (*model.Group, error) {
	var g model.Group
	err := r.db.Where("name = ? AND del_flag = 0", name).First(&g).Error
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *GroupRepo) Create(g *model.Group) error {
	return r.db.Create(g).Error
}

func (r *GroupRepo) Update(g *model.Group) error {
	return r.db.Model(g).Updates(map[string]any{
		"name":       g.Name,
		"sort_order": g.SortOrder,
		"updated_at": gorm.Expr("datetime('now','localtime')"),
	}).Error
}

// Delete 软删除分组（调用方需先校验组内无短链）
func (r *GroupRepo) Delete(gid string) error {
	return r.db.Model(&model.Group{}).
		Where("gid = ?", gid).
		Updates(map[string]any{"del_flag": 1, "updated_at": gorm.Expr("datetime('now','localtime')")}).Error
}

// CountLinksByGID 统计各分组下未删除的短链数量，返回 gid -> count
func (r *GroupRepo) CountLinksByGID() (map[string]int64, error) {
	var rows []struct {
		GID   string `gorm:"column:gid"` // 三个连续大写字母需显式列名，否则 GORM 转成 gi_d
		Count int64  `gorm:"column:count"`
	}
	err := r.db.Table("t_link").
		Select("gid, count(*) as count").
		Where("deleted_at = 0").
		Group("gid").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	m := make(map[string]int64, len(rows))
	for _, row := range rows {
		m[row.GID] = row.Count
	}
	return m, nil
}

// CountLinks 单个分组的短链数量
func (r *GroupRepo) CountLinks(gid string) (int64, error) {
	var n int64
	err := r.db.Model(&model.Link{}).Where("gid = ? AND deleted_at = 0", gid).Count(&n).Error
	return n, err
}
