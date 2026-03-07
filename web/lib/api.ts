import { filtersToQuery, type GlobalFilters } from './filterState'

export const API_QUERY = process.env.NEXT_PUBLIC_API_QUERY || '/api/proxy/query'
export const API_ALERTS = process.env.NEXT_PUBLIC_API_ALERTS || '/api/proxy/alerts'
export const API_GATEWAY = process.env.NEXT_PUBLIC_API_GATEWAY || '/api/proxy/gateway-api'
export const API_GOVERNANCE = process.env.NEXT_PUBLIC_API_GOVERNANCE || '/api/proxy/governance'
export type Role = 'viewer' | 'analyst' | 'admin'

function readCSRFCookie() { if (typeof document === 'undefined') return ''; const m = document.cookie.split(';').map(v => v.trim()).find(v => v.startsWith('sentinel_csrf=')); return m ? decodeURIComponent(m.split('=')[1]) : '' }
async function apiFetch(base: string, path: string, init: RequestInit = {}) {
  const headers = new Headers(init.headers || {})
  if ((init.method || 'GET').toUpperCase() !== 'GET') { const csrf = readCSRFCookie(); if (csrf) headers.set('X-CSRF-Token', csrf); if (!headers.has('Content-Type')) headers.set('Content-Type', 'application/json') }
  const res = await fetch(`${base}${path}`, { credentials: 'include', cache: 'no-store', ...init, headers }); if (!res.ok) throw new Error(`${res.status}`)
  const ct = res.headers.get('content-type') || ''; return ct.includes('json') ? res.json() : {}
}

