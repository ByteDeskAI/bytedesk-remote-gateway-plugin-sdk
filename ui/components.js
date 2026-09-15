import * as contracts from "./contracts.js"
export const componentSnapshotCommand = 'components.snapshot.v1'
export const componentInvokeCommand = 'components.invoke.v1'
export const componentChangedEvent = 'components.changed.v1'
const methodsByFamily = Object.freeze({
 workspace: ['discover'], 'session-list': ['select','reorder','group','launch'],
 'session-tab': ['select','rename','pin','close','sleep','wake'],
 terminal: ['focus','reconnect','setViewMode'], stage: ['move','resize','focus','restore'],
 tasks: ['refresh','start'], 'file-tree': ['select','expand','filter','refresh','createFile','createFolder','rename','remove'],
 'project-tools': ['selectView','refresh','setExpanded'],
})
const snapshotValidators = {
 workspace: contracts.isComponentWorkspaceSnapshot, 'session-list': contracts.isComponentSessionListSnapshot,
 'session-tab': contracts.isComponentSessionTabSnapshot, terminal: contracts.isComponentTerminalSnapshot,
 stage: contracts.isComponentStageSnapshot, tasks: contracts.isComponentTasksSnapshot,
 'file-tree': contracts.isComponentFileTreeSnapshot, 'project-tools': contracts.isComponentProjectToolsSnapshot,
}
function validateSnapshot(family, snapshot) {
 if (!snapshotValidators[family](snapshot)) throw new TypeError('Invalid component snapshot')
}
function validateArguments(family, method, args) {
 const text = value => typeof value === 'string'
 const id = value => text(value) && value.length > 0
 const bool = value => typeof value === 'boolean'
 const natural = value => Number.isSafeInteger(value) && value >= 0
 const pair = checks => args.length === checks.length && checks.every((check,i) => check(args[i]))
 let valid = false
 if (family === 'session-list' && method === 'launch') {
  const value = args[0]
  valid = args.length === 1 && value && typeof value === 'object' && !Array.isArray(value) && id(value.kind) && (value.cwd === undefined || text(value.cwd)) && (value.count === undefined || Number.isSafeInteger(value.count) && value.count > 0) && (value.isolate === undefined || bool(value.isolate))
 } else if (method === 'reorder') valid = pair([v => Array.isArray(v) && v.every(id) && new Set(v).size === v.length])
 else if (family === 'stage' && method === 'move') valid = pair([id,v => natural(v) && v <= 1,natural])
 else if (method === 'resize') valid = pair([id,v => v === 1 || v === 2,v => v === 1 || v === 2])
 else if (method === 'setViewMode') valid = pair([v => v === 'terminal' || v === 'chat'])
 else if (method === 'pin' || method === 'setExpanded') valid = pair([bool])
 else if (method === 'expand') valid = pair([text,bool])
 else if (family === 'file-tree' && method === 'rename') valid = pair([id,id])
 else if (['rename','filter','group','selectView','createFile','createFolder','remove'].includes(method)) valid = pair([text])
 else if (method === 'select' && family !== 'session-tab' || method === 'focus' && family === 'stage') valid = pair([id])
 else valid = args.length === 0
 if (!valid) throw new TypeError(`Invalid arguments for ${family}.${method}`)
}
function immutable(value) {
 const copy = structuredClone(value)
 const freeze = item => { if (item && typeof item === 'object') { Object.values(item).forEach(freeze); Object.freeze(item) } }
 freeze(copy); return copy
}
function validateIdentity(identity) {
 if (!identity || !Object.hasOwn(methodsByFamily,identity.family) || !['id','ownerId','generation'].every(key => typeof identity[key] === 'string' && identity[key].length > 0)) throw new TypeError('Invalid component identity')
}
function sameIdentity(a,b) { return ['id','family','ownerId','generation','projectId','sessionId'].every(key => a?.[key] === b?.[key]) }
export function createComponentController({ identity, snapshot, methods = {}, signal }) {
 validateIdentity(identity)
 validateSnapshot(identity.family, snapshot)
 const allowed = methodsByFamily[identity.family]
 for (const [name, method] of Object.entries(methods)) if (!allowed.includes(name) || typeof method !== 'function') throw new TypeError(`Unsupported component method: ${name}`)
 let current = immutable(snapshot), disposed = false
 const listeners = new Set()
 const assertActive = () => { if (disposed || signal?.aborted) throw new Error('Component disposed') }
 const dispose = () => { disposed = true; listeners.clear(); signal?.removeEventListener('abort',dispose) }
 const handle = {
  identity: immutable(identity), capabilities: Object.freeze(Object.keys(methods)),
  getSnapshot: () => { assertActive(); return current },
  subscribe(selector, listener) {
   assertActive()
   let callback
   if (listener) { let previous = selector(current); callback = () => { const next = selector(current); if (!Object.is(previous,next)) { previous = next; listener(next) } } }
   else callback = selector
   listeners.add(callback); return () => listeners.delete(callback)
  }, dispose,
 }
 for (const [name, method] of Object.entries(methods)) handle[name] = async (...args) => { assertActive(); validateArguments(identity.family,name,args); const result = await method(...args); assertActive(); return result }
 signal?.addEventListener('abort',dispose,{once:true})
 if (signal?.aborted) dispose()
 return Object.freeze({ handle: Object.freeze(handle), update(next) { assertActive(); validateSnapshot(identity.family,next); const value = immutable(next); if (JSON.stringify(value) === JSON.stringify(current)) return; current = value; for (const listener of [...listeners]) listener() }, dispose })
}
export async function connectComponent(host, assignment) {
 validateIdentity(assignment?.identity)
 if (typeof assignment.lease !== 'string' || !assignment.lease) throw new TypeError('Host-issued component assignment required')
 const target = immutable(assignment)
 if (host.signal.aborted) throw new Error('Component host disposed')
 // Subscribe before reading so a concurrent update cannot be lost.
 let controller, queued = [], closed = false
 const unsubscribe = host.subscribe(componentChangedEvent, event => {
  if (!sameIdentity(event?.identity,target.identity) || event?.lease !== target.lease || closed) return
  if (!controller) queued.push(event)
  else if (event.withdrawn) dispose()
  else { try { controller.update(event.snapshot) } catch { dispose() } }
 })
 const dispose = () => { if (closed) return; closed=true; unsubscribe(); controller?.dispose(); host.signal.removeEventListener('abort',dispose) }
 host.signal.addEventListener('abort',dispose,{once:true})
 try {
  const result = await host.request(componentSnapshotCommand,{assignment:target})
  if (closed || host.signal.aborted) throw new Error('Component host disposed')
  if (!sameIdentity(result?.identity,target.identity) || !Array.isArray(result.capabilities)) throw new TypeError('Component response identity mismatch')
  const methods = {}
  for (const name of result.capabilities) {
   if (!methodsByFamily[target.identity.family].includes(name)) throw new TypeError('Unsupported component capability')
   methods[name] = (...args) => host.request(componentInvokeCommand,{assignment:target,method:name,args})
  }
  controller = createComponentController({identity:target.identity,snapshot:result.snapshot,methods,signal:host.signal})
  for (const event of queued) { if (event.withdrawn) { dispose(); break } controller.update(event.snapshot) }
  queued = []
  if (closed) throw new Error('Component assignment withdrawn')
  return Object.freeze({...controller.handle,dispose})
 } catch(error) { dispose(); throw error }
}

export const componentAvailableCommand = 'components.available.v1'
export const componentAssignCommand = 'components.assign.v1'
export const componentContributeCommand = 'components.contribute.v1'
/** Listing targets grants no authority; assigning requires host operator consent. */
export async function availableComponents(host) {
 const result = await host.request(componentAvailableCommand,{})
 if (!contracts.isComponentAvailableResult(result)) throw new TypeError('Invalid component listing')
 return immutable(result.components)
}
export async function assignComponent(host, identity) {
 validateIdentity(identity)
 const result = await host.request(componentAssignCommand,{identity})
 if (!contracts.isComponentAssignResult(result) || !sameIdentity(result.assignment.identity, identity) || !result.assignment.lease) throw new TypeError('Invalid component assignment')
 return immutable(result.assignment)
}
export async function contributeComponent(host, assignment, extension) {
 if (!contracts.isComponentAssignment(assignment) || !assignment.lease) throw new TypeError('Host-issued component assignment required')
 const result = await host.request(componentContributeCommand,{assignment,extension})
 if (!contracts.isComponentContributeResult(result)) throw new TypeError('Invalid component contribution')
 return immutable(result.extension)
}
