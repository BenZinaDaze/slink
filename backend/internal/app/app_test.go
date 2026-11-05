package app

import (
	"bytes"
	"testing"
	"testing/fstest"
)

// TestApplyAdminBase 资产路径改写：HTML 属性、CSS url()、favicon 全部指向前缀路径
func TestApplyAdminBase(t *testing.T) {
	in := []byte(`<link href="/favicon.svg"><script src="/assets/index.js"></script>` +
		`<link href="/assets/index.css">` +
		`@font-face{src:url(/assets/inter.woff2)}`)
	out := string(applyAdminBase(in, "/my-panel"))
	for _, want := range []string{
		`href="/my-panel/favicon.svg"`,
		`src="/my-panel/assets/index.js"`,
		`href="/my-panel/assets/index.css"`,
		`url(/my-panel/assets/inter.woff2)`,
	} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Errorf("改写结果缺少 %s\n实际: %s", want, out)
		}
	}
	if bytes.Contains([]byte(out), []byte(`href="/assets/`)) {
		t.Error("仍存在未改写的根相对资产路径")
	}
}

// TestPrepareAdminIndex index.html 预改写需注入 basename meta（meta 不受 CSP script-src 限制）
func TestPrepareAdminIndex(t *testing.T) {
	webFS := fstest.MapFS{"index.html": &fstest.MapFile{
		Data: []byte(`<html><head><title>x</title></head></html>`),
	}}
	html := prepareAdminIndex(webFS, "/my-panel")
	if html == nil {
		t.Fatal("应返回改写后的 index.html")
	}
	if !bytes.Contains(html, []byte(`<meta name="slink-admin-base" content="/my-panel">`)) {
		t.Errorf("缺少 basename meta: %s", html)
	}
	// 未配置前缀时返回 nil（根路径模式走原逻辑）
	if prepareAdminIndex(webFS, "") != nil {
		t.Error("根路径模式应返回 nil")
	}
}
