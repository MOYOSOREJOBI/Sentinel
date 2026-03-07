'use client'

import { useMemo } from 'react'
import atlas from '../lib/assets/world-atlas-lite.topo.json'
import type { CountryAgg } from '../lib/types'
import { useI18n } from '../lib/i18n'

type Props = {
  data?: CountryAgg[]
  rows?: any[]
  geoIncomplete?: boolean
  selectedCountryCode?: string
  selectedRegion?: string
  onSelectCountry?: (code?: string) => void
  onSelectRegion?: (region: string) => void
}

type TopologyPoint = [number, number]
type TopologyGeometry = {
  type: 'Polygon'
  properties: {
    iso2: string
    name: string
  }
  arcs: number[][]
}

type AtlasTopology = {
  bbox: number[]
  arcs: TopologyPoint[][]
  objects: {
    countries: {
      geometries: TopologyGeometry[]
    }
  }
}

type MapFeature = {
  iso2: string
  name: string
  path: string
}

type NormalizedCountry = {
  countryIso2: string
  countryCode: string
  countryName: string
  incidentCount: number
  avgCompositeRisk: number
  maxSafetyLevel: string
  trustState: string
  topIndustry: string
}

const WORLD_VIEWBOX = '0 0 1000 500'
const topo = atlas as unknown as AtlasTopology

function arcPoints(index: number) {
  const forward = index >= 0
  const arc = topo.arcs[forward ? index : ~index] || []
  const points = forward ? arc : [...arc].reverse()
  return points
}

function polygonPath(rings: number[][]) {
  return rings
    .map((ring) => {
      const points = ring.flatMap((index) => arcPoints(index))
      if (points.length === 0) {
        return ''
      }
      const [startX, startY] = points[0]
      const segments = [`M${startX},${startY}`]
      for (const [x, y] of points.slice(1)) {
        segments.push(`L${x},${y}`)
      }
      segments.push('Z')
      return segments.join(' ')
    })
    .filter(Boolean)
    .join(' ')
}

export function worldAtlasFeatures() {
  return topo.objects.countries.geometries.map((geometry) => ({
    iso2: geometry.properties.iso2.toUpperCase(),
    name: geometry.properties.name,
    path: polygonPath(geometry.arcs),
  }))
}

export function normalizeWorldMapRows(data?: CountryAgg[], rows?: any[]): NormalizedCountry[] {
  const source = Array.isArray(data) ? data : (rows || [])
  return source
    .map((row: any) => ({
      countryIso2: String(row.countryIso2 || row.countryCode || row.country_code || row.country || '').trim().toUpperCase(),
      countryCode: String(row.countryIso2 || row.countryCode || row.country_code || row.country || '').trim().toUpperCase(),
      countryName: String(row.countryName || row.country_name || row.name || '').trim(),
      incidentCount: Number(row.incidentCount || row.incident_count || 0),
      avgCompositeRisk: Number(row.avgCompositeRisk || row.avg_risk || row.incident_pressure || 0),
      maxSafetyLevel: String(row.maxSafetyLevel || row.max_safety_level || 'Stable'),
      trustState: String(row.trustState || row.trust_state || 'healthy'),
      topIndustry: String(row.topIndustry || row.top_sector || row.industry || '').trim(),
    }))
    .filter((row) => row.countryIso2 && row.countryIso2 !== 'XX' && row.countryName)
}

function fillForRow(row: NormalizedCountry | undefined, maxIncidents: number) {
  if (!row) {
    return '#132235'
  }
  const countRatio = Math.min(1, row.incidentCount / Math.max(1, maxIncidents))
  const riskRatio = Math.max(0, Math.min(1, row.avgCompositeRisk))
  const alpha = 0.2 + countRatio * 0.65
  const red = Math.round(68 + riskRatio * 168)
  const green = Math.round(124 + (1 - riskRatio) * 72)
  const blue = Math.round(242 - riskRatio * 154)
  return `rgba(${red}, ${green}, ${blue}, ${alpha.toFixed(3)})`
}

