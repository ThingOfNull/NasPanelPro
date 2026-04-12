package raypanel

import (
	"math"

	"naspanel/internal/layout"

	rl "github.com/gen2brain/raylib-go/raylib"
)

const fadeDurationSec = 0.45

// SceneManager 多场景轮播与渐变状态。
type SceneManager struct {
	layout *layout.LayoutConfig

	active       int
	lastSwitch   float64
	fading       bool
	fadeProgress float32
	fadeFrom     int
	fadeTo       int
}

func NewSceneManager(lc *layout.LayoutConfig) *SceneManager {
	return &SceneManager{layout: lc, lastSwitch: rl.GetTime()}
}

// SetLayout 替换布局指针（每帧来自 Store）。
func (m *SceneManager) SetLayout(lc *layout.LayoutConfig) {
	m.layout = lc
	if m.layout != nil && m.active >= len(m.layout.Scenes) {
		m.active = 0
	}
}

// Tick 推进时间与场景切换 / 渐变。
func (m *SceneManager) Tick() {
	if m.layout == nil || len(m.layout.Scenes) == 0 {
		return
	}
	n := len(m.layout.Scenes)
	now := float64(rl.GetTime())
	interval := m.layout.SwitchIntervalSecs
	if interval <= 0 {
		interval = 15
	}
	if m.active >= 0 && m.active < len(m.layout.Scenes) {
		if d := m.layout.Scenes[m.active].Duration; d > 0 {
			interval = d
		}
	}

	if m.fading {
		m.fadeProgress += rl.GetFrameTime() / fadeDurationSec
		if m.fadeProgress >= 1 {
			m.fadeProgress = 1
			m.active = m.fadeTo
			m.fading = false
			m.lastSwitch = now
		}
		return
	}

	if n <= 1 {
		return
	}
	if now-m.lastSwitch >= interval {
		m.startFade((m.active + 1) % n)
	}
}

func (m *SceneManager) startFade(to int) {
	if m.layout == nil {
		return
	}
	m.fading = true
	m.fadeFrom = m.active
	m.fadeTo = to
	m.fadeProgress = 0
}

// ActiveSceneIndex 当前逻辑活动场景（渐变完成后才切）。
func (m *SceneManager) ActiveSceneIndex() int {
	if m.layout == nil || len(m.layout.Scenes) == 0 {
		return 0
	}
	if m.fading {
		return m.fadeFrom
	}
	return m.active
}

// FadeIndices 返回 (from, to, alpha of to) 供混合；非渐变时 to=-1, alpha=0。
func (m *SceneManager) FadeBlend() (from, to int, alpha float32) {
	if m.layout == nil || !m.fading {
		return m.active, -1, 0
	}
	return m.fadeFrom, m.fadeTo, m.fadeProgress
}

// PollerChartIndices 返回当前轮询应包含的场景下标（渐变时合并两场景）。
func (m *SceneManager) PollerChartIndices() []int {
	if m.layout == nil {
		return nil
	}
	if !m.fading {
		return []int{m.active}
	}
	return []int{m.fadeFrom, m.fadeTo}
}

// EaseInOut 平滑曲线。
func EaseInOut(t float32) float32 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return float32(0.5 - 0.5*math.Cos(float64(t)*math.Pi))
}
