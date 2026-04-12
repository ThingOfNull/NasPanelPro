// Package layout 定义 WebUI 导出与渲染器共用的 config.json / layout.json 结构。
package layout

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// LayoutSettings 与 WebUI PRD `settings` 块对应（可嵌在 layout.json）。
type LayoutSettings struct {
	Width    int `json:"width"`
	Height   int `json:"height"`
	Rotation int `json:"rotation"` // 0/90/180/270，供渲染；改后需重启进程方可靠生效
}

// LayoutConfig 全局布局与场景列表。
type LayoutConfig struct {
	Version            int     `json:"version"`
	ScreenWidth        int     `json:"screen_width"`
	ScreenHeight       int     `json:"screen_height"`
	SwitchIntervalSecs float64 `json:"switch_interval_secs"`
	Scenes             []Scene `json:"scenes"`
	// Settings PRD 嵌套块；载入后会合并到 Screen* / LayoutRotation
	Settings *LayoutSettings `json:"settings,omitempty"`
	// LayoutRotation 逻辑画布旋转（度）；可由 settings.rotation 写入
	LayoutRotation int `json:"layout_rotation,omitempty"`
}

// Scene 单屏场景（轮播项）。
type Scene struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Duration float64  `json:"duration,omitempty"` // 秒；>0 时覆盖本场景停留时间
	Widgets  []Widget `json:"widgets"`
}

// WidgetDataRef PRD `data: { node, chart, dim }`。
type WidgetDataRef struct {
	Node  string `json:"node"`
	Chart string `json:"chart"`
	Dim   string `json:"dim"`
}

// WidgetType 预制组件类型。
type WidgetType string

const (
	WidgetText      WidgetType = "text"
	WidgetGauge     WidgetType = "gauge"
	WidgetLine      WidgetType = "line"
	WidgetProgress  WidgetType = "progress"
	WidgetHistogram WidgetType = "histogram"
)

// Widget 单个可视化组件。
type Widget struct {
	ID         string     `json:"id,omitempty"`
	Type       WidgetType `json:"type"`
	X          float64    `json:"x"`
	Y          float64    `json:"y"`
	W          float64    `json:"w"`
	H          float64    `json:"h"`
	ChartID    string     `json:"chart_id,omitempty"`
	Dimensions []string   `json:"dimensions,omitempty"`
	// 文本类：静态文案或 format 模板
	Label  string `json:"label,omitempty"`
	Format string `json:"format,omitempty"` // 如 "{{.Value}}%" 简化期可只用 label+单 dimension
	// 单位：auto | percent | bytes | none
	Unit string `json:"unit,omitempty"`
	// Gauge
	GaugeArcDegrees int `json:"gauge_arc_degrees,omitempty"` // 180 或 270
	// Progress
	Vertical bool `json:"vertical,omitempty"`
	// 阈值（gauge/progress）
	WarnThreshold     float64 `json:"warn_threshold,omitempty"`
	CriticalThreshold float64 `json:"critical_threshold,omitempty"`
	// Line：DRM 与 WebUI 均按 Netdata 时间窗拉取 points 个采样；0 表示默认 96
	LinePoints int `json:"line_points,omitempty"`
	// DRM / 预览：是否绘制外边框、折线纵轴刻度、是否隐藏标题条（仅图表类；text 仍用正文展示 label）
	ShowBorder bool `json:"show_border,omitempty"`
	ShowYAxis  bool `json:"show_y_axis,omitempty"`
	HideLabel  bool `json:"hide_label,omitempty"`
	// 多节点：Netdata 节点 id（configs/nodes.json）；空则使用节点列表中的默认（首条）节点
	NodeID string         `json:"node_id,omitempty"`
	Data   *WidgetDataRef `json:"data,omitempty"`
	// 样式（DRM 渲染逐步消费；未实现旋转绘制前仍持久化）
	Color       string  `json:"color,omitempty"`     // #RRGGBB
	FontSize    float64 `json:"font_size,omitempty"` // 像素
	RotationDeg float64 `json:"rotation,omitempty"`  // 组件旋转角度
}

// NormalizeBindings 将 data.* 填回 node_id / chart_id / dimensions。
func (w *Widget) NormalizeBindings() {
	if w == nil || w.Data == nil {
		return
	}
	if strings.TrimSpace(w.Data.Node) != "" {
		w.NodeID = strings.TrimSpace(w.Data.Node)
	}
	if strings.TrimSpace(w.Data.Chart) != "" {
		w.ChartID = strings.TrimSpace(w.Data.Chart)
	}
	if strings.TrimSpace(w.Data.Dim) != "" {
		w.Dimensions = []string{strings.TrimSpace(w.Data.Dim)}
	}
}

func (c *LayoutConfig) applySettingsBlock() {
	if c == nil || c.Settings == nil {
		return
	}
	if c.Settings.Width > 0 {
		c.ScreenWidth = c.Settings.Width
	}
	if c.Settings.Height > 0 {
		c.ScreenHeight = c.Settings.Height
	}
	if c.Settings.Rotation != 0 {
		c.LayoutRotation = c.Settings.Rotation
	}
}

