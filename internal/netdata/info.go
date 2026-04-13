package netdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ProbeResult 连通性探测结果（供 WebUI /api/nodes/:id/test）。
type ProbeResult struct {
	Version    string `json:"version"`
	ChartCount int    `json:"chart_count"`
	OK         bool   `json:"ok"`
	Error      string `json:"error,omitempty"`
}

// Probe 请求 /api/v1/info 与 /api/v1/charts，验证节点可达。
func (c *Client) Probe(ctx context.Context) ProbeResult {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		return ProbeResult{Error: "empty base url"}
	}
	cl := c.httpClient()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/info", nil)
	if err != nil {
		return ProbeResult{Error: err.Error()}
	}
	req.Header.Set("Accept", "application/json")
	c.applyAuth(req)
	resp, err := cl.Do(req)
	if err != nil {
		return ProbeResult{Error: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slurp, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return ProbeResult{Error: fmt.Sprintf("info HTTP %s: %s", resp.Status, strings.TrimSpace(string(slurp)))}
	}
	var info map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return ProbeResult{Error: "info json: " + err.Error()}
	}
	ver := ""
	if v, ok := info["version"].(string); ok {
		ver = v
	}
	if ver == "" {
		if m, ok := info["netdata"].(map[string]interface{}); ok {
			if v, ok := m["version"].(string); ok {
				ver = v
			}
		}
	}

	charts, err := c.FetchCharts(ctx)
	if err != nil {
		return ProbeResult{Version: ver, Error: "charts: " + err.Error()}
	}
	return ProbeResult{Version: ver, ChartCount: len(charts), OK: true}
}
