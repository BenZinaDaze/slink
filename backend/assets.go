// Package slink 根包：集中放置 go:embed 资源（迁移脚本、前端产物）。
// embed 指令只能引用包目录内的文件，故放在 backend 根。
package slink

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MigrationsFS 迁移脚本（embed.FS，含 ReadFile）
func MigrationsFS() embed.FS { return migrationsFS }

// Migrations 返回迁移脚本路径列表（按文件名排序执行）
func Migrations() []string {
	entries, _ := migrationsFS.ReadDir("migrations")
	var out []string
	for _, e := range entries {
		out = append(out, "migrations/"+e.Name())
	}
	return out
}

//go:embed all:web/dist
var webFS embed.FS

// WebFS 前端静态资源（构建产物占位文件见 web/dist；make build 会用真实产物替换）
func WebFS() fs.FS {
	sub, err := fs.Sub(webFS, "web/dist")
	if err != nil {
		return nil
	}
	return sub
}
