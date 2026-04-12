package raypanel

import (
	"context"
	"strconv"
	"strings"

	"naspanel/internal/layout"
	"naspanel/internal/netdata"
	"naspanel/internal/nodes"
)

type lineSeriesFetchKey struct {
	base   string
	apiKey string
	chart  string
	points int
}

func pickLineDim(w *layout.Widget, latest map[string]float64) string {
	if w == nil {
		return ""
	}
	for _, d := range w.Dimensions {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if _, ok := latest[d]; ok {
			return d
		}
	}
	for k := range latest {
		return k
	}
	return ""
}

// lineChartAfterSeconds 与 WebUI 预览一致：固定时间窗内多采样点，整段重绘而非每秒 Push 左移。
func lineChartAfterSeconds(points int) string {
	if points <= 0 {
		points = 96
	}
	sec := 180
	if points > 120 {
		sec = points * 2
		if sec > 3600 {
			sec = 3600
		}
	}
	return "-" + strconv.Itoa(sec)
}

// FillLineRings 用 Netdata /api/v1/data 多 points 响应刷新折线图（整段替换）。
func FillLineRings(ctx context.Context, lc *layout.LayoutConfig, sceneIndices []int, nf *nodes.File, rings *LineRings) {
	if lc == nil || rings == nil {
		return
	}
	if nf == nil {
		d := nodes.DefaultFile()
		nf = &d
	}

	type assign struct {
		ringKey string
		w       *layout.Widget
		fk      lineSeriesFetchKey
	}
	var assigns []assign

	uniq := make(map[lineSeriesFetchKey]struct{})
	for _, si := range sceneIndices {
		if si < 0 || si >= len(lc.Scenes) {
			continue
		}
		sc := &lc.Scenes[si]
		for wi := range sc.Widgets {
			w := &sc.Widgets[wi]
			if w.Type != layout.WidgetLine {
				continue
			}
			chart := strings.TrimSpace(w.ChartID)
			if chart == "" {
				continue
			}
			base, apiKey, ok := WidgetNetdataBase(nf, w)
			if !ok || base == "" {
				continue
			}
			points := w.LinePoints
			if points <= 0 {
				points = 96
			}
			fk := lineSeriesFetchKey{base: base, apiKey: apiKey, chart: chart, points: points}
			uniq[fk] = struct{}{}
			assigns = append(assigns, assign{
				ringKey: ringKey(si, wi, w),
				w:       w,
				fk:      fk,
			})
		}
	}
	if len(assigns) == 0 {
		return
	}

	seriesByKey := make(map[lineSeriesFetchKey]*netdata.ChartDataSeries, len(uniq))
	for fk := range uniq {
		cl := &netdata.Client{BaseURL: fk.base, APIKey: fk.apiKey}
		after := lineChartAfterSeconds(fk.points)
		sr, err := cl.FetchChartSeries(ctx, fk.chart, netdata.DataOpts{
			After:  after,
			Points: fk.points,
		})
		if err != nil || sr == nil || len(sr.Series) == 0 {
			continue
		}
		seriesByKey[fk] = sr
	}

	for _, a := range assigns {
		sr, ok := seriesByKey[a.fk]
		if !ok {
			continue
		}
		dim := pickLineDim(a.w, sr.Latest)
		if dim == "" {
			continue
		}
		pts, ok := sr.Series[dim]
		if !ok || len(pts) < 2 {
			continue
		}
		rings.Set(a.ringKey, pts)
	}
}
