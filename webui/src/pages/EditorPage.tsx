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
const STAGE_W = 1280
const STAGE_H = 480
const HANDLE = 8

function applyResize(
  orig: { x: number; y: number; w: number; h: number },
  edge: string,
  dx: number,
  dy: number,
) {
  let { x, y, w, h } = orig
  if (edge.includes('e')) w = orig.w + dx
  if (edge.includes('w')) {
    x = orig.x + dx
    w = orig.w - dx
  }
  if (edge.includes('s')) h = orig.h + dy
  if (edge.includes('n')) {
    y = orig.y + dy
    h = orig.h - dy
  }
  return {
    x,
    y,
    w: Math.max(32, w),
    h: Math.max(24, h),
  }
}

const EDGES = ['n', 's', 'e', 'w', 'nw', 'ne', 'sw', 'se'] as const

function StageWidget({
  si,
  wi,
  w,
  selected,
  chartNodeId,
  onSelect,
  onResizePointerDown,
}: {
  si: number
  wi: number
  w: Widget
  selected: boolean
  chartNodeId: string
  onSelect: () => void
  onResizePointerDown: (edge: string, e: React.PointerEvent) => void
}) {
  const id = `w-${si}-${wi}`
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({ id })

  const edgeStyle = (edge: string): React.CSSProperties => {
    const hit = HANDLE
    switch (edge) {
      case 'n':
        return { top: 0, left: hit, right: hit, height: hit, cursor: 'ns-resize' }
      case 's':
        return { bottom: 0, left: hit, right: hit, height: hit, cursor: 'ns-resize' }
      case 'w':
        return { top: hit, bottom: hit, left: 0, width: hit, cursor: 'ew-resize' }
      case 'e':
        return { top: hit, bottom: hit, right: 0, width: hit, cursor: 'ew-resize' }
      case 'nw':
        return { top: 0, left: 0, width: hit + 2, height: hit + 2, cursor: 'nwse-resize' }
      case 'ne':
        return { top: 0, right: 0, width: hit + 2, height: hit + 2, cursor: 'nesw-resize' }
      case 'sw':
        return { bottom: 0, left: 0, width: hit + 2, height: hit + 2, cursor: 'nesw-resize' }
      case 'se':
        return { bottom: 0, right: 0, width: hit + 2, height: hit + 2, cursor: 'nwse-resize' }
      default:
        return {}
    }
  }

  const title = w.label?.trim() || w.chart_id || w.type

  const outerStyle: React.CSSProperties = {
    position: 'absolute',
    left: w.x,
    top: w.y,
    width: w.w,
    height: w.h,
    boxSizing: 'border-box',
    border: selected ? '2px solid #38bdf8' : '2px dashed #3fb950',
    transform: transform
      ? `translate3d(${transform.x}px,${transform.y}px,0)`
      : undefined,
    zIndex: isDragging ? 30 : 2,
    overflow: 'hidden',
    cursor: 'grab',
  }

  return (
    <div
      ref={setNodeRef}
      style={outerStyle}
      {...listeners}
      {...attributes}
      onClick={(e) => {
        e.stopPropagation()
        onSelect()
      }}
    >
      <div
        className={`absolute inset-0 z-0 pointer-events-none ${w.show_border ? 'ring-1 ring-inset ring-zinc-600' : ''}`}
      >
        <WidgetChartPreview w={w} fallbackNodeId={chartNodeId} />
      </div>
      {!w.hide_label && (
        <div className="absolute top-0 left-0 right-0 z-[4] pointer-events-none px-1 py-0.5 text-[10px] leading-tight text-zinc-200/95 bg-zinc-950/55 border-b border-zinc-700/40 truncate">
          {title}
        </div>
      )}
      {EDGES.map((edge) => (
        <div
          key={edge}
          className="absolute z-[5]"
          style={edgeStyle(edge)}
          onPointerDownCapture={(e) => {
            e.stopPropagation()
            onResizePointerDown(edge, e)
          }}
        />
      ))}
    </div>
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
    si: number
    wi: number
    type: string
    edge: string
    cx: number
    cy: number
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
      tw.x = snap(n.x)
      tw.y = snap(n.y)
      tw.w = snap(n.w)
      tw.h = snap(n.h)
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
      tw.w = side
      tw.h = side
      setLayout(next)
    }
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
    return () => {
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
    }
  }, [scale, setLayout])

  const onResizePointerDown = useCallback(
    (edge: string, e: React.PointerEvent, si: number, wi: number, wtype: string) => {
      const layoutNow = useAppStore.getState().layout
      if (!layoutNow) return
      const tw = layoutNow.scenes[si].widgets[wi]
      resizeRef.current = {
        si,
        wi,
        type: wtype,
        edge,
        cx: e.clientX,
        cy: e.clientY,
        orig: { x: tw.x, y: tw.y, w: tw.w, h: tw.h },
      }
    },
    [],
  )

  useEffect(() => {
    loadLayout().catch((e) => setStatus(String(e)))
    loadNodes().catch(() => {})
  }, [loadLayout, loadNodes, setStatus])

  useEffect(() => {
    const w = new Worker(new URL('../workers/chartSearch.worker.ts', import.meta.url), {
      type: 'module',
    })
    workerRef.current = w
    w.onmessage = (e: MessageEvent<{ type: string; indices?: number[]; n?: number }>) => {
      if (e.data.type === 'result' && e.data.indices) setFilteredIdx(e.data.indices)
    }
    return () => {
      w.terminate()
      workerRef.current = null
    }
  }, [])

  const reloadDiscovery = useCallback(async () => {
    try {
      const r = await api.getDiscovery(chartNodeId)
      setAllRows(r.charts)
      workerRef.current?.postMessage({ type: 'load', rows: r.charts })
      workerRef.current?.postMessage({ type: 'search', q: searchQ, max: 2000 })
    } catch (e) {
      setAllRows([])
      setFilteredIdx([])
      setStatus(String(e))
    }
  }, [chartNodeId, searchQ, setStatus])

  useEffect(() => {
    reloadDiscovery()
  }, [chartNodeId, reloadDiscovery])

  useEffect(() => {
    if (searchT.current) clearTimeout(searchT.current)
    searchT.current = setTimeout(() => {
      workerRef.current?.postMessage({ type: 'search', q: searchQ, max: 2000 })
    }, 120)
    return () => {
      if (searchT.current) clearTimeout(searchT.current)
    }
  }, [searchQ, allRows.length])

  const onDragEnd = (e: DragEndEvent) => {
    const id = String(e.active.id)
    if (!id.startsWith('w-')) return
    const [, a, b] = id.split('-')
    const si = parseInt(a, 10)
    const wi = parseInt(b, 10)
    const dx = e.delta.x / scale
    const dy = e.delta.y / scale
    if (!layout) return
    const next = structuredClone(layout)
    const ww = next.scenes[si].widgets[wi]
    ww.x = snap(ww.x + dx)
    ww.y = snap(ww.y + dy)
    setLayout(next)
  }

  const addWidget = (type: string) => {
    if (!layout) return
    const next = structuredClone(layout)
    const si = sceneIdx
    const w: Widget = {
      id: newWidgetId(),
      type,
      x: snap(64),
      y: snap(64),
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
    updateSelected({
      chart_id: row.id,
      data: {
        node: chartNodeId,
        chart: row.id,
        dim: '',
      },
    })
  }

  const selectedW =
    layout && sel ? layout.scenes[sel.si]?.widgets[sel.wi] : undefined

  const [listH, setListH] = useState(420)
  useEffect(() => {
    const fn = () => setListH(Math.max(200, window.innerHeight - 220))
    fn()
    window.addEventListener('resize', fn)
    return () => window.removeEventListener('resize', fn)
  }, [])

  const [dimOpts, setDimOpts] = useState<string[]>([])
  useEffect(() => {
    if (!selectedW?.chart_id) {
      setDimOpts([])
      return
    }
    api
      .getChartMeta(selectedW.chart_id, chartNodeId)
      .then((m) => setDimOpts(m.dimensions ?? []))
      .catch(() => setDimOpts([]))
  }, [selectedW?.chart_id, chartNodeId])

  const renderChartRow = ({ index, style }: ListChildComponentProps) => {
    const i = filteredIdx[index]
    const row = allRows[i]
    if (!row) return <div style={style} />
    return (
      <div
        style={style}
        className="px-2 text-xs truncate cursor-pointer hover:bg-zinc-800 border-b border-zinc-800/80"
        onClick={() => bindChart(row)}
        title={row.title}
      >
        <span className="text-zinc-500">{row.family}</span> · {row.id}
      </div>
    )
  }

  if (!layout) {
    return <div className="p-6 text-zinc-500">{t('editor.loadingLayout')}</div>
  }

  const sc = layout.scenes[sceneIdx]

  return (
    <DndContext sensors={sensors} onDragEnd={onDragEnd}>
      <div className="flex flex-col h-[calc(100vh-52px)] min-h-[480px]">
        <div className="flex items-center gap-2 px-3 py-2 border-b border-zinc-800 bg-zinc-900/80 shrink-0">
          <span className="text-sm text-zinc-500">{t('editor.scene')}</span>
          <select
            className="bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-sm"
            value={sceneIdx}
            onChange={(e) => {
              setSceneIdx(parseInt(e.target.value, 10))
              setSel(null)
            }}
          >
            {layout.scenes.map((s, i) => (
              <option key={s.id} value={i}>
                {s.name || s.id}
              </option>
            ))}
          </select>
          <button
            type="button"
            className="ml-4 px-3 py-1 rounded bg-emerald-700 text-sm"
            onClick={() => saveLayout().catch((e) => setStatus(String(e)))}
          >
            {t('editor.saveLayout')}
          </button>
          <label className="text-xs text-zinc-500 ml-4 flex items-center gap-1">
            {t('editor.scale')}
            <input
              type="range"
              min={0.35}
              max={1.75}
              step={0.05}
              value={scale}
              onChange={(e) => setScale(parseFloat(e.target.value))}
            />
          </label>
        </div>

        <div className="flex flex-1 min-h-0">
          {/* 资产浏览器 */}
          <aside className="w-72 border-r border-zinc-800 flex flex-col shrink-0 bg-zinc-950">
            <div className="p-2 border-b border-zinc-800 space-y-2">
              <div className="text-xs text-zinc-500">{t('editor.metricNodeHint')}</div>
              <select
                className="w-full bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-sm"
                value={chartNodeId}
                onChange={(e) => setChartNodeId(e.target.value)}
              >
                <option value="">{t('editor.defaultNodeShort')}</option>
                {nodes.map((n) => (
                  <option key={n.id} value={n.id}>
                    {n.name} ({n.id})
                  </option>
                ))}
              </select>
              <input
                className="w-full bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-sm"
                placeholder={t('editor.searchPlaceholder')}
                value={searchQ}
                onChange={(e) => setSearchQ(e.target.value)}
              />
              <div className="text-[10px] text-zinc-600">
                {t('editor.chartListStats', {
                  filtered: filteredIdx.length,
                  total: allRows.length,
                })}
              </div>
            </div>
            <div className="flex-1 min-h-0">
              {filteredIdx.length > 0 ? (
                <List
                  height={listH}
                  itemCount={filteredIdx.length}
                  itemSize={28}
                  width="100%"
                >
                  {renderChartRow}
                </List>
              ) : (
                <div className="p-3 text-xs text-zinc-600">{t('editor.noChartData')}</div>
              )}
            </div>
            <div className="p-2 border-t border-zinc-800 text-[10px] text-zinc-600">
              {t('editor.bindHint')}
            </div>
          </aside>

          {/* 画布 */}
          <div className="flex-1 overflow-auto p-4 flex justify-center items-start bg-zinc-900/40">
            <div
              className="relative shadow-xl border border-zinc-700 bg-[#010409]"
              style={{
                width: STAGE_W * scale,
                height: STAGE_H * scale,
              }}
            >
              <div
                className="origin-top-left"
                style={{
                  width: STAGE_W,
                  height: STAGE_H,
                  transform: `scale(${scale})`,
                }}
                onClick={() => setSel(null)}
              >
                {sc.widgets.map((w, wi) => (
                  <StageWidget
                    key={w.id ?? `${wi}`}
                    si={sceneIdx}
                    wi={wi}
                    w={w}
                    selected={sel?.si === sceneIdx && sel?.wi === wi}
                    chartNodeId={chartNodeId}
                    onSelect={() => setSel({ si: sceneIdx, wi })}
                    onResizePointerDown={(edge, ev) =>
                      onResizePointerDown(edge, ev, sceneIdx, wi, w.type)
                    }
                  />
                ))}
              </div>
            </div>
          </div>

          {/* 组件调色板 + 属性 */}
          <aside className="w-80 border-l border-zinc-800 flex flex-col shrink-0 overflow-auto bg-zinc-950">
            <div className="p-2 border-b border-zinc-800">
              <div className="text-xs text-zinc-500 mb-2">{t('editor.addToScene')}</div>
              <div className="flex flex-wrap gap-1">
                {['text', 'gauge', 'line', 'progress', 'histogram'].map((t) => (
                  <button
                    key={t}
                    type="button"
                    className="px-2 py-1 rounded bg-zinc-800 text-xs hover:bg-zinc-700"
                    onClick={() => addWidget(t)}
                  >
                    +{t}
                  </button>
                ))}
              </div>
            </div>
            <div className="p-3 space-y-2 text-sm">
              <h3 className="text-zinc-400 text-xs font-medium">{t('editor.props')}</h3>
              {!selectedW ? (
                <p className="text-zinc-600 text-xs">{t('editor.selectWidget')}</p>
              ) : (
                <>
                  <label className="block text-xs text-zinc-500">
                    node_id
                    <select
                      className="w-full mt-1 bg-zinc-900 border border-zinc-700 rounded px-2 py-1"
                      value={selectedW.node_id ?? ''}
                      onChange={(e) => updateSelected({ node_id: e.target.value || undefined })}
                    >
                      <option value="">{t('editor.nodeIdDefault')}</option>
                      {nodes.map((n) => (
                        <option key={n.id} value={n.id}>
                          {n.id}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="block text-xs text-zinc-500">
                    chart_id
                    <input
                      className="w-full mt-1 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 font-mono text-xs"
                      value={selectedW.chart_id ?? ''}
                      onChange={(e) => updateSelected({ chart_id: e.target.value })}
                    />
                  </label>
                  <label className="block text-xs text-zinc-500">
                    {t('editor.dimensionsHint')}
                    <input
                      className="w-full mt-1 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-xs"
                      value={(selectedW.dimensions ?? []).join(',')}
                      onChange={(e) =>
                        updateSelected({
                          dimensions: e.target.value
                            .split(',')
                            .map((s) => s.trim())
                            .filter(Boolean),
                        })
                      }
                    />
                  </label>
                  {dimOpts.length > 0 && (
                    <div className="text-xs space-y-1">
                      <div className="text-zinc-500">{t('editor.dimQuick')}</div>
                      <div className="flex flex-wrap gap-1">
                        {dimOpts.map((d) => (
                          <button
                            key={d}
                            type="button"
                            className="px-1.5 py-0.5 rounded bg-zinc-800 text-[10px]"
                            onClick={() => updateSelected({ dimensions: [d] })}
                          >
                            {d}
                          </button>
                        ))}
                      </div>
                    </div>
                  )}
                  <label className="block text-xs text-zinc-500">
                    {t('editor.xywh')}
                    <div className="grid grid-cols-4 gap-1 mt-1">
                      {(['x', 'y', 'w', 'h'] as const).map((k) => (
                        <input
                          key={k}
                          type="number"
                          className="bg-zinc-900 border border-zinc-700 rounded px-1 py-0.5 text-xs"
                          value={Math.round(selectedW[k] as number)}
                          onChange={(e) =>
                            updateSelected({ [k]: parseFloat(e.target.value) || 0 } as Partial<Widget>)
                          }
                        />
                      ))}
                    </div>
                  </label>
                  <label className="block text-xs text-zinc-500">
                    label
                    <input
                      className="w-full mt-1 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-xs"
                      value={selectedW.label ?? ''}
                      onChange={(e) => updateSelected({ label: e.target.value })}
                    />
                  </label>
                  <label className="flex items-center gap-2 text-xs text-zinc-500 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={!!selectedW.show_border}
                      onChange={(e) => updateSelected({ show_border: e.target.checked || undefined })}
                    />
                    {t('editor.showBorder')}
                  </label>
                  <label className="flex items-center gap-2 text-xs text-zinc-500 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={!!selectedW.hide_label}
                      onChange={(e) => updateSelected({ hide_label: e.target.checked || undefined })}
                    />
                    {t('editor.hideLabelBanner')}
                  </label>
                  {selectedW.type === 'line' && (
                    <>
                      <label className="flex items-center gap-2 text-xs text-zinc-500 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={!!selectedW.show_y_axis}
                          onChange={(e) => updateSelected({ show_y_axis: e.target.checked || undefined })}
                        />
                        {t('editor.showYAxis')}
                      </label>
                      <label className="block text-xs text-zinc-500">
                        {t('editor.linePoints')}
                        <input
                          type="number"
                          min={8}
                          max={512}
                          className="w-full mt-1 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-xs"
                          value={selectedW.line_points ?? ''}
                          placeholder="96"
                          onChange={(e) => {
                            const v = e.target.value
                            updateSelected({
                              line_points: v ? Math.min(512, Math.max(8, parseInt(v, 10) || 96)) : undefined,
                            })
                          }}
                        />
                      </label>
                    </>
                  )}
                  <label className="block text-xs text-zinc-500">
                    unit
                    <select
                      className="w-full mt-1 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-xs"
                      value={selectedW.unit ?? ''}
                      onChange={(e) => updateSelected({ unit: e.target.value })}
                    >
                      <option value="">auto</option>
                      <option value="percent">percent</option>
                      <option value="bytes">bytes</option>
                      <option value="none">none</option>
                    </select>
                  </label>
                  <label className="block text-xs text-zinc-500">
                    {t('editor.colorLabel')}
                    <input
                      className="w-full mt-1 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-xs"
                      value={selectedW.color ?? ''}
                      onChange={(e) => updateSelected({ color: e.target.value })}
                      placeholder="#3fb950"
                    />
                  </label>
                  <label className="block text-xs text-zinc-500">
                    {t('editor.fontRotate')}
                    <div className="flex gap-1 mt-1">
                      <input
                        type="number"
                        className="flex-1 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-xs"
                        value={selectedW.font_size ?? ''}
                        placeholder="auto"
                        onChange={(e) =>
                          updateSelected({
                            font_size: e.target.value ? parseFloat(e.target.value) : undefined,
                          })
                        }
                      />
                      <input
                        type="number"
                        className="flex-1 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-xs"
                        value={selectedW.rotation ?? ''}
                        placeholder="0"
                        onChange={(e) =>
                          updateSelected({
                            rotation: e.target.value ? parseFloat(e.target.value) : undefined,
                          })
                        }
                      />
                    </div>
                  </label>
                  <label className="block text-xs text-zinc-500">
                    {t('editor.thresholds')}
                    <div className="flex gap-1 mt-1">
                      <input
                        type="number"
                        className="flex-1 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-xs"
                        value={selectedW.warn_threshold ?? ''}
                        onChange={(e) =>
                          updateSelected({
                            warn_threshold: e.target.value ? parseFloat(e.target.value) : undefined,
                          })
                        }
                      />
                      <input
                        type="number"
                        className="flex-1 bg-zinc-900 border border-zinc-700 rounded px-2 py-1 text-xs"
                        value={selectedW.critical_threshold ?? ''}
                        onChange={(e) =>
                          updateSelected({
                            critical_threshold: e.target.value
                              ? parseFloat(e.target.value)
                              : undefined,
                          })
                        }
                      />
                    </div>
                  </label>
                  <button
                    type="button"
                    className="w-full py-1 rounded bg-red-900/50 text-xs mt-2"
                    onClick={() => {
                      if (!sel) return
                      const next = structuredClone(layout)
                      next.scenes[sel.si].widgets.splice(sel.wi, 1)
                      setLayout(next)
                      setSel(null)
                    }}
                  >
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
