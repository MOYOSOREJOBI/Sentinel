'use client'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import dynamic from 'next/dynamic'
import Link from 'next/link'
import AppShell, { useGlobalFilters } from '../../components/AppShell'
import TrustStrip from '../../components/TrustStrip'
import WorldRiskMap from '../../components/WorldRiskMap'
import { CandleRiskPanel } from '../../components/CandleRiskPanel'
import LoadingState from '../../components/LoadingState'
import DegradedState from '../../components/DegradedState'
import RealtimeStatus from '../../components/RealtimeStatus'
import QuantVisuals from '../../components/QuantVisuals'
import { api } from '../../lib/api'
import { useSSE } from '../../lib/useSSE'
import { deriveQuantMetrics } from '../../lib/quant'

const RiskGlobe = dynamic(() => import('../../components/RiskGlobe').then(m => m.RiskGlobe), { ssr: false })

export default function CommandCenterPage() {
  const { filters, apply } = useGlobalFilters()
  const [data, setData] = useState<any>(null)
  const [queue, setQueue] = useState<any[]>([])
  const [map, setMap] = useState<any>({ countries: [] })
  const [feedItems, setFeedItems] = useState<any[]>([])
  const [degraded, setDegraded] = useState(false)
  const [pollingHealthy, setPollingHealthy] = useState(true)
  const filtersKey = JSON.stringify(filters)

  const loadAll = useCallback(() => {
    return Promise.all([
      api.commandCenter(filters),
      api.worldMap(filters),
      api.queue(filters),
      api.feed(filters, 5),
    ])
      .then(([cc, wm, q, f]) => {
        setData(cc)
        setMap(wm)
        setQueue(Array.isArray(q) ? q : [])
        setFeedItems(f?.items ?? [])
        setDegraded(false)
        setPollingHealthy(true)
      })
      .catch(() => {
        setDegraded(true)
        setPollingHealthy(false)
        setData({ openIncidents: 0, highRiskCount: 0, backlogDelta: 0, trust: { state: 'degraded', fallback_mode: true } })
        setMap({ countries: [] })
        setQueue([])
        setFeedItems([])
      })
  }, [filtersKey]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    loadAll()
  }, [loadAll])

  const [events, setEvents] = useState<Array<{ id: string; at: string; text: string }>>([])
  const seen = useRef<Set<string>>(new Set())
  const lastPushAt = useRef(0)
  const { status: sseStatus } = useSSE<any>('/api/sse/command-center', (evt) => {
    const now = Date.now()
    // Prevent noisy UI churn from ultra-high-frequency updates.
    if (now-lastPushAt.current < 900) return
    const raw = JSON.stringify(evt || {})
    if (seen.current.has(raw)) return
    seen.current.add(raw)
    if (seen.current.size > 80) seen.current.clear()
    lastPushAt.current = now

    const id = String((evt as any)?.id || (evt as any)?.incident?.id || now)
    const text = eventSummary(evt)
    setData((d: any) => ({ ...(d || {}), ...evt }))
    setEvents((prev) => [{ id, at: new Date(now).toLocaleTimeString(), text }, ...prev].slice(0, 10))
  })

  useEffect(() => {
    if (sseStatus === 'open') {
      return
    }
    const iv = setInterval(() => {
      loadAll()
    }, 10_000)
    return () => clearInterval(iv)
  }, [loadAll, sseStatus])

  const globeData = useMemo(() => (map?.countries || []).map((c: any) => ({ iso2: (c.countryCode || c.country_code || 'XX').toUpperCase(), name: c.countryName || c.country_name || 'Unknown', risk: Number(c.avgCompositeRisk || c.avg_risk || 0), highRisk: Number(c.incidentCount || c.incident_count || 0), p95: Number(c.avgCompositeRisk || c.avg_risk || 0) })), [map])
  const quant = useMemo(() => deriveQuantMetrics(queue), [queue])

  if (!data) return <AppShell title="Command Center" subtitle="Real-time risk workspace" titleKey="titleCommandCenter" subtitleKey="subtitleCommandCenter" filters={filters} setFilters={apply}><div data-testid="command-center"><LoadingState /></div></AppShell>
  return <AppShell title="Command Center" subtitle="Real-time risk workspace" titleKey="titleCommandCenter" subtitleKey="subtitleCommandCenter" filters={filters} setFilters={apply}>
    <section data-testid="command-center">
    <TrustStrip trustState={data?.trust?.state} modelUnavailable={Boolean(data?.trust?.fallback_mode)} dqWarning={data?.trust?.dq_status} />
    <div className="kpi-grid">
      <div className="card kpi-card" data-testid="kpi-open-incidents"><h3>Open incidents</h3><p>{data.openIncidents || data.open_incidents || 0}</p><span className="muted">currently active</span></div>
      <div className="card kpi-card" data-testid="kpi-high-risk"><h3>High risk</h3><p>{data.highRiskCount || data.high_risk_count || 0}</p><span className="muted">severity high/critical</span></div>
      <div className="card kpi-card"><h3>Backlog delta</h3><p>{data.backlogDelta || 0}</p><span className="muted">high - open</span></div>
      <div className="card kpi-card"><h3>Avg confidence</h3><p>{(quant.avgConfidence * 100).toFixed(1)}%</p><span className="muted">from ranked queue</span></div>
    </div>

    <div className="analytics-grid">
      <CandleRiskPanel window={filters.window || '24h'} />
      <div className="card">
        <h3>Quantitative risk math</h3>
        <div className="kv"><span>Mean composite risk</span><strong>{quant.meanComposite.toFixed(2)}</strong></div>
        <div className="kv"><span>Tail risk (p95)</span><strong>{quant.p95Composite.toFixed(2)}</strong></div>
        <div className="kv"><span>Dispersion (std dev)</span><strong>{quant.volComposite.toFixed(2)}</strong></div>
        <div className="kv"><span>Expected loss proxy</span><strong>{quant.expectedLossProxy.toFixed(2)}</strong></div>
        <div className="kv"><span>Signal/noise ratio</span><strong>{quant.signalToNoise.toFixed(2)}</strong></div>
        <div className="muted">Formula: 0.28*priority + 0.22*composite + 0.18*escalation + 0.12*confidence + freshness - trust penalty.</div>
      </div>
    </div>

    <div className="analytics-grid">
      <RiskGlobe data={globeData} onSelect={(iso2) => apply({ countryCode: iso2 })} />
      <WorldRiskMap data={map.countries || []} selectedCountryCode={filters.countryCode} onSelectCountry={(c) => apply({ countryCode: c })} />
    </div>

    <QuantVisuals queue={queue} mapRows={map?.countries || []} />

    <div className='card'>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '0.5rem' }}>
        <h3 style={{ margin: 0 }}>Top risk feed</h3>
        <Link href="/feed" style={{ fontSize: '0.82rem', color: 'var(--accent, #3b82f6)' }}>View all →</Link>
      </div>
      {feedItems.length === 0 ? (
        <div className='empty-state'>Feed loading…</div>
      ) : (
        <table className='table'>
          <thead><tr><th>#</th><th>Symbol</th><th>Severity</th><th>Score</th><th>MIR</th></tr></thead>
          <tbody>
            {feedItems.map((it: any) => (
              <tr key={it.id}>
                <td>{it.rank}</td>
                <td style={{ fontWeight: 600 }}>{it.symbol}</td>
                <td>{it.severityBand}</td>
                <td>{Number(it.rankScore ?? 0).toFixed(3)}</td>
                <td>{Number(it.marketImpliedRisk ?? 0).toFixed(3)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>

    <div className='card'>
      <h3>Activity feed</h3>
      <RealtimeStatus sseStatus={sseStatus} polling={sseStatus !== 'open' && pollingHealthy} label="Stream" />
      {events.length === 0 ? (
        <div className='empty-state'>Waiting for command-center updates. Queue and charts stay usable while stream warms.</div>
      ) : (
        <table className='table'>
          <thead><tr><th>Time</th><th>Update</th></tr></thead>
          <tbody>
            {events.map((e, i) => <tr key={`${e.id}-${i}`}><td>{e.at}</td><td>{e.text}</td></tr>)}
          </tbody>
        </table>
      )}
    </div>
    {degraded ? <DegradedState message="Data links degraded. Verify query/alerts services and seed warmup." /> : null}
    </section>
  </AppShell>
}

function eventSummary(evt: any): string {
  if (!evt || typeof evt !== 'object') return 'live patch received'
  if (evt.type === 'degraded' || evt.type === 'degraded-heartbeat') return evt.message || 'upstream degraded'
  if (evt.incident?.id) {
    const risk = Number(evt.incident?.compositeRisk || evt.incident?.composite_risk || 0).toFixed(2)
    return `Incident ${evt.incident.id} updated; composite ${risk}`
  }
  const open = evt.openIncidents ?? evt.open_incidents
  const high = evt.highRiskCount ?? evt.high_risk_count
  if (open !== undefined || high !== undefined) {
    return `Open incidents ${open ?? 0}, high risk ${high ?? 0}`
  }
  if (evt.message) return String(evt.message)
  if (evt.symbol) return `Symbol update ${String(evt.symbol)}`
  return 'command-center patch'
}
