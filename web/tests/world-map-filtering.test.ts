import test from 'node:test'
import assert from 'node:assert/strict'
import { buildFilterAwareHref } from '../components/AppShell'
import { normalizeWorldMapRows, worldAtlasFeatures } from '../components/WorldRiskMap'

test('world atlas ships committed country geometry', () => {
  const features = worldAtlasFeatures()
  assert.ok(features.length >= 12)
  assert.ok(features.some((feature) => feature.iso2 === 'JP'))
  assert.match(features[0].path, /^M/)
})

test('world map normalizes real iso2 rows and drops placeholder XX', () => {
  const rows = normalizeWorldMapRows(undefined, [
    { countryIso2: 'JP', countryName: 'Japan', incidentCount: 9 },
    { countryCode: 'XX', countryName: 'Unknown', incidentCount: 5 },
  ])

  assert.deepEqual(rows.map((row) => row.countryIso2), ['JP'])
})

test('country filters persist when navigating to queue and feed routes', () => {
  assert.equal(buildFilterAwareHref('/queue', 'en', 'window=24h&countryCode=JP', true), '/en/queue?window=24h&countryCode=JP')
  assert.equal(buildFilterAwareHref('/feed', 'en', 'window=24h&countryCode=JP', true), '/en/feed?window=24h&countryCode=JP')
  assert.equal(buildFilterAwareHref('/about', 'en', 'window=24h&countryCode=JP', true), '/en/about')
})
