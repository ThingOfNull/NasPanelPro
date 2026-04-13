package metricexpr

import (
	"sort"

	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/builtin"
)

type identCollector struct {
	seen map[string]struct{}
	ids  []string
}

func (c *identCollector) Visit(node *ast.Node) {
	n, ok := (*node).(*ast.IdentifierNode)
	if !ok {
		return
	}
	if _, isBuiltin := builtin.Index[n.Value]; isBuiltin {
		return
	}
	if _, dup := c.seen[n.Value]; dup {
		return
	}
	c.seen[n.Value] = struct{}{}
	c.ids = append(c.ids, n.Value)
}

// IdentifierNames 返回表达式中的变量名（排除内置函数名），用于公式模式下推断所需 Netdata dimension。
func IdentifierNames(exprStr string) ([]string, error) {
	p, err := compileProgram(exprStr)
	if err != nil {
		return nil, err
	}
	n := p.Node()
	var coll identCollector
	coll.seen = make(map[string]struct{})
	ast.Walk(&n, &coll)
	sort.Strings(coll.ids)
	return coll.ids, nil
}
