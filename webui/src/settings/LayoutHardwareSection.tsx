import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api'
import i18n from '../i18n'
import { useAppStore } from '../store/appStore'

export default function LayoutHardwareSection() {
  const { t } = useTranslation()
  const { layout, setLayout, loadLayout, setStatus } = useAppStore()
  const [lines, setLines] = useState<string[]>([])

  useEffect(() => {
    loadLayout().catch((e) => setStatus(String(e)))
  }, [loadLayout, setStatus])

  const refreshLogs = () => {
    api
      .getLogs(400)
      .then((r) => setLines(r.lines))
      .catch(() => setLines([i18n.t('logs.readFailed')]))
  }

  useEffect(() => {
    refreshLogs()
    const id = window.setInterval(refreshLogs, 5000)
    return () => window.clearInterval(id)
  }, [])

  if (!layout) {
    return <div className="p-6 text-zinc-500">{t('layoutHw.loading')}</div>
  }

  return (
    <>
      <section className="space-y-3 border border-zinc-800 rounded-lg p-4">
        <h2 className="text-sm font-medium text-zinc-400">{t('layoutHw.canvasTitle')}</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <label className="text-xs text-zinc-500">
            {t('layoutHw.width')}
            <input
              type="number"
              className="block w-full mt-1 bg-zinc-900 border border-zinc-700 rounded px-2 py-1"
              value={layout.screen_width}
              onChange={(e) =>
                setLayout({ ...layout, screen_width: parseInt(e.target.value, 10) || 1280 })
              }
            />
          </label>
          <label className="text-xs text-zinc-500">
            {t('layoutHw.height')}
            <input
              type="number"
              className="block w-full mt-1 bg-zinc-900 border border-zinc-700 rounded px-2 py-1"
              value={layout.screen_height}
              onChange={(e) =>
                setLayout({ ...layout, screen_height: parseInt(e.target.value, 10) || 480 })
              }
            />
          </label>
          <label className="text-xs text-zinc-500">
            {t('layoutHw.rotation')}
            <input
              type="number"
              className="block w-full mt-1 bg-zinc-900 border border-zinc-700 rounded px-2 py-1"
              value={layout.layout_rotation ?? 0}
              onChange={(e) =>
                setLayout({ ...layout, layout_rotation: parseInt(e.target.value, 10) || 0 })
              }
            />
          </label>
          <label className="text-xs text-zinc-500">
            {t('layoutHw.switchInterval')}
            <input
              type="number"
              step="0.5"
              className="block w-full mt-1 bg-zinc-900 border border-zinc-700 rounded px-2 py-1"
              value={layout.switch_interval_secs}
              onChange={(e) =>
                setLayout({
                  ...layout,
                  switch_interval_secs: parseFloat(e.target.value) || 15,
                })
              }
            />
          </label>
        </div>
      </section>

      <section className="space-y-2 border border-zinc-800 rounded-lg p-4">
        <h2 className="text-sm font-medium text-zinc-400">{t('layoutHw.sceneDurationTitle')}</h2>
        <p className="text-xs text-zinc-600">{t('layoutHw.sceneDurationHint')}</p>
        {layout.scenes.map((sc, i) => (
          <div key={sc.id} className="flex items-center gap-3 text-sm">
            <span className="text-zinc-500 w-40 truncate">{sc.id}</span>
            <input
              type="number"
              min={0}
              step="0.5"
              className="w-28 bg-zinc-900 border border-zinc-700 rounded px-2 py-1"
              value={sc.duration ?? ''}
              placeholder={t('layoutHw.durationPlaceholder')}
              onChange={(e) => {
                const v = e.target.value
                const next = structuredClone(layout)
                if (v === '') {
                  delete next.scenes[i].duration
                } else {
                  const n = parseFloat(v)
                  if (Number.isFinite(n) && n > 0) {
                    next.scenes[i].duration = n
                  } else {
                    delete next.scenes[i].duration
                  }
                }
                setLayout(next)
              }}
            />
          </div>
        ))}
      </section>

      <section className="space-y-2 border border-zinc-800 rounded-lg p-4">
        <div className="flex justify-between items-center">
          <h2 className="text-sm font-medium text-zinc-400">{t('layoutHw.logsTitle')}</h2>
          <button
            type="button"
            className="text-xs px-2 py-1 rounded bg-zinc-800"
            onClick={refreshLogs}
          >
            {t('layoutHw.logsRefresh')}
          </button>
        </div>
        <pre className="text-xs bg-black/50 border border-zinc-800 rounded p-3 h-64 overflow-auto font-mono text-zinc-400">
          {lines.join('\n')}
        </pre>
      </section>
    </>
  )
}
