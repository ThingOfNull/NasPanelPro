package layout

import (
	"sync/atomic"
)

// Store 保存当前布局快照，供 Gin 与渲染循环并发读写。
type Store struct {
	v atomic.Value // *LayoutConfig
}

// Get 返回内存中布局的浅拷贝（避免调用方修改内部指针共享）。
func (s *Store) Get() LayoutConfig {
	x := s.v.Load()
	if x == nil {
		return DefaultLayout()
	}
	src := x.(*LayoutConfig)
	cp := *src
	return cp
}

// Put 替换快照（c 会被复制到堆上新对象）。
func (s *Store) Put(c LayoutConfig) {
	cp := c
	s.v.Store(&cp)
}

// Ptr 返回只读指针仅供渲染热路径（勿修改 *LayoutConfig 字段）。
func (s *Store) Ptr() *LayoutConfig {
	x := s.v.Load()
	if x == nil {
		d := DefaultLayout()
		return &d
	}
	return x.(*LayoutConfig)
}
