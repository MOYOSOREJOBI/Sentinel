import test from 'node:test'
import assert from 'node:assert/strict'
import { hasCapability, navItemsForRole } from '../../lib/roles'

test('role-gated tabs are enforced by role mapping', () => {
  assert.deepEqual(navItemsForRole('viewer'), ['overview', 'scores'])
  assert.deepEqual(navItemsForRole('analyst'), ['overview', 'alerts', 'scores'])
  assert.deepEqual(navItemsForRole('admin'), ['overview', 'alerts', 'scores', 'governance'])
})

test('capabilities are strictly separated by role', () => {
  assert.equal(hasCapability('viewer', 'alerts:ack'), false)
  assert.equal(hasCapability('analyst', 'alerts:ack'), true)
  assert.equal(hasCapability('analyst', 'audit:verify'), false)
  assert.equal(hasCapability('admin', 'audit:verify'), true)
  assert.equal(hasCapability('admin', 'models:deploy'), true)
})