// DefaultLayout 用于首次启动或缺文件时。
func DefaultLayout() LayoutConfig {
	return LayoutConfig{
		Version:            1,
		ScreenWidth:        1280,
		ScreenHeight:       480,
		SwitchIntervalSecs: 15,
		Scenes: []Scene{
			{
				ID:   "default",
				Name: "Default",
				Widgets: []Widget{
					{
						Type: WidgetText, X: 28, Y: 28, W: 400, H: 40,
						Label: "NAS Panel Pro", ChartID: "", Dimensions: nil,
					},
					{
						Type: WidgetText, X: 28, Y: 80, W: 220, H: 72,
						ChartID: "system.cpu", Dimensions: []string{"user"}, Unit: "percent",
					},
					{
						Type: WidgetGauge, X: 700, Y: 80, W: 200, H: 200,
						ChartID: "system.cpu", Dimensions: []string{"user"}, Unit: "percent",
						GaugeArcDegrees: 180,
					},
				},
			},
			{
				ID:   "scene2",
				Name: "Scene 2",
				Widgets: []Widget{
					{Type: WidgetText, X: 28, Y: 200, W: 600, H: 48, Label: "Scene 2 · 轮播演示"},
					{Type: WidgetLine, X: 40, Y: 260, W: 600, H: 160, ChartID: "system.cpu", Dimensions: []string{"user"}, LinePoints: 60},
				},
			},
		},
	}
}

// Validate 校验配置可加载；warnings 不阻断。
func (c *LayoutConfig) Validate() error {
	if c == nil {
		return errors.New("nil layout")
	}
	if c.Version < 1 {
		c.Version = 1
	}
	if c.ScreenWidth <= 0 {
		c.ScreenWidth = 1280
	}
	if c.ScreenHeight <= 0 {
		c.ScreenHeight = 480
	}
	if c.SwitchIntervalSecs <= 0 {
		c.SwitchIntervalSecs = 15
	}
	if len(c.Scenes) == 0 {
		return errors.New("no scenes")
	}
	c.applySettingsBlock()
	for si := range c.Scenes {
		sc := &c.Scenes[si]
		if strings.TrimSpace(sc.ID) == "" {
			return fmt.Errorf("scene[%d]: empty id", si)
		}
		if sc.Duration < 0 {
			return fmt.Errorf("scene %q: duration cannot be negative", sc.ID)
		}
		for wi := range sc.Widgets {
			sc.Widgets[wi].NormalizeBindings()
			if err := validateWidget(&sc.Widgets[wi]); err != nil {
				return fmt.Errorf("scene %q widget[%d]: %w", sc.ID, wi, err)
			}
		}
	}
	return nil
}

func validateWidget(w *Widget) error {
	switch w.Type {
	case WidgetText, WidgetGauge, WidgetLine, WidgetProgress, WidgetHistogram:
	case "":
		return errors.New("empty widget type")
	default:
		return fmt.Errorf("unknown widget type %q", w.Type)
	}
	if w.W <= 0 || w.H <= 0 {
		return errors.New("widget w/h must be positive")
	}
	switch w.Type {
	case WidgetGauge, WidgetLine, WidgetProgress, WidgetHistogram:
		if strings.TrimSpace(w.ChartID) == "" {
			return fmt.Errorf("type %s requires chart_id", w.Type)
		}
		if len(w.Dimensions) == 0 {
			return fmt.Errorf("type %s requires dimensions", w.Type)
		}
	}
	if w.Type == WidgetText && w.ChartID != "" && len(w.Dimensions) == 0 {
		return errors.New("text with chart_id needs dimensions")
	}
	if w.GaugeArcDegrees != 0 && w.GaugeArcDegrees != 180 && w.GaugeArcDegrees != 270 {
		return errors.New("gauge_arc_degrees must be 180 or 270")
	}
	return nil
}

// LoadFile 读取并校验 JSON。
func LoadFile(path string) (LayoutConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return LayoutConfig{}, err
	}
	var c LayoutConfig
	if err := json.Unmarshal(b, &c); err != nil {
		return LayoutConfig{}, err
	}
	if err := c.Validate(); err != nil {
		return LayoutConfig{}, err
	}
	return c, nil
}

// SaveFile 原子写入（同目录临时文件 + rename）。
func SaveFile(path string, c LayoutConfig) error {
	if err := c.Validate(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ChartsUsed 返回所有场景中出现的 chart id（去重）。
func (c *LayoutConfig) ChartsUsed() []string {
	seen := make(map[string]struct{})
	var out []string
	for _, sc := range c.Scenes {
		for _, w := range sc.Widgets {
			id := strings.TrimSpace(w.ChartID)
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

// ChartsUsedInScene 返回指定场景索引内引用的 chart id（不含节点前缀，供兼容）。
func (c *LayoutConfig) ChartsUsedInScene(sceneIndex int) []string {
	if c == nil || sceneIndex < 0 || sceneIndex >= len(c.Scenes) {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, w := range c.Scenes[sceneIndex].Widgets {
		id := strings.TrimSpace(w.ChartID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// SnapKeysUsedInScenes 返回轮询用的快照键（node\x1fchart 或 chart）。
func (c *LayoutConfig) SnapKeysUsedInScenes(indices ...int) []string {
	if c == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, si := range indices {
		if si < 0 || si >= len(c.Scenes) {
			continue
		}
		for _, w := range c.Scenes[si].Widgets {
			id := strings.TrimSpace(w.ChartID)
			if id == "" {
				continue
			}
			key := ChartSnapKey(w.NodeID, id)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, key)
		}
	}
	return out
}
