package stats

import (
	"log"
	"strings"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

// LocaleResolver IP → 地域解析器。xdb 文件缺失或加载失败时降级为"未知"。
// 默认 xdb 为 IPv4 库；IPv6 地址无法解析，返回"未知"。
type LocaleResolver struct {
	searcher *xdb.Searcher
}

// NewLocaleResolver 加载 ip2region xdb 离线库；路径为空或加载失败时返回降级实例
func NewLocaleResolver(xdbPath string) *LocaleResolver {
	if xdbPath == "" {
		return &LocaleResolver{}
	}
	c, err := xdb.LoadContentFromFile(xdbPath)
	if err != nil {
		log.Printf("[stats] ip2region 库加载失败，地域统计降级为未知: %v", err)
		return &LocaleResolver{}
	}
	s, err := xdb.NewWithBuffer(xdb.IPv4, c)
	if err != nil {
		log.Printf("[stats] ip2region 初始化失败: %v", err)
		return &LocaleResolver{}
	}
	return &LocaleResolver{searcher: s}
}

// Resolve 返回地域描述，如 "中国 广东省 深圳"；内网/未知返回相应标记
func (r *LocaleResolver) Resolve(ip string) string {
	if r.searcher == nil || ip == "" || strings.HasPrefix(ip, "127.") || strings.HasPrefix(ip, "::1") {
		return "本地/未知"
	}
	region, err := r.searcher.Search(ip)
	if err != nil || region == "" {
		return "未知"
	}
	// 原始格式：国家|区域|省份|城市|ISP
	parts := strings.Split(region, "|")
	var out []string
	for _, p := range parts[:4] { // ISP 不计入地域维度
		if p != "" && p != "0" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return "未知"
	}
	return strings.Join(out, " ")
}
