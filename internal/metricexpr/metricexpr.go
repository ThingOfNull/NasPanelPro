// Package metricexpr 使用 expr-lang 对 Netdata 维度值做标量或逐点计算。
package metricexpr

import (
	"fmt"
	"math"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

var (
	compileMu sync.Mutex
	programs  = make(map[string]*vm.Program)
)

func compileProgram(exprStr string) (*vm.Program, error) {
	compileMu.Lock()
	defer compileMu.Unlock()
	if p, ok := programs[exprStr]; ok {
		return p, nil
	}
	p, err := expr.Compile(exprStr,
		expr.AllowUndefinedVariables(),
		expr.AsFloat64(),
	)
	if err != nil {
		return nil, err
	}
	programs[exprStr] = p
	return p, nil
}

// Validate 校验表达式可编译（用于布局保存）。
func Validate(exprStr string) error {
	if exprStr == "" {
		return nil
	}
	_, err := compileProgram(exprStr)
	return err
}

// ClearProgramCache 仅用于测试。
func ClearProgramCache() {
	compileMu.Lock()
	defer compileMu.Unlock()
	programs = make(map[string]*vm.Program)
}

func anyToFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint64:
		return float64(x), true
	default:
		return 0, false
	}
}

func runToFloat64(p *vm.Program, env map[string]any) (float64, error) {
	out, err := expr.Run(p, env)
	if err != nil {
		return 0, err
	}
	f, ok := anyToFloat64(out)
	if !ok {
		return 0, fmt.Errorf("expression result is not numeric")
	}
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return math.NaN(), nil
	}
	return f, nil
}

// EvalScalar 使用 latest 维度值计算单个浮点数。
func EvalScalar(exprStr string, env map[string]float64) (float64, error) {
	if exprStr == "" {
		return 0, fmt.Errorf("empty expression")
	}
	p, err := compileProgram(exprStr)
	if err != nil {
		return 0, err
	}
	m := make(map[string]any, len(env))
	for k, v := range env {
		m[k] = v
	}
	return runToFloat64(p, m)
}

// EvalSeries 对每个下标用各维度在该点的值构造 env 并求值；序列长度取 series 中最短非空长度。
func EvalSeries(exprStr string, series map[string][]float64) ([]float64, error) {
	if exprStr == "" {
		return nil, fmt.Errorf("empty expression")
	}
	p, err := compileProgram(exprStr)
	if err != nil {
		return nil, err
	}
	n := minSeriesLen(series)
	if n == 0 {
		return nil, fmt.Errorf("no series points")
	}
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		env := make(map[string]any, len(series))
		for k, pts := range series {
			if i < len(pts) {
				env[k] = pts[i]
			}
		}
		v, err := runToFloat64(p, env)
		if err != nil {
			out[i] = math.NaN()
			continue
		}
		out[i] = v
	}
	return out, nil
}

func minSeriesLen(series map[string][]float64) int {
	if len(series) == 0 {
		return 0
	}
	min := -1
	for _, pts := range series {
		if len(pts) == 0 {
			return 0
		}
		if min < 0 || len(pts) < min {
			min = len(pts)
		}
	}
	return min
}
