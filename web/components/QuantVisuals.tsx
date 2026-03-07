'use client'

import { useMemo } from 'react'

export function normalizeRiskValue(v: number) {
  if (Number.isNaN(v) || v < 0) return 0
  if (v > 1) return Math.min(v / 100, 1)
  return v
}

export function bucketIndex(v: number) {
  const normalized = normalizeRiskValue(v)
  if (normalized < 0.2) return 0
  if (normalized < 0.4) return 1
  if (normalized < 0.6) return 2
  if (normalized < 0.8) return 3
  return 4
}

export function summarizeQuantVisuals(queue: any[], mapRows: any[], rollups?: any) {
  const severity = (() => {
    const seeded = rollups?.severityBreakdown
    if (seeded && typeof seeded === 'object') {
      return {
        critical: Number(seeded.critical || 0),
        high: Number(seeded.high || 0),
        elevated: Number(seeded.elevated || 0),
        stable: Number(seeded.stable || 0),
      }
    }
    const out = { critical: 0, high: 0, elevated: 0, stable: 0 }
    for (const r of queue) {
      const raw = String(r.severityBand || r.severity || 'stable').toLowerCase()
      if (raw.includes('critical')) out.critical += 1
      else if (raw.includes('high')) out.high += 1
      else if (raw.includes('elevated')) out.elevated += 1
      else out.stable += 1
    }
    return out
  })()

  const histogram = (() => {
    if (Array.isArray(rollups?.compositeHistogram) && rollups.compositeHistogram.length === 5) {
      return rollups.compositeHistogram.map((v: any) => Number(v || 0))
    }
    const bins = [0, 0, 0, 0, 0]
    for (const r of queue) {
      const risk = Number(r.compositeRisk || r.composite_risk || 0)
      bins[bucketIndex(risk)] += 1
    }
    return bins
  })()

  const topCountries = (() => {
    const seeded = Array.isArray(rollups?.topCountries) ? rollups.topCountries : mapRows
    return (seeded || [])
      .map((r: any) => ({
        label: String(r.countryName || r.country_name || r.countryIso2 || r.countryCode || r.country_code || '').trim(),
        code: String(r.countryIso2 || r.countryCode || r.country_code || '').trim().toUpperCase(),
        count: Number(r.incidentCount || r.incident_count || 0),
      }))
      .filter((r: any) => r.count > 0 && r.code !== 'XX' && r.label.toLowerCase() !== 'unknown')
      .sort((a: any, b: any) => b.count - a.count)
      .slice(0, 8)
  })()

  return { severity, histogram, topCountries }
}

export default function QuantVisuals({ queue, mapRows, rollups }: { queue: any[]; mapRows: any[]; rollups?: any }) {
  const { severity, histogram, topCountries } = useMemo(() => summarizeQuantVisuals(queue, mapRows, rollups), [mapRows, queue, rollups])

  const totalSeverity = Math.max(1, severity.critical + severity.high + severity.elevated + severity.stable)
  const sevSeries = [
    { label: 'critical', value: severity.critical, color: '#ef4444' },
    { label: 'high', value: severity.high, color: '#f59e0b' },
    { label: 'elevated', value: severity.elevated, color: '#38bdf8' },
    { label: 'stable', value: severity.stable, color: '#22c55e' },
  ]
  const pieGradient = useMemo(() => {
    let start = 0
    const parts: string[] = []
    for (const s of sevSeries) {
      const ratio = s.value / totalSeverity
      const end = start + ratio * 100
      parts.push(`${s.color} ${start.toFixed(2)}% ${end.toFixed(2)}%`)
      start = end
    }
    return `conic-gradient(${parts.join(', ')})`
  }, [sevSeries, totalSeverity])

  return (
    <div className="quant-grid">
      <section className="card">
        <h3>Country bar chart</h3>
        {topCountries.length === 0 ? <div className="empty-state">No country aggregates yet.</div> : (
          <div className="bars">
            {topCountries.map((c) => (
              <div className="bar-row" key={c.label}>
                <span>{c.label}</span>
                <div className="bar-track"><div className="bar-fill" style={{ width: `${Math.max(4, (c.count / Math.max(1, topCountries[0].count)) * 100)}%` }} /></div>
                <strong>{c.count}</strong>
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="card">
        <h3>Composite risk histogram</h3>
        <div className="hist">
          {histogram.map((v, i) => (
            <div key={i} className="hist-col">
              <div className="hist-bar" style={{ height: `${Math.max(6, v * 18)}px` }} />
              <span>{['0-20%','20-40%','40-60%','60-80%','80-100%'][i]}</span>
              <strong>{v}</strong>
            </div>
          ))}
        </div>
      </section>

      <section className="card">
        <h3>Severity pie chart</h3>
        <div className="pie-wrap">
          <div className="pie-donut" style={{ background: pieGradient }} />
        </div>
        <div className="pie-row">
          {sevSeries.map((s) => (
            <div key={s.label} className="pie-item">
              <span className="dot" style={{ background: s.color }} />
              <span>{s.label}</span>
              <strong>{((s.value / totalSeverity) * 100).toFixed(1)}%</strong>
            </div>
          ))}
        </div>
      </section>

      <section className="card">
        <h3>Comparative figures</h3>
        <div className="kv"><span>Total incidents in queue</span><strong>{queue.length}</strong></div>
        <div className="kv"><span>Critical vs High</span><strong>{severity.critical} / {severity.high}</strong></div>
        <div className="kv"><span>Elevated share</span><strong>{((severity.elevated / totalSeverity) * 100).toFixed(1)}%</strong></div>
        <div className="kv"><span>Country coverage</span><strong>{topCountries.length}</strong></div>
      </section>
    </div>
  )
}
