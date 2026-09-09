import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { projectTerminalPresentation, terminalPresentationCommand } from './index.js'

test('terminal presentation fixture is the generated-contract conformance corpus', () => {
  const vectors = JSON.parse(readFileSync(new URL('./testdata/terminal_presentation.json', import.meta.url), 'utf8'))
  assert.ok(vectors.acceptRequests.length > 0)
  assert.ok(vectors.acceptResults.length > 0)
  assert.ok(vectors.rejectRequests.length > 0)
  assert.ok(vectors.rejectResults.length > 0)
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
