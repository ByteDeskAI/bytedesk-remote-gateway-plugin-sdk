export * from './contracts.js'
import type { HostCapabilities, RuntimeSnapshot } from './contracts.js'

/** Compare lossless nonnegative decimal revisions within one host epoch. */
export function compareRuntimeRevision(a: string, b: string): -1 | 0 | 1
/** Validate untrusted bootstrap/message data before using availability. */
export function isRuntimeSnapshot(value: unknown): value is RuntimeSnapshot

/** Canonical document pattern: literals, :name, terminal one-or-more *name. */
export function isDocumentPath(value: unknown): value is string
/** Decode an escaped pathname exactly once; parameters are data, not URLs. */
export function matchDocumentPath(pattern: string, escapedPath: string): Readonly<Record<string, string>> | null
/** Throws TypeError for invalid declarations. */
export function documentPathsOverlap(a: string, b: string): boolean

/** Framework-independent location of an admitted panel document. */
export interface PluginRouteLocation {
  readonly pathname: string
  /** Original query and fragment, including their leading ? and # if present. */
  readonly search: string
  readonly hash: string
  /** SDK document-path parameters, decoded exactly once. */
  readonly params: Readonly<Record<string, string>>
}

/** Every method is backed by owner/generation-scoped host authority. */
export interface PluginUIHost {
  readonly identity: Readonly<HostCapabilities>
  readonly signal: AbortSignal
  /** Current host-owned route. Changes are delivered as host.location events. */
  location(): Readonly<PluginRouteLocation>
  navigate(path: string, options?: { readonly replace?: boolean }): void
  request(command: string, payload: unknown): Promise<unknown>
  subscribe(eventType: string, handler: (payload: unknown) => void): () => void
}

/** A module owns its rendering runtime. No private host React instance is passed. */
export interface PluginUIModule {
  mount(element: HTMLElement, host: PluginUIHost): (() => void) | Promise<() => void>
}
