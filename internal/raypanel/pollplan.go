package raypanel

import (
	"strings"

	"naspanel/internal/layout"
	"naspanel/internal/nodes"
)

// PollTarget 单节点上的 chart 拉取任务。
type PollTarget struct {
	NodeKey string
	BaseURL string
	APIKey  string
	Charts  []string
}

type pollGroup struct {
	base   string
	apiKey string
	charts []string
}

// BuildPollTargets 按布局与节点表构造轮询计划。
func BuildPollTargets(lc *layout.LayoutConfig, ns *nodes.Store) []PollTarget {
	if lc == nil {
		return nil
	}
	var nf *nodes.File
	if ns != nil {
		nf = ns.Ptr()
	} else {
		d := nodes.DefaultFile()
		nf = &d
	}

	var fallbackBase, fallbackAPIKey string
	if fn, ok := nf.First(); ok {
		fallbackBase = strings.TrimSpace(fn.BaseURL())
		fallbackAPIKey = fn.APIKey
	}

	type pair struct{ nk, chart string }
	seen := make(map[pair]struct{})
	groups := make(map[string]*pollGroup)

	add := func(nodeKey, base, apiKey, chart string) {
		chart = strings.TrimSpace(chart)
		base = strings.TrimSpace(base)
		if chart == "" || base == "" {
			return
		}
		k := pair{nodeKey, chart}
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		g := groups[nodeKey]
		if g == nil {
			g = &pollGroup{base: base, apiKey: apiKey}
			groups[nodeKey] = g
		}
		g.charts = append(g.charts, chart)
	}

	for _, sc := range lc.Scenes {
		for i := range sc.Widgets {
			w := &sc.Widgets[i]
			chart := strings.TrimSpace(w.ChartID)
			if chart == "" {
				continue
			}
			nid := strings.TrimSpace(w.NodeID)
			if nid == "" {
				if fallbackBase != "" {
					add("", fallbackBase, fallbackAPIKey, chart)
				}
				continue
			}
			n, ok := nf.ByID(nid)
			if !ok {
				if fallbackBase != "" {
					add("", fallbackBase, fallbackAPIKey, chart)
				}
				continue
			}
			add(nid, n.BaseURL(), n.APIKey, chart)
		}
	}

	if len(groups) == 0 {
		return nil
	}
	out := make([]PollTarget, 0, len(groups))
	for nk, g := range groups {
		if len(g.charts) == 0 {
			continue
		}
		out = append(out, PollTarget{
			NodeKey: nk,
			BaseURL: g.base,
			APIKey:  g.apiKey,
			Charts:  g.charts,
		})
	}
	return out
}

// WidgetNetdataBase 解析某组件拉取 Netdata 时使用的 BaseURL 与 API Key（与 BuildPollTargets 规则一致）。
func WidgetNetdataBase(nf *nodes.File, w *layout.Widget) (base, apiKey string, ok bool) {
	if w == nil || strings.TrimSpace(w.ChartID) == "" {
		return "", "", false
	}
	if nf == nil {
		d := nodes.DefaultFile()
		nf = &d
	}
	nid := strings.TrimSpace(w.NodeID)
	if nid == "" {
		if fn, ok := nf.First(); ok {
			return fn.BaseURL(), fn.APIKey, true
		}
		return "", "", false
	}
	n, ok := nf.ByID(nid)
	if !ok {
		if fn, ok := nf.First(); ok {
			return fn.BaseURL(), fn.APIKey, true
		}
		return "", "", false
	}
	return n.BaseURL(), n.APIKey, true
}
