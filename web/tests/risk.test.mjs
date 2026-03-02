import test from 'node:test'
import assert from 'node:assert/strict'

function summarizeScoreSeverity(scores) {
  return scores.reduce((acc, score) => {
    if (score.severity === 'critical') acc.critical += 1
    if (score.severity === 'high') acc.high += 1
    return acc
  }, { critical: 0, high: 0 })
}

test('summarizeScoreSeverity counts high and critical severities', () => {
  const out = summarizeScoreSeverity([
    { severity: 'critical' },
    { severity: 'high' },
    { severity: 'high' },
    { severity: 'medium' },
  ])
  assert.deepEqual(out, { critical: 1, high: 2 })
})
