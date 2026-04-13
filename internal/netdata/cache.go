package netdata

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ChartMeta 供搜索与 API 列表返回的轻量字段。
type ChartMeta struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ChartCache 进程内 charts 元数据缓存（发现-订阅模型）。
type ChartCache struct {
	Client *Client
	TTL    time.Duration

	mu        sync.RWMutex
	charts    map[string]ChartDef
	fetchedAt time.Time
	fetchErr  error
}

func (cc *ChartCache) client() *Client {
	if cc.Client == nil {
		return &Client{}
	}
	return cc.Client
}

func (cc *ChartCache) ttl() time.Duration {
	if cc.TTL <= 0 {
		return 5 * time.Minute
	}
	return cc.TTL
}

// SetClient 更新下游 Netdata 客户端；URL 或密钥变化时清空 charts 缓存。
func (cc *ChartCache) SetClient(c *Client) {
	if cc == nil {
		return
	}
	cc.mu.Lock()
	defer cc.mu.Unlock()
	old := cc.Client
	cc.Client = c
	if c == nil {
		cc.charts = nil
		cc.fetchErr = nil
		cc.fetchedAt = time.Time{}
		return
	}
	changed := old == nil
	if !changed {
		changed = strings.TrimSpace(old.BaseURL) != strings.TrimSpace(c.BaseURL) ||
			strings.TrimSpace(old.APIKey) != strings.TrimSpace(c.APIKey)
	}
	if changed {
		cc.charts = nil
		cc.fetchErr = nil
		cc.fetchedAt = time.Time{}
	}
}

// Refresh 强制拉取 /api/v1/charts。
func (cc *ChartCache) Refresh(ctx context.Context) error {
	c := cc.client()
	charts, err := c.FetchCharts(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.fetchErr = err
	if err != nil {
		return err
	}
	cc.charts = charts
	cc.fetchedAt = time.Now()
	return nil
}

// Charts 返回缓存；若过期或空则 Refresh（ctx 可为 context.Background）。
func (cc *ChartCache) Charts(ctx context.Context) (map[string]ChartDef, error) {
	// 第一次加读锁：判断是否 stale，不 stale 时在持锁状态下直接返回，避免 TOCTOU。
	cc.mu.RLock()
	stale := cc.charts == nil || time.Since(cc.fetchedAt) > cc.ttl()
	if !stale && cc.fetchErr == nil {
		charts := cc.charts
		cc.mu.RUnlock()
		return charts, nil
	}
	err := cc.fetchErr
	cc.mu.RUnlock()

	// 需要刷新。
	if refreshErr := cc.Refresh(ctx); refreshErr != nil {
		cc.mu.RLock()
		defer cc.mu.RUnlock()
		if cc.charts != nil {
			return cc.charts, nil
		}
		if err != nil {
			return nil, err
		}
		return nil, refreshErr
	}
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return cc.charts, nil
}

// SearchCharts 按 id / title 子串过滤（大小写不敏感），最多 limit 条（<=0 则 50）。
func (cc *ChartCache) SearchCharts(ctx context.Context, query string, limit int) ([]ChartMeta, error) {
	all, err := cc.Charts(ctx)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var out []ChartMeta
	for id, ch := range all {
		title := ch.Title
		if title == "" {
			title = ch.Name
		}
		if q != "" {
			if !strings.Contains(strings.ToLower(id), q) && !strings.Contains(strings.ToLower(title), q) {
				continue
			}
		}
		out = append(out, ChartMeta{ID: id, Title: title})
		if len(out) >= limit {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// ChartDiscoveryRow 供 WebUI Worker 全量索引（category ≈ family）。
type ChartDiscoveryRow struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Family  string `json:"family"`
	Context string `json:"context"`
}

// ListChartDiscovery 返回全部 chart 元数据行（可能上万条）。
func (cc *ChartCache) ListChartDiscovery(ctx context.Context) ([]ChartDiscoveryRow, error) {
	all, err := cc.Charts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ChartDiscoveryRow, 0, len(all))
	for id, ch := range all {
		title := ch.Title
		if title == "" {
			title = ch.Name
		}
		out = append(out, ChartDiscoveryRow{
			ID:      id,
			Title:   title,
			Family:  ch.Family,
			Context: ch.Context,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// ChartByID 从缓存取单图定义。
func (cc *ChartCache) ChartByID(ctx context.Context, id string) (*ChartDef, error) {
	all, err := cc.Charts(ctx)
	if err != nil {
		return nil, err
	}
	ch, ok := all[id]
	if !ok {
		return nil, fmt.Errorf("chart not found: %s", id)
	}
	return &ch, nil
}
