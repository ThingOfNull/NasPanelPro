package metricexpr

import (
	"sort"
	"strings"
)

// NonEmptyExprLines 按换行分割，trim 后丢弃空行。
func NonEmptyExprLines(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	var out []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// CompositeEnvDimensionIDs 返回所有表达式行中变量名的并集（去重、排序）。
func CompositeEnvDimensionIDs(lines []string) ([]string, error) {
	seen := make(map[string]struct{})
	for _, ln := range lines {
		ids, err := IdentifierNames(ln)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			seen[id] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}
