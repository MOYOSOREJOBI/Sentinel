'use client'
import { useCallback, useEffect, useMemo, useState } from 'react'
import AppShell, { useGlobalFilters } from '../../components/AppShell'
import LoadingState from '../../components/LoadingState'
import EmptyState from '../../components/EmptyState'
import DegradedState from '../../components/DegradedState'
import RealtimeStatus from '../../components/RealtimeStatus'
import QuantVisuals from '../../components/QuantVisuals'
import { api } from '../../lib/api'
import { deriveQuantMetrics } from '../../lib/quant'
import { useSSE } from '../../lib/useSSE'

export default function QueuePage() {
  const { filters, apply } = useGlobalFilters()
  const [rows, setRows] = useState<any[] | null>(null)
  const [feed, setFeed] = useState<any>({ items: [] })
  const [degraded, setDegraded] = useState(false)
  const [pollingHealthy, setPollingHealthy] = useState(true)
  const filtersKey = JSON.stringify(filters)

  const loadQueue = useCallback(() => {
    Promise.all([api.queue(filters), api.feed(filters, 100)])
      .then(([queueRows, riskFeed]) => {
        setRows(Array.isArray(queueRows) ? queueRows : [])
        setFeed(riskFeed || { items: [] })
        setDegraded(false)
        setPollingHealthy(true)
      })
      .catch(() => {
        setRows([])
        setFeed({ items: [] })
        setDegraded(true)
        setPollingHealthy(false)
      })
  }, [filtersKey]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    loadQueue()
  }, [loadQueue])

  const { status: sseStatus } = useSSE<any>('/api/sse/queue', (patch) => {
    setRows((prev) => {
      const arr = [...(prev || [])]
      const idx = arr.findIndex((r) => String(r.id) === String(patch.incident?.id))
      if (patch.type === 'delete' && idx >= 0) arr.splice(idx, 1)
      if (patch.type === 'upsert') {
        if (idx >= 0) arr[idx] = patch.incident
        else arr.push(patch.incident)
      }
      return arr
    })
  })

  useEffect(() => {
    if (sseStatus === 'open') {
      return
    }
    const iv = setInterval(() => loadQueue(), 10_000)
    return () => clearInterval(iv)
  }, [loadQueue, sseStatus])

  const sorted = useMemo(() => (rows || []).slice().sort((a, b) => (b.priorityScore || b.priority_score || 0) - (a.priorityScore || a.priority_score || 0)), [rows])
  const rankByIncidentID = useMemo(() => {
    const out = new Map<string, any>()
    for (const item of feed?.items || []) out.set(String(item.id), item)
    return out
  }, [feed])
  const feedRows = useMemo(() => {
    return (feed?.items || []).map((f: any) => ({
      id: f.id,
      symbol: f.symbol,
      countryIso2: f.countryIso2 || f.countryCode,
      countryCode: f.countryCode,
      countryName: f.countryName,
      region: f.region,
      sector: f.sector,
      industry: f.industry,
      venue: f.venue,
      priorityScore: f.priorityScore,
      compositeRisk: f.compositeRisk,
      confidence: f.confidence,
      severityBand: f.severityBand,
      recommendedAction: 'triage',
      rankReason: f.rankReason,
    }))
  }, [feed])
  const displayRows = sorted.length > 0 ? sorted : feedRows
  const usingFeedFallback = sorted.length === 0 && feedRows.length > 0
  const quant = useMemo(() => deriveQuantMetrics(displayRows), [displayRows])

  const mapRows = useMemo(() => {
    const by = new Map<string, number>()
    for (const r of displayRows) {
      const c = String(r.countryIso2 || r.countryCode || 'XX').toUpperCase()
      by.set(c, (by.get(c) || 0) + 1)
    }
    return Array.from(by.entries()).map(([countryIso2, incidentCount]) => ({ countryIso2, countryCode: countryIso2, incidentCount }))
  }, [displayRows])

  return <AppShell title="Queue" subtitle="Ranked incident queue" titleKey="titleQueue" subtitleKey="subtitleQueue" filters={filters} setFilters={apply}>
    <section data-testid="queue">
    {rows === null ? <LoadingState label="Loading queue" /> :
      <>
        <RealtimeStatus sseStatus={sseStatus} polling={sseStatus !== 'open' && pollingHealthy} />
        {sorted.length === 0 && !usingFeedFallback ? <EmptyState message="Queue is clear for current filters. Try Clear, broader time windows, or remove country/topic constraints." /> : null}
        {usingFeedFallback ? (
          <div className="card">
            <h3>Queue source</h3>
            <p className="muted">Primary queue returned no rows for this slice. Showing ranked feed candidates from live incidents to keep triage moving.</p>
          </div>
        ) : null}
        <div className="card">
          <h3>Queue quality snapshot</h3>
          <div className="kv"><span>Mean Priority</span><strong>{quant.meanPriority.toFixed(2)}</strong></div>
          <div className="kv"><span>Tail Composite (p95)</span><strong>{quant.p95Composite.toFixed(2)}</strong></div>
          <div className="kv"><span>Confidence-weighted Risk</span><strong>{quant.confidenceWeightedRisk.toFixed(2)}</strong></div>
        </div>
        <QuantVisuals queue={displayRows} mapRows={mapRows} />
        <div className="card">
          <table className="table">
            <thead>
              <tr>
                <th>ID</th>
                <th>Instrument</th>
                <th>Region</th>
                <th>Priority</th>
                <th>Composite</th>
                <th>Confidence</th>
                <th>Rank reason</th>
                <th>Recommended Action</th>
              </tr>
            </thead>
            <tbody>
              {displayRows.map((r) => {
                const id = String(r.id)
                const feedRow = rankByIncidentID.get(id)
                const priority = Number(r.priorityScore || r.priority_score || 0)
                const composite = Number(r.compositeRisk || r.composite_risk || 0)
                const confidence = Number(r.confidence || 0)
                return (
                  <tr key={id}>
                    <td>{id}</td>
                    <td>
                      <strong>{r.symbol || '-'}</strong>
                      <div className="muted">{r.severityBand || r.severity || 'elevated'}</div>
                    </td>
                    <td>{r.countryName ? `${r.countryName} (${r.countryIso2 || r.countryCode || '-'})` : (r.countryIso2 || r.countryCode || r.region || '-')}</td>
                    <td>{priority.toFixed(1)}</td>
                    <td>{composite.toFixed(1)}</td>
                    <td>{(confidence * 100).toFixed(1)}%</td>
                    <td className="muted">{r.rankReason || feedRow?.rankReason || 'ranked by priority/composite/escalation/confidence'}</td>
                    <td>{r.recommendedAction || r.recommended_action || 'watch'}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      </>}
    {degraded && <DegradedState />}
    </section>
  </AppShell>
}
