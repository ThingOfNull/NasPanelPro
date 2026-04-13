package raypanel

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"naspanel/internal/layout"
	"naspanel/internal/metricexpr"
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
	if w.CompositeDimsExpr {
		lines := metricexpr.NonEmptyExprLines(w.ValueExpr)
		if len(lines) == 0 {
			return 0, false
		}
		line := lines[0]
		ids, err := metricexpr.CompositeEnvDimensionIDs(lines)
		if err != nil {
			return 0, false
		}
		env := make(map[string]float64)
		for _, id := range ids {
			if v, ok := dims[id]; ok {
				env[id] = v
			}
		}
		if len(env) == 0 {
			return 0, false
		}
		v, err := metricexpr.EvalScalar(line, env)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, false
		}
		return v, true
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
// finiteMinMax 忽略 NaN/Inf，若无有限值则 ok=false。
func finiteMinMax(pts []float64) (minV, maxV float64, ok bool) {
	first := true
	for _, p := range pts {
		if math.IsNaN(p) || math.IsInf(p, 0) {
			continue
		}
		if first {
			minV, maxV = p, p
			first = false
			continue
		}
		if p < minV {
			minV = p
		}
		if p > maxV {
			maxV = p
		}
	}
	return minV, maxV, !first
}

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
	sz := float32(13)
	if th < 20 {
		sz = 11
	}
	drawUIText(panelUIFont, panelUIFontOk, lbl, x+6, y+4, sz, colText)
}

func drawScene(lc *layout.LayoutConfig, sceneIndex int, snap netdata.DataSnapshot, rings *LineRings, texCache *ChartTexCache, lw, lh int32) {
	if lc == nil || sceneIndex < 0 || sceneIndex >= len(lc.Scenes) {
		return
	}
	sc := &lc.Scenes[sceneIndex]
	rl.DrawRectangle(0, 0, lw, lh, colBackground)
	for wi := range sc.Widgets {
		w := &sc.Widgets[wi]
		key := ringKey(sceneIndex, wi, w)
		drawWidget(w, snap, rings, texCache, key, lw, lh)
	}
}

func ringKey(sceneIndex, widgetIndex int, w *layout.Widget) string {
	if w.ID != "" {
		return w.ID
	}
	return fmt.Sprintf("%d:%d", sceneIndex, widgetIndex)
}

