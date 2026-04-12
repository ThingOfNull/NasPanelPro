// Package core 存放与具体渲染后端无关的共享类型（Ebitengine / Raylib 等均可依赖，不得依赖任何图形库）。
package core

import "math"

// Snapshot 为监控数据快照，由 monitor 写入、由各 UI 引擎读取。
type Snapshot struct {
	CPUPercent    float64
	CPUTempC      float64 // 无传感器时为 NaN
	MemUsedBytes  uint64
	MemTotalBytes uint64
}

// HasValidTemp 判断温度是否可用于展示。
func HasValidTemp(c float64) bool {
	return !math.IsNaN(c) && c > 0
}
