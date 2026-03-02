import test from 'node:test'
import assert from 'node:assert/strict'
import { navItemsForRole } from '../../lib/navigation'

test('role-gated tabs are enforced by role mapping', () => {
  assert.deepEqual(navItemsForRole('viewer'), ['overview', 'scores'])
  assert.deepEqual(navItemsForRole('analyst'), ['overview', 'alerts', 'scores'])
  assert.deepEqual(navItemsForRole('admin'), ['overview', 'alerts', 'scores', 'governance'])
})
