// Field names use snake_case to match the backend JSON serialization (Go struct tags).
// Renaming to camelCase would require changes across the full API layer.

export interface NetdataNode {
  id: string
  name: string
  host: string
  port: number
  api_key?: string
  /** 为 true 且 host 未写 scheme 时使用 https://（如反代 TLS 端口） */
  secure?: boolean
}

export interface NodesFile {
  nodes: NetdataNode[]
}

export interface WidgetDataRef {
  node: string
  chart: string
  dim: string
}

export interface Widget {
  id?: string
  type: string
  x: number
  y: number
  w: number
  h: number
  chart_id?: string
  dimensions?: string[]
  node_id?: string
  data?: WidgetDataRef
  label?: string
  unit?: string
  color?: string
  font_size?: number
  rotation?: number
  warn_threshold?: number
  critical_threshold?: number
  gauge_arc_degrees?: number
  vertical?: boolean
  line_points?: number
  /** DRM：外边框 */
  show_border?: boolean
  /** 折线：左侧纵轴刻度 */
  show_y_axis?: boolean
  /** 隐藏标题条（编排器画布与屏幕一致） */
  hide_label?: boolean
}

export interface Scene {
  id: string
  name?: string
  duration?: number
  widgets: Widget[]
}

export interface LayoutSettings {
  width: number
  height: number
  rotation: number
}

export interface LayoutConfig {
  version: number
  screen_width: number
  screen_height: number
  switch_interval_secs: number
  layout_rotation?: number
  settings?: LayoutSettings
  scenes: Scene[]
}

export interface ChartDiscoveryRow {
  id: string
  title: string
  family: string
  context: string
}

export interface ProbeResult {
  version: string
  chart_count: number
  ok: boolean
  error?: string
}
