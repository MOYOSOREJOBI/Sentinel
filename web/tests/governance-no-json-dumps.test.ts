import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'

test('governance page has no JSON dump pre block', () => {
  const s = fs.readFileSync('app/governance/page.tsx', 'utf8')
  assert.equal(s.includes('JSON.stringify'), false)
})
