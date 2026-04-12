package layout

import "strings"

// ChartSnapSep 分隔 node_id 与 chart_id，避免与 chart id 内字符冲突。
const ChartSnapSep = "\x1f"

// ChartSnapKey 生成 DataSnapshot 中的 chart 键；nodeID 为空时仅为 chartID。
func ChartSnapKey(nodeID, chartID string) string {
	chartID = strings.TrimSpace(chartID)
	if chartID == "" {
		return ""
	}
	n := strings.TrimSpace(nodeID)
	if n == "" {
		return chartID
	}
	return n + ChartSnapSep + chartID
}
