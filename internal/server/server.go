// Package server 提供 Gin HTTP：节点、布局、Netdata 代理、静态 SPA。
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"naspanel/internal/layout"
	"naspanel/internal/logbuf"
	"naspanel/internal/netdata"
	"naspanel/internal/nodes"

	"github.com/gin-gonic/gin"
)

// Options 服务启动参数。
type Options struct {
	Addr        string
	LayoutPath  string
	NodesPath   string
	LayoutStore *layout.Store
	NodesStore  *nodes.Store
	ChartCache  *netdata.ChartCache
	LogBuf      *logbuf.Buffer
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func chartCacheForNode(nodeID string, opt Options) *netdata.ChartCache {
	if opt.NodesStore == nil {
		return opt.ChartCache
	}
	nf := opt.NodesStore.Get()
	nid := strings.TrimSpace(nodeID)
	if nid != "" {
		if n, ok := nf.ByID(nid); ok {
			return &netdata.ChartCache{
				Client: &netdata.Client{BaseURL: n.BaseURL(), APIKey: n.APIKey},
				TTL:    0,
			}
		}
	}
	if n, ok := nf.First(); ok && opt.ChartCache != nil {
		opt.ChartCache.SetClient(&netdata.Client{BaseURL: n.BaseURL(), APIKey: n.APIKey})
		return opt.ChartCache
	}
	return opt.ChartCache
}

func netdataClientForNodeID(nodeID string, opt Options) (*netdata.Client, bool) {
	if opt.NodesStore == nil {
		return nil, false
	}
	nf := opt.NodesStore.Get()
	nid := strings.TrimSpace(nodeID)
	if nid != "" {
		if n, ok := nf.ByID(nid); ok {
			return &netdata.Client{BaseURL: n.BaseURL(), APIKey: n.APIKey}, true
		}
	}
	if n, ok := nf.First(); ok {
		return &netdata.Client{BaseURL: n.BaseURL(), APIKey: n.APIKey}, true
	}
	return nil, false
}

// Start 在后台监听；ctx 取消时关闭 http.Server。
func Start(ctx context.Context, opt Options) error {
	if opt.LayoutStore == nil {
		return fmt.Errorf("LayoutStore required")
	}
	if opt.NodesStore == nil {
		return fmt.Errorf("NodesStore required")
	}
	if opt.ChartCache == nil {
		opt.ChartCache = &netdata.ChartCache{TTL: 0}
	}
	if opt.ChartCache.Client == nil && opt.NodesStore != nil {
		nf := opt.NodesStore.Get()
		if n, ok := nf.First(); ok {
			opt.ChartCache.SetClient(&netdata.Client{BaseURL: n.BaseURL(), APIKey: n.APIKey})
		}
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(cors(), gin.Recovery())

	indexHTML, staticFS := loadWebUI()
	serveSPA := func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	}
	r.GET("/", serveSPA)
	r.GET("/index.html", serveSPA)

	if staticFS != nil {
		if sub, err := fs.Sub(staticFS, "assets"); err == nil {
			r.StaticFS("/assets", http.FS(sub))
		}
	}

	api := r.Group("/api")
	{
		api.GET("/layout", func(c *gin.Context) {
			c.JSON(http.StatusOK, opt.LayoutStore.Get())
		})
		api.PUT("/layout", func(c *gin.Context) {
			var body layout.LayoutConfig
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if err := body.Validate(); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if opt.LayoutPath != "" {
				if err := layout.SaveFile(opt.LayoutPath, body); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
			opt.LayoutStore.Put(body)
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		api.GET("/nodes", func(c *gin.Context) {
			c.JSON(http.StatusOK, opt.NodesStore.Get())
		})
		api.PUT("/nodes", func(c *gin.Context) {
			var body nodes.File
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if err := body.Validate(); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if opt.NodesPath != "" {
				if err := nodes.SaveFile(opt.NodesPath, body); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
			opt.NodesStore.Put(body)
			if opt.ChartCache != nil {
				if n, ok := body.First(); ok {
					opt.ChartCache.SetClient(&netdata.Client{BaseURL: n.BaseURL(), APIKey: n.APIKey})
				} else {
					opt.ChartCache.SetClient(nil)
				}
			}
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
		api.POST("/nodes/:id/test", func(c *gin.Context) {
			id := strings.TrimSpace(c.Param("id"))
			var probeBody struct {
				Host   string `json:"host"`
				Port   int    `json:"port"`
				APIKey string `json:"api_key"`
				Secure bool   `json:"secure"`
			}
			raw, _ := io.ReadAll(c.Request.Body)
			if len(bytes.TrimSpace(raw)) > 0 {
				if err := json.Unmarshal(raw, &probeBody); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json: " + err.Error()})
					return
				}
			}

			nf := opt.NodesStore.Get()
			n, ok := nf.ByID(id)
			if strings.TrimSpace(probeBody.Host) != "" {
				// 使用请求体中的当前表单（无需先保存即可探测）
				n = nodes.Node{
					ID:     id,
					Host:   strings.TrimSpace(probeBody.Host),
					Port:   probeBody.Port,
					APIKey: probeBody.APIKey,
					Secure: probeBody.Secure,
				}
			} else if !ok {
				c.JSON(http.StatusNotFound, gin.H{"error": "unknown node: save first or send host in JSON body"})
				return
			}
			base := n.BaseURL()
			if base == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "empty host"})
				return
			}
			cl := &netdata.Client{BaseURL: base, APIKey: n.APIKey}
			res := cl.Probe(c.Request.Context())
			c.JSON(http.StatusOK, res)
		})

		api.GET("/logs", func(c *gin.Context) {
			if opt.LogBuf == nil {
				c.JSON(http.StatusOK, gin.H{"lines": []string{}})
				return
			}
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "500"))
			c.JSON(http.StatusOK, gin.H{"lines": opt.LogBuf.Snapshot(limit)})
		})

		api.GET("/netdata/charts", func(c *gin.Context) {
			q := c.Query("q")
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "80"))
			if limit <= 0 {
				limit = 80
			}
			cc := chartCacheForNode(c.Query("node_id"), opt)
			list, err := cc.SearchCharts(c.Request.Context(), q, limit)
			if err != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"charts": list})
		})

		api.GET("/netdata/discovery", func(c *gin.Context) {
			cc := chartCacheForNode(c.Query("node_id"), opt)
			rows, err := cc.ListChartDiscovery(c.Request.Context())
			if err != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"charts": rows})
		})

		api.GET("/netdata/chart/:id", func(c *gin.Context) {
			id := c.Param("id")
			id = strings.TrimPrefix(id, "/")
			cc := chartCacheForNode(c.Query("node_id"), opt)
			ch, err := cc.ChartByID(c.Request.Context(), id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"id":               ch.ID,
				"title":            ch.Title,
				"name":             ch.Name,
				"units":            ch.Units,
				"context":          ch.Context,
				"dimensions":       netdata.DimensionIDs(ch),
				"dimension_meta":   ch.Dimensions,
			})
		})

		api.GET("/netdata/data", func(c *gin.Context) {
			chart := strings.TrimSpace(c.Query("chart"))
			if chart == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "missing chart"})
				return
			}
			points, _ := strconv.Atoi(c.DefaultQuery("points", "72"))
			if points < 1 {
				points = 1
			}
			if points > 1200 {
				points = 1200
			}
			after := strings.TrimSpace(c.DefaultQuery("after", "-120"))
			if after == "" {
				after = "-120"
			}
			cl, ok := netdataClientForNodeID(c.Query("node_id"), opt)
			if !ok || cl == nil || strings.TrimSpace(cl.BaseURL) == "" {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no netdata node"})
				return
			}
			series, err := cl.FetchChartSeries(c.Request.Context(), chart, netdata.DataOpts{
				After:  after,
				Points: points,
			})
			if err != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, series)
		})
	}

	r.Any("/proxy/:node_id/*filepath", func(c *gin.Context) {
		id := strings.TrimSpace(c.Param("node_id"))
		nf := opt.NodesStore.Get()
		n, ok := nf.ByID(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown node"})
			return
		}
		target, err := url.Parse(n.BaseURL())
		if err != nil || target.Host == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node url"})
			return
		}
		suffix := strings.TrimPrefix(c.Param("filepath"), "/")
		proxy := httputil.NewSingleHostReverseProxy(target)
		orig := proxy.Director
		proxy.Director = func(req *http.Request) {
			orig(req)
			req.URL.Path = "/" + suffix
			req.URL.RawQuery = c.Request.URL.RawQuery
			if strings.TrimSpace(n.APIKey) != "" {
				req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(n.APIKey))
			}
		}
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = io.WriteString(w, e.Error())
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	})

	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api") || strings.HasPrefix(p, "/proxy") {
			c.Status(http.StatusNotFound)
			return
		}
		if strings.Contains(p, ".") && staticFS != nil {
			rel := strings.TrimPrefix(strings.TrimPrefix(p, "/"), "webui-dist/")
			if b, err := fs.ReadFile(staticFS, rel); err == nil {
				ct := http.DetectContentType(b)
				if strings.HasSuffix(p, ".js") {
					ct = "application/javascript"
				} else if strings.HasSuffix(p, ".css") {
					ct = "text/css"
				}
				c.Data(http.StatusOK, ct, b)
				return
			}
		}
		serveSPA(c)
	})

	srv := &http.Server{Addr: opt.Addr, Handler: r}
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			_, _ = gin.DefaultWriter.Write([]byte("server: " + err.Error() + "\n"))
		}
	}()
	return nil
}
