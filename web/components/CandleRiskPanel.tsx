"use client";

import { useEffect, useRef, useState } from 'react'
import { api } from '../lib/api'

interface ScorePoint {
  ts: string
  score_norm: number
  composite_risk: number
  escalation_prob: number
  model_version: string
  symbol?: string
}

interface CandlePoint {
  ts: string
  open: number
  high: number
  low: number
  close: number
  volume: number
  symbol?: string
}

const RESOLUTIONS = ['1m', '5m', '1h', '1d'] as const

function normalizeSeriesSymbol(symbol?: string) {
  return String(symbol || '').trim().toUpperCase()
}

export function pickOverlaySymbol(preferred: string, scores: Array<{ symbol?: string }>, candles: Array<{ symbol?: string }>) {
  const scoreCounts = new Map<string, number>()
  const candleCounts = new Map<string, number>()
  for (const point of scores) {
    const next = normalizeSeriesSymbol(point.symbol)
    if (!next) continue
    scoreCounts.set(next, (scoreCounts.get(next) || 0) + 1)
  }
  for (const point of candles) {
    const next = normalizeSeriesSymbol(point.symbol)
    if (!next) continue
    candleCounts.set(next, (candleCounts.get(next) || 0) + 1)
  }

  const chosen = normalizeSeriesSymbol(preferred)
  if (chosen && Math.min(scoreCounts.get(chosen) || 0, candleCounts.get(chosen) || 0) > 0) {
    return chosen
  }

  let best = ''
  let bestCoverage = -1
  let bestTotal = -1
  for (const candidate of new Set([...scoreCounts.keys(), ...candleCounts.keys()])) {
    const scoreCount = scoreCounts.get(candidate) || 0
    const candleCount = candleCounts.get(candidate) || 0
    const coverage = Math.min(scoreCount, candleCount)
    const total = scoreCount + candleCount
    if (coverage > bestCoverage || (coverage === bestCoverage && total > bestTotal)) {
      best = candidate
      bestCoverage = coverage
      bestTotal = total
    }
  }
  return best
}

export function filterSeriesBySymbol<T extends { symbol?: string }>(series: T[], symbol: string) {
  const normalized = normalizeSeriesSymbol(symbol)
  if (!normalized) {
    return series
  }
  return series.filter((point) => normalizeSeriesSymbol(point.symbol) === normalized)
}

export function sortSeriesByTs<T extends { ts: string }>(series: T[]) {
  const toChartTime = (value: string) => Math.floor(new Date(value).getTime() / 1000)
  const sorted = [...series].sort((a, b) => toChartTime(a.ts) - toChartTime(b.ts))
  const deduped: T[] = []
  for (const point of sorted) {
    const nextTime = toChartTime(point.ts)
    const last = deduped[deduped.length - 1]
    if (last && toChartTime(last.ts) === nextTime) {
      deduped[deduped.length - 1] = point
      continue
    }
    deduped.push(point)
  }
  return deduped
}

