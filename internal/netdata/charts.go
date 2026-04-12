// Package netdata 实现 Netdata HTTP API 的发现与解析（PRD 2.0 第一阶段）。
package netdata

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// ChartDef 对应 /api/v1/charts 中单个图定义；dimensions 结构因插件而异，用 map 承接。
type ChartDef struct {
	ID          string                    `json:"id"`
	Name        string                    `json:"name"`
	Title       string                    `json:"title"`
	Family      string                    `json:"family"`
	Context     string                    `json:"context"`
	Units       string                    `json:"units"`
	Type        string                    `json:"type"`
	Dimensions  map[string]DimensionEntry `json:"dimensions"`
}

// DimensionEntry 为 dimensions 下单个键的值；可能是 {"name":"user"} 或带 multiplier 等字段。
type DimensionEntry map[string]interface{}

// UnmarshalJSON 允许 dimensions 中值为任意 JSON 对象并展平为 map。
func (d *DimensionEntry) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		*d = nil
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	*d = m
	return nil
}

// Client 最小 HTTP 客户端。
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	APIKey     string // 可选；设置后请求带 Authorization: Bearer
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (c *Client) applyAuth(req *http.Request) {
	if c == nil || strings.TrimSpace(c.APIKey) == "" {
		return
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(c.APIKey))
}

// FetchCharts 请求 GET /api/v1/charts 并解析。
func (c *Client) FetchCharts() (map[string]ChartDef, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	req, err := http.NewRequest(http.MethodGet, base+"/api/v1/charts", nil)
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
		return nil, fmt.Errorf("netdata charts: HTTP %s: %s", resp.Status, strings.TrimSpace(string(slurp)))
	}
	return DecodeChartsResponse(resp.Body)
}

// DecodeChartsResponse 从 Reader 解析 charts（自动识别是否包裹在 charts 键下）。
func DecodeChartsResponse(r io.Reader) (map[string]ChartDef, error) {
	dec := json.NewDecoder(r)
	var root map[string]json.RawMessage
	if err := dec.Decode(&root); err != nil {
		return nil, err
	}
	if raw, ok := root["charts"]; ok {
		var out map[string]ChartDef
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, fmt.Errorf("decode .charts: %w", err)
		}
		return out, nil
	}
	// 整包即 id -> chart
	out := make(map[string]ChartDef)
	skip := map[string]struct{}{
		"hostname": {}, "version": {}, "release_channel": {},
		"os": {}, "timezone": {}, "abbrev_timezone": {},
	}
	for k, v := range root {
		if _, s := skip[k]; s {
			continue
		}
		var ch ChartDef
		if err := json.Unmarshal(v, &ch); err != nil {
			continue
		}
		if ch.ID == "" {
			ch.ID = k
		}
		out[k] = ch
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no charts found in response")
	}
	return out, nil
}

// DimensionIDs 返回某 chart 下所有 dimension 键名（用于聚合 data 请求与 WebUI 勾选）。
func DimensionIDs(c *ChartDef) []string {
	if c == nil || len(c.Dimensions) == 0 {
		return nil
	}
	keys := make([]string, 0, len(c.Dimensions))
	for k := range c.Dimensions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// DimensionStringField 从 metadata 中读取常见字符串字段（如 name）。
func DimensionStringField(entry DimensionEntry, field string) string {
	if entry == nil {
		return ""
	}
	v, ok := entry[field]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}
