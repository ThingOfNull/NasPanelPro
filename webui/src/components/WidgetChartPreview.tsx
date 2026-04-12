import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api'
import type { Widget } from '../types'

export type ChartSeriesPayload = {
  labels: string[]
  series: Record<string, number[]>
  latest: Record<string, number>
}

function pickDim(w: Widget, latest: Record<string, number>): string | undefined {
  const d0 = w.dimensions?.find(Boolean)
  if (d0 && latest[d0] !== undefined) {
    return d0
  }
  const keys = Object.keys(latest)
  return keys[0]
}

function clamp01(x: number, max: number) {
  if (!Number.isFinite(x)) {
    return 0
  }
  if (max <= 0) {
    return 0
  }
  return Math.max(0, Math.min(1, x / max))
}

/** Match drawWidgetGauge percent scaling (internal/raypanel/widgets_draw.go). */
function gaugePercentValue(w: Widget, raw: number): number {
  let v = raw
  if (!Number.isFinite(v)) {
    return 0
  }
  const u = w.unit ?? ''
  if (u !== '' && u.toLowerCase() === 'percent') {
    return Math.max(0, Math.min(100, v))
  }
  v = Math.abs(v)
  if (v > 1.5) {
    v /= 100
  }
  if (v > 1) {
    v = 1
  }
  return v * 100
}

/**
 * Line geometry matches drawWidgetLine: vertical padding 4px each side of box height,
 * x spread across full width (see internal/raypanel/widgets_draw.go).
 * Data source matches DRM：Netdata 时间窗多 points，非逐秒单点追加。
 */
function linePathD(pts: number[], boxW: number, boxH: number): string {
  if (pts.length < 2) {
    return ''
  }
  let minV = pts[0]
  let maxV = pts[0]
  for (const p of pts) {
    if (p < minV) {
      minV = p
    }
    if (p > maxV) {
      maxV = p
    }
  }
  if (maxV === minV) {
    maxV = minV + 1
  }
  const py = (val: number) => {
    const tt = (val - minV) / (maxV - minV)
    return boxH - 4 - tt * (boxH - 8)
  }
  let den = pts.length - 1
  if (den < 1) {
    den = 1
  }
  const parts: string[] = []
  for (let i = 1; i < pts.length; i++) {
    const x0 = ((i - 1) / den) * boxW
    const x1 = (i / den) * boxW
    const y0 = py(pts[i - 1])
    const y1 = py(pts[i])
    if (i === 1) {
      parts.push(`M${x0.toFixed(1)},${y0.toFixed(1)}`)
    }
    parts.push(`L${x1.toFixed(1)},${y1.toFixed(1)}`)
  }
  return parts.join(' ')
}

