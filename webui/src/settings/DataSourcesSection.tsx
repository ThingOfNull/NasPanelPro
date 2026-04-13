import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../api'
import type { NetdataNode, ProbeResult } from '../types'
import { useAppStore } from '../store/appStore'

function StatusIcon({ r, tOnline, tOffline }: { r?: ProbeResult; tOnline: string; tOffline: string }) {
  if (!r) {
    return <span className="text-zinc-500">—</span>
  }
  if (r.ok) {
    return (
      <span className="text-emerald-400" title={`${r.version} · ${r.chart_count} charts`}>
        ● {tOnline}
      </span>
    )
  }
  return (
    <span className="text-red-400" title={r.error}>
      ● {tOffline}
    </span>
  )
}

export default function DataSourcesSection() {
  const { t } = useTranslation()
  const { nodes, setNodes, loadNodes, setStatus, renameNodeId } = useAppStore()
  const [probe, setProbe] = useState<Record<string, ProbeResult>>({})
  const [busy, setBusy] = useState<string | null>(null)

  useEffect(() => {
    loadNodes().catch((e) => setStatus(String(e)))
  }, [loadNodes, setStatus])

  const update = (i: number, patch: Partial<NetdataNode>) => {
    // 节点 ID 变更时，同步更新 layout 中所有引用该旧 ID 的 widget。
    if ('id' in patch && patch.id !== nodes[i].id) {
      renameNodeId(nodes[i].id, patch.id ?? '')
    }
    const next = nodes.map((n, j) => (j === i ? { ...n, ...patch } : n))
    setNodes(next)
  }

  const add = () => {
    setNodes([
      ...nodes,
      {
        id: `n${Date.now()}`,
        name: t('dataSources.newNodeName'),
        host: '127.0.0.1',
        port: 19999,
        secure: false,
      },
    ])
  }

  const remove = (i: number) => {
    setNodes(nodes.filter((_, j) => j !== i))
  }

  const testOne = async (n: NetdataNode) => {
    setBusy(n.id)
    try {
      const r = await api.testNode(n.id, {
        host: n.host,
        port: n.port,
        api_key: n.api_key,
        secure: n.secure,
      })
      setProbe((p) => ({ ...p, [n.id]: r }))
    } catch (e) {
      setProbe((p) => ({
        ...p,
        [n.id]: { ok: false, version: '', chart_count: 0, error: String(e) },
      }))
    } finally {
      setBusy(null)
    }
  }

  return (
    <section className="space-y-4 border border-zinc-800 rounded-lg p-4">
      <div>
        <h2 className="text-lg font-semibold text-zinc-100">{t('dataSources.title')}</h2>
        <p className="text-sm text-zinc-500 mt-1">{t('dataSources.intro')}</p>
      </div>

      <div className="flex gap-2">
        <button
          type="button"
          onClick={add}
          className="px-3 py-1.5 rounded bg-emerald-700 hover:bg-emerald-600 text-sm"
        >
          {t('dataSources.addNode')}
        </button>
      </div>

      <div className="overflow-x-auto border border-zinc-800 rounded-lg">
        <table className="w-full text-sm">
          <thead className="bg-zinc-900 text-zinc-400">
            <tr>
              <th className="text-left p-2">{t('dataSources.colId')}</th>
              <th className="text-left p-2">{t('dataSources.colName')}</th>
              <th className="text-left p-2">{t('dataSources.colHost')}</th>
              <th className="text-left p-2">{t('dataSources.colPort')}</th>
              <th className="text-center p-2 w-20" title={t('dataSources.colTlsTitle')}>
                {t('dataSources.colTls')}
              </th>
              <th className="text-left p-2">{t('dataSources.colApiKey')}</th>
              <th className="text-left p-2">{t('dataSources.colStatus')}</th>
              <th className="p-2">{t('dataSources.colActions')}</th>
            </tr>
          </thead>
          <tbody>
            {nodes.map((n, i) => (
              <tr key={n.id || i} className="border-t border-zinc-800">
                <td className="p-2">
                  <input
                    className="w-full bg-zinc-900 border border-zinc-700 rounded px-2 py-1"
                    value={n.id}
                    onChange={(e) => update(i, { id: e.target.value })}
                  />
                </td>
                <td className="p-2">
                  <input
                    className="w-full bg-zinc-900 border border-zinc-700 rounded px-2 py-1"
                    value={n.name}
                    onChange={(e) => update(i, { name: e.target.value })}
                  />
                </td>
                <td className="p-2">
                  <input
                    className="w-full bg-zinc-900 border border-zinc-700 rounded px-2 py-1"
                    value={n.host}
                    onChange={(e) => update(i, { host: e.target.value })}
                  />
                </td>
                <td className="p-2 w-24">
                  <input
                    type="number"
                    className="w-full bg-zinc-900 border border-zinc-700 rounded px-2 py-1"
                    value={n.port || 0}
                    onChange={(e) => update(i, { port: parseInt(e.target.value, 10) || 0 })}
                  />
                </td>
                <td className="p-2 text-center">
                  <input
                    type="checkbox"
                    className="accent-sky-500"
                    checked={!!n.secure}
                    onChange={(e) => update(i, { secure: e.target.checked })}
                  />
                </td>
                <td className="p-2">
                  <input
                    className="w-full bg-zinc-900 border border-zinc-700 rounded px-2 py-1"
                    type="password"
                    autoComplete="off"
                    value={n.api_key ?? ''}
                    onChange={(e) => update(i, { api_key: e.target.value })}
                    placeholder={t('dataSources.apiKeyPlaceholder')}
                  />
                </td>
                <td className="p-2 whitespace-nowrap">
                  <StatusIcon
                    r={probe[n.id]}
                    tOnline={t('dataSources.online')}
                    tOffline={t('dataSources.offline')}
                  />
                </td>
                <td className="p-2 whitespace-nowrap space-x-1">
                  <button
                    type="button"
                    disabled={busy === n.id}
                    className="px-2 py-1 rounded bg-sky-800 hover:bg-sky-700 text-xs disabled:opacity-50"
                    onClick={() => testOne(n)}
                  >
                    {t('dataSources.test')}
                  </button>
                  <button
                    type="button"
                    className="px-2 py-1 rounded bg-red-900/60 hover:bg-red-800 text-xs"
                    onClick={() => remove(i)}
                  >
                    {t('dataSources.delete')}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}
