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

Install the reviewed source tag as an exact git dependency during prerelease integration, or distribute `npm pack` through the established package release channel. No local sibling path is needed in consumer manifests. `npm test` checks lossless revision ordering and untrusted snapshot validation; `go test ./...` verifies RPC negotiation and that `ui/contracts.d.ts` exactly matches the pinned common SDK generator. Runtime snapshot revisions are decimal strings and compare only within the same epoch.


`Requirement.MatchesVersion(actual)` and `ValidateVersionConstraint()` are inherited
from common SDK v0.4.0-rc.4. Hosts must check version compatibility alongside runtime
availability; empty constraints preserve legacy versions. Range syntax and prerelease
behavior are documented by the common SDK. This adds a pinned semantic-version parser
through the common dependency, without extending Host or Plugin methods.
