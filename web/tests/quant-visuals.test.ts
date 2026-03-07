import test from 'node:test'
import assert from 'node:assert/strict'
import { bucketIndex, normalizeRiskValue, summarizeQuantVisuals } from '../components/QuantVisuals'

test('normalizeRiskValue supports normalized and percentage-like inputs', () => {
  assert.equal(normalizeRiskValue(0.62), 0.62)
  assert.equal(normalizeRiskValue(62), 0.62)
  assert.equal(normalizeRiskValue(140), 1)
  assert.equal(normalizeRiskValue(-1), 0)
})

test('bucketIndex uses 0..1 risk buckets', () => {
  assert.equal(bucketIndex(0.05), 0)
  assert.equal(bucketIndex(0.21), 1)
  assert.equal(bucketIndex(0.59), 2)
  assert.equal(bucketIndex(0.61), 3)
  assert.equal(bucketIndex(0.91), 4)
  assert.equal(bucketIndex(91), 4)
})

test('summarizeQuantVisuals prefers backend rollups and filters placeholder countries', () => {
  const summary = summarizeQuantVisuals([], [], {
    severityBreakdown: { critical: 2, high: 3, elevated: 4, stable: 5 },
    compositeHistogram: [1, 2, 3, 4, 5],
    topCountries: [
      { countryCode: 'XX', incidentCount: 99 },
      { countryCode: 'JP', incidentCount: 7 },
      { countryCode: 'AU', incidentCount: 5 },
    ],
  })

  assert.deepEqual(summary.severity, { critical: 2, high: 3, elevated: 4, stable: 5 })
  assert.deepEqual(summary.histogram, [1, 2, 3, 4, 5])
  assert.deepEqual(summary.topCountries, [
    { label: 'JP', code: 'JP', count: 7 },
    { label: 'AU', code: 'AU', count: 5 },
  ])
})
