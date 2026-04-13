import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api'
import {
  buildLatestEnvForComposite,
  buildSeriesForComposite,
  evalScalar,
  evalSeries,
  nonEmptyExprLines,
} from '../metricexpr'
import type { Widget } from '../types'

export type ChartSeriesPayload = {
  labels: string[]
  series: Record<string, number[]>
  latest: Record<string, number>
}

// 多维度颜色序列（与 DRM draw.go dimColors 保持一致）
const DIM_COLORS = ['#58d68d', '#58a6ff', '#ff9f43', '#c792ea', '#2ecccc']
const ACCENT = '#58d68d'
// 未导出模块内常量使用 camelCase
const trackColor = '#3d444d'
const panelColor = '#1e242b'
const textColor = '#e6edf3'
const mutedColor = '#adbac7'

function pickDim(w: Widget, latest: Record<string, number>): string | undefined {
  const d0 = w.dimensions?.find(Boolean)
  if (d0 && latest[d0] !== undefined) {
    return d0
  }
  return Object.keys(latest)[0]
}

function clamp01(x: number, max: number) {
  if (!Number.isFinite(x) || max <= 0) return 0
  return Math.max(0, Math.min(1, x / max))
}

/** Match drawWidgetGauge percent scaling. */
function gaugePercentValue(w: Widget, raw: number): number {
  let v = raw
  if (!Number.isFinite(v)) return 0
  const u = w.unit ?? ''
  if (u !== '' && u.toLowerCase() === 'percent') {
    return Math.max(0, Math.min(100, v))
  }
  v = Math.abs(v)
  if (v > 1.5) v /= 100
  if (v > 1) v = 1
  return v * 100
}

function linePathD(pts: number[], boxW: number, boxH: number): string {
  if (pts.length < 2) return ''
  let minV = pts[0], maxV = pts[0]
  for (const p of pts) {
    if (p < minV) minV = p
    if (p > maxV) maxV = p
  }
  if (maxV === minV) maxV = minV + 1
  const py = (val: number) => boxH - 4 - ((val - minV) / (maxV - minV)) * (boxH - 8)
  let den = pts.length - 1
  if (den < 1) den = 1
  const parts: string[] = []
  for (let i = 1; i < pts.length; i++) {
    const x0 = ((i - 1) / den) * boxW
    const x1 = (i / den) * boxW
    const y0 = py(pts[i - 1])
    const y1 = py(pts[i])
    if (i === 1) parts.push(`M${x0.toFixed(1)},${y0.toFixed(1)}`)
    parts.push(`L${x1.toFixed(1)},${y1.toFixed(1)}`)
  }
  return parts.join(' ')
}

/** 填充路径：折线 + 底部封闭 */
function lineFillD(pts: number[], boxW: number, boxH: number): string {
  if (pts.length < 2) return ''
  const stroke = linePathD(pts, boxW, boxH)
  if (!stroke) return ''
  const den = pts.length - 1 || 1
  const lastX = ((pts.length - 1) / den) * boxW
  const firstX = 0
  return `${stroke} L${lastX.toFixed(1)},${(boxH - 4).toFixed(1)} L${firstX.toFixed(1)},${(boxH - 4).toFixed(1)} Z`
}

function formatAxisCompact(v: number, unit?: string): string {
  const u = (unit ?? '').toLowerCase()
  if (!Number.isFinite(v)) return '—'
  if (u === 'percent') return `${Math.round(v)}%`
  if (u === 'bytes') {
    const bytes = v * 1024
    const g = bytes / 1024 ** 3
    if (g >= 1) return `${g.toFixed(1)}G`
    const m = bytes / 1024 ** 2
    if (m >= 1) return `${Math.round(m)}M`
    return `${Math.round(bytes / 1024)}K`
  }
  if (Math.abs(v) >= 1000) return `${Math.round(v)}`
  return v.toFixed(1)
}

