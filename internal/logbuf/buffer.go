// Package logbuf 提供环形日志缓冲，供 WebUI「系统日志」页展示。
package logbuf

import (
	"bytes"
	"sync"
)

const defaultMax = 2000

// Buffer 线程安全的按行环形缓冲。
// 内部使用固定大小环形数组，Write 和 Snapshot 均为 O(1) 摊销，不做 O(N) 移位。
type Buffer struct {
	mu   sync.Mutex
	ring []string // 长度固定为 max
	head int      // 下一条写入的位置
	size int      // 当前有效条目数
	max  int
}

// New 创建缓冲；maxLines<=0 则用 2000。
func New(maxLines int) *Buffer {
	if maxLines <= 0 {
		maxLines = defaultMax
	}
	return &Buffer{
		ring: make([]string, maxLines),
		max:  maxLines,
	}
}

// Write 实现 io.Writer，按换行切分。
func (b *Buffer) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, line := range bytes.Split(p, []byte{'\n'}) {
		s := string(bytes.TrimRight(line, "\r"))
		if s == "" {
			continue
		}
		b.ring[b.head] = s
		b.head = (b.head + 1) % b.max
		if b.size < b.max {
			b.size++
		}
	}
	return len(p), nil
}

// Snapshot 返回最近若干行（最旧在前）。
func (b *Buffer) Snapshot(limit int) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if limit <= 0 || limit > b.size {
		limit = b.size
	}
	if limit == 0 {
		return []string{}
	}
	out := make([]string, limit)
	// 从最旧的那条开始：起始位置 = (head - size + max) % max
	start := (b.head - b.size + b.max) % b.max
	// 跳过最老的 (size - limit) 条
	skip := b.size - limit
	start = (start + skip) % b.max
	for i := 0; i < limit; i++ {
		out[i] = b.ring[(start+i)%b.max]
	}
	return out
}
