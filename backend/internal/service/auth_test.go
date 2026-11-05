package service

import (
	"testing"
	"time"
)

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
