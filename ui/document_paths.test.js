import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { isDocumentPath, matchDocumentPath, documentPathsOverlap } from './index.js'

test('document matching uses the pinned common SDK vectors', () => {
  const vectors = JSON.parse(readFileSync(new URL('./testdata/document_paths.json', import.meta.url), 'utf8'))
  for (const { pattern, path, params } of vectors.matches) {
    assert.deepEqual(matchDocumentPath(pattern, path), params, `${pattern} at ${path}`)
  }
  for (const { a, b, overlap } of vectors.overlaps) {
    assert.equal(documentPathsOverlap(a, b), overlap, `${a} vs ${b}`)
  }
})

test('malformed wire patterns cannot acquire document paths', () => {
  for (const pattern of [null, {}, [], 0, '', '/', '/a/', '//a', '/a//b', '/a/.', '/a/..', '/a%20b', '/a?b', '/a#b', '/a\\b', '/a/:', '/a/:9x', '/a/:x/:x', '/a/:x/*x', '/a/*', '/a/*rest/more', '/a/pre:x', '/a/*rest*', '/é', '/a\n', '/a/:x\n', '/a/*x\n']) {
    assert.equal(isDocumentPath(pattern), false, String(pattern))
    assert.equal(matchDocumentPath(pattern, '/a/b'), null)
    assert.throws(() => documentPathsOverlap(pattern, '/a'), TypeError)
  }
  for (const pattern of ['/a', '/a/:id', '/a/*rest', '/a/:id/*rest']) assert.equal(isDocumentPath(pattern), true)
  assert.equal(matchDocumentPath('/a/:id', '/a/\ud800'), null)
  assert.deepEqual(matchDocumentPath('/a/:id', '/a/😀'), { id: '😀' })
})

test('overlap agrees with enumerated matching language', () => {
  const patterns = ['/a', '/b', '/:x', '/*tail', '/a/:x', '/a/*tail', '/a/b', '/:x/b', '/:x/*tail', '/a/:x/b', '/a/:x/*tail', '/a/b/:x', '/a/b/*tail', '/a/b/x/:id', '/a/b/x/*tail']
  const witnesses = new Set()
  function walk(parts) {
    if (parts.length) {
      const path = '/' + parts.join('/')
      const matched = patterns.filter(pattern => matchDocumentPath(pattern, path) !== null)
      for (const a of matched) for (const b of matched) witnesses.add(JSON.stringify([a, b]))
    }
    if (parts.length === 5) return
    for (const part of ['a', 'b', 'x', 'z']) walk([...parts, part])
  }
  walk([])
  for (const a of patterns) for (const b of patterns) {
    assert.equal(documentPathsOverlap(a, b), witnesses.has(JSON.stringify([a, b])), `${a} versus ${b}`)
  }
})
