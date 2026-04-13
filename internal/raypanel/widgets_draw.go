package raypanel

import (
	"fmt"
	"math"
	"strings"

	"naspanel/internal/layout"
	"naspanel/internal/netdata"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func widgetSnapKey(w *layout.Widget) string {
	if w == nil {
		return ""
	}
	return layout.ChartSnapKey(w.NodeID, w.ChartID)
}

func widgetPrimaryValue(snap netdata.DataSnapshot, w *layout.Widget) (float64, bool) {
	if snap == nil || w == nil {
		return 0, false
	}
	dims, ok := snap[widgetSnapKey(w)]
	if !ok {
		return 0, false
	}
	for _, d := range w.Dimensions {
		if v, ok := dims[d]; ok {
			return v, true
		}
	}
	return 0, false
}

func formatVal(v float64, unit string) string {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "percent":
		return fmt.Sprintf("%.1f%%", v)
	case "bytes":
		return formatBytesFromKiB(v)
	case "none", "":
		return fmt.Sprintf("%.2f", v)
	default:
		return fmt.Sprintf("%.2f", v)
	}
}

// formatValCompact 用于纵轴窄栏。
func formatValCompact(v float64, unit string) string {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "percent":
		return fmt.Sprintf("%.0f%%", v)
	case "bytes":
		return formatBytesCompactKiB(v)
	case "none", "":
		if math.Abs(v) >= 1000 {
			return fmt.Sprintf("%.0f", v)
		}
		return fmt.Sprintf("%.1f", v)
	default:
		if math.Abs(v) >= 1000 {
			return fmt.Sprintf("%.0f", v)
		}
		return fmt.Sprintf("%.1f", v)
	}
}

// formatBytesFromKiB 将 Netdata 常见的 KiB 绝对值转为可读字符串。
func formatBytesFromKiB(kib float64) string {
	bytes := kib * 1024
	const k = 1024.0
	gi := bytes / (k * k * k)
	if gi >= 1 {
		return fmt.Sprintf("%.2f GiB", gi)
	}
	mi := bytes / (k * k)
	return fmt.Sprintf("%.1f MiB", mi)
}

func formatBytesCompactKiB(kib float64) string {
	bytes := kib * 1024
	const k = 1024.0
	gi := bytes / (k * k * k)
	if gi >= 1 {
		return fmt.Sprintf("%.1fG", gi)
	}
	mi := bytes / (k * k)
	if mi >= 1 {
		return fmt.Sprintf("%.0fM", mi)
	}
	ki := bytes / k
	return fmt.Sprintf("%.0fK", ki)
}

func widgetDisplayTitle(w *layout.Widget) string {
	if w == nil {
		return ""
	}
	if t := strings.TrimSpace(w.Label); t != "" {
		return t
	}
	if t := strings.TrimSpace(w.ChartID); t != "" {
		return t
	}
	return string(w.Type)
}

func widgetTitleHeight(w *layout.Widget) float32 {
	if w == nil || w.HideLabel || w.Type == layout.WidgetText {
		return 0
	}
	if widgetDisplayTitle(w) == "" {
		return 0
	}
	return 22
}

func drawWidgetTitleBar(w *layout.Widget, x, y, fw, th float32) {
	lbl := widgetDisplayTitle(w)
	if lbl == "" {
		return
	}
	sz := float32(15)
	if th < 20 {
		sz = 12
	}
	drawUIText(panelUIFont, panelUIFontOk, lbl, x+6, y+3, sz, colText)
}

func drawScene(lc *layout.LayoutConfig, sceneIndex int, snap netdata.DataSnapshot, rings *LineRings, lw, lh int32) {
	if lc == nil || sceneIndex < 0 || sceneIndex >= len(lc.Scenes) {
		return
	}
	sc := &lc.Scenes[sceneIndex]
	rl.DrawRectangle(0, 0, lw, lh, colBackground)
	for wi := range sc.Widgets {
		w := &sc.Widgets[wi]
		key := ringKey(sceneIndex, wi, w)
		drawWidget(w, snap, rings, key, lw, lh)
	}
}

func ringKey(sceneIndex, widgetIndex int, w *layout.Widget) string {
	if w.ID != "" {
		return w.ID
	}
	return fmt.Sprintf("%d:%d", sceneIndex, widgetIndex)
}

