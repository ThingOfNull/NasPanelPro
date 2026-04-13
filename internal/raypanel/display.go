// Package raypanel 使用 raylib-go 绘制监控 UI，仅应由 cmd/naspanel 导入。
package raypanel

import (
	"sync/atomic"
)

// Display 挂起状态，供 Supervisor 与渲染循环协作。
type Display struct {
	suspended atomic.Bool
}

// NewDisplay 构造显示状态机。
func NewDisplay() *Display {
	return &Display{}
}

// SuspendRender / ResumeRender / RenderSuspended 供 Supervisor 对接。
func (d *Display) SuspendRender() { d.suspended.Store(true) }
func (d *Display) ResumeRender()  { d.suspended.Store(false) }
func (d *Display) RenderSuspended() bool {
	return d.suspended.Load()
}
