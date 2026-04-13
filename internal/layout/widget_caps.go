package layout

// WidgetAllowsMultiDims 是否允许 dimensions 中多个维度（英文逗号分隔）。
func WidgetAllowsMultiDims(t WidgetType) bool {
	return t == WidgetHistogram
}

// WidgetAllowsMultiExpr 是否允许 value_expr 多行（复合模式下多条表达式）。
func WidgetAllowsMultiExpr(t WidgetType) bool {
	return t == WidgetHistogram
}
