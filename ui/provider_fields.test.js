import assert from 'node:assert/strict'
import { test } from 'node:test'
import { isConfigField } from './contracts.js'

test('provider field browser guard checks point and capability structure', () => {
  const field = { key: 'store', kind: 'provider', point: 'host.data.store', requires: ['transactions', 'blobs'] }
  assert.equal(isConfigField(field), true)
  assert.equal(isConfigField({ key: 'store', kind: 'string' }), true)
  assert.equal(isConfigField({ ...field, point: 42 }), false)
  assert.equal(isConfigField({ ...field, requires: 'transactions' }), false)
  assert.equal(isConfigField({ ...field, requires: ['transactions', 42] }), false)
  // These guards check wire structure; Go validates declaration semantics and
  // the host resolves registry membership and runtime authorization.
})
