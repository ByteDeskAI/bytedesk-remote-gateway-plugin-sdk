const decimal = /^(0|[1-9][0-9]*)$/
const record = value => value !== null && typeof value === 'object' && !Array.isArray(value)

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
