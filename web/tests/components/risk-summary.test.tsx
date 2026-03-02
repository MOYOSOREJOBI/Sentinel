import test from 'node:test'
import assert from 'node:assert/strict'
import { summarizeScoreSeverity } from '../../lib/risk'

test('score summary counts high and critical severities', () => {
  const out = summarizeScoreSeverity([
    { severity: 'critical' },
    { severity: 'high' },
    { severity: 'high' },
    { severity: 'medium' },
  ])
  assert.deepEqual(out, { critical: 1, high: 2 })
})
