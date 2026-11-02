// Package version 版本信息，由构建时注入：
//
//	go build -ldflags "-X github.com/benzi/slink/backend/internal/version.Version=v1.0.0"
package version

// Version 版本号，CI 构建时注入 tag；未注入时为 dev
var Version = "dev"

// Get 返回版本号（经函数封装，避免编译器对直接使用点的常量折叠）
func Get() string { return Version }
