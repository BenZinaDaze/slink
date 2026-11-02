package repo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	slink "github.com/benzi/slink/backend"
)

var _ = slink.Migrations // 确保 embed 资源包被引用

// Open 打开 SQLite（WAL 模式）并执行内嵌迁移脚本
func Open(path string) (*gorm.DB, error) {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建数据目录失败: %w", err)
		}
	}
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite 失败: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	// SQLite 单写多读：单连接 + WAL，避免 database locked
	sqlDB.SetMaxOpenConns(1)

	if err := migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

func migrate(db *gorm.DB) error {
	mfs := slink.MigrationsFS()
	for _, name := range slink.Migrations() {
		sql, err := mfs.ReadFile(name)
		if err != nil {
			return err
		}
		if err := db.Exec(string(sql)).Error; err != nil {
			return fmt.Errorf("执行迁移 %s 失败: %w", name, err)
		}
	}
	return nil
}
