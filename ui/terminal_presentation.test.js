import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { isPresentationItem, projectTerminalPresentation, terminalPresentationCommand } from './index.js'

test('terminal presentation fixture is the generated-contract conformance corpus', () => {
  const vectors = JSON.parse(readFileSync(new URL('./testdata/terminal_presentation.json', import.meta.url), 'utf8'))
  assert.ok(vectors.acceptRequests.length > 0)
  assert.ok(vectors.acceptResults.length > 0)
  assert.ok(vectors.rejectRequests.length > 0)
  assert.ok(vectors.rejectResults.length > 0)
  assert.equal(vectors.acceptResults[0].items[0].agentId, 'codex-qi')
  assert.equal(vectors.acceptResults[0].items[0].displayName, 'Codex Qi')
})

test('presentation item accepts optional agent identity and rejects wrong identity types', () => {
  const item = {
    terminalId: 'terminal-1',
    groupPath: [],
    badges: [],
    priority: 0,
    freshness: 'fresh'
  }
  assert.equal(isPresentationItem(item), true)
  assert.equal(isPresentationItem({ ...item, agentId: 'codex-qi', displayName: 'Codex Qi' }), true)
  assert.equal(isPresentationItem({ ...item, agentId: 42 }), false)
  assert.equal(isPresentationItem({ ...item, displayName: false }), false)
})

test('browser adapter dispatches only through the selected owner-scoped host', async () => {
  const request = JSON.parse(readFileSync(new URL('./testdata/terminal_presentation.json', import.meta.url), 'utf8')).acceptRequests[0]
  const calls = []
  const result = { lease: request.lease, maxAgeMs: 1000, items: [] }
  const host = { request: async (...args) => { calls.push(args); return result } }
  assert.equal(await projectTerminalPresentation(host, request), result)
  assert.deepEqual(calls, [[terminalPresentationCommand, request]])
  await assert.rejects(() => projectTerminalPresentation({}, request), TypeError)
})
