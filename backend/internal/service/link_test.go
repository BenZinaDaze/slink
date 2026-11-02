package service

import (
	"testing"
)

func TestExtractTitle(t *testing.T) {
	cases := []struct {
		name string
		html string
		want string
	}{
		{"标准", "<html><head><title>Hello</title></head><body></body></html>", "Hello"},
		{"带空白", "<html><head><title>  短链接  </title></head></html>", "短链接"},
		{"body中title标签", "<html><head></head><body><title>X</title></body></html>", "X"},
		{"无标题", "<html><head></head><body>hello</body></html>", ""},
		{"中文", "<title>短链接系统</title>", "短链接系统"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := extractTitle([]byte(c.html)); got != c.want {
				t.Fatalf("extractTitle = %q, want %q", got, c.want)
			}
		})
	}
}

func TestExtractTitleTruncated(t *testing.T) {
	long := make([]byte, 300)
	for i := range long {
		long[i] = 'a'
	}
	html := "<title>" + string(long) + "</title>"
	got := extractTitle([]byte(html))
	if len(got) > 256 {
		t.Fatalf("标题未截断: %d", len(got))
	}
}
