import test from 'node:test'
import assert from 'node:assert/strict'

test('upsert replaces matching row only', () => {
  const rows: any[] = [{ id: 1, priorityScore: 1 }, { id: 2, priorityScore: 1 }]
  const patch = { type: 'upsert', incident: { id: 1, priorityScore: 2 } }
  const idx = rows.findIndex((r) => r.id === patch.incident.id)
  if (idx >= 0) rows[idx] = patch.incident
  assert.equal(rows[0].priorityScore, 2)
  assert.equal(rows[1].priorityScore, 1)
})