func drawWidget(w *layout.Widget, snap netdata.DataSnapshot, rings *LineRings, ringKey string, lw, lh int32) {
	x, y := float32(w.X), float32(w.Y)
	fw, fh := float32(w.W), float32(w.H)
	th := widgetTitleHeight(w)
	ix, iy := x, y+th
	iw, ih := fw, fh-th

	if th > 0 {
		rl.DrawRectangle(int32(x), int32(y), int32(fw), int32(th), colTitleBar)
		drawWidgetTitleBar(w, x, y, fw, th)
	}

	switch w.Type {
	case layout.WidgetText:
		drawWidgetText(w, snap, ix, iy, iw, ih)
	case layout.WidgetGauge:
		drawWidgetGauge(w, snap, ix, iy, iw, ih)
	case layout.WidgetLine:
		drawWidgetLine(w, rings, ringKey, ix, iy, iw, ih)
	case layout.WidgetProgress:
		drawWidgetProgress(w, snap, ix, iy, iw, ih)
	case layout.WidgetHistogram:
		drawWidgetHistogram(w, snap, ix, iy, iw, ih)
	}

	if w.ShowBorder {
		rl.DrawRectangleLines(int32(x), int32(y), int32(fw), int32(fh), colMuted)
	}
}

func drawWidgetText(w *layout.Widget, snap netdata.DataSnapshot, x, y, fw, fh float32) {
	var msg string
	if strings.TrimSpace(w.ChartID) == "" {
		msg = w.Label
	} else {
		if v, ok := widgetPrimaryValue(snap, w); ok {
			if strings.EqualFold(w.Unit, "bytes") {
				msg = formatBytesFromKiB(v)
			} else {
				msg = formatVal(v, w.Unit)
			}
		} else {
			msg = "—"
		}
		if w.Label != "" {
			msg = w.Label + "\n" + msg
		}
	}
	sz := int32(fh * 0.45)
	if sz < 16 {
		sz = 16
	}
	if sz > 72 {
		sz = 72
	}
	drawUIMultiline(panelUIFont, panelUIFontOk, msg, x+4, y+4, float32(sz), colText)
}

func drawWidgetGauge(w *layout.Widget, snap netdata.DataSnapshot, x, y, fw, fh float32) {
	v, ok := widgetPrimaryValue(snap, w)
	if !ok {
		v = 0
	}
	if strings.EqualFold(w.Unit, "percent") {
		// 已是 0-100，直接使用
	} else {
		// unit 为空或非 percent：值 > 1 视为 0-100 量纲，除以 100 得比例后再乘回；
		// 值本身已在 0-1 范围内（如归一化比例）则直接乘 100。
		v = math.Abs(v)
		if v > 1 {
			v = v / 100.0
		}
		if v > 1 {
			v = 1
		}
		v *= 100
	}
	cx := x + fw/2
	cy := y + fh*0.55
	r := float32(math.Min(float64(fw), float64(fh))) * 0.38
	inner := r * 0.65
	spanStart := float32(180)
	spanEnd := float32(360)
	rl.DrawRingLines(rl.NewVector2(cx, cy), inner, r, spanStart, spanEnd, 32, colBarTrack)
	p := float32(math.Max(0, math.Min(100, float64(v)))) / 100
	end := spanStart + (spanEnd-spanStart)*p
	col := colAccent
	if w.CriticalThreshold > 0 && v >= w.CriticalThreshold {
		col = rl.NewColor(248, 81, 73, 255)
	} else if w.WarnThreshold > 0 && v >= w.WarnThreshold {
		col = rl.NewColor(210, 153, 34, 255)
	}
	if p > 0.001 {
		rl.DrawRing(rl.NewVector2(cx, cy), inner, r, spanStart, end, 32, col)
	}
	drawUIText(panelUIFont, panelUIFontOk, fmt.Sprintf("%.0f%%", v), cx-r, cy, 18, colMuted)
}

