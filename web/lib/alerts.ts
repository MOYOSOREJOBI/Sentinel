export type AlertStatus = 'open' | 'ack' | 'resolved' | 'suppressed' | string

export type AlertLike = {
  id: number
  symbol: string
  status: AlertStatus
  severity?: string
  explanation?: string
  score?: number
  ts?: string
}

export function filterAlerts(alerts: AlertLike[], filters: { severity: string; status: string; symbol: string }): AlertLike[] {
  const symbol = filters.symbol.trim().toLowerCase()
  return alerts.filter((alert) => {
    const severityMatch = filters.severity === 'all' || (alert.severity || 'unknown') === filters.severity
    const statusMatch = filters.status === 'all' || alert.status === filters.status
    const symbolMatch = !symbol || alert.symbol.toLowerCase().includes(symbol)
    return severityMatch && statusMatch && symbolMatch
  })
}

export function groupAlerts(alerts: AlertLike[], mode: 'status' | 'symbol'): Record<string, AlertLike[]> {
  return alerts.reduce<Record<string, AlertLike[]>>((acc, alert) => {
    const key = mode === 'symbol' ? alert.symbol : alert.status
    if (!acc[key]) acc[key] = []
    acc[key].push(alert)
    return acc
  }, {})
}
