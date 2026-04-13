import { create } from 'zustand'
import { api } from '../api'
import i18n from '../i18n'
import type { LayoutConfig, NetdataNode } from '../types'

type AppState = {
  layout: LayoutConfig | null
  nodes: NetdataNode[]
  status: string
  setStatus: (s: string) => void
  loadLayout: () => Promise<void>
  loadNodes: () => Promise<void>
  setLayout: (l: LayoutConfig) => void
  setNodes: (n: NetdataNode[]) => void
  saveLayout: () => Promise<void>
  saveNodes: () => Promise<void>
  saveSettings: () => Promise<void>
  /** 节点 ID 变更时，同步更新所有 widget 中引用该旧 ID 的 node_id 字段。 */
  renameNodeId: (oldId: string, newId: string) => void
}

export const useAppStore = create<AppState>((set, get) => ({
  layout: null,
  nodes: [],
  status: '',
  setStatus: (s) => set({ status: s }),
  loadLayout: async () => {
    const l = await api.getLayout()
    set({ layout: l })
  },
  loadNodes: async () => {
    const f = await api.getNodes()
    set({ nodes: f.nodes ?? [] })
  },
  setLayout: (l) => set({ layout: l }),
  setNodes: (n) => set({ nodes: n }),
  saveLayout: async () => {
    const { layout } = get()
    if (!layout) return
    await api.putLayout(layout)
    set({ status: i18n.t('status.layoutSaved') })
  },
  saveNodes: async () => {
    const { nodes } = get()
    await api.putNodes({ nodes })
    set({ status: i18n.t('status.nodesSaved') })
  },
  saveSettings: async () => {
    const { layout, nodes } = get()
    if (!layout) return
    // 直接保存顶层字段（screen_width/screen_height/layout_rotation），不再写冗余的 settings 块。
    await api.putLayout(layout)
    await api.putNodes({ nodes })
    set({ status: i18n.t('status.settingsSaved') })
  },
  renameNodeId: (oldId, newId) => {
    const { layout } = get()
    if (!layout || oldId === newId) return
    const next = structuredClone(layout)
    for (const scene of next.scenes) {
      for (const w of scene.widgets) {
        if (w.node_id === oldId) {
          w.node_id = newId || undefined
        }
      }
    }
    set({ layout: next })
  },
}))
