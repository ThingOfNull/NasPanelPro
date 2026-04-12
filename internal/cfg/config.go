// Package cfg 为各二进制共享的运行参数，仅依赖标准库，避免通过 config 把 X11/Raylib 绑在一起。
package cfg

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 可由环境变量覆盖（ApplyEnv）。
type Config struct {
	ScreenWidth  int
	ScreenHeight int
	TTYGUI       int
	TTYConsole   int
	VCSPath      string
	KMsgPath     string
	VCSPoll      int
	VCSConfirmN  int
	RecoverSec   int
	ChvtShield   int

	FullScreen     bool
	SampleInterval time.Duration
	MaxTPS         int
	RotateDeg      int

	HTTPListenAddr string // 如 ":8090"；空为默认 :8090；"-" 关闭 HTTP
	LayoutPath     string // 默认 configs/layout.json
}

// DefaultConfig 与 PRD MVP 默认值一致。
func DefaultConfig() Config {
	return Config{
		ScreenWidth:    1280,
		ScreenHeight:   480,
		TTYGUI:         7,
		TTYConsole:     1,
		VCSPath:        "/dev/vcs1",
		KMsgPath:       "/dev/kmsg",
		VCSPoll:        500,
		VCSConfirmN:    2,
		RecoverSec:     30,
		ChvtShield:     2000,
		FullScreen:     true,
		SampleInterval: time.Second,
		MaxTPS:         30,
	}
}

// ApplyEnv 用环境变量覆盖字段。
func ApplyEnv(c *Config) {
	if os.Getenv("NASPANEL_WINDOWED") == "1" {
		c.FullScreen = false
	}
	if v := os.Getenv("NASPANEL_MAXTPS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			c.MaxTPS = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("NASPANEL_ROTATE")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.RotateDeg = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("NASPANEL_HTTP_ADDR")); v != "" {
		c.HTTPListenAddr = v
	}
	if v := strings.TrimSpace(os.Getenv("NASPANEL_LAYOUT_PATH")); v != "" {
		c.LayoutPath = v
	}
}
