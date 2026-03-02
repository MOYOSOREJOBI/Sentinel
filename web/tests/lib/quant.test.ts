import test from 'node:test'
import assert from 'node:assert/strict'
import { deriveQuantMetrics, percentile } from '../../lib/quant'

test('percentile returns bounded value', () => {
  assert.equal(percentile([], 0.95), 0)
  assert.equal(percentile([1, 2, 3, 4, 5], 0.5), 3)
})

test('deriveQuantMetrics computes meaningful positive values', () => {
  const out = deriveQuantMetrics([
    { compositeRisk: 90, priorityScore: 88, confidence: 0.9, escalationProbability: 0.8 },
    { compositeRisk: 70, priorityScore: 65, confidence: 0.75, escalationProbability: 0.55 },
    { compositeRisk: 40, priorityScore: 44, confidence: 0.6, escalationProbability: 0.3 },
  ])

  assert.ok(out.meanComposite > 0)
  assert.ok(out.p95Composite >= out.meanComposite)
  assert.ok(out.expectedLossProxy > 0)
  assert.ok(out.avgConfidence > 0)
})
