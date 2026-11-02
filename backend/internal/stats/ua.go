// Package stats 统计管道：消费 Redis Stream 中的访问事件，
// 解析 UA / IP 地域，聚合 PV/UV/UIP 与维度统计，批量落库。
package stats

import (
	"crypto/sha1"
	"encoding/hex"

	ua "github.com/mssola/useragent"
)

// Visitor 访问者画像
type Visitor struct {
	Browser string // 浏览器
	OS      string // 操作系统
	Device  string // 设备类型：PC / 移动设备
	UVKey   string // 访客指纹（ua 哈希），配合 IP 用于 UV 去重
}

// ParseUA 解析 User-Agent
func ParseUA(userAgent string) Visitor {
	if userAgent == "" {
		return Visitor{Browser: "未知", OS: "未知", Device: "未知"}
	}
	parsed := ua.New(userAgent)
	browserName, browserVersion := parsed.Browser()
	browser := "未知"
	if browserName != "" {
		browser = browserName
		if browserVersion != "" {
			browser = browserName + " " + majorVersion(browserVersion)
		}
	}
	osName := parsed.OS()
	if osName == "" {
		osName = "未知"
	}
	device := "PC"
	if parsed.Mobile() {
		device = "移动设备"
	}
	sum := sha1.Sum([]byte(userAgent))
	return Visitor{
		Browser: browser,
		OS:      osName,
		Device:  device,
		UVKey:   hex.EncodeToString(sum[:8]),
	}
}

// majorVersion 只保留主版本号，避免维度值过碎（Chrome 120/121... 各自一行）
func majorVersion(v string) string {
	for i := 0; i < len(v); i++ {
		if v[i] == '.' || v[i] == '-' || v[i] == '_' {
			return v[:i]
		}
	}
	return v
}
