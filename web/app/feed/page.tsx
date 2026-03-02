'use client'
import { useCallback, useEffect, useRef, useState } from 'react'
import AppShell, { useGlobalFilters } from '../../components/AppShell'
import RealtimeStatus from '../../components/RealtimeStatus'
import { api } from '../../lib/api'
import { useSSE } from '../../lib/useSSE'

type FeedMode = 'trending' | 'for_you'

const MODES: { value: FeedMode; label: string }[] = [
  { value: 'for_you', label: 'For You' },
  { value: 'trending', label: 'Trending' },
]

const SEV_COLORS: Record<string, string> = {
  critical: '#ef4444',
  high: '#f97316',
  elevated: '#eab308',
}

export default function FeedPage() {
  const { filters, apply } = useGlobalFilters()
  const [mode, setMode] = useState<FeedMode>('for_you')
  const [items, setItems] = useState<any[]>([])
  const [cursor, setCursor] = useState('')
  const [hasMore, setHasMore] = useState(false)
  const [historyMeta, setHistoryMeta] = useState<any>(null)
  const [loading, setLoading] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [pollingHealthy, setPollingHealthy] = useState(true)
  const [notifications, setNotifications] = useState<any[]>([])
  const [forecast, setForecast] = useState<any>(null)
  const sentinel = useRef<HTMLDivElement>(null)
  const filtersKey = JSON.stringify(filters)

  const loadPage = useCallback(async (nextCursor: string, replace: boolean) => {
    if (!nextCursor) setLoading(true)
    else setLoadingMore(true)
    try {
      const res = await api.feed(filters, 50, nextCursor, mode)
      const newItems: any[] = res?.items ?? []
      setItems(prev => replace ? newItems : [...prev, ...newItems])
      setCursor(String(res?.next_cursor || ''))
      setHasMore(Boolean(res?.next_cursor))
      setHistoryMeta({
        limited: Boolean(res?.history_limited),
        history: res?.history || null,
        backfillPath: res?.dev_backfill_path || '',
      })
      setPollingHealthy(true)
    } catch {
      setPollingHealthy(false)
    } finally {
      setLoading(false)
      setLoadingMore(false)
    }
  }, [filtersKey, mode]) // eslint-disable-line react-hooks/exhaustive-deps

  // Reset on filter/mode change
  useEffect(() => {
    setItems([])
    setCursor('')
    setHasMore(false)
    loadPage('', true)
    api.notifications().then(setNotifications).catch(() => setNotifications([]))
    api.forecast().then(setForecast).catch(() => setForecast(null))
  }, [filtersKey, mode]) // eslint-disable-line react-hooks/exhaustive-deps

  // SSE: reload feed when the queue emits new incidents (debounced 3s)
  const reloadTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const { status: sseStatus } = useSSE('/api/sse/command-center', () => {
    if (reloadTimer.current) clearTimeout(reloadTimer.current)
    reloadTimer.current = setTimeout(() => {
      setItems([])
      setCursor('')
      setHasMore(false)
      loadPage('', true)
    }, 3000)
  })

  // Polling fallback: refresh every 10s when SSE is degraded
  useEffect(() => {
    if (sseStatus === 'open') return
    const iv = setInterval(() => loadPage('', true), 10_000)
    return () => clearInterval(iv)
  }, [sseStatus, loadPage])

  // Infinite scroll observer
  useEffect(() => {
    if (!sentinel.current) return
    const obs = new IntersectionObserver((entries) => {
      if (entries[0]?.isIntersecting && hasMore && !loadingMore && !loading) {
        loadPage(cursor, false)
      }
    }, { rootMargin: '200px' })
    obs.observe(sentinel.current)
    return () => obs.disconnect()
  }, [hasMore, loadingMore, loading, cursor, loadPage])

  const activeFilters = [
    filters.q ? `Query: ${filters.q}` : '',
    filters.symbol ? `Symbol: ${filters.symbol}` : '',
    filters.countryCode ? `Country: ${filters.countryCode}` : '',
    filters.region ? `Region: ${filters.region}` : '',
    filters.sector ? `Sector: ${filters.sector}` : '',
    filters.industry ? `Industry: ${filters.industry}` : '',
    filters.venue ? `Venue: ${filters.venue}` : '',
    filters.window === 'custom'
      ? (filters.start && filters.end ? `Range: ${new Date(filters.start).toLocaleDateString()} to ${new Date(filters.end).toLocaleDateString()}` : 'Range: custom')
      : `Window: ${filters.window || '24h'}`,
  ].filter(Boolean)

  async function runBackfill() {
    try {
      await api.triggerBackfill(20)
      await loadPage('', true)
    } catch {}
  }

  function shareBrief() {
    const lines = items.slice(0, 10).map((it: any) =>
      `[${String(it.rank).padStart(2, '0')}] ${it.symbol} | ${String(it.severity ?? it.severityBand ?? '-').toUpperCase()} | score ${Number(it.rank_score ?? it.rankScore ?? 0).toFixed(3)} | ${(it.topDrivers ?? []).join(', ') || 'no drivers'}`
    )
    const text = `Sentinel Risk Feed — ${new Date().toISOString()}\nMode: ${mode || 'default'}\n\n${lines.join('\n')}`
    navigator.clipboard.writeText(text).catch(() => {})
  }

  return (
    <AppShell title="Risk Feed" subtitle="Ranked real-time risk stream" titleKey="titleFeed" subtitleKey="subtitleFeed" filters={filters} setFilters={apply}>
      <section data-testid="feed">
        <div style={{ display: 'flex', alignItems: 'center', gap: '1rem', marginBottom: '1rem', flexWrap: 'wrap' }}>
          <div style={{ display: 'flex', gap: '0.5rem' }}>
            {MODES.map(m => (
              <button
                key={m.value}
                data-testid={`feed-mode-${m.value || 'default'}`}
                onClick={() => setMode(m.value)}
                style={{
                  padding: '0.25rem 0.75rem',
                  borderRadius: '1rem',
                  border: '1px solid var(--border, #334)',
                  background: mode === m.value ? 'var(--accent, #3b82f6)' : 'transparent',
                  color: mode === m.value ? '#fff' : 'inherit',
                  cursor: 'pointer',
                  fontSize: '0.85rem',
                }}
              >
                {m.label}
              </button>
            ))}
          </div>
          <button
            data-testid="feed-share-brief"
            onClick={shareBrief}
            style={{
              marginLeft: 'auto',
              padding: '0.25rem 0.75rem',
              borderRadius: '0.5rem',
              border: '1px solid var(--border, #334)',
              background: 'transparent',
              cursor: 'pointer',
              fontSize: '0.85rem',
            }}
          >
            Share brief
          </button>
          <RealtimeStatus sseStatus={sseStatus} polling={sseStatus !== 'open' && pollingHealthy} label="Feed" />
        </div>

        {activeFilters.length > 0 && (
          <div data-testid="feed-filter-chips" style={{ display: 'flex', flexWrap: 'wrap', gap: '0.4rem', marginBottom: '0.75rem' }}>
            {activeFilters.map((chip) => (
              <span
                key={chip}
                style={{
                  border: '1px solid var(--border, #334)',
                  borderRadius: '999px',
                  padding: '0.15rem 0.55rem',
                  fontSize: '0.75rem',
                  color: 'var(--muted)',
                }}
              >
                {chip}
              </span>
            ))}
          </div>
        )}

        {historyMeta?.limited && (
          <div
            data-testid="feed-history-banner"
            style={{
              marginBottom: '0.75rem',
              border: '1px solid rgba(245, 158, 11, 0.35)',
              background: 'rgba(245, 158, 11, 0.08)',
              borderRadius: '0.5rem',
              padding: '0.75rem 1rem',
              display: 'flex',
              alignItems: 'center',
              gap: '0.75rem',
              flexWrap: 'wrap',
            }}
          >
            <span>
              History limited to {Number(historyMeta?.history?.available_days || 0)} days in this environment. Use backfill to generate more.
            </span>
            <button
              type="button"
              data-testid="feed-backfill"
              onClick={runBackfill}
              style={{
                padding: '0.25rem 0.75rem',
                borderRadius: '0.5rem',
                border: '1px solid var(--border, #334)',
                background: 'transparent',
                cursor: 'pointer',
              }}
            >
              Backfill 20 years
            </button>
          </div>
        )}

        {loading && <div style={{ textAlign: 'center', padding: '2rem', color: 'var(--muted)' }}>Loading feed…</div>}

        {!loading && items.length === 0 && (
          <div style={{ textAlign: 'center', padding: '2rem', color: 'var(--muted)' }}>No incidents in feed for current filters.</div>
        )}

        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
          {items.map((item: any) => (
            <FeedCard key={`${item.id}-${item.rank}`} item={item} />
          ))}
        </div>

        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit,minmax(18rem,1fr))', gap: '0.75rem', marginTop: '1rem' }}>
          <section
            style={{
              border: '1px solid var(--border, #334)',
              borderRadius: '0.5rem',
              padding: '0.75rem 1rem',
              background: 'var(--card-bg, #1a1a2e)',
            }}
          >
            <div style={{ fontWeight: 700, marginBottom: '0.35rem' }}>Comms</div>
            <div style={{ fontSize: '0.8rem', color: 'var(--muted)', marginBottom: '0.5rem' }}>Recent status updates and notification stubs.</div>
            {(notifications || []).slice(0, 5).map((n: any) => (
              <div key={n.id} style={{ padding: '0.35rem 0', borderTop: '1px solid rgba(255,255,255,0.04)' }}>
                <div style={{ fontWeight: 600 }}>Incident #{n.incidentId} · {n.status}</div>
                <div style={{ fontSize: '0.75rem', color: 'var(--muted)' }}>{n.target} via {n.channel}</div>
              </div>
            ))}
            {(!notifications || notifications.length === 0) && <div style={{ fontSize: '0.8rem', color: 'var(--muted)' }}>No comms events yet.</div>}
          </section>

          <section
            style={{
              border: '1px solid var(--border, #334)',
              borderRadius: '0.5rem',
              padding: '0.75rem 1rem',
              background: 'var(--card-bg, #1a1a2e)',
            }}
          >
            <div style={{ fontWeight: 700, marginBottom: '0.35rem' }}>Next 24h forecast</div>
            {forecast?.incidentVolumeBand ? (
              <>
                <div style={{ fontSize: '0.85rem' }}>
                  Volume band: {forecast.incidentVolumeBand.low} / {forecast.incidentVolumeBand.base} / {forecast.incidentVolumeBand.high}
                </div>
                <div style={{ fontSize: '0.75rem', color: 'var(--muted)', marginTop: '0.35rem' }}>
                  Watchlist: {(forecast.watchlistRisks || []).join(' · ') || 'none'}
                </div>
                <div style={{ fontSize: '0.75rem', color: 'var(--muted)', marginTop: '0.35rem' }}>
                  {(forecast.whyMoved || []).join(' · ')}
                </div>
              </>
            ) : <div style={{ fontSize: '0.8rem', color: 'var(--muted)' }}>Forecast unavailable.</div>}
          </section>
        </div>

        {loadingMore && <div style={{ textAlign: 'center', padding: '1rem', color: 'var(--muted)' }}>Loading more…</div>}

        {/* sentinel element for IntersectionObserver */}
        <div ref={sentinel} style={{ height: '1px' }} />

        {!hasMore && items.length > 0 && (
          <div style={{ textAlign: 'center', padding: '1rem', color: 'var(--muted)', fontSize: '0.8rem' }}>
            All {items.length} items loaded
          </div>
        )}
      </section>
    </AppShell>
  )
}

