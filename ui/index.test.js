import assert from 'node:assert/strict'
import { test } from 'node:test'
import { compareRuntimeRevision, isRuntimeSnapshot } from './index.js'

test('revision ordering preserves values above JavaScript integer precision', () => {
  assert.equal(compareRuntimeRevision('9007199254740993', '9007199254740992'), 1)
  assert.equal(compareRuntimeRevision('9', '10'), -1)
  assert.equal(compareRuntimeRevision('0', '0'), 0)
  for (const value of ['-1', '01', '1e3', '', 1, null]) {
    assert.throws(() => compareRuntimeRevision(value, '0'), TypeError)
  }
})

test('untrusted availability cannot invent an installed or active generation', () => {
  const p = { id: 'sessions', installed: true, desiredState: 'enabled', observedState: 'running', available: true, generation: 'g1' }
  const snapshot = plugins => ({ epoch: 'host1', revision: '1', plugins })
  assert.equal(isRuntimeSnapshot(snapshot([p])), true)
  assert.equal(isRuntimeSnapshot(snapshot(null)), true)
  for (const bad of [{ ...p, installed: false }, { ...p, generation: '' }, { ...p, desiredState: 'disabled' }, { ...p, available: 'true' }]) {
    assert.equal(isRuntimeSnapshot(snapshot([bad])), false)
  }
  assert.equal(isRuntimeSnapshot(snapshot([p, p])), false)
  assert.equal(isRuntimeSnapshot({ ...snapshot([p]), revision: 1 }), false)
  assert.equal(isRuntimeSnapshot({ ...snapshot([p]), epoch: '' }), false)
  assert.equal(isRuntimeSnapshot(snapshot([{ ...p, available: false, observedState: 'degraded' }])) , true)
})

test('unknown durable intent is a valid recovery state but never available', () => {
  const p = { id: 'sessions', installed: true, desiredState: 'unknown', observedState: 'failed', available: false, generation: '', reason: 'intent store unreadable' }
  const value = { epoch: 'host1', revision: '2', plugins: [p] }
  assert.equal(isRuntimeSnapshot(value), true)
  assert.equal(isRuntimeSnapshot({ ...value, plugins: [{ ...p, available: true, generation: 'g1' }] }), false)
})
