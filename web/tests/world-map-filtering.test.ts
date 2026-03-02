import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'

test('world map has click binding for filters', () => {
  const s = fs.readFileSync('components/WorldRiskMap.tsx', 'utf8')
  assert.equal(s.includes('onSelectCountry?.('), true)
  assert.equal(s.includes('onSelectCountry?.(g.code)'), true)
})