function FeedCard({ item }: { item: any }) {
  const sev = String(item.severity ?? item.severityBand ?? '').toLowerCase()
  const color = SEV_COLORS[sev] ?? '#6b7280'
  const score = Number(item.rank_score ?? item.rankScore ?? 0)
  const composite = Number(item.composite ?? item.compositeRisk ?? 0)
  const mir = Number(item.marketImpliedRisk ?? 0)
  const drivers: string[] = item.topDrivers ?? []
  const stamp = item.timestamp || item.lastActivityAt ? new Date(item.timestamp || item.lastActivityAt).toLocaleString() : 'unknown'

  return (
    <div
      data-testid="feed-item"
      style={{
        border: `1px solid var(--border, #334)`,
        borderLeft: `4px solid ${color}`,
        borderRadius: '0.5rem',
        padding: '0.75rem 1rem',
        background: 'var(--card-bg, #1a1a2e)',
        display: 'grid',
        gridTemplateColumns: '2.5rem 1fr auto',
        gap: '0.75rem',
        alignItems: 'start',
      }}
    >
      <div style={{ fontWeight: 700, color: 'var(--muted)', fontSize: '0.85rem', paddingTop: '0.2rem' }}>
        #{item.rank}
      </div>
      <div>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: '0.5rem', flexWrap: 'wrap' }}>
          <span style={{ fontWeight: 700, fontSize: '1rem' }}>{item.symbol || 'Market risk event'}</span>
          <span style={{
            fontSize: '0.72rem',
            fontWeight: 600,
            color,
            textTransform: 'uppercase',
            letterSpacing: '0.05em',
          }}>{item.severity ?? item.severityBand}</span>
          {item.status === 'ack' && <span style={{ fontSize: '0.7rem', color: 'var(--muted)' }}>ACK</span>}
        </div>
        <div style={{ fontSize: '0.76rem', color: 'var(--muted)', marginTop: '0.15rem' }}>
          Composite {composite.toFixed(3)} · {stamp}
        </div>
        {drivers.length > 0 && (
          <div style={{ fontSize: '0.8rem', color: 'var(--muted)', marginTop: '0.2rem' }}>
            {drivers.join(' · ')}
          </div>
        )}
        <div style={{ fontSize: '0.72rem', color: 'var(--muted)', marginTop: '0.15rem', fontFamily: 'monospace' }}>
          {Array.isArray(item.rank_reason || item.rankReason) ? (item.rank_reason || item.rankReason).join(' · ') : String(item.rank_reason || item.rankReason || '')}
        </div>
      </div>
      <div style={{ textAlign: 'right', minWidth: '6rem' }}>
        <div style={{ fontSize: '1.1rem', fontWeight: 700 }}>{score.toFixed(3)}</div>
        <div style={{ fontSize: '0.72rem', color: 'var(--muted)' }}>rank score</div>
        <div style={{ fontSize: '0.8rem', marginTop: '0.25rem' }}>MIR {mir.toFixed(3)}</div>
        <div style={{ fontSize: '0.7rem', color: 'var(--muted)' }}>market implied</div>
      </div>
    </div>
  )
}
