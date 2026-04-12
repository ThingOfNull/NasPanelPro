// Package logbuf 提供环形日志缓冲，供 WebUI「系统日志」页展示。
package logbuf

import (
	"bytes"
	"sync"
)

const defaultMax = 2000

// Buffer 线程安全的按行环形缓冲。
type Buffer struct {
	mu    sync.Mutex
	lines []string
	max   int
}

// New 创建缓冲；maxLines<=0 则用 2000。
func New(maxLines int) *Buffer {
	if maxLines <= 0 {
		maxLines = defaultMax
	}
	return &Buffer{max: maxLines}
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
		if len(b.lines) >= b.max {
			copy(b.lines, b.lines[1:])
			b.lines = b.lines[:len(b.lines)-1]
		}
		b.lines = append(b.lines, s)
	}
	return len(p), nil
}

// Snapshot 返回最近若干行（最旧在前）。
func (b *Buffer) Snapshot(limit int) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if limit <= 0 || limit > len(b.lines) {
		limit = len(b.lines)
	}
	start := len(b.lines) - limit
	out := make([]string, limit)
	copy(out, b.lines[start:])
	return out
}
