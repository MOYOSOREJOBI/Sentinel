import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'

test('governance page retains admin-gated controls', () => {
  const s = fs.readFileSync('app/governance/page.tsx', 'utf8')
  assert.equal(s.includes('Admin role required'), true)
})
