package netdata

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// NumericFromDimensionMeta 尝试从 charts 接口里 dimension 的 metadata 对象中抽出数值。
// 说明：多数 Netdata 版本在 /charts 里维度值是描述性对象（name、multiplier 等），
// 真正的测量值在 /api/v1/data；本函数用于少数扁平结构或后续扩展。
func NumericFromDimensionMeta(entry DimensionEntry) (float64, bool) {
	if entry == nil {
		return 0, false
	}
	for _, key := range []string{"value", "last", "v"} {
		if v, ok := entry[key]; ok {
			if f, ok := toFloat64(v); ok {
				return f, true
			}
		}
	}
	return 0, false
}

func toFloat64(v interface{}) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case uint64:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(t, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// ParseDataLabelsValues 解析 Netdata /api/v1/data 常见二维数组格式：
//
//	{ "labels": ["time", "dim1", "dim2"], "data": [[t, v1, v2], ...] }
//
// 返回最后一行中各维度列的 float64（跳过首列时间）；若 points=1 则只有一行。
func ParseDataLabelsValues(payload map[string]interface{}) (cols map[string]float64, err error) {
	labelsRaw, ok := payload["labels"]
	if !ok {
		return nil, fmt.Errorf("missing labels")
	}
	labels, ok := toStringSlice(labelsRaw)
	if !ok || len(labels) < 2 {
		return nil, fmt.Errorf("invalid labels")
	}
	dataRaw, ok := payload["data"]
	if !ok {
		return nil, fmt.Errorf("missing data")
	}
	rows, ok := dataRaw.([]interface{})
	if !ok || len(rows) == 0 {
		return nil, fmt.Errorf("invalid data rows")
	}
	lastRow, ok := rows[len(rows)-1].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid data row type")
	}
	cols = make(map[string]float64)
	for i := 1; i < len(labels) && i < len(lastRow); i++ {
		name := labels[i]
		f, ok := toFloat64(lastRow[i])
		if !ok {
			continue
		}
		cols[name] = f
	}
	return cols, nil
}

// ParseDataTimeSeries 解析 /api/v1/data 中全部采样点（各维度一条序列，时间列在 labels[0]）。
func ParseDataTimeSeries(payload map[string]interface{}) (*ChartDataSeries, error) {
	labelsRaw, ok := payload["labels"]
	if !ok {
		return nil, fmt.Errorf("missing labels")
	}
	labels, ok := toStringSlice(labelsRaw)
	if !ok || len(labels) < 2 {
		return nil, fmt.Errorf("invalid labels")
	}
	dataRaw, ok := payload["data"]
	if !ok {
		return nil, fmt.Errorf("missing data")
	}
	rows, ok := dataRaw.([]interface{})
	if !ok || len(rows) == 0 {
		return nil, fmt.Errorf("invalid data rows")
	}
	series := make(map[string][]float64)
	for i := 1; i < len(labels); i++ {
		series[labels[i]] = make([]float64, 0, len(rows))
	}
	for _, row := range rows {
		rowArr, ok := row.([]interface{})
		if !ok {
			continue
		}
		for i := 1; i < len(labels) && i < len(rowArr); i++ {
			f, ok := toFloat64(rowArr[i])
			if !ok {
				continue
			}
			name := labels[i]
			series[name] = append(series[name], f)
		}
	}
	latest, err := ParseDataLabelsValues(payload)
	if err != nil {
		latest = map[string]float64{}
		for k, vals := range series {
			if len(vals) > 0 {
				latest[k] = vals[len(vals)-1]
			}
		}
	}
	return &ChartDataSeries{Labels: labels, Series: series, Latest: latest}, nil
}

func toStringSlice(v interface{}) ([]string, bool) {
	a, ok := v.([]interface{})
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(a))
	for _, x := range a {
		out = append(out, fmt.Sprint(x))
	}
	return out, true
}
