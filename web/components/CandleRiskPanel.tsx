"use client";

import { useEffect, useRef, useState } from 'react'
import { api } from '../lib/api'

interface ScorePoint {
  ts: string
  score_norm: number
  composite_risk: number
  escalation_prob: number
  model_version: string
}

interface CandlePoint {
  ts: string
  open: number
  high: number
  low: number
  close: number
  volume: number
}

const RESOLUTIONS = ['1m', '5m', '1h', '1d'] as const

export function CandleRiskPanel({ symbol = '', window = '24h' }: { symbol?: string; window?: string }) {
  const chartContainerRef = useRef<HTMLDivElement>(null)
  const chartRef = useRef<any>(null)
  const candleSeriesRef = useRef<any>(null)
  const compositeSeriesRef = useRef<any>(null)
  const escalationSeriesRef = useRef<any>(null)
  const volumeSeriesRef = useRef<any>(null)
  const [scoreCount, setScoreCount] = useState(0)
  const [modelVersion, setModelVersion] = useState('')
  const [resolution, setResolution] = useState<typeof RESOLUTIONS[number]>('5m')
  const [showVolume, setShowVolume] = useState(true)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const container = chartContainerRef.current
    if (!container) return
    let chart: any

    import('lightweight-charts').then(({ createChart, ColorType, CrosshairMode }) => {
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

      const ro = new ResizeObserver(() => {
        if (container && chart) {
          chart.applyOptions({ width: container.clientWidth })
        }
      })
      ro.observe(container)
      return () => ro.disconnect()
    })

    return () => {
      chart?.remove()
      chartRef.current = null
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
      try {
        const [candleResp, scoreResp] = await Promise.all([
          api.candles(symbol, window, 320, resolution),
          api.scores(symbol, window, 320),
        ])

        if (cancelled) {
          return
        }

        const candles: CandlePoint[] = Array.isArray(candleResp?.series) ? candleResp.series : []
        const scores: ScorePoint[] = Array.isArray(scoreResp?.series) ? scoreResp.series : []

        candleSeriesRef.current?.setData(
          candles.map((p) => ({
            time: Math.floor(new Date(p.ts).getTime() / 1000) as any,
            open: p.open,
            high: p.high,
            low: p.low,
            close: p.close,
          }))
        )

        compositeSeriesRef.current?.setData(
          scores.map((p) => ({
            time: Math.floor(new Date(p.ts).getTime() / 1000) as any,
            value: Number(p.composite_risk ?? p.score_norm ?? 0),
          }))
        )

        escalationSeriesRef.current?.setData(
          scores.map((p) => ({
            time: Math.floor(new Date(p.ts).getTime() / 1000) as any,
            value: Number(p.escalation_prob ?? 0),
          }))
        )

        volumeSeriesRef.current?.setData(
          showVolume
            ? candles.map((p) => ({
                time: Math.floor(new Date(p.ts).getTime() / 1000) as any,
                value: Number(p.volume ?? 0),
                color: 'rgba(123, 154, 255, 0.35)',
              }))
            : []
        )

        setScoreCount(scores.length)
        setModelVersion(scores[scores.length - 1]?.model_version ?? '')
        chartRef.current?.timeScale().fitContent()
      } catch {
        setScoreCount(0)
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
  }, [resolution, showVolume, symbol, window])

  return (
    <div className='card' data-testid="candle-risk-panel">
      <div className="chart-toolbar">
        <div>
          <h3 style={{ margin: 0 }}>Market + risk overlay</h3>
          <div className='muted chart-subtitle'>
            {loading ? 'Loading live market slices…' : `${scoreCount} score pts · ${resolution} resolution`}
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
      {scoreCount === 0 && !loading ? (
        <div className='empty-state' style={{ marginBottom: 10 }}>
          No score points for the current slice yet. Chart chrome stays mounted so resolution changes and new ticks render immediately.
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