func drawWidgetLine(w *layout.Widget, rings *LineRings, key string, x, y, fw, fh float32) {
	axisW := float32(0)
	if w.ShowYAxis {
		axisW = 42
		if axisW > fw*0.35 {
			axisW = fw * 0.35
		}
	}
	px := x + axisW
	pw := fw - axisW
	if pw < 8 {
		pw = 8
	}
	py0 := y + 4
	ph := fh - 8
	if ph < 4 {
		ph = 4
	}

	pts := rings.Get(key)
	if len(pts) < 2 {
		rl.DrawRectangleLines(int32(x), int32(y), int32(fw), int32(fh), colMuted)
		drawUIText(panelUIFont, panelUIFontOk, "line", x+4, y+4, 14, colMuted)
		return
	}
	minV, maxV := pts[0], pts[0]
	for _, p := range pts {
		if p < minV {
			minV = p
		}
		if p > maxV {
			maxV = p
		}
	}
	if maxV == minV {
		maxV = minV + 1
	}
	py := func(val float64) int32 {
		t := (val - minV) / (maxV - minV)
		return int32(py0 + ph - float32(t)*ph)
	}
	den := float32(len(pts) - 1)
	if den < 1 {
		den = 1
	}
	for i := 1; i < len(pts); i++ {
		x0 := int32(px + float32(i-1)/den*pw)
		x1 := int32(px + float32(i)/den*pw)
		rl.DrawLine(x0, py(pts[i-1]), x1, py(pts[i]), colAccent)
	}

	if w.ShowYAxis && axisW > 4 {
		sMax := formatValCompact(maxV, w.Unit)
		sMid := formatValCompact((minV+maxV)/2, w.Unit)
		sMin := formatValCompact(minV, w.Unit)
		tx := x + 2
		drawUIText(panelUIFont, panelUIFontOk, sMax, tx, py0, 11, colMuted)
		midY := py0 + ph/2 - 6
		drawUIText(panelUIFont, panelUIFontOk, sMid, tx, midY, 11, colMuted)
		drawUIText(panelUIFont, panelUIFontOk, sMin, tx, py0+ph-12, 11, colMuted)
		if axisW > 6 {
			gx := int32(px - 1)
			rl.DrawLine(gx, int32(py0), gx, int32(py0+ph), colBarTrack)
		}
	}
}

func drawWidgetProgress(w *layout.Widget, snap netdata.DataSnapshot, x, y, fw, fh float32) {
	v, ok := widgetPrimaryValue(snap, w)
	if !ok {
		v = 0
	}
	p := v
	if strings.EqualFold(w.Unit, "percent") {
		// 已是 0-100，归一化为比例
		p = v / 100
	} else if v > 1 {
		// 非 percent 且值 > 1：视为 0-100 量纲，同 Gauge 逻辑
		p = v / 100
	}
	// 值在 0-1 范围内时 p 直接使用
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	col := colAccent
	if w.CriticalThreshold > 0 && v >= w.CriticalThreshold {
		col = rl.NewColor(248, 81, 73, 255)
	} else if w.WarnThreshold > 0 && v >= w.WarnThreshold {
		col = rl.NewColor(210, 153, 34, 255)
	}
	rl.DrawRectangle(int32(x), int32(y), int32(fw), int32(fh), colBarTrack)
	if w.Vertical {
		hfill := float32(p) * fh
		rl.DrawRectangle(int32(x), int32(y+fh-hfill), int32(fw), int32(hfill), col)
	} else {
		wfill := float32(p) * fw
		rl.DrawRectangle(int32(x), int32(y), int32(wfill), int32(fh), col)
	}
}

func drawWidgetHistogram(w *layout.Widget, snap netdata.DataSnapshot, x, y, fw, fh float32) {
	dims, ok := snap[widgetSnapKey(w)]
	if !ok || len(w.Dimensions) == 0 {
		return
	}
	vals := make([]float64, 0, len(w.Dimensions))
	for _, d := range w.Dimensions {
		if v, ok := dims[d]; ok {
			vals = append(vals, v)
		}
	}
	if len(vals) == 0 {
		return
	}
	maxV := vals[0]
	for _, v := range vals {
		if v > maxV {
			maxV = v
		}
	}
	if maxV <= 0 {
		maxV = 1
	}
	bw := fw / float32(len(vals))
	for i, v := range vals {
		h := float32(v/maxV) * (fh - 8)
		bx := x + float32(i)*bw + 2
		rl.DrawRectangle(int32(bx), int32(y+fh-h), int32(bw-4), int32(h), colAccent)
	}
}
