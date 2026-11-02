package repo

import (
	"gorm.io/gorm"

	"github.com/benzi/slink/backend/internal/model"
)

type SettingRepo struct{ db *gorm.DB }

func NewSettingRepo(db *gorm.DB) *SettingRepo { return &SettingRepo{db: db} }

func (r *SettingRepo) Get(key string) (string, bool, error) {
	var s model.Setting
	err := r.db.Where("k = ?", key).First(&s).Error
	if err == gorm.ErrRecordNotFound {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return s.V, true, nil
}

func (r *SettingRepo) Set(key, value string) error {
	return r.db.Exec(`
		INSERT INTO t_settings (k, v, updated_at) VALUES (?, ?, datetime('now','localtime'))
		ON CONFLICT (k) DO UPDATE SET v = excluded.v, updated_at = excluded.updated_at
	`, key, value).Error
}
