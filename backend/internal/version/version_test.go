package version

import "testing"

func TestDefaultVersion(t *testing.T) {
	if Get() == "" {
		t.Fatal("Version 不能为空")
	}
}
