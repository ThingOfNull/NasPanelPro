package raypanel

import (
	"sync"
)

// LineRings 为折线图维护每键最多 cap 个采样点。
type LineRings struct {
	mu       sync.Mutex
	cap      int
	data     map[string][]float64
	versions map[string]int64 // 每 key 数据版本，Set 时自增
}

func NewLineRings(capacity int) *LineRings {
	if capacity <= 0 {
		capacity = 60
	}
	return &LineRings{
		cap:      capacity,
		data:     make(map[string][]float64),
		versions: make(map[string]int64),
	}
}

// Set 用一整段序列替换该键（按 Netdata 时间窗取 points，而非逐帧 Push）。
// 每次调用都会递增该 key 的版本号。
func (r *LineRings) Set(key string, pts []float64) {
	if key == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]float64, len(pts))
	copy(cp, pts)
	r.data[key] = cp
	r.versions[key]++
}

// Version 返回该 key 当前的数据版本号；未初始化时返回 0。
func (r *LineRings) Version(key string) int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.versions[key]
}

func (r *LineRings) Get(key string) []float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.data[key]
	out := make([]float64, len(s))
	copy(out, s)
	return out
}
