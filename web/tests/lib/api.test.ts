import test from 'node:test'
import assert from 'node:assert/strict'
import { normalizeAlertCreatedAt } from '../../lib/api'

test('normalizeAlertCreatedAt prefers created_at and falls back to ts', () => {
  assert.equal(normalizeAlertCreatedAt({ id: 1, symbol: 'AAPL', status: 'open', ts: 't1', created_at: 't2' }), 't2')
  assert.equal(normalizeAlertCreatedAt({ id: 2, symbol: 'MSFT', status: 'open', ts: 't3' }), 't3')
})
