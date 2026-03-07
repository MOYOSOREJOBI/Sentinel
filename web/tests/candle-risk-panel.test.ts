import test from 'node:test'
import assert from 'node:assert/strict'
import { filterSeriesBySymbol, pickOverlaySymbol, sortSeriesByTs } from '../components/CandleRiskPanel'

test('pickOverlaySymbol prefers the symbol shared by candles and scores', () => {
  const chosen = pickOverlaySymbol('', [
    { symbol: 'AAPL' },
    { symbol: 'AAPL' },
    { symbol: 'MSFT' },
  ], [
    { symbol: 'MSFT' },
    { symbol: 'AAPL' },
    { symbol: 'AAPL' },
  ])

  assert.equal(chosen, 'AAPL')
})

test('filterSeriesBySymbol keeps only the focused instrument when one is selected', () => {
  const filtered = filterSeriesBySymbol([
    { symbol: 'AAPL', value: 1 },
    { symbol: 'MSFT', value: 2 },
    { symbol: 'AAPL', value: 3 },
  ], 'aapl')

  assert.deepEqual(filtered, [
    { symbol: 'AAPL', value: 1 },
    { symbol: 'AAPL', value: 3 },
  ])
})

test('pickOverlaySymbol falls back when the preferred symbol has no series data', () => {
  const chosen = pickOverlaySymbol('REIT-010', [
    { symbol: 'CONS-041' },
    { symbol: 'CONS-041' },
  ], [
    { symbol: 'CONS-041' },
  ])

  assert.equal(chosen, 'CONS-041')
})

test('pickOverlaySymbol falls back when the preferred symbol has no candle coverage', () => {
  const chosen = pickOverlaySymbol('REIT-010', [
    { symbol: 'REIT-010' },
    { symbol: 'CONS-041' },
    { symbol: 'CONS-041' },
  ], [
    { symbol: 'CONS-041' },
    { symbol: 'CONS-041' },
  ])

  assert.equal(chosen, 'CONS-041')
})

test('sortSeriesByTs orders chart points from oldest to newest', () => {
  const ordered = sortSeriesByTs([
    { ts: '2026-03-03T12:05:00.900Z', value: 2 },
    { ts: '2026-03-03T12:01:00Z', value: 1 },
    { ts: '2026-03-03T12:03:00.100Z', value: 3 },
    { ts: '2026-03-03T12:03:00.900Z', value: 4 },
  ])

  assert.deepEqual(ordered.map((point) => point.value), [1, 4, 2])
})
