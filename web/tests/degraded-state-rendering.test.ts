import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'

test('degraded state component exists', () => {
  const s = fs.readFileSync('components/DegradedState.tsx', 'utf8')
  assert.equal(s.includes('Service degraded'), true)
})
