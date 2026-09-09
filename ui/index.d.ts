export * from './contracts.js'
import type { HostCapabilities, RuntimeSnapshot } from './contracts.js'

/** Compare lossless nonnegative decimal revisions within one host epoch. */
export function compareRuntimeRevision(a: string, b: string): -1 | 0 | 1
/** Validate untrusted bootstrap/message data before using availability. */
export function isRuntimeSnapshot(value: unknown): value is RuntimeSnapshot

/** Every method is backed by owner/generation-scoped host authority. */
export interface PluginUIHost {
  readonly identity: Readonly<HostCapabilities>
  readonly signal: AbortSignal
  navigate(path: string): void
  request(command: string, payload: unknown): Promise<unknown>
  subscribe(eventType: string, handler: (payload: unknown) => void): () => void
}

/** A module owns its rendering runtime. No private host React instance is passed. */
export interface PluginUIModule {
  mount(element: HTMLElement, host: PluginUIHost): (() => void) | Promise<() => void>
}
