// Package monitor 从 gopsutil 采集指标并推送到 SnapshotSink，不依赖任何图形后端。
package monitor

import (
	"context"
	"math"
	"time"

	"naspanel/internal/core"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

// SnapshotSink 由 Raylib 等 UI 实现；monitor 仅依赖此接口。
type SnapshotSink interface {
	UpdateSnapshot(*core.Snapshot)
}

// Service 周期性采集并推送。
type Service struct {
	sink     SnapshotSink
	Interval time.Duration
}

// NewService 构造采集服务。
func NewService(sink SnapshotSink, interval time.Duration) *Service {
	if interval <= 0 {
		interval = time.Second
	}
	return &Service{sink: sink, Interval: interval}
}

// Run 阻塞直到 ctx 结束。
func (s *Service) Run(ctx context.Context) error {
	_, _ = cpu.Percent(0, false)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(200 * time.Millisecond):
	}

	t := time.NewTicker(s.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			s.sampleOnce()
		}
	}
}

func (s *Service) sampleOnce() {
	snap := &core.Snapshot{}

	if v, err := mem.VirtualMemory(); err == nil {
		snap.MemUsedBytes = v.Used
		snap.MemTotalBytes = v.Total
	}

	if pct, err := cpu.Percent(0, false); err == nil && len(pct) > 0 {
		snap.CPUPercent = pct[0]
	}

	snap.CPUTempC = math.NaN()
	if temps, err := host.SensorsTemperatures(); err == nil {
		var maxC float64
		var found bool
		for _, t := range temps {
			if t.Temperature <= 0 {
				continue
			}
			if !found || t.Temperature > maxC {
				maxC = t.Temperature
				found = true
			}
		}
		if found {
			snap.CPUTempC = maxC
		}
	}

	s.sink.UpdateSnapshot(snap)
}
