package raypanel

import (
	"bytes"
	"sync"

	chart "github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
	rl "github.com/gen2brain/raylib-go/raylib"
)

// dimColorHex 多维度折线颜色序列（与 draw.go dimColors 对应）。
var dimColorHex = []drawing.Color{
	{R: 0x58, G: 0xd6, B: 0x8d, A: 0xff},
	{R: 0x58, G: 0xa6, B: 0xff, A: 0xff},
	{R: 0xff, G: 0x9f, B: 0x43, A: 0xff},
	{R: 0xc7, G: 0x92, B: 0xea, A: 0xff},
	{R: 0x2e, G: 0xcc, B: 0xcc, A: 0xff},
}

// chartBG 折线/柱状图背景色。
var chartBG = drawing.Color{R: 0x0d, G: 0x11, B: 0x17, A: 0xff}

// ChartTexEntry 单个纹理缓存项。
type ChartTexEntry struct {
	Tex     rl.Texture2D
	Version int64
	W, H    int32
}

// ChartTexCache 按 ringKey 缓存 go-chart 渲染出的 GPU 纹理。
// 所有纹理操作（Load/Unload）必须在 Raylib 主线程调用。
type ChartTexCache struct {
	mu      sync.Mutex
	entries map[string]*ChartTexEntry
}

// NewChartTexCache 构造空缓存。
func NewChartTexCache() *ChartTexCache {
	return &ChartTexCache{entries: make(map[string]*ChartTexEntry)}
}

// UnloadAll 释放全部 GPU 纹理。必须在 Raylib 主线程调用。
func (c *ChartTexCache) UnloadAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, e := range c.entries {
		if e.Tex.ID > 0 {
			rl.UnloadTexture(e.Tex)
		}
	}
	c.entries = make(map[string]*ChartTexEntry)
}

// getEntry 返回 key 对应的缓存项（无则新建）。调用方须持 mu 锁。
func (c *ChartTexCache) getEntry(key string) *ChartTexEntry {
	e := c.entries[key]
	if e == nil {
		e = &ChartTexEntry{}
		c.entries[key] = e
	}
	return e
}

// UpdateLine 若版本号变化则重新渲染折线图纹理，返回当前纹理。
// pts: dim → 时间序列；dims: 维度顺序；w/h: widget 像素尺寸；version: LineRings 版本号。
func (c *ChartTexCache) UpdateLine(key string, pts map[string][]float64, dims []string, w, h int32, version int64) rl.Texture2D {
	c.mu.Lock()
	e := c.getEntry(key)
	if e.Version == version && e.W == w && e.H == h && e.Tex.ID > 0 {
		tex := e.Tex
		c.mu.Unlock()
		return tex
	}
	c.mu.Unlock()

	tex := renderLineTexture(pts, dims, w, h)

	c.mu.Lock()
	e = c.getEntry(key)
	if e.Tex.ID > 0 {
		rl.UnloadTexture(e.Tex)
	}
	e.Tex = tex
	e.Version = version
	e.W = w
	e.H = h
	c.mu.Unlock()
	return tex
}

// UpdateHistogram 若版本号变化则重新渲染柱状图纹理。
func (c *ChartTexCache) UpdateHistogram(key string, vals map[string]float64, dims []string, w, h int32, version int64) rl.Texture2D {
	c.mu.Lock()
	e := c.getEntry(key)
	if e.Version == version && e.W == w && e.H == h && e.Tex.ID > 0 {
		tex := e.Tex
		c.mu.Unlock()
		return tex
	}
	c.mu.Unlock()

	tex := renderHistogramTexture(vals, dims, w, h)

	c.mu.Lock()
	e = c.getEntry(key)
	if e.Tex.ID > 0 {
		rl.UnloadTexture(e.Tex)
	}
	e.Tex = tex
	e.Version = version
	e.W = w
	e.H = h
	c.mu.Unlock()
	return tex
}

// dimColor 返回第 i 个维度的颜色（循环取）。
func dimColor(i int) drawing.Color {
	return dimColorHex[i%len(dimColorHex)]
}

// pngBytesToTexture 将 PNG bytes 转为 rl.Texture2D（必须在主线程调用）。
func pngBytesToTexture(pngBytes []byte) rl.Texture2D {
	if len(pngBytes) == 0 {
		return rl.Texture2D{}
	}
	img := rl.LoadImageFromMemory(".png", pngBytes, int32(len(pngBytes)))
	if img == nil || img.Width == 0 {
		return rl.Texture2D{}
	}
	tex := rl.LoadTextureFromImage(img)
	rl.UnloadImage(img)
	rl.SetTextureFilter(tex, rl.FilterBilinear)
	return tex
}

// renderLineTexture 用 go-chart 渲染折线图为 rl.Texture2D。
func renderLineTexture(pts map[string][]float64, dims []string, w, h int32) rl.Texture2D {
	if w <= 0 {
		w = 200
	}
	if h <= 0 {
		h = 100
	}

	var series []chart.Series
	for i, d := range dims {
		p, ok := pts[d]
		if !ok || len(p) < 2 {
			continue
		}
		col := dimColor(i)
		xv := make([]float64, len(p))
		for j := range p {
			xv[j] = float64(j)
		}
		fillCol := drawing.Color{R: col.R, G: col.G, B: col.B, A: 0x28}
		series = append(series, chart.ContinuousSeries{
			XValues: xv,
			YValues: p,
			Style: chart.Style{
				StrokeColor: col,
				StrokeWidth: 1.5,
				FillColor:   fillCol,
			},
		})
	}
	if len(series) == 0 {
		return rl.Texture2D{}
	}

	g := chart.Chart{
		Width:  int(w),
		Height: int(h),
		Background: chart.Style{
			FillColor: chartBG,
			Padding:   chart.Box{Top: 4, Left: 4, Right: 4, Bottom: 4},
		},
		Canvas: chart.Style{
			FillColor: chartBG,
		},
		XAxis:  chart.XAxis{Style: chart.Hidden()},
		YAxis:  chart.YAxis{Style: chart.Hidden()},
		Series: series,
	}

	var buf bytes.Buffer
	if err := g.Render(chart.PNG, &buf); err != nil {
		return rl.Texture2D{}
	}
	return pngBytesToTexture(buf.Bytes())
}

// renderHistogramTexture 用 go-chart 渲染柱状图为 rl.Texture2D。
func renderHistogramTexture(vals map[string]float64, dims []string, w, h int32) rl.Texture2D {
	if w <= 0 {
		w = 200
	}
	if h <= 0 {
		h = 100
	}
	if len(dims) == 0 {
		return rl.Texture2D{}
	}

	var bars []chart.Value
	for i, d := range dims {
		v := vals[d]
		if v < 0 {
			v = -v
		}
		col := dimColor(i)
		bars = append(bars, chart.Value{
			Value: v,
			Style: chart.Style{
				FillColor:   col,
				StrokeColor: col,
				StrokeWidth: 0,
			},
		})
	}

	g := chart.BarChart{
		Width:  int(w),
		Height: int(h),
		Background: chart.Style{
			FillColor: chartBG,
			Padding:   chart.Box{Top: 4, Left: 4, Right: 4, Bottom: 4},
		},
		Canvas: chart.Style{
			FillColor: chartBG,
		},
		XAxis: chart.Style{Hidden: true},
		YAxis: chart.YAxis{Style: chart.Hidden()},
		Bars:  bars,
	}

	var buf bytes.Buffer
	if err := g.Render(chart.PNG, &buf); err != nil {
		return rl.Texture2D{}
	}
	return pngBytesToTexture(buf.Bytes())
}
