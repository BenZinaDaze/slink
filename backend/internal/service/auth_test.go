package service

import (
	"testing"
	"time"
)

// TestReservedPathsFor 保留字随 admin.path 模式动态变化：
// 根路径模式保留全部前端路由；隐藏入口模式只保留真实路由与管理入口本身
func TestReservedPathsFor(t *testing.T) {
	root := ReservedPathsFor("")
	for _, want := range []string{"login", "links", "settings", "api", "assets"} {
		if _, ok := root[want]; !ok {
			t.Errorf("根路径模式应保留 %q", want)
		}
	}

	hidden := ReservedPathsFor("my-panel")
	for _, want := range []string{"my-panel", "api", "health"} {
		if _, ok := hidden[want]; !ok {
			t.Errorf("隐藏入口模式应保留 %q", want)
		}
	}
	// 隐藏模式下原前端路由词可作短码
	for _, code := range []string{"login", "links", "settings", "assets"} {
		if _, ok := hidden[code]; ok {
			t.Errorf("隐藏入口模式 %q 不应保留（可作短码）", code)
		}
	}
}

// TestLoginLockDuration 连续失败 n 次后的锁定时长：满 5 次起锁、30s 起步翻倍、封顶 1h
func TestLoginLockDuration(t *testing.T) {
	cases := []struct {
		fails int
		want  time.Duration
	}{
		{0, 0}, {1, 0}, {4, 0},
		{5, 30 * time.Second},
		{6, time.Minute},
		{7, 2 * time.Minute},
		{8, 4 * time.Minute},
		{10, 16 * time.Minute},
		{11, 32 * time.Minute},
		{12, time.Hour}, // 64min → 封顶
		{21, time.Hour},
		{10000, time.Hour}, // 防移位溢出
	}
	for _, tc := range cases {
		if got := loginLockDuration(tc.fails); got != tc.want {
			t.Errorf("loginLockDuration(%d) = %v, 期望 %v", tc.fails, got, tc.want)
		}
	}
}
