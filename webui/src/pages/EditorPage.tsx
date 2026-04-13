import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { DragEndEvent } from '@dnd-kit/core'
import {
  DndContext,
  PointerSensor,
  useSensor,
  useSensors,
  useDraggable,
} from '@dnd-kit/core'
import { FixedSizeList as List } from 'react-window'
import type { ListChildComponentProps } from 'react-window'
import WidgetChartPreview from '../components/WidgetChartPreview'
import { api } from '../api'
import { newWidgetId } from '../lib/id'
import { useAppStore } from '../store/appStore'
import type { ChartDiscoveryRow, Widget } from '../types'

const SNAP = 8
const snap = (v: number) => Math.round(v / SNAP) * SNAP
const HANDLE = 8

function applyResize(
  orig: { x: number; y: number; w: number; h: number },
  edge: string,
  dx: number,
  dy: number,
) {
  let { x, y, w, h } = orig
  if (edge.includes('e')) w = orig.w + dx
  if (edge.includes('w')) { x = orig.x + dx; w = orig.w - dx }
  if (edge.includes('s')) h = orig.h + dy
  if (edge.includes('n')) { y = orig.y + dy; h = orig.h - dy }
  return { x, y, w: Math.max(32, w), h: Math.max(24, h) }
}

const EDGES = ['n', 's', 'e', 'w', 'nw', 'ne', 'sw', 'se'] as const

function StageWidget({
  si, wi, w, selected, chartNodeId, onSelect, onResizePointerDown,
}: {
  si: number; wi: number; w: Widget; selected: boolean; chartNodeId: string
  onSelect: () => void
  onResizePointerDown: (edge: string, e: React.PointerEvent) => void
}) {
  const id = `w-${si}-${wi}`
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({ id })

  const edgeStyle = (edge: string): React.CSSProperties => {
    const hit = HANDLE
    switch (edge) {
      case 'n': return { top: 0, left: hit, right: hit, height: hit, cursor: 'ns-resize' }
      case 's': return { bottom: 0, left: hit, right: hit, height: hit, cursor: 'ns-resize' }
      case 'w': return { top: hit, bottom: hit, left: 0, width: hit, cursor: 'ew-resize' }
      case 'e': return { top: hit, bottom: hit, right: 0, width: hit, cursor: 'ew-resize' }
      case 'nw': return { top: 0, left: 0, width: hit + 2, height: hit + 2, cursor: 'nwse-resize' }
      case 'ne': return { top: 0, right: 0, width: hit + 2, height: hit + 2, cursor: 'nesw-resize' }
      case 'sw': return { bottom: 0, left: 0, width: hit + 2, height: hit + 2, cursor: 'nesw-resize' }
      case 'se': return { bottom: 0, right: 0, width: hit + 2, height: hit + 2, cursor: 'nwse-resize' }
      default: return {}
    }
  }

  const title = w.label?.trim() || w.chart_id || w.type
  const outerStyle: React.CSSProperties = {
    position: 'absolute', left: w.x, top: w.y, width: w.w, height: w.h,
    boxSizing: 'border-box',
    border: selected ? '2px solid #58a6ff' : '1px dashed #58d68d44',
    transform: transform ? `translate3d(${transform.x}px,${transform.y}px,0)` : undefined,
    zIndex: isDragging ? 30 : 2,
    overflow: 'hidden', cursor: 'grab',
  }

  return (
    <div ref={setNodeRef} style={outerStyle} {...listeners} {...attributes}
      onClick={(e) => { e.stopPropagation(); onSelect() }}>
      <div className={`absolute inset-0 z-0 pointer-events-none ${w.show_border ? 'ring-1 ring-inset ring-zinc-600' : ''}`}>
        <WidgetChartPreview w={w} fallbackNodeId={chartNodeId} />
      </div>
      {!w.hide_label && (
        <div className="absolute top-0 left-0 right-0 z-[4] pointer-events-none px-1.5 py-0.5 text-[10px] leading-tight text-zinc-200/90 bg-zinc-950/60 border-b border-zinc-700/30 truncate">
          {title}
        </div>
      )}
      {EDGES.map((edge) => (
        <div key={edge} className="absolute z-[5]" style={edgeStyle(edge)}
          onPointerDownCapture={(e) => { e.stopPropagation(); onResizePointerDown(edge, e) }} />
      ))}
    </div>
  )
}

