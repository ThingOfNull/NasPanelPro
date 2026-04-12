export type ChartRow = {
  id: string
  title: string
  family: string
  context: string
}

let rows: ChartRow[] = []

self.onmessage = (e: MessageEvent<{ type: string; rows?: ChartRow[]; q?: string; max?: number }>) => {
  const d = e.data
  if (d.type === 'load' && d.rows) {
    rows = d.rows
    self.postMessage({ type: 'loaded', n: rows.length })
    return
  }
  if (d.type === 'search') {
    const q = (d.q ?? '').toLowerCase().trim()
    const max = d.max && d.max > 0 ? d.max : 800
    const indices: number[] = []
    if (!q) {
      const n = Math.min(rows.length, max)
      for (let i = 0; i < n; i++) indices.push(i)
    } else {
      for (let i = 0; i < rows.length && indices.length < max; i++) {
        const r = rows[i]
        const hay = `${r.id}\t${r.title}\t${r.family}\t${r.context}`.toLowerCase()
        if (hay.includes(q)) indices.push(i)
      }
    }
    self.postMessage({ type: 'result', indices })
  }
}
