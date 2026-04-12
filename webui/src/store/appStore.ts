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
    if (!layout) {
      return
    }
    const toSave = {
      ...layout,
      settings: {
        width: layout.screen_width,
        height: layout.screen_height,
        rotation: layout.layout_rotation ?? 0,
      },
    }
    await api.putLayout(toSave)
    await api.putNodes({ nodes })
    set({ layout: toSave, status: i18n.t('status.settingsSaved') })
  },
}))