func drawWidget(w *layout.Widget, snap netdata.DataSnapshot, rings *LineRings, texCache *ChartTexCache, ringKey string, lw, lh int32) {
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
		drawWidgetLine(w, rings, texCache, ringKey, ix, iy, iw, ih)
	case layout.WidgetProgress:
		drawWidgetProgress(w, snap, ix, iy, iw, ih)
	case layout.WidgetHistogram:
		drawWidgetHistogram(w, snap, rings, texCache, ringKey, ix, iy, iw, ih)
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

// gaugeArcAngles 根据 gauge_arc_degrees 返回 (spanStart, spanEnd)。
// 180°: 从正左（180°）到正右（360°），即下半弧。
// 270°: 从左下（135°）到右下（405°），即 3/4 圆弧。
func gaugeArcAngles(arcDeg int) (start, end float32) {
	if arcDeg == 270 {
		return 135, 405
	}
	// 默认 180°
	return 180, 360
}

func drawWidgetGauge(w *layout.Widget, snap netdata.DataSnapshot, x, y, fw, fh float32) {
	v, ok := widgetPrimaryValue(snap, w)
	if !ok {
		arcDeg := w.GaugeArcDegrees
		if arcDeg != 270 {
			arcDeg = 180
		}
		spanStart, spanEnd := gaugeArcAngles(arcDeg)
		cx := x + fw/2
		var cy float32
		if arcDeg == 270 {
			cy = y + fh/2
		} else {
			cy = y + fh*0.60
		}
		r := float32(math.Min(float64(fw), float64(fh))) * 0.38
		if arcDeg == 270 {
			r = float32(math.Min(float64(fw), float64(fh))) * 0.42
		}
		inner := r * 0.68
		// thick := r - inner
		rl.DrawRingLines(rl.NewVector2(cx, cy), inner, r, spanStart, spanEnd, 48, colBarTrack)
		cardW := float32(52)
		cardH := float32(22)
		cardX := cx - cardW/2
		cardY := cy - cardH/2
		rl.DrawRectangleRounded(rl.NewRectangle(cardX, cardY, cardW, cardH), 0.4, 8, colPanel)
		drawUIText(panelUIFont, panelUIFontOk, "—", cardX+cardW/2-6, cardY+3, 14, colText)
		return
	}
	if strings.EqualFold(w.Unit, "percent") {
		// 已是 0-100，直接使用
	} else {
		v = math.Abs(v)
		if v > 1 {
			v = v / 100.0
		}
		if v > 1 {
			v = 1
		}
		v *= 100
	}

	arcDeg := w.GaugeArcDegrees
	if arcDeg != 270 {
		arcDeg = 180
	}
	spanStart, spanEnd := gaugeArcAngles(arcDeg)

	cx := x + fw/2
	// 270° 时圆心居中；180° 时圆心偏下
	var cy float32
	if arcDeg == 270 {
		cy = y + fh/2
	} else {
		cy = y + fh*0.60
	}
	r := float32(math.Min(float64(fw), float64(fh))) * 0.38
	if arcDeg == 270 {
		r = float32(math.Min(float64(fw), float64(fh))) * 0.42
	}
	inner := r * 0.68
	thick := r - inner

	// 轨道弧
	rl.DrawRingLines(rl.NewVector2(cx, cy), inner, r, spanStart, spanEnd, 48, colBarTrack)

	// 填充弧
	p := float32(math.Max(0, math.Min(100, float64(v)))) / 100
	end := spanStart + (spanEnd-spanStart)*p
	col := colAccent
	if w.CriticalThreshold > 0 && v >= w.CriticalThreshold {
		col = rl.NewColor(248, 81, 73, 255)
	} else if w.WarnThreshold > 0 && v >= w.WarnThreshold {
		col = rl.NewColor(210, 153, 34, 255)
	}
	if p > 0.001 {
		rl.DrawRing(rl.NewVector2(cx, cy), inner, r, spanStart, end, 48, col)
	}

	// 端点圆形装饰（起点始终显示，终点仅有值时显示）
	startRadRL := float64(spanStart) * math.Pi / 180
	tipR := thick * 0.5
	epx := cx + (inner+thick/2)*float32(math.Cos(startRadRL))
	epy := cy + (inner+thick/2)*float32(math.Sin(startRadRL))
	rl.DrawCircle(int32(epx), int32(epy), tipR, colBarTrack)
	if p > 0.001 {
		endRadRL := float64(end) * math.Pi / 180
		epx2 := cx + (inner+thick/2)*float32(math.Cos(endRadRL))
		epy2 := cy + (inner+thick/2)*float32(math.Sin(endRadRL))
		rl.DrawCircle(int32(epx2), int32(epy2), tipR, col)
	}

	// 中心值文字（背景圆角矩形 + 白色数字）
	valStr := fmt.Sprintf("%.0f%%", v)
	textSz := float32(14)
	if r > 50 {
		textSz = 16
	}
	// 小背景卡片
	cardW := float32(52)
	cardH := float32(22)
	cardX := cx - cardW/2
	cardY := cy - cardH/2
	rl.DrawRectangleRounded(rl.NewRectangle(cardX, cardY, cardW, cardH), 0.4, 8, colPanel)
	drawUIText(panelUIFont, panelUIFontOk, valStr, cardX+cardW/2-float32(len(valStr))*textSz*0.3, cardY+3, textSz, colText)
}

func drawWidgetLine(w *layout.Widget, rings *LineRings, texCache *ChartTexCache, key string, x, y, fw, fh float32) {
	version := rings.Version(key)
	pts := rings.Get(key)

	if len(pts) < 2 {
		rl.DrawRectangleLines(int32(x), int32(y), int32(fw), int32(fh), colMuted)
		drawUIText(panelUIFont, panelUIFontOk, "no data", x+4, y+4, 12, colMuted)
		return
	}

	// 构建多维度 map（单维度兼容）；复合表达式结果用统一维度名
	dim := "value"
	if !w.CompositeDimsExpr && len(w.Dimensions) > 0 && strings.TrimSpace(w.Dimensions[0]) != "" {
		dim = strings.TrimSpace(w.Dimensions[0])
	}
	ptsMap := map[string][]float64{dim: pts}
	dims := []string{dim}

	tex := texCache.UpdateLine(key, ptsMap, dims, int32(fw), int32(fh), version)
	if tex.ID == 0 {
		// 回退：简单折线手绘
		drawSimpleLine(pts, x, y, fw, fh)
		return
	}

	src := rl.NewRectangle(0, 0, float32(tex.Width), -float32(tex.Height))
	dst := rl.NewRectangle(x, y, fw, fh)
	rl.DrawTexturePro(tex, src, dst, rl.NewVector2(0, 0), 0, tintWhite)

	// Y 轴刻度（叠加在纹理之上）
	if w.ShowYAxis {
		minV, maxV, ok := finiteMinMax(pts)
		if !ok {
			minV, maxV = 0, 1
		}
		sMax := formatValCompact(maxV, w.Unit)
		sMid := formatValCompact((minV+maxV)/2, w.Unit)
		sMin := formatValCompact(minV, w.Unit)
		drawUIText(panelUIFont, panelUIFontOk, sMax, x+2, y+2, 10, colMuted)
		drawUIText(panelUIFont, panelUIFontOk, sMid, x+2, y+fh/2-6, 10, colMuted)
		drawUIText(panelUIFont, panelUIFontOk, sMin, x+2, y+fh-14, 10, colMuted)
	}
}

// drawSimpleLine 回退：无纹理时简单手绘折线。
func drawSimpleLine(pts []float64, x, y, fw, fh float32) {
	py0 := y + 4
	ph := fh - 8
	if ph < 4 {
		ph = 4
	}
	minV, maxV, ok := finiteMinMax(pts)
	if !ok {
		return
	}
	if maxV == minV {
		maxV = minV + 1
	}
	pyFn := func(val float64) int32 {
		t := (val - minV) / (maxV - minV)
		return int32(py0 + ph - float32(t)*ph)
	}
	den := float32(len(pts) - 1)
	if den < 1 {
		den = 1
	}
	for i := 1; i < len(pts); i++ {
		a, b := pts[i-1], pts[i]
		if math.IsNaN(a) || math.IsNaN(b) || math.IsInf(a, 0) || math.IsInf(b, 0) {
			continue
		}
		x0 := int32(x + float32(i-1)/den*fw)
		x1 := int32(x + float32(i)/den*fw)
		rl.DrawLine(x0, pyFn(a), x1, pyFn(b), colAccent)
	}
}

func drawWidgetProgress(w *layout.Widget, snap netdata.DataSnapshot, x, y, fw, fh float32) {
	v, ok := widgetPrimaryValue(snap, w)
	if !ok {
		roundness := float32(0.35)
		segments := int32(8)
		trackRect := rl.NewRectangle(x, y, fw, fh)
		rl.DrawRectangleRounded(trackRect, roundness, segments, colBarTrack)
		textSz := float32(fh * 0.55)
		if textSz < 10 {
			textSz = 10
		}
		if textSz > 18 {
			textSz = 18
		}
		drawUIText(panelUIFont, panelUIFontOk, "—", x+fw/2-4, y+(fh-textSz)/2, textSz, colText)
		return
	}
	p := v
	if strings.EqualFold(w.Unit, "percent") {
		p = v / 100
	} else if v > 1 {
		p = v / 100
	}
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

	roundness := float32(0.35)
	segments := int32(8)
	trackRect := rl.NewRectangle(x, y, fw, fh)
	rl.DrawRectangleRounded(trackRect, roundness, segments, colBarTrack)

	if w.Vertical {
		hfill := float32(p) * fh
		if hfill > 1 {
			fillRect := rl.NewRectangle(x, y+fh-hfill, fw, hfill)
			rl.DrawRectangleRounded(fillRect, roundness, segments, col)
		}
	} else {
		wfill := float32(p) * fw
		if wfill > 1 {
			fillRect := rl.NewRectangle(x, y, wfill, fh)
			rl.DrawRectangleRounded(fillRect, roundness, segments, col)
		}
	}

	// 百分比文字叠加
	valStr := fmt.Sprintf("%.0f%%", p*100)
	textSz := float32(fh * 0.55)
	if textSz < 10 {
		textSz = 10
	}
	if textSz > 18 {
		textSz = 18
	}
	drawUIText(panelUIFont, panelUIFontOk, valStr, x+fw/2-float32(len(valStr))*textSz*0.28, y+(fh-textSz)/2, textSz, colText)
}

func drawWidgetHistogram(w *layout.Widget, snap netdata.DataSnapshot, rings *LineRings, texCache *ChartTexCache, key string, x, y, fw, fh float32) {
	dims, ok := snap[widgetSnapKey(w)]
	if !ok {
		return
	}
	if w.CompositeDimsExpr {
		lines := metricexpr.NonEmptyExprLines(w.ValueExpr)
		if len(lines) == 0 {
			return
		}
		ids, err := metricexpr.CompositeEnvDimensionIDs(lines)
		if err != nil {
			return
		}
		env := make(map[string]float64)
		for _, id := range ids {
			if v, ok := dims[id]; ok {
				env[id] = v
			}
		}
		if len(env) == 0 {
			return
		}
		vals := make(map[string]float64)
		var dimNames []string
		for i, line := range lines {
			v, err := metricexpr.EvalScalar(line, env)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				return
			}
			k := strconv.Itoa(i)
			vals[k] = v
			dimNames = append(dimNames, k)
		}
		version := rings.Version(key)
		tex := texCache.UpdateHistogram(key, vals, dimNames, int32(fw), int32(fh), version)
		if tex.ID == 0 {
			drawSimpleHistogram(vals, dimNames, x, y, fw, fh)
			return
		}
		src := rl.NewRectangle(0, 0, float32(tex.Width), -float32(tex.Height))
		dst := rl.NewRectangle(x, y, fw, fh)
		rl.DrawTexturePro(tex, src, dst, rl.NewVector2(0, 0), 0, tintWhite)
		return
	}

	if len(w.Dimensions) == 0 {
		return
	}

	vals := make(map[string]float64, len(w.Dimensions))
	for _, d := range w.Dimensions {
		if v, ok := dims[d]; ok {
			vals[d] = v
		}
	}
	if len(vals) == 0 {
		return
	}

	version := rings.Version(key)
	tex := texCache.UpdateHistogram(key, vals, w.Dimensions, int32(fw), int32(fh), version)
	if tex.ID == 0 {
		// 回退：简单手绘柱状图
		drawSimpleHistogram(vals, w.Dimensions, x, y, fw, fh)
		return
	}

	src := rl.NewRectangle(0, 0, float32(tex.Width), -float32(tex.Height))
	dst := rl.NewRectangle(x, y, fw, fh)
	rl.DrawTexturePro(tex, src, dst, rl.NewVector2(0, 0), 0, tintWhite)
}

// drawSimpleHistogram 回退：无纹理时简单手绘。
func drawSimpleHistogram(vals map[string]float64, dims []string, x, y, fw, fh float32) {
	maxV := 1.0
	for _, v := range vals {
		if v > maxV {
			maxV = v
		}
	}
	bw := fw / float32(len(dims))
	for i, d := range dims {
		v := vals[d]
		h := float32(v/maxV) * (fh - 8)
		bx := x + float32(i)*bw + 2
		col := dimColors[i%len(dimColors)]
		rl.DrawRectangleRounded(rl.NewRectangle(bx, y+fh-h, bw-4, h), 0.15, 4, col)
	}
}
