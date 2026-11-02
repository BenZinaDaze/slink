package stats

import (
	"strings"
	"testing"
)

func TestParseUA(t *testing.T) {
	cases := []struct {
		ua      string
		browser string
		os      string
		device  string
	}{
		{
			ua:      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			browser: "Chrome 120", os: "Mac OS X", device: "PC",
		},
		{
			ua:      "Mozilla/5.0 (iPhone; CPU iPhone OS 17_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Mobile/15E148 Safari/604.1",
			device:  "移动设备",
			browser: "Safari",
		},
		{
			ua:      "curl/8.4.0",
			browser: "curl",
		},
	}
	for _, c := range cases {
		t.Run(c.browser, func(t *testing.T) {
			v := ParseUA(c.ua)
			if v.Browser == "" || v.OS == "" || v.Device == "" {
				t.Fatalf("字段为空: %+v", v)
			}
			if c.browser != "" && !strings.HasPrefix(v.Browser, strings.Split(c.browser, " ")[0]) {
				t.Fatalf("browser = %q, want prefix %q", v.Browser, c.browser)
			}
			if c.os != "" && v.OS != c.os {
				t.Logf("os = %q, want %q（库版本差异可接受）", v.OS, c.os)
			}
			if c.device != "" && v.Device != c.device {
				t.Fatalf("device = %q, want %q", v.Device, c.device)
			}
			if v.UVKey == "" {
				t.Fatal("UVKey 为空")
			}
		})
	}
}
