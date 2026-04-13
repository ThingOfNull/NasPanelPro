/** 与 internal/layout/widget_caps.go 对齐：多维度 / 多行表达式能力 */

export function widgetAllowsMultiDims(widgetType: string): boolean {
  return widgetType === 'histogram';
}

export function widgetAllowsMultiExpr(widgetType: string): boolean {
  return widgetType === 'histogram';
}
