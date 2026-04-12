package raypanel

import (
	"naspanel/internal/cfg"
	"naspanel/internal/layout"
	"naspanel/internal/nodes"
)

// Options Raylib 主循环参数（PRD 2.0）。
type Options struct {
	Config      cfg.Config
	LayoutStore *layout.Store
	LayoutPath  string // 非空则定期从磁盘重载
	NodesPath   string // 非空则定期重载 configs/nodes.json
	NodeStore *nodes.Store
	Display   *Display
}