export default function WorldRiskMap({ data, rows, geoIncomplete = false, selectedCountryCode, onSelectCountry }: Props) {
  const { tr } = useI18n()
  const normalized = useMemo(() => normalizeWorldMapRows(data, rows), [data, rows])
  const features = useMemo(() => worldAtlasFeatures(), [])
  const byCode = useMemo(() => new Map(normalized.map((row) => [row.countryIso2, row])), [normalized])
  const maxIncidents = useMemo(() => normalized.reduce((max, row) => Math.max(max, row.incidentCount), 1), [normalized])
  const topCountries = normalized.slice().sort((a, b) => {
    if (b.incidentCount === a.incidentCount) {
      return a.countryIso2.localeCompare(b.countryIso2)
    }
    return b.incidentCount - a.incidentCount
  }).slice(0, 8)

  return <div className='card' data-testid='world-map'>
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12, marginBottom: 8 }}>
      <h3 style={{ margin: 0 }}>{tr('worldMapTitle', 'World risk map')}</h3>
      {selectedCountryCode ? (
        <button
          type='button'
          className='ghost-btn small'
          data-testid='world-map-clear'
          onClick={() => onSelectCountry?.('')}
        >
          {tr('filterClear', 'Clear')}
        </button>
      ) : null}
    </div>
    {geoIncomplete ? (
      <div className='badge degraded' style={{ display: 'inline-flex', marginBottom: 8 }}>
        {tr('worldMapIncompleteGeo', 'Geo enrichment incomplete — map uses fallback codes.')}
      </div>
    ) : null}
    <svg viewBox={WORLD_VIEWBOX} role='img' aria-label='world risk map'>
      <rect x='0' y='0' width='1000' height='500' rx='14' fill='#09111e' />
      <g opacity='0.18'>
        <path d='M22,126 L978,126' stroke='#2c4669' strokeWidth='1.5' />
        <path d='M22,252 L978,252' stroke='#2c4669' strokeWidth='1.5' />
        <path d='M22,378 L978,378' stroke='#2c4669' strokeWidth='1.5' />
      </g>
      {features.map((feature) => {
        const row = byCode.get(feature.iso2)
        const selected = selectedCountryCode === feature.iso2
        return (
          <path
            key={feature.iso2}
            d={feature.path}
            data-country-code={feature.iso2}
            fill={fillForRow(row, maxIncidents)}
            stroke={selected ? '#f8fafc' : '#27435d'}
            strokeWidth={selected ? 4 : 2}
            vectorEffect='non-scaling-stroke'
            style={{ cursor: 'pointer', transition: 'fill 180ms ease, stroke 180ms ease' }}
            onClick={() => onSelectCountry?.(feature.iso2)}
          >
            <title>
              {row
                ? `${row.countryName} | incident count ${row.incidentCount} | avg risk ${row.avgCompositeRisk.toFixed(2)} | ${row.maxSafetyLevel} | ${row.trustState} | ${row.topIndustry || 'n/a'}`
                : feature.name}
            </title>
          </path>
        )
      })}
    </svg>
    {normalized.length === 0 ? <div className='empty-state'>{tr('worldMapEmpty', 'No regional aggregates yet. Keep filters broad to populate global heat.')}</div> : null}
    {topCountries.length > 0 ? (
      <div className='map-canvas' style={{ marginTop: 10 }}>
        {topCountries.map((row) => {
          const intensity = Math.max(1, Math.min(5, Math.ceil((row.incidentCount / Math.max(1, maxIncidents)) * 5)))
          return (
            <button
              key={row.countryIso2}
              type='button'
              data-testid={`world-map-chip-${row.countryIso2}`}
              className={`map-cell map-intensity-${intensity} ${selectedCountryCode === row.countryIso2 ? 'selected' : ''}`}
              onClick={() => onSelectCountry?.(row.countryIso2)}
            >
              <strong>{row.countryIso2}</strong>
              <div>{row.countryName}</div>
              <div className='muted'>incidents {row.incidentCount}</div>
            </button>
          )
        })}
      </div>
    ) : null}
    <aside className='map-legend'>{tr('worldMapTopRegions', 'Top regions')}</aside>
  </div>
}
