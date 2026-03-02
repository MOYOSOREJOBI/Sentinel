'use client'
import { useEffect, useMemo, useRef, useState } from 'react'
import AppShell, { useGlobalFilters } from '../../components/AppShell'
import { api } from '../../lib/api'
import { CandleRiskPanel } from '../../components/CandleRiskPanel'

// ── helpers ──────────────────────────────────────────────────────────────────

function sparkPath(vals: number[], w = 200, h = 48): string {
  if (vals.length === 0) return ''
  const max = Math.max(...vals, 0.001)
  return vals.map((v, i) => {
    const x = (i / Math.max(1, vals.length - 1)) * w
    const y = h - (v / max) * (h - 4)
    return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`
  }).join(' ')
}

function clamp(v: number, lo = 0, hi = 1) { return Math.min(hi, Math.max(lo, v)) }
function riskColor(r: number) {
  return r > 0.7 ? '#ef4444' : r > 0.4 ? '#f97316' : r > 0.15 ? '#eab308' : '#22c55e'
}

// ── sub-components ─────────────────────────────────────────────────────────

function RiskHistogram({ scores }: { scores: number[] }) {
  const buckets = 20
  const counts = Array(buckets).fill(0)
  scores.forEach(s => { const i = Math.min(buckets - 1, Math.floor(clamp(s) * buckets)); counts[i]++ })
  const maxC = Math.max(...counts, 1)
  const W = 360, H = 90
  const bw = W / buckets
  return (
    <div className='card' data-testid='viz-histogram'>
      <h3>Risk score distribution</h3>
      <svg viewBox={`0 0 ${W} ${H}`} style={{ width: '100%' }}>
        <rect width={W} height={H} fill='#081426' rx='4' />
        {counts.map((c, i) => {
          const bh = (c / maxC) * (H - 8)
          const x = i * bw + 1
          const y = H - bh
          const bucket = i / buckets
          return <rect key={i} x={x} y={y} width={bw - 2} height={bh} fill={riskColor(bucket)} opacity={0.8} rx='1' />
        })}
        <line x1={0} y1={H - 1} x2={W} y2={H - 1} stroke='rgba(155,181,219,0.2)' />
        {[0, 0.25, 0.5, 0.75, 1.0].map(t => (
          <text key={t} x={t * W} y={H - 2} fontSize='8' fill='rgba(155,181,219,0.5)' textAnchor='middle'>{t.toFixed(2)}</text>
        ))}
      </svg>
      <div className='muted' style={{ fontSize: 11 }}>{scores.length} score samples · composite risk [0–1]</div>
    </div>
  )
}

function CountryBars({ countries }: { countries: Array<{ name: string; iso2: string; risk: number; count: number }> }) {
  const top = countries.slice(0, 12)
  const maxR = Math.max(...top.map(c => c.risk), 0.001)
  return (
    <div className='card' data-testid='viz-country-bars'>
      <h3>Country risk concentration</h3>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
        {top.length === 0 && <div className='muted'>No country data yet</div>}
        {top.map(c => (
          <div key={c.iso2} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ width: 36, fontSize: 11, color: 'rgba(155,181,219,0.7)' }}>{c.iso2}</span>
            <div style={{ flex: 1, height: 10, background: '#0d1f38', borderRadius: 2 }}>
              <div style={{ height: '100%', width: `${(c.risk / maxR) * 100}%`, background: riskColor(c.risk), borderRadius: 2, transition: 'width 0.4s' }} />
            </div>
            <span style={{ width: 36, fontSize: 11, textAlign: 'right' }}>{c.risk.toFixed(2)}</span>
            <span style={{ width: 28, fontSize: 10, color: 'rgba(155,181,219,0.5)' }}>({c.count})</span>
          </div>
        ))}
      </div>
    </div>
  )
}

function IncidentVelocitySparkline({ queue }: { queue: any[] }) {
  // Derive velocity from creation timestamps bucketed by hour
  const now = Date.now()
  const buckets = 12
  const counts = Array(buckets).fill(0)
  queue.forEach(inc => {
    const t = inc.lastActivityAt || inc.last_activity_at || inc.createdAt || inc.created_at
    if (!t) return
    const age = (now - new Date(t).getTime()) / 3600_000 // hours ago
    const b = buckets - 1 - Math.floor(clamp(age / buckets, 0, 0.999) * buckets)
    if (b >= 0 && b < buckets) counts[b]++
  })
  const path = sparkPath(counts, 360, 56)
  return (
    <div className='card' data-testid='viz-velocity'>
      <h3>Incident velocity (last 12 hours)</h3>
      <svg viewBox={`0 0 360 56`} style={{ width: '100%' }}>
        <rect width={360} height={56} fill='#081426' rx='4' />
        {path && <path d={path} fill='none' stroke='#5bc4ff' strokeWidth='2' />}
        {path && <path d={path + ` L360,56 L0,56 Z`} fill='rgba(91,196,255,0.08)' />}
      </svg>
      <div className='muted' style={{ fontSize: 11 }}>incidents created per hour bucket · {queue.length} total</div>
    </div>
  )
}

// ── main page ─────────────────────────────────────────────────────────────

export default function VizPage() {
  const { filters } = useGlobalFilters()
  const [queue, setQueue] = useState<any[]>([])
  const [map, setMap] = useState<any>({ countries: [] })
  const [scores, setScores] = useState<number[]>([])

  useEffect(() => {
    api.queue(filters).then(q => setQueue(Array.isArray(q) ? q : [])).catch(() => {})
    api.worldMap(filters).then(m => setMap(m || { countries: [] })).catch(() => {})
    api.scores('', filters.window || '24h', 500).then(r => {
      const pts = (r?.series || []).map((s: any) => Number(s.score_norm ?? s.composite_risk ?? 0)).filter((v: number) => !isNaN(v))
      setScores(pts)
    }).catch(() => {})
  }, [JSON.stringify(filters)])

  const countries = useMemo(() => (map.countries || []).map((c: any) => ({
    name: c.countryName || c.country_name || c.name || 'Unknown',
    iso2: (c.countryCode || c.country_code || c.iso2 || 'XX').toUpperCase(),
    risk: Number(c.avgCompositeRisk || c.avg_risk || c.risk || 0),
    count: Number(c.incidentCount || c.incident_count || c.highRisk || 0),
  })).sort((a: any, b: any) => b.risk - a.risk), [map])

  return (
    <AppShell title="Visualization Gallery" subtitle="Real data from live pipeline" titleKey="titleViz" subtitleKey="subtitleViz">
      <section data-testid="viz-gallery">
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit,minmax(340px,1fr))', gap: '1rem' }}>
          <div style={{ gridColumn: '1 / -1' }}>
            <CandleRiskPanel window={filters.window || '24h'} />
          </div>
          <RiskHistogram scores={scores} />
          <CountryBars countries={countries} />
          <div style={{ gridColumn: '1 / -1' }}>
            <IncidentVelocitySparkline queue={queue} />
          </div>
        </div>
        <div className='muted' style={{ fontSize: 11, marginTop: '1rem' }}>
          All charts populated from live backend: /scores, /candles, /world-map, /queue. Updates every page load.
        </div>
      </section>
    </AppShell>
  )
}
