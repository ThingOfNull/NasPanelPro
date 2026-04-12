package raypanel

import (
	"sync"
)

// LineRings 为折线图维护每键最多 cap 个采样点。
type LineRings struct {
	mu   sync.Mutex
	cap  int
	data map[string][]float64
}

func NewLineRings(capacity int) *LineRings {
	if capacity <= 0 {
		capacity = 60
	}
	return &LineRings{cap: capacity, data: make(map[string][]float64)}
}

func (r *LineRings) Push(key string, v float64) {
	if key == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.data[key]
	s = append(s, v)
	if len(s) > r.cap {
		s = s[len(s)-r.cap:]
	}
	r.data[key] = s
}

// Set 用一整段序列替换该键（与 WebUI 一样按 Netdata 时间窗取 points，而非逐帧 Push）。
func (r *LineRings) Set(key string, pts []float64) {
	if key == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]float64, len(pts))
	copy(cp, pts)
	r.data[key] = cp
}

func (r *LineRings) Get(key string) []float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.data[key]
	out := make([]float64, len(s))
	copy(out, s)
	return out
}
