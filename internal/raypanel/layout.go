package raypanel

import (
	"os"

	"naspanel/internal/cfg"
	"naspanel/internal/core"
)

// WindowPixelSize：90°/270° 时交换宽高，与窗口物理像素一致。
func WindowPixelSize(c cfg.Config) (w, h int) {
	if c.ScreenWidth <= 0 {
		c.ScreenWidth = 1280
	}
	if c.ScreenHeight <= 0 {
		c.ScreenHeight = 480
	}
	r := core.NormalizeRotationDegrees(c.RotateDeg)
	if r == 90 || r == 270 {
		return c.ScreenHeight, c.ScreenWidth
	}
	return c.ScreenWidth, c.ScreenHeight
}

// DRMInitWindowSize 返回传给 rl.InitWindow 的宽高。Raylib DRM 会按该尺寸匹配连接器 mode；
// 若只存在竖屏 mode（如 480×1280），却请求 1280×480 会报 “Failed to find a suitable DRM connector mode”。
// 设 NASPANEL_DRM_USE_LOGICAL_SIZE=1 可强制使用 WindowPixelSize 的原始值（真横屏显示器时用）。
func DRMInitWindowSize(c cfg.Config, winW, winH int32, rotDeg int) (initW, initH int32) {
	initW, initH = winW, winH
	if os.Getenv("NASPANEL_DRM_USE_LOGICAL_SIZE") == "1" {
		return initW, initH
	}
	r := core.NormalizeRotationDegrees(rotDeg)
	if (r == 0 || r == 180) && winW > winH {
		lw := int32(c.ScreenWidth)
		lh := int32(c.ScreenHeight)
		if lw <= 0 {
			lw = 1280
		}
		if lh <= 0 {
			lh = 480
		}
		initW, initH = lh, lw
	}
	return initW, initH
}
