import test from 'node:test'
import assert from 'node:assert/strict'
import { filterAlerts, groupAlerts } from '../../lib/alerts'

const alerts = [
  { id: 1, symbol: 'AAPL', status: 'open', severity: 'high' },
  { id: 2, symbol: 'MSFT', status: 'ack', severity: 'critical' },
  { id: 3, symbol: 'AAPL', status: 'resolved', severity: 'low' },
]

test('filterAlerts applies severity status and symbol filters', () => {
  assert.equal(filterAlerts(alerts, { severity: 'high', status: 'all', symbol: '' }).length, 1)
  assert.equal(filterAlerts(alerts, { severity: 'all', status: 'ack', symbol: '' }).length, 1)
  assert.equal(filterAlerts(alerts, { severity: 'all', status: 'all', symbol: 'aap' }).length, 2)
})

test('groupAlerts groups by status and symbol', () => {
  const byStatus = groupAlerts(alerts, 'status')
  assert.equal(byStatus.open.length, 1)
  const bySymbol = groupAlerts(alerts, 'symbol')
  assert.equal(bySymbol.AAPL.length, 2)
})
