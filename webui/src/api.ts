import type { LayoutConfig, NodesFile, ProbeResult, ChartDiscoveryRow } from './types'

async function parse<T>(r: Response | Promise<Response>): Promise<T> {
  const res = await r
  if (!res.ok) {
    const t = await res.text()
    throw new Error(t || res.statusText)
  }
  return res.json() as Promise<T>
}

export const api = {
  getLayout: () => parse<LayoutConfig>(fetch('/api/layout')),
  putLayout: (body: LayoutConfig) =>
    parse<{ ok: boolean }>(
      fetch('/api/layout', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      }),
    ),
  getNodes: () => parse<NodesFile>(fetch('/api/nodes')),
  putNodes: (body: NodesFile) =>
    parse<{ ok: boolean }>(
      fetch('/api/nodes', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      }),
    ),
  testNode: (
    id: string,
    draft?: { host: string; port: number; api_key?: string; secure?: boolean },
  ) =>
    parse<ProbeResult>(
      fetch(`/api/nodes/${encodeURIComponent(id)}/test`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(draft ?? {}),
      }),
    ),
  getLogs: (limit = 500) =>
    parse<{ lines: string[] }>(fetch(`/api/logs?limit=${limit}`)),
  getDiscovery: (nodeId: string) =>
    parse<{ charts: ChartDiscoveryRow[] }>(
      fetch(`/api/netdata/discovery?node_id=${encodeURIComponent(nodeId)}`),
    ),
  getChartMeta: (chartId: string, nodeId: string) =>
    parse<{
      id: string
      title: string
      dimensions: string[]
    }>(
      fetch(
        `/api/netdata/chart/${encodeURIComponent(chartId)}?node_id=${encodeURIComponent(nodeId)}`,
      ),
    ),
  getChartSeries: (chart: string, nodeId: string, points = 72, after = '-120') =>
    parse<{
      labels: string[]
      series: Record<string, number[]>
      latest: Record<string, number>
    }>(
      fetch(
        `/api/netdata/data?chart=${encodeURIComponent(chart)}&node_id=${encodeURIComponent(nodeId)}&points=${points}&after=${encodeURIComponent(after)}`,
      ),
    ),
}