/** 根据 arc 度数（180 或 270）计算 SVG arc path */
function gaugeArcPath(cx: number, cy: number, r: number, arcDeg: number, pct: number): { track: string; fill: string } {
  // 180°: 从左(π) 到右(0)，逆时针（数学坐标）
  // 270°: 从左下(225°) 到右下(315°逆时针 = 225°+270°)，顺时针
  // SVG: 0°=右，顺时针
  if (arcDeg === 270) {
    // 起点角度: 135°（左下），终点: 135° + 270° = 405° = 45°
    const startDeg = 135
    const totalDeg = 270
    const toRad = (d: number) => (d * Math.PI) / 180
    const startRad = toRad(startDeg)
    const endRad = toRad(startDeg + totalDeg)
    const sx = cx + r * Math.cos(startRad)
    const sy = cy + r * Math.sin(startRad)
    const ex = cx + r * Math.cos(endRad)
    const ey = cy + r * Math.sin(endRad)
    const track = `M${sx.toFixed(2)},${sy.toFixed(2)} A${r},${r} 0 1,1 ${ex.toFixed(2)},${ey.toFixed(2)}`

    const fillDeg = startDeg + totalDeg * Math.max(0, Math.min(1, pct))
    const fRad = toRad(fillDeg)
    const fx = cx + r * Math.cos(fRad)
    const fy = cy + r * Math.sin(fRad)
    const large = totalDeg * pct > 180 ? 1 : 0
    const fill = `M${sx.toFixed(2)},${sy.toFixed(2)} A${r},${r} 0 ${large},1 ${fx.toFixed(2)},${fy.toFixed(2)}`
    return { track, fill }
  }
  // 默认 180°
  const sx = cx - r
  const ex = cx + r
  const track = `M${sx} ${cy} A${r} ${r} 0 0 1 ${ex} ${cy}`
  const tArc = Math.max(0, Math.min(1, pct))
  const a0 = Math.PI * (1 - tArc)
  const x2 = cx + r * Math.cos(a0)
  const y2 = cy - r * Math.sin(a0)
  const large = tArc > 0.5 ? 1 : 0
  const fill = `M${sx} ${cy} A${r} ${r} 0 ${large} 1 ${x2.toFixed(2)} ${y2.toFixed(2)}`
  return { track, fill }
}

