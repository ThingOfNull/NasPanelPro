/** 兼容无 crypto.randomUUID 的环境（如部分 HTTP 内网页）。 */
export function newWidgetId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `w-${Date.now()}-${Math.random().toString(36).slice(2, 11)}`
}
