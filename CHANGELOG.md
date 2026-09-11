# Changelog

## [Unreleased]

### Added

- `ProfileCommand` (`GET /cmd.profile.v1.cpu?seconds=N`, 1–60): every plugin served by `ServePlugin` now answers a host request to capture a CPU profile of its own process. A spawned plugin has its own Go runtime, so this is the only way the gateway can profile it (gateway TM-285). The route is a plain path check in front of the plugin's handler, so no other routing changes, and a plugin without a handler keeps `/healthz`. The `cmd.` prefix marks it host-only; the gateway refuses browser requests to `/p/<id>/cmd.*`. Untagged: iterated on a pseudo-version and released with the single EP-019 SDK tag.
- Re-exports of the common SDK's lifecycle hook constants and `DeclaredHooks`.

### Changed

- **Breaking:** settings section commands are `cmd.host.settings.section.v1.snapshot` and `cmd.host.settings.section.v1.patch`, following the common SDK's renamed `SettingsSectionPoint`.
- `ServePlugin` advertises the lifecycle hooks a plugin implements and serves `POST /cmd.lifecycle.v1.hook` for those the host acknowledges, without running them locally. Against an older host that rejects the hook set, it retries negotiation without hooks and runs them locally as before.
- Regenerated `ui/contracts.d.ts` from the combined common SDK (typed settings field schema and lifecycle hook fields).
- Consumes `bytedesk-sdk-dependencies` by pseudo-version `v0.4.0-rc.8.0.20260911150628-cc6fc9b776e6` while the contract iterates untagged.

## [0.4.0-rc.9] - 2026-09-10

- Added optional configured agent identity fields to terminal presentation items.
- Updated the common SDK pin to `v0.4.0-rc.8`.

## [0.4.0-rc.8] - 2026-09-10

### Added

- `messaging` package re-exporting the common SDK's asynchronous messaging contract, so a plugin can consume it through the gateway SDK alone.
- `ui/contracts.js`, the generated TypeScript runtime validators, added to the published `files` list. `ui/index.d.ts` has always declared `export * from './contracts.js'`; until now that resolved to the type declarations because the package was types-only.

### Changed

- Pinned `bytedesk-sdk-dependencies` to `v0.4.0-rc.7`, which brings the typed capability layer, per-package classification unions and `bd.schema-id.v1`.
- Regenerated `ui/contracts.d.ts` from the newly pinned generator: byte-identical, so no consumer renegotiates.

### Notes

- `isRuntimeSnapshot` in `ui/index.js` deliberately shadows the generated validator of the same name. Only the local one enforces `available => installed && generation && desiredState === 'enabled'`, the invariant the gateway relies on to fail closed on optional-plugin availability. A structural validator cannot derive it.

## [0.4.0-rc.7] - 2026-09-09

### Added

- Adopt canonical `terminal.presentation.v1` types, validators, generated TypeScript declarations and conformance vectors from common SDK v0.4.0-rc.6.
- Selected-owner Go dispatch and spawned-plugin HTTP adapters for `terminal.presentation.project.v1`, plus an owner-scoped browser request helper. Multiple providers do not compete for a global `CommandHandler` name.

## [0.4.0-rc.6] - 2026-09-09

### Added

- Adopt canonical document-path contributions and helpers from common SDK v0.4.0-rc.5, with matching browser helpers and pinned parity vectors.
- Framework-independent UI location and replace-navigation ports for the negotiated module mount contract; no host router or rendering context crosses the SDK boundary.

## [0.4.0-rc.5] - 2026-09-09

### Added

- Adopt common SDK v0.4.0-rc.4 and expose canonical required-peer version matching through the existing Requirement alias. Malformed manifest version constraints are rejected by shared validation.

## [0.4.0-rc.4] - 2026-09-08

### Added

- Re-export canonical unknown desired state and validate unavailable recovery snapshots in the UI SDK. Unknown intent can never grant availability.

## [0.4.0-rc.3] - 2026-09-08

### Fixed

