const decimal = /^(0|[1-9][0-9]*)$/
const record = value => value !== null && typeof value === 'object' && !Array.isArray(value)

function documentSegments(pattern) {
  if (typeof pattern !== 'string' || !pattern.startsWith('/') || pattern === '/') return null
  const parts = pattern.slice(1).split('/')
  const names = new Set()
  const result = []
  for (const [i, part] of parts.entries()) {
    if (!part || part === '.' || part === '..') return null
    const kind = part[0] === ':' || part[0] === '*' ? part[0] : ''
    const value = kind ? part.slice(1) : part
    if (kind) {
      if (!/^[A-Za-z]/.test(value) || /[^A-Za-z0-9_]/.test(value) || names.has(value) || (kind === '*' && i !== parts.length - 1)) return null
      names.add(value)
    } else if (/[^A-Za-z0-9._~-]/.test(value)) return null
    result.push({ kind, value })
  }
  return result
}

/** Canonical SDK pattern grammar, including a terminal one-or-more *name. */
export function isDocumentPath(pattern) {
  return documentSegments(pattern) !== null
}

/** Input is an escaped pathname; returned parameters are decoded data. */
export function matchDocumentPath(pattern, escapedPath) {
  const segments = documentSegments(pattern)
  if (!segments || typeof escapedPath !== 'string' || !escapedPath.startsWith('/') || /[?#\\]/.test(escapedPath)) return null
  const parts = escapedPath.slice(1).split('/')
  if (parts.length < segments.length || (segments.at(-1).kind !== '*' && parts.length !== segments.length)) return null
  try {
    for (let i = 0; i < parts.length; i++) {
      const value = decodeURIComponent(parts[i])
      if (!value || value === '.' || value === '..' || /[/\\\u0000-\u001f\u007f-\u009f\uD800-\uDFFF]/u.test(value)) return null
      parts[i] = value
    }
  } catch {
    return null
  }
  const params = {}
  for (const [i, segment] of segments.entries()) {
    if (segment.kind === ':') params[segment.value] = parts[i]
    else if (segment.kind === '*') params[segment.value] = parts.slice(i).join('/')
    else if (segment.value !== parts[i]) return null
  }
  return params
}

/** Reject invalid patterns instead of treating them as safe non-overlap. */
export function documentPathsOverlap(a, b) {
  const left = documentSegments(a)
  const right = documentSegments(b)
  if (!left || !right) throw new TypeError('invalid document path')
  if ((left.at(-1).kind !== '*' && left.length < right.length) || (right.at(-1).kind !== '*' && right.length < left.length)) return false
  for (let i = 0; i < Math.min(left.length, right.length); i++) {
    if (left[i].kind === '*' || right[i].kind === '*') return true
    if (!left[i].kind && !right[i].kind && left[i].value !== right[i].value) return false
  }
  return true
}

export function compareRuntimeRevision(a, b) {
  if (typeof a !== 'string' || typeof b !== 'string' || !decimal.test(a) || !decimal.test(b)) {
    throw new TypeError('runtime revisions must be canonical nonnegative decimal strings')
  }
  if (a === b) return 0
  return a.length !== b.length ? (a.length > b.length ? 1 : -1) : (a > b ? 1 : -1)
}

export function isRuntimeSnapshot(value) {
  if (!record(value) || typeof value.epoch !== 'string' || !value.epoch ||
      typeof value.revision !== 'string' || !decimal.test(value.revision) ||
      !(value.plugins === null || Array.isArray(value.plugins))) return false
  const seen = new Set()
  for (const plugin of value.plugins || []) {
    if (!record(plugin) || typeof plugin.id !== 'string' || !plugin.id || seen.has(plugin.id) ||
        typeof plugin.installed !== 'boolean' || typeof plugin.available !== 'boolean' ||
        !['unknown', 'absent', 'disabled', 'enabled'].includes(plugin.desiredState) ||
        typeof plugin.observedState !== 'string' || !plugin.observedState ||
        typeof plugin.generation !== 'string' ||
        (plugin.reason !== undefined && typeof plugin.reason !== 'string') ||
        (plugin.available && (!plugin.installed || !plugin.generation || plugin.desiredState !== 'enabled'))) return false
    seen.add(plugin.id)
  }
  return true
}
