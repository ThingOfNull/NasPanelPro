// naspanel：Raylib DRM + Netdata 布局渲染 + Gin 配置 API + Supervisor。
package main

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"naspanel/internal/cfg"
	"naspanel/internal/layout"
	"naspanel/internal/logbuf"
	"naspanel/internal/netdata"
	"naspanel/internal/nodes"
	"naspanel/internal/platform"
	"naspanel/internal/raypanel"
	"naspanel/internal/server"
	"naspanel/internal/supervisor"
)

// blockedExitWaiter：SIGINT/SIGTERM 后若主线程长时间卡在 raylib C（如 DRM Present），
// context 虽已取消，但 Run 无法回到 Go 检查 stopFrame。此时在短延迟后 os.Exit，避免 SSH 下「^C 有回显却杀不掉」。
// NASPANEL_BLOCKED_EXIT_MS=0 可关闭该行为（仅调试用，可能永远无法退出）。
func blockedExitWaiter(ctx context.Context) (done func()) {
	ms := 600
	if v := os.Getenv("NASPANEL_BLOCKED_EXIT_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			ms = n
		}
	}
	if ms <= 0 {
		return func() {}
	}
	ch := make(chan struct{})
	var once sync.Once
	go func() {
		<-ctx.Done()
		t := time.NewTimer(time.Duration(ms) * time.Millisecond)
		defer t.Stop()
		select {
		case <-t.C:
			log.Printf("naspanel: DRM still blocked after %dms, forcing exit", ms)
			os.Exit(0)
		case <-ch:
		}
	}()
	return func() { once.Do(func() { close(ch) }) }
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	stopBlockedExit := blockedExitWaiter(ctx)
	defer stopBlockedExit()

	c := cfg.DefaultConfig()
	cfg.ApplyEnv(&c)

	layoutPath := c.LayoutPath
	if layoutPath == "" {
		layoutPath = filepath.Join("configs", "layout.json")
	}
	nodesPath := filepath.Join("configs", "nodes.json")
	if v := os.Getenv("NASPANEL_NODES_PATH"); v != "" {
		nodesPath = v
	}
	if err := os.MkdirAll(filepath.Dir(layoutPath), 0o755); err != nil {
		log.Printf("naspanel: mkdir layout dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(nodesPath), 0o755); err != nil {
		log.Printf("naspanel: mkdir nodes dir: %v", err)
	}

	logRing := logbuf.New(2000)
	log.SetOutput(io.MultiWriter(os.Stderr, logRing))

	st := &layout.Store{}
	loaded, err := layout.LoadFile(layoutPath)
	if err != nil {
		if os.IsNotExist(err) {
			def := layout.DefaultLayout()
			st.Put(def)
			if err := layout.SaveFile(layoutPath, def); err != nil {
				log.Printf("naspanel: write default layout.json: %v", err)
			}
		} else {
			log.Printf("naspanel: load layout failed, using default: %v", err)
			st.Put(layout.DefaultLayout())
		}
	} else {
		st.Put(loaded)
	}

	ns := &nodes.Store{}
	if nf, err := nodes.LoadFile(nodesPath); err != nil {
		if os.IsNotExist(err) {
			defN := nodes.DefaultFile()
			ns.Put(defN)
			if err := nodes.SaveFile(nodesPath, defN); err != nil {
				log.Printf("naspanel: write default nodes.json: %v", err)
			}
		} else {
			log.Printf("naspanel: load nodes.json failed, using empty list: %v", err)
			ns.Put(nodes.DefaultFile())
		}
	} else {
		ns.Put(nf)
	}

	cache := &netdata.ChartCache{TTL: 5 * time.Minute}
	nf := ns.Get()
	if n, ok := nf.First(); ok {
		cache.SetClient(&netdata.Client{BaseURL: n.BaseURL(), APIKey: n.APIKey})
	}
	if c.HTTPListenAddr != "-" {
		httpAddr := c.HTTPListenAddr
		if httpAddr == "" {
			httpAddr = ":8090"
		}
		if err := server.Start(ctx, server.Options{
			Addr:        httpAddr,
			LayoutPath:  layoutPath,
			NodesPath:   nodesPath,
			LayoutStore: st,
			NodesStore:  ns,
			ChartCache:  cache,
			LogBuf:      logRing,
		}); err != nil {
			log.Printf("naspanel: HTTP server: %v", err)
		}
	}

	disp := raypanel.NewDisplay()
	sup := supervisor.New(c, supervisor.Callbacks{
		OnYield: func() error {
			disp.SuspendRender()
			return platform.Chvt(c.TTYConsole)
		},
		OnRecover: func() error {
			if err := platform.Chvt(c.TTYGUI); err != nil {
				return err
			}
			disp.ResumeRender()
			return nil
		},
	})
	go func() {
		if err := sup.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("supervisor: %v", err)
		}
	}()

	opt := raypanel.Options{
		Config:      c,
		LayoutStore: st,
		LayoutPath:  layoutPath,
		NodesPath:   nodesPath,
		NodeStore:   ns,
		Display:     disp,
	}
	if err := raypanel.Run(ctx, opt); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("naspanel: exit: %v", err)
		os.Exit(1)
	}
}