export const api = {
  me: async () => {
    const r = await fetch(`${API_GATEWAY}/me`, { credentials: 'include', cache: 'no-store' })
    if (!r.ok) {
      const msg = await r.text().catch(() => '')
      const e = new Error(`me() failed: ${r.status} ${msg}`)
      ;(e as any).status = r.status
      throw e
    }
    const body = await r.json()
    return { ...body, email: body?.email || body?.user || '' }
  },
  login: (Email: string, Password: string) => apiFetch(API_GATEWAY, '/auth/login', { method: 'POST', body: JSON.stringify({ Email, Password }) }),
  logout: () => apiFetch(API_GATEWAY, '/auth/logout', { method: 'POST' }).catch(() => ({})),
  commandCenter: (f: GlobalFilters = {}) => apiFetch(API_QUERY, `/command-center${filtersToQuery(f)}`).catch(() => ({ openIncidents: 0, highRiskCount: 0 })),
  feed: (f: GlobalFilters = {}, limit = 50, cursor = '', mode = '') => {
    const base = filtersToQuery({ ...f, limit })
    const extras: string[] = []
    if (cursor) extras.push(`cursor=${encodeURIComponent(cursor)}`)
    if (mode) extras.push(`mode=${encodeURIComponent(mode)}`)
    const sep = base ? '&' : '?'
    const url = `/feed${base}${extras.length ? sep + extras.join('&') : ''}`
    return apiFetch(API_QUERY, url).catch(() => ({ items: [], limit, next_cursor: '', history_limited: false, history: { available_days: 0 }, window: f.window || '24h' }))
  },
  searchSuggest: (q: string, limit = 12, k = '') => {
    const base = filtersToQuery({ q, limit })
    const path = `/search-suggest${base}${k ? `${base ? '&' : '?'}k=${encodeURIComponent(k)}` : ''}`
    return apiFetch(API_QUERY, path).catch(() => ({ items: [] }))
  },
  queue: (f: GlobalFilters = {}) => apiFetch(API_QUERY, `/queue${filtersToQuery(f)}`).catch(() => ([])),
  incident: (id: string | number) => apiFetch(API_QUERY, `/incident/${id}`).catch(() => ({ id })),
  incidentBrief: async (id: string | number, format: 'json' | 'md' = 'json') => {
    const res = await fetch(`${API_QUERY}/incident/${id}/brief?format=${format}`, { credentials: 'include', cache: 'no-store' })
    if (!res.ok) throw new Error(`${res.status}`)
    if (format === 'md') return res.text()
    return res.json()
  },
  incidentTransition: (id: string | number, command: string, owner = '') => apiFetch(API_ALERTS, `/incidents/${id}/transition`, { method: 'POST', body: JSON.stringify({ command, owner }) }),
  alertAck: (id: string | number) => apiFetch(API_ALERTS, `/alerts/${id}/ack`, { method: 'POST' }),
  notifyOnCall: (id: string | number) => apiFetch(API_ALERTS, `/incidents/${id}/notify-oncall`, { method: 'POST' }),
  trust: (f: GlobalFilters = {}) => apiFetch(API_QUERY, `/trust${filtersToQuery(f)}`).catch(() => ({})),
  forecast: () => apiFetch(API_QUERY, '/forecast').catch(() => ({})),
  notifications: () => apiFetch(API_ALERTS, '/notifications').catch(() => ([])),
  replay: (job: string, mode = '') => apiFetch(API_QUERY, `/replay/${job}${mode ? `?mode=${encodeURIComponent(mode)}` : ''}`).catch(() => ({ id: job, status: 'unknown' })),
  replaySummary: (job: string) => apiFetch(API_QUERY, `/replay/${job}/summary`).catch(() => ({ id: job, status: 'unknown', summary: {} })),
  replayBrief: async (job: string, format: 'json' | 'md' = 'json') => {
    const res = await fetch(`${API_QUERY}/replay/${job}/brief?format=${format}`, { credentials: 'include', cache: 'no-store' })
    if (!res.ok) throw new Error(`${res.status}`)
    if (format === 'md') return res.text()
    return res.json()
  },
  worldMap: (f: GlobalFilters = {}) => apiFetch(API_QUERY, `/world-map${filtersToQuery(f)}`).catch(() => ({ countries: [], geoEnriched: false, missingGeoCount: 0 })),
  governanceSummary: () => apiFetch(API_QUERY, '/governance/summary').catch(() => ({ modelLineage: [], replayJobs: [] })),
  governanceSnapshot: () => apiFetch(API_GOVERNANCE, '/governance/summary').catch(() => ({ modelLineage: [], modelDeployments: [], replayJobs: [], escalationPolicies: [] })),
  models: () => apiFetch(API_GOVERNANCE, '/models').catch(() => ([])),
  modelDeployments: () => apiFetch(API_GOVERNANCE, '/models/deployments').catch(() => ([])),
  replayJobs: () => apiFetch(API_GOVERNANCE, '/replay/jobs').catch(() => ([])),
  escalationPolicies: () => apiFetch(API_GOVERNANCE, '/escalation-policies').catch(() => ([])),
  deployModel: (model_name: string, version: string) => apiFetch(API_GOVERNANCE, '/models/deploy', { method: 'POST', body: JSON.stringify({ model_name, version }) }),
  startReplay: (payload: Record<string, unknown> = {}) => apiFetch(API_GOVERNANCE, '/replay/start', { method: 'POST', body: JSON.stringify(payload) }),
  saveEscalationPolicy: (payload: Record<string, unknown>) => apiFetch(API_GOVERNANCE, '/escalation-policies', { method: 'PUT', body: JSON.stringify(payload) }),
  createEscalationPolicy: (payload: Record<string, unknown>) => apiFetch(API_GOVERNANCE, '/escalation-policies', { method: 'POST', body: JSON.stringify(payload) }),
  deleteEscalationPolicy: (id: string) => apiFetch(API_GOVERNANCE, `/escalation-policies/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  currentOnCall: () => apiFetch(API_GOVERNANCE, '/current-oncall').catch(() => ({})),
  executiveSummary: (f: GlobalFilters = {}) => apiFetch(API_QUERY, `/executive-summary${filtersToQuery(f)}`).catch(() => ({})),
  case: (id: string | number) => apiFetch(API_QUERY, `/case/${id}`).catch(() => ({ id })),
  cases: () => apiFetch(API_QUERY, '/cases').catch(() => ([])),
  caseDetail: (id: string | number) => apiFetch(API_ALERTS, `/cases/${id}`).catch(() => ({ id, notes: [], evidence: [], actions: [] })),
  caseStatus: (id: string | number, status: string) => apiFetch(API_ALERTS, `/cases/${id}`, { method: 'PATCH', body: JSON.stringify({ status }) }),
  caseNote: (id: string | number, note: string) => apiFetch(API_ALERTS, `/cases/${id}/notes`, { method: 'POST', body: JSON.stringify({ note }) }),
  caseEvidence: (id: string | number, evidence_type: string, reference_id: string, metadata: any = {}) => apiFetch(API_ALERTS, `/cases/${id}/evidence`, { method: 'POST', body: JSON.stringify({ evidence_type, reference_id, metadata }) }),
  caseDisposition: (id: string | number, status: string, reason: string) => apiFetch(API_ALERTS, `/cases/${id}/disposition`, { method: 'POST', body: JSON.stringify({ status, reason }) }),
  promoteIncidentCase: (incidentId: string | number, reason: string) => apiFetch(API_ALERTS, `/incidents/${incidentId}/promote-case`, { method: 'POST', body: JSON.stringify({ reason }) }),
  scores: (symbol = '', window = '24h', maxPoints = 300) =>
    apiFetch(API_QUERY, `/scores?symbol=${encodeURIComponent(symbol)}&window=${window}&maxPoints=${maxPoints}`),
  triggerBackfill: (years = 20) => apiFetch(API_QUERY, `/dev/backfill?years=${years}`, { method: 'POST' }),
  candles: (symbol = '', window = '24h', maxPoints = 300, res = '1m') =>
    apiFetch(API_QUERY, `/candles?symbol=${encodeURIComponent(symbol)}&window=${window}&maxPoints=${maxPoints}&res=${encodeURIComponent(res)}`),
  updateLocalePreference: (locale: string) => apiFetch(API_GATEWAY, '/me/locale', { method: 'PUT', body: JSON.stringify({ locale }) }),
}

export const getCommandCenter = api.commandCenter
export const getQueue = api.queue
export const getIncident = api.incident
export const getTrust = api.trust
export const getReplay = api.replay
export const getWorldMap = api.worldMap
export function normalizeAlertCreatedAt(v: any): string { if (!v) return ''; if (typeof v === 'object') return String(v.created_at || v.ts || ''); return String(v) }
export type { GlobalFilters } from './filterState'
