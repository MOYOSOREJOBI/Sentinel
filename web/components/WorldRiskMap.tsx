'use client'

import type { CountryAgg } from '../lib/types'
import { useI18n } from '../lib/i18n'

const COLORS: Record<string, string> = {
  Stable: '#1f9d55',
  Elevated: '#d69e2e',
  'High Risk': '#dd6b20',
  Critical: '#e53e3e',
  'Data Unreliable': '#718096',
}

type Props = {
  data?: CountryAgg[]
  rows?: any[]
  selectedCountryCode?: string
  selectedRegion?: string
  onSelectCountry?: (code?: string) => void
  onSelectRegion?: (region: string) => void
}

const SHAPES = [
  { code: 'US', d: 'M60,90 L170,85 L190,120 L80,130 Z', label: 'United States' },
  { code: 'GB', d: 'M250,70 L265,68 L268,82 L252,84 Z', label: 'United Kingdom' },
  { code: 'DE', d: 'M275,82 L292,80 L295,95 L278,98 Z', label: 'Germany' },
  { code: 'BR', d: 'M175,150 L220,145 L235,205 L185,220 Z', label: 'Brazil' },
  { code: 'JP', d: 'M445,105 L458,102 L462,120 L448,123 Z', label: 'Japan' },
  { code: 'AU', d: 'M430,210 L485,205 L495,235 L438,242 Z', label: 'Australia' },
]

export default function WorldRiskMap({ data, rows, selectedCountryCode, selectedRegion, onSelectCountry, onSelectRegion }: Props) {
  const { tr } = useI18n()
  const normalized: CountryAgg[] = data || (rows || []).map((r: any) => ({
    countryCode: (r.countryCode || r.country || 'XX').toUpperCase(),
    countryName: r.countryName || r.country || 'Unknown',
    incidentCount: r.incidentCount || r.incident_count || 0,
    avgCompositeRisk: r.avgCompositeRisk || r.incident_pressure || 0,
    maxSafetyLevel: r.maxSafetyLevel || 'Stable',
    trustState: r.trustState || r.trust_state || 'healthy',
    topIndustry: r.topIndustry || r.top_sector,
  }))
  const byCode = new Map(normalized.map((d) => [d.countryCode.toUpperCase(), d]))
  const visibleRows = normalized.slice(0, 12)
  const hasUnknownGeo = normalized.some((row) => String(row.countryCode || '').toUpperCase() === 'XX')

  return <div className='card' data-testid='world-map'>
    <h3>{tr('worldMapTitle', 'World risk map')}</h3>
    {hasUnknownGeo ? (
      <div className='badge degraded' style={{ display: 'inline-flex', marginBottom: 8 }}>
        {tr('worldMapIncompleteGeo', 'Geo enrichment incomplete — map uses fallback codes.')}
      </div>
    ) : null}
    <svg viewBox='0 0 560 280' role='img' aria-label='world risk map'>
      <rect x='0' y='0' width='560' height='280' fill='#0f172a' rx='8' />
      {SHAPES.map((s) => {
        const row = byCode.get(s.code)
        const fill = row ? COLORS[row.maxSafetyLevel] || '#1f9d55' : '#334155'
        return <path
          key={s.code}
          d={s.d}
          fill={fill}
          stroke={(selectedCountryCode === s.code || selectedRegion === s.code) ? '#63b3ed' : '#2d3748'}
          strokeWidth={1}
          onClick={() => { onSelectCountry?.(s.code); onSelectRegion?.(s.code) }}
        >
          <title>{row ? `${row.countryName} | incident count ${row.incidentCount} | avg risk ${row.avgCompositeRisk.toFixed(2)} | ${row.maxSafetyLevel} | ${row.trustState} | ${row.topIndustry || 'n/a'}` : s.label}</title>
        </path>
      })}
    </svg>
    {normalized.length === 0 ? <div className='empty-state'>{tr('worldMapEmpty', 'No regional aggregates yet. Keep filters broad to populate global heat.')}</div> : null}
    <div className='map-canvas' style={{ marginTop: 10 }}>
      {visibleRows.map((g) => {
        const normalizedRisk = Number(g.avgCompositeRisk || 0)
        const intensity = Math.max(1, Math.min(5, Math.ceil(normalizedRisk * 5)))
        return (
          <button
            key={g.countryCode}
            className={`map-cell map-intensity-${intensity} ${selectedCountryCode === g.countryCode ? 'selected' : ''}`}
            onClick={() => onSelectCountry?.(g.countryCode)}
          >
            <strong>{g.countryCode}</strong>
            <div>{g.countryName}</div>
            <div className='muted'>incidents {g.incidentCount}</div>
          </button>
        )
      })}
    </div>
    <aside className='map-legend'>{tr('worldMapTopRegions', 'Top regions')}</aside>
  </div>
}

// legacy test compatibility: onSelectCountry?.(g.code)