// ── 可折叠属性分组 ──────────────────────────────────────────────────────────
function PropGroup({ title, children }: { title: string; children: React.ReactNode }) {
  const [open, setOpen] = useState(true)
  return (
    <div className="border border-zinc-800 rounded-md overflow-hidden">
      <button
        type="button"
        className="w-full flex items-center justify-between px-3 py-1.5 bg-zinc-900 hover:bg-zinc-800 text-xs font-medium text-zinc-400 transition-colors"
        onClick={() => setOpen((v) => !v)}
      >
        <span>{title}</span>
        <span className="text-zinc-600">{open ? '▾' : '▸'}</span>
      </button>
      {open && <div className="p-2 space-y-2 bg-zinc-950">{children}</div>}
    </div>
  )
}

// ── 属性输入行 ──────────────────────────────────────────────────────────────
function PropLabel({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block text-[11px] text-zinc-500">
      <span className="block mb-0.5">{label}</span>
      {children}
    </label>
  )
}

export default function EditorPage() {
  const { t } = useTranslation()
  const { layout, setLayout, loadLayout, saveLayout, nodes, loadNodes, setStatus } = useAppStore()
  const [sceneIdx, setSceneIdx] = useState(0)
  const [sel, setSel] = useState<{ si: number; wi: number } | null>(null)
  const [scale, setScale] = useState(0.88)
  const [chartNodeId, setChartNodeId] = useState('')
  const [searchQ, setSearchQ] = useState('')
  const [allRows, setAllRows] = useState<ChartDiscoveryRow[]>([])
  const [filteredIdx, setFilteredIdx] = useState<number[]>([])
  const workerRef = useRef<Worker | null>(null)
  const searchT = useRef<ReturnType<typeof setTimeout>>(null)

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }))

  const resizeRef = useRef<{
    si: number; wi: number; type: string; edge: string; cx: number; cy: number
    orig: { x: number; y: number; w: number; h: number }
  } | null>(null)

  useEffect(() => {
    const onMove = (e: PointerEvent) => {
      const r = resizeRef.current
      if (!r) return
      const layoutNow = useAppStore.getState().layout
      if (!layoutNow) return
      const dx = (e.clientX - r.cx) / scale
      const dy = (e.clientY - r.cy) / scale
      let n = applyResize(r.orig, r.edge, dx, dy)
      if (r.type === 'gauge') {
        const side = Math.max(48, snap(Math.min(n.w, n.h)))
        n = { ...n, w: side, h: side }
      }
      const next = structuredClone(layoutNow)
      const tw = next.scenes[r.si].widgets[r.wi]
      tw.x = snap(n.x); tw.y = snap(n.y); tw.w = snap(n.w); tw.h = snap(n.h)
      setLayout(next)
    }
    const onUp = () => {
      const r = resizeRef.current
      resizeRef.current = null
      if (!r || r.type !== 'gauge') return
      const layoutNow = useAppStore.getState().layout
      if (!layoutNow) return
      const next = structuredClone(layoutNow)
      const tw = next.scenes[r.si].widgets[r.wi]
      const side = Math.max(48, snap(Math.min(tw.w, tw.h)))
      tw.w = side; tw.h = side
      setLayout(next)
    }
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
    return () => { window.removeEventListener('pointermove', onMove); window.removeEventListener('pointerup', onUp) }
  }, [scale, setLayout])

  const onResizePointerDown = useCallback(
    (edge: string, e: React.PointerEvent, si: number, wi: number, wtype: string) => {
      const layoutNow = useAppStore.getState().layout
      if (!layoutNow) return
      const tw = layoutNow.scenes[si].widgets[wi]
      resizeRef.current = { si, wi, type: wtype, edge, cx: e.clientX, cy: e.clientY, orig: { x: tw.x, y: tw.y, w: tw.w, h: tw.h } }
    }, [],
  )

  useEffect(() => {
    loadLayout().catch((e) => setStatus(String(e)))
    loadNodes().catch(() => {})
  }, [loadLayout, loadNodes, setStatus])

  useEffect(() => {
    const w = new Worker(new URL('../workers/chartSearch.worker.ts', import.meta.url), { type: 'module' })
    workerRef.current = w
    w.onmessage = (e: MessageEvent<{ type: string; indices?: number[]; n?: number }>) => {
      if (e.data.type === 'result' && e.data.indices) setFilteredIdx(e.data.indices)
    }
    return () => { w.terminate(); workerRef.current = null }
  }, [])

  const reloadDiscovery = useCallback(async () => {
    try {
      const r = await api.getDiscovery(chartNodeId)
      setAllRows(r.charts)
      workerRef.current?.postMessage({ type: 'load', rows: r.charts })
      workerRef.current?.postMessage({ type: 'search', q: searchQ, max: 2000 })
    } catch (e) {
      setAllRows([]); setFilteredIdx([]); setStatus(String(e))
    }
  }, [chartNodeId, searchQ, setStatus])

  useEffect(() => { reloadDiscovery() }, [chartNodeId, reloadDiscovery])

  useEffect(() => {
    if (searchT.current) clearTimeout(searchT.current)
    searchT.current = setTimeout(() => {
      workerRef.current?.postMessage({ type: 'search', q: searchQ, max: 2000 })
    }, 120)
    return () => { if (searchT.current) clearTimeout(searchT.current) }
  }, [searchQ, allRows.length])

  const onDragEnd = (e: DragEndEvent) => {
    const id = String(e.active.id)
    if (!id.startsWith('w-')) return
    const [, a, b] = id.split('-')
    const si = parseInt(a, 10); const wi = parseInt(b, 10)
    const dx = e.delta.x / scale; const dy = e.delta.y / scale
    if (!layout) return
    const next = structuredClone(layout)
    const ww = next.scenes[si].widgets[wi]
    ww.x = snap(ww.x + dx); ww.y = snap(ww.y + dy)
    setLayout(next)
  }

  const addWidget = (type: string) => {
    if (!layout) return
    const next = structuredClone(layout)
    const si = sceneIdx
    const w: Widget = {
      id: newWidgetId(), type,
      x: snap(64), y: snap(64),
      w: type === 'text' ? 220 : 140,
      h: type === 'progress' ? 160 : type === 'gauge' ? 120 : 52,
      chart_id: type === 'text' ? '' : 'system.cpu',
      dimensions: type === 'text' ? [] : ['user'],
      label: type === 'text' ? t('chartPreview.textDefault') : '',
      unit: type === 'text' ? '' : 'percent',
    }
    next.scenes[si].widgets.push(w)
    setLayout(next)
    setSel({ si, wi: next.scenes[si].widgets.length - 1 })
  }

  const addScene = () => {
    if (!layout) return
    const next = structuredClone(layout)
    const id = `scene${Date.now()}`
    next.scenes.push({ id, name: `Scene ${next.scenes.length + 1}`, widgets: [] })
    setLayout(next)
    setSceneIdx(next.scenes.length - 1)
    setSel(null)
  }

  const updateSelected = (patch: Partial<Widget>) => {
    if (!layout || !sel) return
    const next = structuredClone(layout)
    const tw = next.scenes[sel.si].widgets[sel.wi]
    Object.assign(tw, patch)
    if (patch.data) {
      const d = patch.data
      if (d.node) tw.node_id = d.node
      if (d.chart) tw.chart_id = d.chart
      if (d.dim) tw.dimensions = [d.dim]
    }
    setLayout(next)
  }

  const bindChart = (row: ChartDiscoveryRow) => {
    if (!sel) return
    updateSelected({ chart_id: row.id, data: { node: chartNodeId, chart: row.id, dim: '' } })
  }

  const selectedW = layout && sel ? layout.scenes[sel.si]?.widgets[sel.wi] : undefined

  const [listH, setListH] = useState(420)
  useEffect(() => {
    const fn = () => setListH(Math.max(200, window.innerHeight - 220))
    fn(); window.addEventListener('resize', fn)
    return () => window.removeEventListener('resize', fn)
  }, [])

  const [dimOpts, setDimOpts] = useState<string[]>([])
  useEffect(() => {
    if (!selectedW?.chart_id) { setDimOpts([]); return }
    api.getChartMeta(selectedW.chart_id, chartNodeId)
      .then((m) => setDimOpts(m.dimensions ?? []))
      .catch(() => setDimOpts([]))
  }, [selectedW?.chart_id, chartNodeId])

  const renderChartRow = ({ index, style }: ListChildComponentProps) => {
    const i = filteredIdx[index]
    const row = allRows[i]
    if (!row) return <div style={style} />
    return (
      <div style={style}
        className="px-2 text-xs truncate cursor-pointer hover:bg-zinc-800/80 border-b border-zinc-800/60 flex items-center gap-1"
        onClick={() => bindChart(row)} title={row.title}>
        <span className="text-zinc-600 shrink-0">{row.family}</span>
        <span className="text-zinc-400 truncate">· {row.id}</span>
      </div>
    )
  }

  if (!layout) {
    return <div className="p-6 text-zinc-500">{t('editor.loadingLayout')}</div>
  }

  const sc = layout.scenes[sceneIdx] ?? layout.scenes[0]
  const stageW = layout.screen_width || 1280
  const stageH = layout.screen_height || 480

  // 棋盘格背景 CSS
  const checkerBg: React.CSSProperties = {
    backgroundImage: 'repeating-conic-gradient(#18181b 0% 25%, #09090b 0% 50%)',
    backgroundSize: '16px 16px',
  }

  return (
    <DndContext sensors={sensors} onDragEnd={onDragEnd}>
      <div className="flex flex-col h-[calc(100vh-52px)] min-h-[480px]">
        {/* 顶部工具栏 */}
        <div className="flex items-center gap-2 px-3 py-2 border-b border-zinc-800 bg-zinc-900/90 shrink-0 flex-wrap">
          <span className="text-xs text-zinc-500">{t('editor.scene')}</span>
          <select
            className="bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-sm text-zinc-200"
            value={sceneIdx}
            onChange={(e) => { setSceneIdx(parseInt(e.target.value, 10)); setSel(null) }}
          >
            {layout.scenes.map((s, i) => (
              <option key={s.id} value={i}>{s.name || s.id}</option>
            ))}
          </select>
          {/* 新建场景 */}
          <button type="button"
            className="px-2 py-1 rounded bg-zinc-700 hover:bg-zinc-600 text-xs text-zinc-300 transition-colors"
            onClick={addScene} title={t('editor.addScene') ?? '+ Scene'}>
            + {t('editor.addScene') ?? 'Scene'}
          </button>
          <button type="button"
            className="ml-2 px-3 py-1 rounded bg-emerald-700 hover:bg-emerald-600 text-sm transition-colors"
            onClick={() => saveLayout().catch((e) => setStatus(String(e)))}>
            {t('editor.saveLayout')}
          </button>
          <label className="text-xs text-zinc-500 ml-2 flex items-center gap-1">
            {t('editor.scale')}
            <input type="range" min={0.3} max={1.8} step={0.05} value={scale}
              onChange={(e) => setScale(parseFloat(e.target.value))} />
            <span className="w-8 text-right text-zinc-600">{Math.round(scale * 100)}%</span>
          </label>
        </div>

        <div className="flex flex-1 min-h-0">
          {/* ── 资产浏览器 ── */}
          <aside className="w-68 border-r border-zinc-800 flex flex-col shrink-0 bg-zinc-950" style={{ width: 272 }}>
            <div className="p-2 border-b border-zinc-800 space-y-1.5">
              <div className="text-[10px] text-zinc-600">{t('editor.metricNodeHint')}</div>
              <select
                className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs text-zinc-200"
                value={chartNodeId} onChange={(e) => setChartNodeId(e.target.value)}>
                <option value="">{t('editor.defaultNodeShort')}</option>
                {nodes.map((n) => (
                  <option key={n.id} value={n.id}>{n.name} ({n.id})</option>
                ))}
              </select>
              <input
                className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs text-zinc-200 placeholder-zinc-600"
                placeholder={t('editor.searchPlaceholder')}
                value={searchQ} onChange={(e) => setSearchQ(e.target.value)} />
              <div className="text-[10px] text-zinc-700">
                {t('editor.chartListStats', { filtered: filteredIdx.length, total: allRows.length })}
              </div>
            </div>
            <div className="flex-1 min-h-0">
              {filteredIdx.length > 0 ? (
                <List height={listH} itemCount={filteredIdx.length} itemSize={26} width="100%">
                  {renderChartRow}
                </List>
              ) : (
                <div className="p-3 text-xs text-zinc-700">{t('editor.noChartData')}</div>
              )}
            </div>
            <div className="p-2 border-t border-zinc-800 text-[10px] text-zinc-700">
              {t('editor.bindHint')}
            </div>
          </aside>

          {/* ── 画布 ── */}
          <div className="flex-1 overflow-auto p-4 flex justify-center items-start" style={checkerBg}>
            <div className="relative shadow-2xl border border-zinc-700/60" style={{ width: stageW * scale, height: stageH * scale, background: '#0d1117' }}>
              <div className="origin-top-left" style={{ width: stageW, height: stageH, transform: `scale(${scale})` }}
                onClick={() => setSel(null)}>
                {sc.widgets.map((w, wi) => (
                  <StageWidget
                    key={w.id ?? `${wi}`} si={sceneIdx} wi={wi} w={w}
                    selected={sel?.si === sceneIdx && sel?.wi === wi}
                    chartNodeId={chartNodeId}
                    onSelect={() => setSel({ si: sceneIdx, wi })}
                    onResizePointerDown={(edge, ev) => onResizePointerDown(edge, ev, sceneIdx, wi, w.type)}
                  />
                ))}
              </div>
            </div>
          </div>

          {/* ── 组件调色板 + 属性 ── */}
          <aside className="w-80 border-l border-zinc-800 flex flex-col shrink-0 overflow-auto bg-zinc-950">
            {/* 添加组件 */}
            <div className="p-2 border-b border-zinc-800">
              <div className="text-[11px] text-zinc-500 mb-1.5 font-medium">{t('editor.addToScene')}</div>
              <div className="flex flex-wrap gap-1">
                {['text', 'gauge', 'line', 'progress', 'histogram'].map((tp) => (
                  <button key={tp} type="button"
                    className="px-2 py-0.5 rounded bg-zinc-800 hover:bg-zinc-700 text-xs text-zinc-300 border border-zinc-700 transition-colors"
                    onClick={() => addWidget(tp)}>
                    +{tp}
                  </button>
                ))}
              </div>
            </div>

            {/* 属性面板 */}
            <div className="flex-1 p-2 space-y-1.5 overflow-auto">
              <div className="text-[11px] text-zinc-500 font-medium px-1 py-1">{t('editor.props')}</div>
              {!selectedW ? (
                <p className="text-zinc-700 text-xs px-1">{t('editor.selectWidget')}</p>
              ) : (
                <>
                  {/* ── 数据绑定 ── */}
                  <PropGroup title={t('editor.groupDataBinding') ?? 'Data Binding'}>
                    <PropLabel label="node_id">
                      <select
                        className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs"
                        value={selectedW.node_id ?? ''}
                        onChange={(e) => updateSelected({ node_id: e.target.value || undefined })}>
                        <option value="">{t('editor.nodeIdDefault')}</option>
                        {nodes.map((n) => (
                          <option key={n.id} value={n.id}>{n.id}</option>
                        ))}
                      </select>
                    </PropLabel>
                    <PropLabel label="chart_id">
                      <input
                        className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 font-mono text-xs"
                        value={selectedW.chart_id ?? ''}
                        onChange={(e) => updateSelected({ chart_id: e.target.value })} />
                    </PropLabel>
                    <PropLabel label={t('editor.dimensionsHint')}>
                      <input
                        className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs"
                        value={(selectedW.dimensions ?? []).join(',')}
                        onChange={(e) => updateSelected({
                          dimensions: e.target.value.split(',').map((s) => s.trim()).filter(Boolean),
                        })} />
                    </PropLabel>
                    {dimOpts.length > 0 && (
                      <div className="space-y-1">
                        <div className="text-[10px] text-zinc-600">{t('editor.dimQuick')}</div>
                        <div className="flex flex-wrap gap-1">
                          {dimOpts.map((d) => (
                            <button key={d} type="button"
                              className="px-1.5 py-0.5 rounded bg-zinc-800 hover:bg-zinc-700 text-[10px] border border-zinc-700"
                              onClick={() => updateSelected({ dimensions: [d] })}>
                              {d}
                            </button>
                          ))}
                        </div>
                      </div>
                    )}
                  </PropGroup>

                  {/* ── 几何布局 ── */}
                  <PropGroup title={t('editor.groupLayout') ?? 'Layout'}>
                    <PropLabel label={t('editor.xywh')}>
                      <div className="grid grid-cols-4 gap-1">
                        {(['x', 'y', 'w', 'h'] as const).map((k) => (
                          <div key={k} className="relative">
                            <span className="absolute left-1.5 top-1 text-[9px] text-zinc-600">{k}</span>
                            <input type="number"
                              className="w-full bg-zinc-800 border border-zinc-700 rounded pt-4 pb-0.5 px-1 text-xs"
                              value={Math.round(selectedW[k] as number)}
                              onChange={(e) => updateSelected({ [k]: parseFloat(e.target.value) || 0 } as Partial<Widget>)} />
                          </div>
                        ))}
                      </div>
                    </PropLabel>
                    {selectedW.type === 'line' && (
                      <>
                        <label className="flex items-center gap-2 text-[11px] text-zinc-500 cursor-pointer">
                          <input type="checkbox" checked={!!selectedW.show_y_axis}
                            onChange={(e) => updateSelected({ show_y_axis: e.target.checked || undefined })} />
                          {t('editor.showYAxis')}
                        </label>
                        <PropLabel label={t('editor.linePoints')}>
                          <input type="number" min={8} max={512}
                            className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs"
                            value={selectedW.line_points ?? ''} placeholder="96"
                            onChange={(e) => {
                              const v = e.target.value
                              updateSelected({ line_points: v ? Math.min(512, Math.max(8, parseInt(v, 10) || 96)) : undefined })
                            }} />
                        </PropLabel>
                      </>
                    )}
                    {selectedW.type === 'progress' && (
                      <label className="flex items-center gap-2 text-[11px] text-zinc-500 cursor-pointer">
                        <input type="checkbox" checked={!!selectedW.vertical}
                          onChange={(e) => updateSelected({ vertical: e.target.checked || undefined })} />
                        Vertical
                      </label>
                    )}
                    {selectedW.type === 'gauge' && (
                      <PropLabel label="gauge_arc_degrees">
                        <select
                          className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs"
                          value={selectedW.gauge_arc_degrees ?? 180}
                          onChange={(e) => updateSelected({ gauge_arc_degrees: parseInt(e.target.value, 10) })}>
                          <option value={180}>180°</option>
                          <option value={270}>270°</option>
                        </select>
                      </PropLabel>
                    )}
                  </PropGroup>

                  {/* ── 外观样式 ── */}
                  <PropGroup title={t('editor.groupAppearance') ?? 'Appearance'}>
                    <PropLabel label="label">
                      <input
                        className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs"
                        value={selectedW.label ?? ''}
                        onChange={(e) => updateSelected({ label: e.target.value })} />
                    </PropLabel>
                    <PropLabel label="unit">
                      <select
                        className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs"
                        value={selectedW.unit ?? ''}
                        onChange={(e) => updateSelected({ unit: e.target.value })}>
                        <option value="">auto</option>
                        <option value="percent">percent</option>
                        <option value="bytes">bytes</option>
                        <option value="none">none</option>
                      </select>
                    </PropLabel>
                    <PropLabel label={t('editor.colorLabel')}>
                      <input
                        className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs"
                        value={selectedW.color ?? ''} placeholder="#58d68d"
                        onChange={(e) => updateSelected({ color: e.target.value })} />
                    </PropLabel>
                    <div className="grid grid-cols-2 gap-1">
                      <PropLabel label="font_size">
                        <input type="number"
                          className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs"
                          value={selectedW.font_size ?? ''} placeholder="auto"
                          onChange={(e) => updateSelected({ font_size: e.target.value ? parseFloat(e.target.value) : undefined })} />
                      </PropLabel>
                      <PropLabel label="rotation">
                        <input type="number"
                          className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs"
                          value={selectedW.rotation ?? ''} placeholder="0"
                          onChange={(e) => updateSelected({ rotation: e.target.value ? parseFloat(e.target.value) : undefined })} />
                      </PropLabel>
                    </div>
                    <div className="grid grid-cols-2 gap-1">
                      <PropLabel label="warn">
                        <input type="number"
                          className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs"
                          value={selectedW.warn_threshold ?? ''}
                          onChange={(e) => updateSelected({ warn_threshold: e.target.value ? parseFloat(e.target.value) : undefined })} />
                      </PropLabel>
                      <PropLabel label="critical">
                        <input type="number"
                          className="w-full bg-zinc-800 border border-zinc-700 rounded px-2 py-1 text-xs"
                          value={selectedW.critical_threshold ?? ''}
                          onChange={(e) => updateSelected({ critical_threshold: e.target.value ? parseFloat(e.target.value) : undefined })} />
                      </PropLabel>
                    </div>
                    <div className="flex flex-col gap-1 pt-0.5">
                      <label className="flex items-center gap-2 text-[11px] text-zinc-500 cursor-pointer">
                        <input type="checkbox" checked={!!selectedW.show_border}
                          onChange={(e) => updateSelected({ show_border: e.target.checked || undefined })} />
                        {t('editor.showBorder')}
                      </label>
                      <label className="flex items-center gap-2 text-[11px] text-zinc-500 cursor-pointer">
                        <input type="checkbox" checked={!!selectedW.hide_label}
                          onChange={(e) => updateSelected({ hide_label: e.target.checked || undefined })} />
                        {t('editor.hideLabelBanner')}
                      </label>
                    </div>
                  </PropGroup>

                  {/* 删除按钮 */}
                  <button type="button"
                    className="w-full py-1.5 rounded bg-red-950/60 hover:bg-red-900/60 text-xs text-red-400 border border-red-900/40 transition-colors mt-1"
                    onClick={() => {
                      if (!sel) return
                      const next = structuredClone(layout)
                      next.scenes[sel.si].widgets.splice(sel.wi, 1)
                      setLayout(next); setSel(null)
                    }}>
                    {t('editor.deleteWidget')}
                  </button>
                </>
              )}
            </div>
          </aside>
        </div>
      </div>
    </DndContext>
  )
}