export default function WidgetChartPreview({
  w,
  fallbackNodeId,
}: {
  w: Widget
  fallbackNodeId: string
}) {
  const { t } = useTranslation()
  const [payload, setPayload] = useState<ChartSeriesPayload | null>(null)
  const [failed, setFailed] = useState(false)

  const nodeId = w.node_id ?? fallbackNodeId ?? ''

  useEffect(() => {
    if (w.type === 'text' || !w.chart_id?.trim()) {
      setPayload(null)
      setFailed(false)
      return
    }
    let cancelled = false
    const points = w.type === 'line' ? 96 : 1
    const after = w.type === 'line' ? '-180' : '-1'
    const tick = () => {
      api
        .getChartSeries(w.chart_id!, nodeId, points, after)
        .then((r) => {
          if (!cancelled) { setPayload(r); setFailed(false) }
        })
        .catch(() => {
          if (!cancelled) { setPayload(null); setFailed(true) }
        })
    }
    tick()
    const id = window.setInterval(tick, 8000)
    return () => { cancelled = true; window.clearInterval(id) }
  }, [
    w.type,
    w.chart_id,
    w.node_id,
    fallbackNodeId,
    nodeId,
    w.value_expr,
    w.composite_dims_expr,
    w.dimensions?.join('\x1e'),
  ])

  if (w.type === 'text') {
    return (
      <div className="flex items-center justify-center h-full text-zinc-400 text-xs px-1 text-center leading-tight">
        {w.label || t('chartPreview.textDefault')}
      </div>
    )
  }

  if (!w.chart_id?.trim()) {
    return (
      <div className="flex items-center justify-center h-full text-zinc-600 text-[10px]">
        {t('chartPreview.noChartBound')}
      </div>
    )
  }

  if (failed || !payload) {
    return (
      <div className="flex items-center justify-center h-full text-zinc-600 text-[10px]">
        {failed ? t('chartPreview.noData') : t('chartPreview.loading')}
      </div>
    )
  }

  const composite = !!w.composite_dims_expr
  const exprLines = composite ? nonEmptyExprLines(w.value_expr ?? '') : []
  const firstLine = exprLines[0] ?? ''
  let val: number = NaN
  let pts: number[] = []
  if (composite && firstLine) {
    try {
      const env = buildLatestEnvForComposite(payload.latest, exprLines)
      val = evalScalar(firstLine, env)
    } catch {
      val = NaN
    }
    if (w.type === 'line') {
      try {
        const sub = buildSeriesForComposite(payload.series, exprLines)
        pts = evalSeries(firstLine, sub)
      } catch {
        pts = []
      }
    }
  } else if (!composite) {
    const dim = pickDim(w, payload.latest)
    val = dim !== undefined ? payload.latest[dim] : NaN
    pts = dim && payload.series[dim] ? payload.series[dim] : []
  }

  // ── Gauge ──────────────────────────────────────────────────────────────────
  if (w.type === 'gauge') {
    const vPct = gaugePercentValue(w, val)
    const arcDeg = w.gauge_arc_degrees === 270 ? 270 : 180
    const r = 38
    const cx = 50
    const cy = arcDeg === 270 ? 50 : 72
    const vh = arcDeg === 270 ? 100 : 90
    const { track, fill } = gaugeArcPath(cx, cy, r, arcDeg, vPct / 100)

    return (
      <svg className="w-full h-full" viewBox={`0 0 100 ${vh}`} preserveAspectRatio="xMidYMid meet">
        <path d={track} fill="none" stroke={trackColor} strokeWidth="7" strokeLinecap="round" />
        {vPct > 0.5 && (
          <path d={fill} fill="none" stroke={ACCENT} strokeWidth="7" strokeLinecap="round" />
        )}
        {/* 中心值卡片 */}
        <rect x={cx - 20} y={cy - 9} width={40} height={18} rx={4} fill={panelColor} />
        <text x={cx} y={cy + 5} textAnchor="middle" fill={textColor} fontSize="10" fontWeight="500">
          {Number.isFinite(val) ? `${Math.round(vPct)}%` : '—'}
        </text>
      </svg>
    )
  }

  // ── Line ───────────────────────────────────────────────────────────────────
  if (w.type === 'line' && pts.length < 2) {
    return (
      <div className="flex items-center justify-center h-full text-zinc-600 text-[10px]">
        {t('chartPreview.noData')}
      </div>
    )
  }
  if (w.type === 'line' && pts.length > 1) {
    const W = 100, H = 100
    let minV = pts[0], maxV = pts[0]
    for (const p of pts) {
      if (p < minV) minV = p
      if (p > maxV) maxV = p
    }
    if (maxV === minV) maxV = minV + 1
    const mid = (minV + maxV) / 2
    const axis = !!w.show_y_axis
    const d = linePathD(pts, W, H)
    const fd = lineFillD(pts, W, H)

    return (
      <div className={`flex w-full h-full min-h-0 ${axis ? 'flex-row' : ''}`}>
        {axis && (
          <div className="flex flex-col justify-between text-[8px] leading-tight py-1 pl-0.5 w-[22px] shrink-0" style={{ color: mutedColor }}>
            <span className="truncate text-right">{formatAxisCompact(maxV, w.unit)}</span>
            <span className="truncate text-right">{formatAxisCompact(mid, w.unit)}</span>
            <span className="truncate text-right">{formatAxisCompact(minV, w.unit)}</span>
          </div>
        )}
        <svg className="w-full h-full min-w-0" viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none">
          {/* 面积填充 */}
          <path d={fd} fill={ACCENT} fillOpacity="0.15" stroke="none" />
          {/* 折线 */}
          <path d={d} fill="none" stroke={ACCENT} strokeWidth="1.5" />
        </svg>
      </div>
    )
  }

  // ── Progress ───────────────────────────────────────────────────────────────
  if (w.type === 'progress') {
    if (composite && !Number.isFinite(val)) {
      return (
        <div className="flex items-center justify-center h-full text-zinc-600 text-[10px]">
          {t('chartPreview.noData')}
        </div>
      )
    }
    const maxV = w.unit === 'percent' ? 100 : Math.max(1, Number.isFinite(val) ? Math.abs(val) : 1)
    const tFill = clamp01(val, maxV)
    const fillColor = ACCENT
    if (w.vertical) {
      return (
        <svg className="w-full h-full" viewBox="0 0 40 100" preserveAspectRatio="xMidYMid meet">
          <rect x="10" y="4" width="20" height="92" rx="4" fill={trackColor} />
          {tFill > 0.01 && (
            <rect x="10" y={4 + 92 * (1 - tFill)} width="20"
              height={Math.max(3, 92 * tFill)} rx="4" fill={fillColor} />
          )}
          <text x="20" y="52" textAnchor="middle" fill={textColor} fontSize="9">
            {`${Math.round(tFill * 100)}%`}
          </text>
        </svg>
      )
    }
    return (
      <svg className="w-full h-full" viewBox="0 0 100 24" preserveAspectRatio="none">
        <rect x="4" y="7" width="92" height="10" rx="4" fill={trackColor} />
        {tFill > 0.01 && (
          <rect x="4" y="7" width={Math.max(5, 92 * tFill)} height="10" rx="4" fill={fillColor} />
        )}
        <text x="50" y="15" textAnchor="middle" fill={textColor} fontSize="7">
          {`${Math.round(tFill * 100)}%`}
        </text>
      </svg>
    )
  }

  // ── Histogram ──────────────────────────────────────────────────────────────
  if (w.type === 'histogram') {
    if (composite && exprLines.length === 0) {
      return (
        <div className="flex items-center justify-center h-full text-zinc-600 text-[10px]">
          {t('chartPreview.noData')}
        </div>
      )
    }
    if (composite && exprLines.length > 0) {
      const env = buildLatestEnvForComposite(payload.latest, exprLines)
      const vals = exprLines.map((line) => {
        try {
          const x = evalScalar(line, env)
          return Number.isFinite(x) ? Math.abs(x) : 0
        } catch {
          return 0
        }
      })
      const mx = Math.max(1, ...vals)
      const n = vals.length || 1
      const gap = 3
      const bw = Math.max(4, (100 - 16 - gap * (n - 1)) / n)
      return (
        <svg className="w-full h-full" viewBox="0 0 100 60" preserveAspectRatio="none">
          {vals.map((v, i) => {
            const bh = (Math.abs(v) / mx) * 48
            const bx = 8 + i * (bw + gap)
            return (
              <rect key={i} x={bx} y={56 - bh} width={bw} height={Math.max(2, bh)}
                fill={DIM_COLORS[i % DIM_COLORS.length]} rx="2" />
            )
          })}
        </svg>
      )
    }
    const dims = (w.dimensions?.length ? w.dimensions : Object.keys(payload.latest)).slice(0, 6)
    const vals = dims.map((d) => payload.latest[d] ?? 0)
    const mx = Math.max(1, ...vals.map(Math.abs))
    const n = vals.length || 1
    const gap = 3
    const bw = Math.max(4, (100 - 16 - gap * (n - 1)) / n)
    return (
      <svg className="w-full h-full" viewBox="0 0 100 60" preserveAspectRatio="none">
        {vals.map((v, i) => {
          const bh = (Math.abs(v) / mx) * 48
          const bx = 8 + i * (bw + gap)
          return (
            <rect key={i} x={bx} y={56 - bh} width={bw} height={Math.max(2, bh)}
              fill={DIM_COLORS[i % DIM_COLORS.length]} rx="2" />
          )
        })}
      </svg>
    )
  }

  return (
    <div className="flex items-center justify-center h-full text-zinc-500 text-[10px]">
      {Number.isFinite(val) ? val.toFixed(1) : '—'}
    </div>
  )
}
