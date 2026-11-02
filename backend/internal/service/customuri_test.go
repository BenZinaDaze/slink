package service

import "testing"

// TestCustomURIRule 自定义后缀规则：2-16 位，字母/数字/-/_
func TestCustomURIRule(t *testing.T) {
	valid := []string{
		"go", "cv",                                    // 短码便于记忆
		"install-script", "my_deploy_2026",            // 语义化长后缀
		"Ab3", "a1", "A-b_9",
	}
	invalid := []string{
		"", "g",                          // 长度不足 2
		"install-script-tool-x9",         // 超过 16 位
		"has.dot",                        // 含点（与静态资源路径冲突）
		"带中文", "with space", "a/b",   // 非法字符
	}
	for _, s := range valid {
		if !customURIRe.MatchString(s) {
			t.Errorf("%q 应为合法后缀", s)
		}
	}
	for _, s := range invalid {
		if customURIRe.MatchString(s) {
			t.Errorf("%q 应为非法后缀", s)
		}
	}
}
