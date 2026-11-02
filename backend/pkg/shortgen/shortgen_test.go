package shortgen

import (
	"regexp"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	cases := []uint64{0, 1, 61, 62, 3844, 1 << 46, ^uint64(0) >> 18}
	for _, n := range cases {
		s := Encode(n)
		d, err := Decode(s)
		if err != nil {
			t.Fatalf("Decode(%q) 错误: %v", s, err)
		}
		if d != n {
			t.Fatalf("Encode/Decode 不一致: %d -> %q -> %d", n, s, d)
		}
	}
	if Encode(62) != "10" {
		t.Fatalf("62 应编码为 10，得到 %q", Encode(62))
	}
}

func TestGeneratorUniqueAndShape(t *testing.T) {
	g := New(1)
	seen := map[string]struct{}{}
	re := regexp.MustCompile(`^[0-9a-zA-Z]{5,8}$`)
	for i := 0; i < 5000; i++ {
		code := g.NewCode()
		if !re.MatchString(code) {
			t.Fatalf("短码 %q 形态不合法", code)
		}
		if _, dup := seen[code]; dup {
			t.Fatalf("短码 %q 重复", code)
		}
		seen[code] = struct{}{}
	}
}

func TestGeneratorTrendIncreasing(t *testing.T) {
	g := New(2)
	prev := g.Next()
	for i := 0; i < 1000; i++ {
		next := g.Next()
		if next <= prev {
			t.Fatalf("ID 非递增: %d -> %d", prev, next)
		}
		prev = next
	}
}
