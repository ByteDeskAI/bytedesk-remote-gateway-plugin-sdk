# bytedesk-gateway-sdk

Gateway **plugin SDK**. Inherits **all** common objects and plugin requirements
from `bytedesk-sdk-dependencies` (Manifest, Validate, LoadDir, pack, serve,
bus.Envelope, semver). This module only orchestrates gateway host differences:

- env: `GATEWAY_PLUGIN_SOCKET`, `GATEWAY_PLUGIN_ID`
- `plugin.json` `targets` must include `gateway` (empty targets default to gateway)
- authoring CLI + stdio MCP (`cmd/plugin-sdk`)

```go
import gatewaysdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk"

gatewaysdk.Serve(gatewaysdk.Config{Handler: mux})
// Manifest, NavItem, Envelope, … are type aliases of sdk-dependencies
```

Module path stays `bytedesk-remote-gateway-plugin-sdk` until a rename cutover.
The product name is **gateway SDK**.

## Versioning

This SDK’s SemVer (`VERSION`) is independent of `bytedesk-sdk-dependencies`.
`go.mod` `require`s the dependency revision to use; those version numbers
need not match. Bump this repo when the gateway SDK changes. Bump the
`require` when adopting a newer common contract.

Vault plugins use `bytedesk-vault-sdk` (same inherited Manifest, `VAULT_PLUGIN_*`).

## Live contracts prerelease

Version 0.4 re-exports the common runtime model and optional `ActivationChecker`/`Negotiator`. Call `Negotiate` before using any required protocol feature; an older host that lacks the endpoint fails the negotiation instead of silently granting a capability. Existing `Host`/`Plugin` methods and callers remain compatible. Protocol support and granted authority are distinct.

The root npm package `@bytedesk/gateway-plugin-ui` has no framework dependencies. Its shared types are generated from the pinned common Go SDK, and its `PluginUIModule.mount(element, host)` contract returns a cleanup function. The host supplies scoped identity, an abort signal, navigation and brokered request/subscription operations. Each plugin owns its renderer; no private React instance crosses the contract. Privileged in-page modules still require explicit trust; this interface alone is not a sandbox.

Modules using that mount contract require `ui.mount.v1` through protocol
negotiation. `host.location()` exposes the current admitted document location
as raw `pathname`, `search`, `hash` and once-decoded `params`; changes arrive as
`host.location` subscription events with the same location shape. Navigation
accepts an optional `{replace: true}` without exposing the host router. A
plugin-owned React root can bundle its own React and does not share host context.
Hosts must abort the facade, revoke its operations/subscriptions and call mount
cleanup on withdrawal, including cleanup returned after an asynchronous mount
resolves late. The SDK type declaration does not prove a host implements these
behaviors; unsupported hosts must not advertise the capability.

### Terminal presentation owner dispatch

`terminal.presentation.v1` is inherited from common SDK v0.4.0-rc.6. A host may
admit multiple providers for `terminal.presentation.project.v1`; it selects one
owner/provider first and then calls:

```go
result, err := gatewaysdk.DispatchTerminalPresentation(
    ctx,
    gatewaysdk.TerminalPresentationSelection{
        PluginID: pluginID, ProviderID: providerID,
        Generation: generation, Provider: selectedProvider,
    },
    request,
    resolveCurrentAuthorizedTerminals,
)
```

`selectedProvider` implements the canonical aliased
`TerminalPresentationProvider`. The selection identity must exactly match the
request lease. The helper validates the request, applies the canonical two-second
deadline, invokes only that provider instance, resolves current authority after
the provider returns, and validates the complete replacement against the captured
request and current authorized terminal incarnations. It never registers
`TerminalPresentationCommand` in the global `CommandHandler` namespace.

Spawned plugins use `TerminalPresentationHTTPHandler(selection)` as their
`HTTPPlugin.Handler()` (or mount it in that handler). The host selects
the owner-specific plugin socket and POSTs canonical JSON to
`/terminal.presentation.project.v1`. This is the supported external transport;
the adapter intentionally does not implement `CommandHandler`. Because the
spawned process cannot authoritatively resolve the host's current principal set,
the host must decode and validate the returned result against its current
authorized terminal incarnations before admission. In a UI module,
`projectTerminalPresentation(ownerScopedHost, request)` uses the already
owner-scoped `PluginUIHost.request` facade and does not perform provider selection.

Panel `documentPaths` and the Go `ValidateDocumentPath`, `MatchDocumentPath`,
`DocumentPathsOverlap` helpers come from the pinned common SDK. Browser helpers
`isDocumentPath`, `matchDocumentPath`, `documentPathsOverlap` implement the same
grammar and use the pinned shared vectors, checked for drift by Go tests. A
terminal `*name` matches **one or more** segments; `/files` and `/files/*path`
cover root and descendants. Match an escaped pathname, never a whole URL or a
router-decoded value. Parameters are decoded data and must not be decoded or
cleaned again. Root `/` and reserved host infrastructure are not plugin claims.
Manifests declaring document paths must require `ui.document-paths.v1`.

Install the reviewed source tag as an exact git dependency during prerelease integration, or distribute `npm pack` through the established package release channel. No local sibling path is needed in consumer manifests. `npm test` checks lossless revision ordering and untrusted snapshot validation; `go test ./...` verifies RPC negotiation and that `ui/contracts.d.ts` exactly matches the pinned common SDK generator. Runtime snapshot revisions are decimal strings and compare only within the same epoch.


`Requirement.MatchesVersion(actual)` and `ValidateVersionConstraint()` are inherited
from common SDK v0.4.0-rc.4. Hosts must check version compatibility alongside runtime
availability; empty constraints preserve legacy versions. Range syntax and prerelease
behavior are documented by the common SDK. This adds a pinned semantic-version parser
through the common dependency, without extending Host or Plugin methods.
