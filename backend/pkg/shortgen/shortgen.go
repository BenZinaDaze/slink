// Package shortgen 短码生成器：秒级雪花 ID（秒级时间戳 + 节点 + 序列，
// 共 46 位）经 Base62 编码后得到 7-8 位短码。趋势递增、单节点内不冲突，
// 配合布隆过滤器/数据库唯一约束兜底。单机创建吞吐上限约 1024 个/秒。
package shortgen

import (
	"strconv"
	"sync"
	"time"
)

// epochSeconds 起始纪元 2020-01-01 UTC（保证 2026 年后生成的短码 >= 7 位）
const epochSeconds int64 = 1_577_836_800

const (
	nodeBits = 4
	seqBits  = 10
	seqMask  = (1 << seqBits) - 1
)

// Generator 秒级雪花 ID 生成器
type Generator struct {
	mu      sync.Mutex
	node    int64 // 节点编号，0-15
	lastSec int64
	seq     int64
}

func New(node int64) *Generator {
	return &Generator{node: node & ((1 << nodeBits) - 1)}
}

// Next 生成 46 位以内的唯一 ID
func (g *Generator) Next() uint64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now().Unix() - epochSeconds
	switch {
	case now < g.lastSec:
		now = g.lastSec // 时钟回拨时退化为沿用上一秒
	case now > g.lastSec:
		g.lastSec = now
		g.seq = 0
	default:
		g.seq = (g.seq + 1) & seqMask
		if g.seq == 0 { // 当前秒序列耗尽，自旋到下一秒
			for now <= g.lastSec {
				time.Sleep(time.Millisecond)
				now = time.Now().Unix() - epochSeconds
			}
			g.lastSec = now
		}
	}
	return uint64(now<<(nodeBits+seqBits) | g.node<<seqBits | g.seq)
}

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Encode 将非负整数编码为 Base62 字符串
func Encode(n uint64) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 11)
	for n > 0 {
		buf = append(buf, alphabet[n%62])
		n /= 62
	}
	// 反转
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

// Decode 将 Base62 字符串解码回整数
func Decode(s string) (uint64, error) {
	var n uint64
	for _, c := range s {
		var v int
		switch {
		case c >= '0' && c <= '9':
			v = int(c - '0')
		case c >= 'A' && c <= 'Z':
			v = int(c-'A') + 10
		case c >= 'a' && c <= 'z':
			v = int(c-'a') + 36
		default:
			return 0, strconv.ErrSyntax
		}
		n = n*62 + uint64(v)
	}
	return n, nil
}

// NewCode 生成一个短码
func (g *Generator) NewCode() string {
	return Encode(g.Next())
}
