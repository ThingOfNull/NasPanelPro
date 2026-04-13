package netdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// ChartDataSeries 供 WebUI 预览：多采样点时间序列 + 最后一帧。
type ChartDataSeries struct {
	Labels []string             `json:"labels"`
	Series map[string][]float64 `json:"series"`
	Latest map[string]float64   `json:"latest"`
}

// DataOpts 控制 /api/v1/data 查询参数。
type DataOpts struct {
	After  string // 默认 "-1"（最近数据）
	Points int    // 默认 1
	Group  string // 默认 "average"
}

func (o DataOpts) normalized() DataOpts {
	if o.After == "" {
		o.After = "-1"
	}
	if o.Points <= 0 {
		o.Points = 1
	}
	if o.Group == "" {
		o.Group = "average"
	}
	return o
}

// buildDataURL 构造 /api/v1/data 请求 URL，供 FetchChartData 和 FetchChartSeries 共用。
func buildDataURL(baseURL, chart string, opts DataOpts) (*url.URL, error) {
	base := strings.TrimRight(baseURL, "/")
	u, err := url.Parse(base + "/api/v1/data")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("chart", chart)
	q.Set("after", opts.After)
	q.Set("points", fmt.Sprintf("%d", opts.Points))
	q.Set("group", opts.Group)
	u.RawQuery = q.Encode()
	return u, nil
}

// FetchChartData 拉取单个 chart 的最新点（labels + data 格式）。
func (c *Client) FetchChartData(ctx context.Context, chart string, opts DataOpts) (map[string]float64, error) {
	opts = opts.normalized()
	u, err := buildDataURL(c.BaseURL, chart, opts)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	c.applyAuth(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slurp, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("netdata data %s: HTTP %s: %s", chart, resp.Status, strings.TrimSpace(string(slurp)))
	}
	var root map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return nil, fmt.Errorf("chart %s: %w", chart, err)
	}
	return ParseDataLabelsValues(root)
}

// FetchChartSeries 拉取 chart 的多个采样点（after/points 与 Netdata 一致）。
func (c *Client) FetchChartSeries(ctx context.Context, chart string, opts DataOpts) (*ChartDataSeries, error) {
	opts = opts.normalized()
	u, err := buildDataURL(c.BaseURL, chart, opts)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	c.applyAuth(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slurp, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("netdata data %s: HTTP %s: %s", chart, resp.Status, strings.TrimSpace(string(slurp)))
	}
	var root map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return nil, fmt.Errorf("chart %s: %w", chart, err)
	}
	return ParseDataTimeSeries(root)
}

// DataSnapshot chartID -> dimension -> 最后一帧数值。
type DataSnapshot map[string]map[string]float64

// FetchChartsData 对多个 chart 并发各请求一次 /api/v1/data（去重）。
// Netdata 各版本对多 chart 单请求参数不一致，此处用并行单图保证兼容。
func (c *Client) FetchChartsData(ctx context.Context, charts []string, opts DataOpts) (DataSnapshot, error) {
	seen := make(map[string]struct{})
	var uniq []string
	for _, id := range charts {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return DataSnapshot{}, nil
	}

	var mu sync.Mutex
	out := make(DataSnapshot)
	var firstErr error

	var wg sync.WaitGroup
	for _, ch := range uniq {
		wg.Add(1)
		go func(chartID string) {
			defer wg.Done()
			cols, err := c.FetchChartData(ctx, chartID, opts)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			out[chartID] = cols
		}(ch)
	}
	wg.Wait()
	if len(out) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

// Lookup 从快照中取单值；无则 ok=false。
func (s DataSnapshot) Lookup(chartID, dimension string) (v float64, ok bool) {
	if s == nil {
		return 0, false
	}
	dims, ok := s[chartID]
	if !ok {
		return 0, false
	}
	v, ok = dims[dimension]
	return v, ok
}
