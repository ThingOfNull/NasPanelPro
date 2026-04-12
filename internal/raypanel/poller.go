package raypanel

import (
	"context"
	"strings"
	"sync"
	"time"

	"naspanel/internal/layout"
	"naspanel/internal/netdata"
	"naspanel/internal/nodes"
)

// Poller 轮询 Netdata /api/v1/data；支持多节点（Plan）或单 BaseURL 回退。
type Poller struct {
	mu sync.Mutex
	// 单节点回退（Plan 为 nil 时使用）
	baseURL string
	client  *netdata.Client
	charts  []string
	// 多节点计划（非 nil 时优先）
	plan func() []PollTarget

	// 折线图：按时间窗拉取序列写入 LineRings（在轮询协程内调用，与 Snapshot 同一节拍）
	lineRings      *LineRings
	layoutScenes   func() (*layout.LayoutConfig, []int)
	lineNodeStore  LineNodeStore

	snapMu sync.RWMutex
	snap   netdata.DataSnapshot
}

// LineNodeStore 供折线序列解析节点 BaseURL（与 *nodes.Store 一致的最小接口）。
type LineNodeStore interface {
	Ptr() *nodes.File
}

func NewPoller(baseURL string) *Poller {
	return &Poller{
		client: &netdata.Client{BaseURL: baseURL},
		snap:   netdata.DataSnapshot{},
	}
}

// SetPlan 设置多节点轮询计划生成器；设为 nil 则退回 SetBaseURL/SetCharts。
func (p *Poller) SetPlan(fn func() []PollTarget) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.plan = fn
}

// SetBaseURL 每帧由渲染循环更新（Plan 为 nil 时有效）。
func (p *Poller) SetBaseURL(url string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.baseURL = strings.TrimSpace(url)
	if p.client == nil {
		p.client = &netdata.Client{BaseURL: p.baseURL}
		return
	}
	p.client.BaseURL = p.baseURL
}

// SetCharts Plan 为 nil 时使用。
// SetLineFill 在轮询 tick 内刷新折线序列（layoutScenes 返回当前布局与参与轮询的场景下标）。
func (p *Poller) SetLineFill(rings *LineRings, layoutScenes func() (*layout.LayoutConfig, []int), ns LineNodeStore) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.lineRings = rings
	p.layoutScenes = layoutScenes
	p.lineNodeStore = ns
}

func (p *Poller) SetCharts(ids []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	seen := make(map[string]struct{})
	var out []string
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	p.charts = out
}

func (p *Poller) chartsCopy() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.charts))
	copy(out, p.charts)
	return out
}

// Snapshot 返回最近一次成功的数据快照。
func (p *Poller) Snapshot() netdata.DataSnapshot {
	p.snapMu.RLock()
	defer p.snapMu.RUnlock()
	if p.snap == nil {
		return netdata.DataSnapshot{}
	}
	out := make(netdata.DataSnapshot, len(p.snap))
	for k, v := range p.snap {
		inner := make(map[string]float64, len(v))
		for dk, dv := range v {
			inner[dk] = dv
		}
		out[k] = inner
	}
	return out
}

func (p *Poller) refreshLineRings(ctx context.Context) {
	p.mu.Lock()
	rings := p.lineRings
	ls := p.layoutScenes
	ns := p.lineNodeStore
	p.mu.Unlock()
	if rings == nil || ls == nil {
		return
	}
	lc, si := ls()
	var nf *nodes.File
	if ns != nil {
		nf = ns.Ptr()
	}
	FillLineRings(ctx, lc, si, nf, rings)
}

// Run 阻塞直到 ctx 取消。
func (p *Poller) Run(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Second
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.mu.Lock()
			planFn := p.plan
			base := p.baseURL
			charts := p.charts
			legacyClient := p.client
			p.mu.Unlock()

			var merged netdata.DataSnapshot
			if planFn != nil {
				targets := planFn()
				merged = make(netdata.DataSnapshot)
				for _, tg := range targets {
					if tg.BaseURL == "" || len(tg.Charts) == 0 {
						continue
					}
					cl := &netdata.Client{BaseURL: tg.BaseURL, APIKey: tg.APIKey}
					snap, err := cl.FetchChartsData(ctx, tg.Charts, netdata.DataOpts{})
					if err != nil || snap == nil {
						continue
					}
					for ck, dims := range snap {
						key := layout.ChartSnapKey(tg.NodeKey, ck)
						merged[key] = dims
					}
				}
			} else {
				if len(charts) == 0 || base == "" {
					p.refreshLineRings(ctx)
					continue
				}
				if legacyClient == nil {
					legacyClient = &netdata.Client{BaseURL: base}
				} else {
					legacyClient.BaseURL = base
				}
				snap, err := legacyClient.FetchChartsData(ctx, charts, netdata.DataOpts{})
				if err != nil || snap == nil {
					p.refreshLineRings(ctx)
					continue
				}
				merged = snap
			}
			if len(merged) > 0 {
				p.snapMu.Lock()
				p.snap = merged
				p.snapMu.Unlock()
			}
			p.refreshLineRings(ctx)
		}
	}
}

// ChartsForLayoutScene 从布局提取某场景的 chart id。
func ChartsForLayoutScene(lc *layout.LayoutConfig, sceneIndex int) []string {
	if lc == nil {
		return nil
	}
	return lc.ChartsUsedInScene(sceneIndex)
}

// ChartsForLayoutScenes 合并多场景的 chart（去重）。
func ChartsForLayoutScenes(lc *layout.LayoutConfig, indices ...int) []string {
	if lc == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, si := range indices {
		for _, id := range lc.ChartsUsedInScene(si) {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}
