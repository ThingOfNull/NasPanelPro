import type { LayoutConfig, Widget } from './types';
import { widgetAllowsMultiDims, widgetAllowsMultiExpr } from './widgetCaps';
import { nonEmptyExprLines } from './metricexpr';

function widgetHasDimensionList(w: Widget): boolean {
  return (w.dimensions ?? []).some((d) => d.trim() !== '');
}

function widgetDimensionCount(w: Widget): number {
  return (w.dimensions ?? []).filter((d) => d.trim() !== '').length;
}

/** 保存前校验（与 Go validateWidget 对齐）；通过返回 null，否则返回错误文案（可直接 setStatus） */
export function validateLayoutConfig(c: LayoutConfig): string | null {
  for (let si = 0; si < c.scenes.length; si++) {
    const sc = c.scenes[si];
    for (let wi = 0; wi < sc.widgets.length; wi++) {
      const err = validateWidget(sc.widgets[wi]);
      if (err) {
        return `${sc.name || sc.id} · widget[${wi}]: ${err}`;
      }
    }
  }
  return null;
}

function validateWidget(w: Widget): string | null {
  const t = w.type;
  if (!t) {
    return 'empty widget type';
  }
  const okType =
    t === 'text' ||
    t === 'gauge' ||
    t === 'line' ||
    t === 'progress' ||
    t === 'histogram';
  if (!okType) {
    return `unknown widget type ${t}`;
  }
  if (w.w <= 0 || w.h <= 0) {
    return 'widget w/h must be positive';
  }

  const composite = !!w.composite_dims_expr;
  const hasDims = widgetHasDimensionList(w);
  const veTrim = (w.value_expr ?? '').trim();
  const lines = nonEmptyExprLines(w.value_expr ?? '');

  if (composite && hasDims) {
    return 'composite_dims_expr requires empty dimensions';
  }
  if (!composite && veTrim !== '') {
    return 'value_expr is only allowed when composite_dims_expr is true';
  }
  if (!composite && widgetDimensionCount(w) > 1 && !widgetAllowsMultiDims(t)) {
    return `type ${t} allows only one dimension`;
  }
  if (composite) {
    if (lines.length === 0) {
      return 'composite_dims_expr requires at least one non-empty expression line';
    }
    if (!widgetAllowsMultiExpr(t) && lines.length > 1) {
      return `type ${t} allows only one expression line`;
    }
    // 语法由后端强校验；前端可再试 parse
  }

  const chartTypes = t === 'gauge' || t === 'line' || t === 'progress' || t === 'histogram';
  if (chartTypes) {
    if (!(w.chart_id ?? '').trim()) {
      return `type ${t} requires chart_id`;
    }
    if (composite) {
      if (lines.length === 0) {
        return `type ${t} requires value_expr when composite_dims_expr is set`;
      }
    } else {
      if (!hasDims) {
        return `type ${t} requires dimensions when not using composite_dims_expr`;
      }
    }
  }
  if (t === 'text' && (w.chart_id ?? '').trim()) {
    if (composite) {
      if (lines.length === 0) {
        return 'text with chart_id requires value_expr when composite_dims_expr is set';
      }
    } else {
      if (!hasDims) {
        return 'text with chart_id requires dimensions when not using composite_dims_expr';
      }
    }
  }
  const g = w.gauge_arc_degrees ?? 0;
  if (g !== 0 && g !== 180 && g !== 270) {
    return 'gauge_arc_degrees must be 180 or 270';
  }
  return null;
}