export function CandleRiskPanel({ symbol = '', window = '24h' }: { symbol?: string; window?: string }) {
  const chartContainerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<any>(null)
  const candleSeriesRef = useRef<any>(null)
  const compositeSeriesRef = useRef<any>(null)
  const escalationSeriesRef = useRef<any>(null)
  const volumeSeriesRef = useRef<any>(null)
  const [scoreCount, setScoreCount] = useState(0)
  const [candleCount, setCandleCount] = useState(0)
  const [modelVersion, setModelVersion] = useState('')
  const [activeSymbol, setActiveSymbol] = useState('')
  const [resolution, setResolution] = useState<typeof RESOLUTIONS[number]>('5m')
  const [showVolume, setShowVolume] = useState(true)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [reloadKey, setReloadKey] = useState(0)

  useEffect(() => {
    const container = chartContainerRef.current
    if (!container) return
    let disposed = false
    let chart: any
    let ro: ResizeObserver | null = null

    import('lightweight-charts').then(({ createChart, ColorType, CrosshairMode }) => {
      if (disposed) return
      chart = createChart(container, {
        width: container.clientWidth || 640,
        height: 280,
        layout: {
          background: { type: ColorType.Solid, color: '#09111e' },
          textColor: '#9bb5db',
        },
        grid: {
          vertLines: { color: 'rgba(155,181,219,0.06)' },
          horzLines: { color: 'rgba(155,181,219,0.06)' },
        },
        crosshair: { mode: CrosshairMode.Normal },
        rightPriceScale: { borderColor: 'rgba(155,181,219,0.18)', scaleMargins: { top: 0.08, bottom: 0.32 } },
        timeScale: { borderColor: 'rgba(155,181,219,0.18)', timeVisible: true, secondsVisible: resolution === '1m' },
      })

      candleSeriesRef.current = chart.addCandlestickSeries({
        upColor: '#2dd4bf',
        downColor: '#fb7185',
        borderVisible: false,
        wickUpColor: '#2dd4bf',
        wickDownColor: '#fb7185',
        priceScaleId: 'right',
      })

      compositeSeriesRef.current = chart.addLineSeries({
        color: '#5bc4ff',
        lineWidth: 2,
        priceScaleId: 'risk',
      })

      escalationSeriesRef.current = chart.addLineSeries({
        color: '#f59e0b',
        lineWidth: 2,
        lineStyle: 2,
        priceScaleId: 'risk',
      })

      volumeSeriesRef.current = chart.addHistogramSeries({
        color: 'rgba(123, 154, 255, 0.35)',
        priceScaleId: 'volume',
      })

      chart.priceScale('risk').applyOptions({
        scaleMargins: { top: 0.74, bottom: 0.08 },
        borderVisible: false,
      })
      chart.priceScale('volume').applyOptions({
        scaleMargins: { top: 0.84, bottom: 0.0 },
        borderVisible: false,
      })

      chartRef.current = chart

      ro = new ResizeObserver(() => {
        if (container && chart) {
          chart.applyOptions({ width: container.clientWidth })
        }
      })
      ro.observe(container)
    })

    return () => {
      disposed = true
      ro?.disconnect()
      chart?.remove()
      chartRef.current = null
      candleSeriesRef.current = null
      compositeSeriesRef.current = null
      escalationSeriesRef.current = null
      volumeSeriesRef.current = null
    }
  }, [])

  useEffect(() => {
    chartRef.current?.applyOptions({
      timeScale: { timeVisible: true, secondsVisible: resolution === '1m' },
    })
  }, [resolution])

  useEffect(() => {
    let cancelled = false

    async function load() {
      setLoading(true)
      setLoadError('')
      try {
        const loadSlices = async (requestedSymbol: string) => {
          const [candleResp, scoreResp] = await Promise.all([
            api.candles(requestedSymbol, window, 320, resolution),
            api.scores(requestedSymbol, window, 320),
          ])
          return {
            candles: Array.isArray(candleResp?.series) ? candleResp.series as CandlePoint[] : [],
            scores: Array.isArray(scoreResp?.series) ? scoreResp.series as ScorePoint[] : [],
          }
        }

        const preferredSymbol = normalizeSeriesSymbol(symbol)
        let { candles, scores } = await loadSlices(preferredSymbol)
        const preferredCoverage = preferredSymbol
          ? Math.min(
              filterSeriesBySymbol(scores, preferredSymbol).length,
              filterSeriesBySymbol(candles, preferredSymbol).length
            )
          : 0
        if (preferredSymbol && preferredCoverage === 0) {
          ;({ candles, scores } = await loadSlices(''))
        }

        if (cancelled) {
          return
        }

        const selectedSymbol = pickOverlaySymbol(symbol, scores, candles)
        const filteredCandles = sortSeriesByTs(filterSeriesBySymbol(candles, selectedSymbol))
        const filteredScores = sortSeriesByTs(filterSeriesBySymbol(scores, selectedSymbol))

        candleSeriesRef.current?.setData(
          filteredCandles.map((p) => ({
            time: Math.floor(new Date(p.ts).getTime() / 1000) as any,
            open: p.open,
            high: p.high,
            low: p.low,
            close: p.close,
          }))
        )

        compositeSeriesRef.current?.setData(
          filteredScores.map((p) => ({
            time: Math.floor(new Date(p.ts).getTime() / 1000) as any,
            value: Number(p.composite_risk ?? p.score_norm ?? 0),
          }))
        )

        escalationSeriesRef.current?.setData(
          filteredScores.map((p) => ({
            time: Math.floor(new Date(p.ts).getTime() / 1000) as any,
            value: Number(p.escalation_prob ?? 0),
          }))
        )

        volumeSeriesRef.current?.setData(
          showVolume
            ? filteredCandles.map((p) => ({
                time: Math.floor(new Date(p.ts).getTime() / 1000) as any,
                value: Number(p.volume ?? 0),
                color: 'rgba(123, 154, 255, 0.35)',
              }))
            : []
        )

        setActiveSymbol(selectedSymbol)
        setScoreCount(filteredScores.length)
        setCandleCount(filteredCandles.length)
        setModelVersion(filteredScores[filteredScores.length - 1]?.model_version ?? '')
        chartRef.current?.timeScale().fitContent()
      } catch (err: any) {
        candleSeriesRef.current?.setData([])
        compositeSeriesRef.current?.setData([])
        escalationSeriesRef.current?.setData([])
        volumeSeriesRef.current?.setData([])
        setActiveSymbol(normalizeSeriesSymbol(symbol))
        setScoreCount(0)
        setCandleCount(0)
        setModelVersion('')
        setLoadError(String(err?.message || 'upstream unavailable'))
      }
      if (!cancelled) {
        setLoading(false)
      }
    }

    load()
    const iv = setInterval(load, 10_000)
    return () => {
      cancelled = true
      clearInterval(iv)
    }
  }, [reloadKey, resolution, showVolume, symbol, window])

  return (
    <div className='card' data-testid="candle-risk-panel">
      <div className="chart-toolbar">
        <div>
          <h3 style={{ margin: 0 }}>Market + risk overlay</h3>
          <div className='muted chart-subtitle'>
            {loading ? 'Loading live market slices…' : `${scoreCount} score pts · ${candleCount} candle pts · ${resolution} resolution`}
            {activeSymbol ? ` · ${activeSymbol}` : ''}
            {modelVersion ? ` · ${modelVersion}` : ''}
          </div>
        </div>
        <div className="chart-actions">
          <div className="segmented-control" data-testid="chart-resolution">
            {RESOLUTIONS.map((res) => (
              <button
                key={res}
                type="button"
                className={resolution === res ? 'active' : ''}
                onClick={() => setResolution(res)}
              >
                {res}
              </button>
            ))}
          </div>
          <button type="button" className={`ghost-btn small ${showVolume ? 'active' : ''}`} onClick={() => setShowVolume((prev) => !prev)}>
            Volume
          </button>
        </div>
      </div>
      {loadError && !loading ? (
        <div className='empty-state' style={{ marginBottom: 10 }}>
          Chart data unavailable for the current slice ({loadError}).
          <button type="button" className="ghost-btn small" style={{ marginLeft: 10 }} onClick={() => setReloadKey((prev) => prev + 1)}>
            Retry
          </button>
        </div>
      ) : null}
      {scoreCount === 0 && candleCount === 0 && !loading && !loadError ? (
        <div className='empty-state' style={{ marginBottom: 10 }}>
          No score or candle points exist for the current slice yet. Use Generate data / backfill in the demo if you need a fuller history window.
        </div>
      ) : null}
      <div ref={chartContainerRef} style={{ width: '100%', height: 280 }} />
      <div className="chart-legend">
        <span><i className="legend-swatch candle" /> Candles</span>
        <span><i className="legend-swatch composite" /> Composite risk</span>
        <span><i className="legend-swatch escalation" /> Escalation probability</span>
        <span><i className="legend-swatch volume" /> Volume</span>
      </div>
    </div>
  )
}