- Callback stream admission rejects HTTP errors, respects caller cancellation and permits retry after failed admission. Concurrent startup waits for readiness.
- Host RPC calls have a five-second upper deadline and reject redirects. Empty NewHost socket reads GATEWAY_HOST_SOCKET as documented.

## [Unreleased]

### Added

- **Settings sections for spawned plugins.** `SettingsSection`, `SettingsSectionHTTPHandler` and `MountSettingsSection` let a plugin serve `cmd.settings.section.v1.snapshot` and `cmd.settings.section.v1.patch` on its own socket, which the gateway bridges onto its settings API. `SettingsSectionPoint` re-exports the common SDK constant. A PATCH body is bounded at 64 KiB, a read-only section returns `ErrSettingsReadOnly` and is answered 405, and any method but POST is refused.
- **`ExtensionRegistrar`.** The spawned-plugin host implements `RegisterExtension(ctx, point, id, providerID)`, which asks the host to admit a live provider at a point the plugin declared in `Manifest.Implements`. The host still decides: the point must be open to the bridge, declared in the admitted manifest, and consented to for an installed plugin.

### Changed

- Consumes `bytedesk-sdk-dependencies` by pseudo-version (`v0.4.0-rc.8.0.20260911022014-d1fb318b7a44`) while the contract iterates untagged, and regenerates `ui/contracts.d.ts` from it (adds `Config` and `ConfigSection`).


## [0.4.0-rc.2] - 2026-09-08

### Added

- ServePlugin runs the SDK Plugin lifecycle, negotiates required features, rolls back failed starts and activation, and performs bounded-deadline shutdown.

### Fixed

- Adopt canonical browser declarations with normalized file endings from common SDK v0.4.0-rc.2.

## [0.4.0-rc.1] - 2026-09-08

### Added

- Re-export live runtime, activation, permissions, protocol and UI contribution contracts from common SDK v0.4.0-rc.1.
- Optional bounded RPC negotiation rejects unsupported required features and missing generation identity.
- Portable @bytedesk/gateway-plugin-ui package with generated shared declarations, module mount/cleanup contract and lossless snapshot validation.

## [0.3.0] - 2026-09-06

### Added

- Re-export the plugin contract: `Plugin`, `Host`, `Logger`, and the optional
  `HTTPPlugin`, `HealthContributor`, `CommandHandler`, `Validator`, `Readier`.
  A plugin is written once against these; whether the host links it in or spawns
  it is the host's decision, not the author's.
- `rpcHost` and `EnvHostSocket` — a `plugin.Host` for a spawned plugin, carrying
  each call to the host over `GATEWAY_HOST_SOCKET`. The plugin dials once and
  holds a streaming connection open for callbacks, so a dropped connection ends
  every subscription and timer on it with no liveness tracking on either side.
- `HostStarter` / `HostCloser`, with `LastFault()` and `Counters()`. `Subscribe`
  and `Every` return only a cancel function and so have no way to report a
  failure; without these a dropped registration is indistinguishable from an
  event that never happened.

### Changed

- Registration ids are generated by the plugin and sent to the host, not
  assigned by the host and returned. The host begins delivering as soon as it
  registers, so a host-assigned id left a window in which a callback could
  arrive before this side knew which handler it belonged to.
- require `bytedesk-sdk-dependencies v0.3.0`

### Changed

- docs: SDK SemVer is independent of `sdk-dependencies`; `go.mod` `require` is the pin
- require `bytedesk-sdk-dependencies v0.1.3` (role / requires / trial)

## [0.1.3] - 2026-08-13

### Changed

- require `bytedesk-sdk-dependencies v0.1.3`; re-export `RoleSystem`, `RoleExtension`, `Requirement`

## [0.1.2] - 2026-08-12

### Changed

- Inherit Manifest, pack, serve, bus, and semver from `bytedesk-sdk-dependencies@v0.1.2`
- `ValidateDir` / `PackDir` require `targets` include `gateway` via `LoadDirForHost`
- `Serve` wraps common unix Serve and reads `GATEWAY_PLUGIN_SOCKET` / `GATEWAY_PLUGIN_ID`

### Added

- Type aliases for the common plugin contract (`Manifest`, `Envelope`, …)