function formatAxisCompact(v: number, unit?: string): string {
  const u = (unit ?? '').toLowerCase()
  if (!Number.isFinite(v)) {
    return '—'
  }
  if (u === 'percent') {
    return `${Math.round(v)}%`
  }
  if (u === 'bytes') {
    const bytes = v * 1024
    const g = bytes / 1024 ** 3
    if (g >= 1) {
      return `${g.toFixed(1)}G`
    }
    const m = bytes / 1024 ** 2
    if (m >= 1) {
      return `${Math.round(m)}M`
    }
    const k = bytes / 1024
    return `${Math.round(k)}K`
  }
  if (Math.abs(v) >= 1000) {
    return `${Math.round(v)}`
  }
  return v.toFixed(1)
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
          if (!cancelled) {
            setPayload(r)
            setFailed(false)
          }
        })
        .catch(() => {
          if (!cancelled) {
            setPayload(null)
            setFailed(true)
          }
        })
    }
    tick()
    const id = window.setInterval(tick, 8000)
    return () => {
      cancelled = true
      window.clearInterval(id)
    }
  }, [w.type, w.chart_id, w.node_id, fallbackNodeId, nodeId])

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

  const dim = pickDim(w, payload.latest)
  const val = dim !== undefined ? payload.latest[dim] : NaN
  const pts = dim && payload.series[dim] ? payload.series[dim] : []

  const maxV =
    w.unit === 'percent'
      ? 100
      : Math.max(
          1,
          ...pts.filter(Number.isFinite),
          Number.isFinite(val) ? Math.abs(val) : 0,
        )

  if (w.type === 'gauge') {
    const vPct = gaugePercentValue(w, val)
    const tArc = clamp01(vPct, 100)
    const arc = 180
    const r0 = 42
    const cx = 50
    const cy = 78
    const a0 = Math.PI * (1 - tArc * (arc / 180))
    const x2 = cx + r0 * Math.cos(a0)
    const y2 = cy - r0 * Math.sin(a0)
    const large = tArc > 0.5 ? 1 : 0
    const d = `M ${cx - r0} ${cy} A ${r0} ${r0} 0 ${large} 1 ${x2} ${y2}`
    return (
      <svg className="w-full h-full" viewBox="0 0 100 90" preserveAspectRatio="xMidYMid meet">
        <path
          d={`M ${cx - r0} ${cy} A ${r0} ${r0} 0 0 1 ${cx + r0} ${cy}`}
          fill="none"
          stroke="#30363d"
          strokeWidth="6"
        />
        <path d={d} fill="none" stroke="#38bdf8" strokeWidth="6" strokeLinecap="round" />
        <text x={cx} y={cy + 8} textAnchor="middle" fill="#8b949e" fontSize="10">
          {Number.isFinite(val) ? `${Math.round(vPct)}%` : '—'}
        </text>
      </svg>
    )
  }

  if (w.type === 'line' && pts.length > 1) {
    const W = 100
    const H = 100
    let minV = pts[0]
    let maxV = pts[0]
    for (const p of pts) {
      if (p < minV) {
        minV = p
      }
      if (p > maxV) {
        maxV = p
      }
    }
    if (maxV === minV) {
      maxV = minV + 1
    }
    const mid = (minV + maxV) / 2
    const axis = !!w.show_y_axis
    const d = linePathD(pts, W, H)
    return (
      <div className={`flex w-full h-full min-h-0 ${axis ? 'flex-row' : ''}`}>
        {axis && (
          <div className="flex flex-col justify-between text-[8px] text-zinc-500 leading-tight py-1 pl-0.5 w-[22px] shrink-0">
            <span className="truncate text-right">{formatAxisCompact(maxV, w.unit)}</span>
            <span className="truncate text-right">{formatAxisCompact(mid, w.unit)}</span>
            <span className="truncate text-right">{formatAxisCompact(minV, w.unit)}</span>
          </div>
        )}
        <svg className="w-full h-full min-w-0" viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none">
          <path d={d} fill="none" stroke="#38bdf8" strokeWidth="1.2" />
        </svg>
      </div>
    )
  }

  if (w.type === 'progress') {
    const tFill = clamp01(val, maxV)
    if (w.vertical) {
      return (
        <svg className="w-full h-full" viewBox="0 0 40 100" preserveAspectRatio="xMidYMid meet">
          <rect x="12" y="4" width="16" height="92" rx="3" fill="#21262d" />
          <rect
            x="12"
            y={4 + 92 * (1 - tFill)}
            width="16"
            height={Math.max(2, (92 * tFill) | 0)}
            rx="3"
            fill="#3fb950"
          />
        </svg>
      )
    }
    return (
      <svg className="w-full h-full" viewBox="0 0 100 24" preserveAspectRatio="none">
        <rect x="4" y="8" width="92" height="10" rx="3" fill="#21262d" />
        <rect x="4" y="8" width={Math.max(4, 92 * tFill)} height="10" rx="3" fill="#3fb950" />
      </svg>
    )
  }

  if (w.type === 'histogram') {
    const dims = (w.dimensions?.length ? w.dimensions : Object.keys(payload.latest)).slice(0, 6)
    const vals = dims.map((d) => payload.latest[d] ?? 0)
    const mx = Math.max(1, ...vals.map(Math.abs))
    const n = vals.length || 1
    const gap = 2
    const bw = Math.max(4, (100 - 16 - gap * (n - 1)) / n)
    return (
      <svg className="w-full h-full" viewBox="0 0 100 56" preserveAspectRatio="none">
        {vals.map((v, i) => {
          const bh = (Math.abs(v) / mx) * 44
          const x = 8 + i * (bw + gap)
          return <rect key={i} x={x} y={52 - bh} width={bw} height={bh} fill="#58a6ff" rx="1" />
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
