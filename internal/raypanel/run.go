package raypanel

import (
	"context"
	"fmt"
	"log"
	"os"
	"slices"
	"sync/atomic"
	"time"

	"naspanel/internal/core"
	"naspanel/internal/layout"
	"naspanel/internal/nodes"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Run 进入 Raylib DRM 主循环（布局 + Netdata + 多场景）。LayoutStore 必填。
func Run(ctx context.Context, opt Options) error {
	if opt.LayoutStore == nil {
		return fmt.Errorf("raypanel: LayoutStore is required")
	}
	d := opt.Display
	if d == nil {
		d = NewDisplay()
	}

	var stopFrame atomic.Bool
	go func() {
		<-ctx.Done()
		stopFrame.Store(true)
	}()

	lc0 := opt.LayoutStore.Ptr()
	logicW := int32(opt.Config.ScreenWidth)
	logicH := int32(opt.Config.ScreenHeight)
	if logicW <= 0 {
		logicW = 1280
	}
	if logicH <= 0 {
		logicH = 480
	}
	if lc0 != nil {
		if lc0.ScreenWidth > 0 {
			logicW = int32(lc0.ScreenWidth)
		}
		if lc0.ScreenHeight > 0 {
			logicH = int32(lc0.ScreenHeight)
		}
	}

	c := opt.Config
	winW, winH := WindowPixelSize(c)
	rot := core.NormalizeRotationDegrees(c.RotateDeg)
	if lc0 != nil && lc0.LayoutRotation != 0 {
		rot = core.NormalizeRotationDegrees(lc0.LayoutRotation)
	}

	initW, initH := DRMInitWindowSize(c, int32(winW), int32(winH), c.RotateDeg)
	if initW != int32(winW) || initH != int32(winH) {
		log.Printf("raypanel: DRM InitWindow %dx%d to match connector (logical canvas %dx%d)", initW, initH, logicW, logicH)
	}

	detachStdinFromTTY()

	// 开启 MSAA 4× 抗锯齿（须在 InitWindow 之前设置）
	rl.SetConfigFlags(rl.FlagMsaa4xHint)
	if c.FullScreen {
		rl.SetConfigFlags(rl.FlagMsaa4xHint | rl.FlagFullscreenMode | rl.FlagWindowUndecorated)
	}
	rl.InitWindow(initW, initH, "NAS Panel (DRM)")
	defer func() {
		unloadPanelUIFont()
		rl.CloseWindow()
	}()
	if !rl.IsWindowReady() {
		return fmt.Errorf("raypanel: DRM InitWindow not ready")
	}

	InitPanelUIFont(lc0)

	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())
	if rot == 0 && os.Getenv("NASPANEL_DRM_NO_AUTOROTATE") != "1" {
		if int(sw) == c.ScreenHeight && int(sh) == c.ScreenWidth && c.ScreenWidth > c.ScreenHeight {
			log.Printf("raypanel: DRM size %dx%d vs logical %dx%d, using 90 offscreen rotation", sw, sh, c.ScreenWidth, c.ScreenHeight)
			rot = 90
		}
	}

	if c.MaxTPS > 0 {
		rl.SetTargetFPS(int32(c.MaxTPS))
	} else {
		rl.SetTargetFPS(0)
	}

	var rt rl.RenderTexture2D
	if rot != 0 {
		rt = rl.LoadRenderTexture(logicW, logicH)
		defer rl.UnloadRenderTexture(rt)
		rl.SetTextureFilter(rt.Texture, rl.FilterPoint)
	}

	fade0 := rl.LoadRenderTexture(logicW, logicH)
	fade1 := rl.LoadRenderTexture(logicW, logicH)
	rl.SetTextureFilter(fade0.Texture, rl.FilterPoint)
	rl.SetTextureFilter(fade1.Texture, rl.FilterPoint)
	defer rl.UnloadRenderTexture(fade0)
	defer rl.UnloadRenderTexture(fade1)

	rings := NewLineRings(60)
	sceneMgr := NewSceneManager(lc0)
	texCache := NewChartTexCache()

	poller := NewPoller("")
	poller.SetPlan(func() []PollTarget {
		return BuildPollTargets(opt.LayoutStore.Ptr(), opt.NodeStore)
	})
	poller.SetLineFill(rings, func() (*layout.LayoutConfig, []int) {
		return opt.LayoutStore.Ptr(), sceneMgr.PollerChartIndices()
	}, opt.NodeStore)
	pollerCtx, pollerStop := context.WithCancel(ctx)
	defer pollerStop()
	// 直接启动，不需要 sync.Once（此处只会执行一次）
	go poller.Run(pollerCtx, time.Second)

	var layoutModTime time.Time
	var nodesModTime time.Time
	reloadEvery := 0

	// 缓存上一帧 poller 参数，避免每帧不必要的 alloc 和锁争用。
	var lastBase string
	var lastCharts []string

	for !rl.WindowShouldClose() && !stopFrame.Load() {
		if rl.IsKeyPressed(rl.KeyEscape) {
			break
		}

		reloadEvery++
		if reloadEvery%120 == 0 {
			if opt.LayoutPath != "" {
				if st, err := os.Stat(opt.LayoutPath); err == nil && st.ModTime() != layoutModTime {
					if loaded, err := layout.LoadFile(opt.LayoutPath); err == nil {
						opt.LayoutStore.Put(loaded)
						layoutModTime = st.ModTime()
						InitPanelUIFont(&loaded)
						texCache.UnloadAll() // layout 重载时释放所有缓存纹理
					}
				}
			}
			if opt.NodesPath != "" && opt.NodeStore != nil {
				if st, err := os.Stat(opt.NodesPath); err == nil && st.ModTime() != nodesModTime {
					if nf, err := nodes.LoadFile(opt.NodesPath); err == nil {
						opt.NodeStore.Put(nf)
						nodesModTime = st.ModTime()
					}
				}
			}
		}

		lc := opt.LayoutStore.Ptr()
		sceneMgr.SetLayout(lc)

		// 仅在内容变化时才调用 SetBaseURL / SetCharts，避免每帧持锁写 Poller。
		firstBase := ""
		if opt.NodeStore != nil {
			nf := opt.NodeStore.Get()
			if fn, ok := nf.First(); ok {
				firstBase = fn.BaseURL()
			}
		}
		if firstBase != lastBase {
			poller.SetBaseURL(firstBase)
			lastBase = firstBase
		}
		charts := ChartsForLayoutScenes(lc, sceneMgr.PollerChartIndices()...)
		if !slices.Equal(charts, lastCharts) {
			poller.SetCharts(charts)
			lastCharts = charts
		}

		sceneMgr.Tick()
		snap := poller.Snapshot()

		if d.RenderSuspended() {
			rl.BeginDrawing()
			rl.ClearBackground(colBackground)
			rl.EndDrawing()
			continue
		}

		from, to, fadeA := sceneMgr.FadeBlend()
		useFade := len(lc.Scenes) > 1 && to >= 0 && fadeA > 0.001

		drawToTexture := func(tex rl.RenderTexture2D, si int) {
			rl.BeginTextureMode(tex)
			rl.ClearBackground(colBackground)
			drawScene(lc, si, snap, rings, texCache, logicW, logicH)
			rl.EndTextureMode()
		}

		if useFade {
			a := EaseInOut(fadeA)
			drawToTexture(fade0, from)
			drawToTexture(fade1, to)
			if rot != 0 {
				rl.BeginTextureMode(rt)
				rl.ClearBackground(colBackground)
				blendTwoRenderTextures(fade0.Texture, fade1.Texture, logicW, logicH, a)
				rl.EndTextureMode()
				rl.BeginDrawing()
				rl.ClearBackground(colBackground)
				if os.Getenv("NASPANEL_DRM_BLIT_PRO") == "1" {
					blitRotatedPro(rt, rot, sw, sh, logicW, logicH)
				} else {
					blitRotatedCamera(rt, rot, sw, sh, logicW, logicH)
				}
				rl.EndDrawing()
			} else {
				rl.BeginDrawing()
				rl.ClearBackground(colBackground)
				blendTwoRenderTextures(fade0.Texture, fade1.Texture, logicW, logicH, a)
				rl.EndDrawing()
			}
			continue
		}

		si := sceneMgr.ActiveSceneIndex()
		if rot != 0 {
			rl.BeginTextureMode(rt)
			rl.ClearBackground(colBackground)
			drawScene(lc, si, snap, rings, texCache, logicW, logicH)
			rl.EndTextureMode()
			rl.BeginDrawing()
			rl.ClearBackground(colBackground)
			if os.Getenv("NASPANEL_DRM_BLIT_PRO") == "1" {
				blitRotatedPro(rt, rot, sw, sh, logicW, logicH)
			} else {
				blitRotatedCamera(rt, rot, sw, sh, logicW, logicH)
			}
			rl.EndDrawing()
		} else {
			rl.BeginDrawing()
			rl.ClearBackground(colBackground)
			drawScene(lc, si, snap, rings, texCache, logicW, logicH)
			rl.EndDrawing()
		}
	}

	if stopFrame.Load() {
		return ctx.Err()
	}
	return nil
}
