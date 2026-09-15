import type { ComponentIdentity, ComponentAssignment, ComponentWorkspaceSnapshot, ComponentSessionListSnapshot, ComponentSessionTabSnapshot, ComponentTerminalSnapshot, ComponentStageSnapshot, ComponentTasksSnapshot, ComponentFileTreeSnapshot, ComponentProjectToolsSnapshot } from './contracts.js'
import type { PluginUIHost } from './index.js'
export interface ComponentSnapshotMap {
 workspace: ComponentWorkspaceSnapshot
 'session-list': ComponentSessionListSnapshot
 'session-tab': ComponentSessionTabSnapshot
 terminal: ComponentTerminalSnapshot
 stage: ComponentStageSnapshot
 tasks: ComponentTasksSnapshot
 'file-tree': ComponentFileTreeSnapshot
 'project-tools': ComponentProjectToolsSnapshot
}
export type ComponentFamily = keyof ComponentSnapshotMap
export interface ComponentCommandOptions { readonly signal?: AbortSignal }
export interface ComponentLaunchOptions { readonly kind: string; readonly cwd?: string; readonly count?: number; readonly isolate?: boolean }
export interface ComponentMethodsMap {
 workspace: { discover(): Promise<readonly ComponentIdentity[]> }
 'session-list': { select(id: string): Promise<void>; reorder(ids: readonly string[]): Promise<void>; group(group: string): Promise<void>; launch(options: ComponentLaunchOptions): Promise<void> }
 'session-tab': { select(): Promise<void>; rename(title: string): Promise<void>; pin(pinned: boolean): Promise<void>; close(): Promise<void>; sleep(): Promise<void>; wake(): Promise<void> }
 terminal: { focus(): Promise<void>; reconnect(): Promise<void>; setViewMode(mode: 'terminal' | 'chat'): Promise<void> }
 stage: { move(sessionId: string, column: number, row: number): Promise<void>; resize(sessionId: string, width: 1 | 2, height: 1 | 2): Promise<void>; focus(sessionId: string): Promise<void>; restore(): Promise<void> }
 tasks: { refresh(): Promise<void>; start(): Promise<void> }
 'file-tree': { select(path: string): Promise<void>; expand(path: string, expanded: boolean): Promise<void>; filter(value: string): Promise<void>; refresh(): Promise<void>; createFile(path: string): Promise<void>; createFolder(path: string): Promise<void>; rename(path: string, newPath: string): Promise<void>; remove(path: string): Promise<void> }
 'project-tools': { selectView(view: string): Promise<void>; refresh(): Promise<void>; setExpanded(expanded: boolean): Promise<void> }
}
export type ComponentHandle<F extends ComponentFamily> = Readonly<{
 identity: ComponentIdentity & { readonly family: F }
 capabilities: readonly (keyof ComponentMethodsMap[F])[]
 getSnapshot(): ComponentSnapshotMap[F]
 subscribe(listener: () => void): () => void
 subscribe<T>(selector: (snapshot: ComponentSnapshotMap[F]) => T, listener: (value: T) => void): () => void
 dispose(): void
}> & Partial<ComponentMethodsMap[F]>
export interface ComponentController<F extends ComponentFamily> {
 readonly handle: ComponentHandle<F>
 update(snapshot: ComponentSnapshotMap[F]): void
 dispose(): void
}
export function createComponentController<F extends ComponentFamily>(options: {
 readonly identity: ComponentIdentity & { readonly family: F }
 readonly snapshot: ComponentSnapshotMap[F]
 readonly methods?: Partial<ComponentMethodsMap[F]>
 readonly signal?: AbortSignal
}): ComponentController<F>
/** Assignment must be host-issued; an identity is not a grant. Host validates every invocation. */
export function connectComponent<F extends ComponentFamily>(host: Pick<PluginUIHost, 'request' | 'subscribe' | 'signal'>, assignment: ComponentAssignment & { readonly identity: ComponentIdentity & {readonly family: F} }): Promise<ComponentHandle<F>>
export const componentSnapshotCommand: 'components.snapshot.v1'
export const componentInvokeCommand: 'components.invoke.v1'
export const componentChangedEvent: 'components.changed.v1'

export const componentAvailableCommand: 'components.available.v1'
export const componentAssignCommand: 'components.assign.v1'
export const componentContributeCommand: 'components.contribute.v1'
export function availableComponents(host: Pick<PluginUIHost, 'request'>): Promise<readonly ComponentIdentity[]>
export function assignComponent(host: Pick<PluginUIHost, 'request'>, identity: ComponentIdentity): Promise<ComponentAssignment>
export function contributeComponent(host: Pick<PluginUIHost, 'request'>, assignment: ComponentAssignment, extension: Omit<import('./contracts.js').ComponentExtension, 'ownerId' | 'generation'>): Promise<import('./contracts.js').ComponentExtension>
