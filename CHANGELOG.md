# Changelog

## [Unreleased]

### Added

- **SDK v2 (`v2/`), a new module beside v1.** `github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2` re-exports the `bytedesk-sdk-dependencies/v2` contract and adds this side of it: the v2 handshake, the NATS transport and the packaging tool. v1 keeps building and keeps receiving fixes.
- `v2/transport/natsconn` — the NATS transport, and the only package in either SDK that imports `github.com/nats-io/*`. A unix custom dialer (a contained plugin has no network namespace, so TCP is impossible by construction), `UserCredentials`, `CustomInboxPrefix("_INBOX.<id>")` so a reply to one plugin can never reach another, unlimited reconnect, and an error handler that maps a broker's asynchronous `Permissions Violation` onto `FaultDenied` — on the subscription for a subscribe refusal, and on the next call for that subject for a publish refusal, which a core publish can only report out of band.
- `v2/plugin_serve.go` — `ServePlugin` runs the whole ordered handshake: identity, manifest, a ≤5s wait for `GATEWAY_BUS_CREDS`, dial, `cmd.plugin.v1.negotiate`, `CheckProtocol` (fail-closed on major, required features and `needs`), `Bind`, `Validate`, `Start`, lifecycle endpoints, HTTP. HTTP routes on `GATEWAY_PLUGIN_SOCKET` are unchanged from v1.
- Lifecycle hooks are service endpoints, not negotiated callbacks: `svc.<id>.lifecycle.v1.{activation.check,ready,health}`, mounted for whichever optional interfaces the plugin implements.
- `v2/cmd/plugin-sdk` — `pack` prints the grants digest an operator's consent is keyed by, and `digest` prints it without packing. The CLI's version is the embedded `VERSION` file rather than a constant that had drifted four releases from it.
- `v2/v1compat` re-exports the common SDK's v1 `Host` facade over a bound v2 `Base`, so a v1 plugin runs unchanged while it is converted.
- `v2/internal/fakebroker` — a NATS peer spoken by hand over a unix socket, so the SDK's own tests can watch a real CONNECT be refused without depending on `nats-server` and without putting a broker import outside the transport.

### Changed

- **Breaking, v2 only.** `Protocol.Major` is 2. `Start` and `Validate` take no host argument: `Bind` has already installed the generation's bus, logger, profiler, state dir and identity. `Host`, `Kit`, `CommandHandler`, `StatusSubscriber`, `ObservableRegistrar`, `Negotiator` and `ExtensionRegistrar` are retired. Every `dev-grants.json` entry re-approves once, because the digest now covers `serves`, `streams`, `kv`, `objects` and `needs`.
- A spawned v2 plugin logs to stderr, which the host already captures. v1 forwarded log records over the host RPC wire; v2 has no such wire, and routing them over the bus would drop a line whenever nothing happened to be subscribed.

### Known gaps

- `natsconn` implements core pub/sub, request/reply, services and correlation. Streams, KV, objects and schedules are reported **false** by `Capabilities()`, so a manifest that needs one fails closed at the handshake naming the capability rather than at first use. The JetStream surfaces land with the durable re-platforms.
- A spawned v2 plugin has no host-driven profiling switch: the subject that carries it is the gateway's to define when it adopts v2.

## [0.4.0-rc.17] - 2026-09-16

### Added

- Re-export `Profiler` and `NopProfiler`. `rpcHost.Profiling()` reads and writes the host RPC `/profiling` switch so a spawned plugin can enable or disable its own profiler at runtime.

### Changed

- Adopt common SDK `v0.4.0-rc.16` for `Host.Profiling()`. Host implementations must satisfy the new method.

## [0.4.0-rc.16] - 2026-09-15

### Added

- Carry one host-minted subject lease from a spawned plugin's private HTTP request context to `Host.Request` through `X-Bytedesk-Subject-Lease`. The lease is never added to the bus envelope or forwarded by publish, subscribe, timer, negotiation, logging or extension calls. Missing headers retain autonomous behavior for older hosts; malformed or duplicate present values remain distinguishable so the host can refuse them instead of silently downgrading authority.
- Re-export the common SDK's bounded, fixed-format tmux availability, session, window and pane read contracts and generated descriptors.

### Changed

- Adopt common SDK `v0.4.0-rc.15` for the subject-classified tmux read contract.

## [0.4.0-rc.15] - 2026-09-15

### Added

- `HeaderCorrelationID`, `CorrelationID`, and `LoggerForRequest` bind one canonical host-minted request id to structured plugin logs. The id crosses spawned-plugin RPC in the existing log arguments, while missing headers from older hosts keep the ordinary logger unchanged.

### Changed

- Adopt common SDK `v0.4.0-rc.14` for the additive correlation logger helper.

## [0.4.0-rc.14] - 2026-09-15

### Fixed

- Assigned component clients validate host revisions and ignore stale or duplicate change events, including events racing the first snapshot read. Invalid revisions withdraw clients; queued withdrawal and host cancellation retain cleanup guarantees.

## [0.4.0-rc.13] - 2026-09-15

### Added

- Framework-independent component controllers, readonly typed handles, selective subscriptions and disposal for eight core component families. Optional named methods advertise only implemented capabilities.
- Assigned component clients over the negotiated UI host, with exact identity checks, structural snapshot validation, withdrawal cleanup and explicit host-issued leases. Additive contribution helpers preserve host-owned contributor identity; no DOM, renderer or terminal I/O is exposed.

## [0.4.0-rc.12] - 2026-09-15

### Added

- Re-export manifest settings types as `ManifestConfig`, `ConfigSection` and `ConfigField`, all seven kind constants including `ConfigKindProvider`, and `ConfigFieldsFromStruct`. `Config` remains the server configuration type. Provider declarations name an extension point and required capabilities; the host owns registry resolution and selection validation.
- Re-export canonical contribution-role eligibility types, constants and lookup/authorization helpers. The host supplies compiled provenance and explicit per-point consent; declarations alone never grant either.

- **Host-attested external HTTP origin (gateway TM-331).** `HeaderExternalOrigin` and `ExternalOrigin` read exactly one canonical HTTP/HTTPS origin from the authenticated private host transport. They reject ambiguous or malformed values and never fall back to client forwarding headers. The host must strip client copies and stamp its validated origin; this helper alone neither authenticates a request nor enables a LAN-origin fallback.

### Changed

- Adopt released common SDK `v0.4.0-rc.11` and regenerate both browser contract files. Applications scan and registration schema hashes change with the subject-classified resolved executable path; coordinate host and plugin adoption rather than assuming optional JSON fields preserve typed-schema compatibility.

### Fixed

- Allocate active Unix-socket test fixtures in short private directories so the complete suite also runs under long CI temporary paths. Verify permissions and cleanup without changing production transport.

- Verify generated browser JavaScript against the pinned common SDK alongside TypeScript declarations, so validator drift fails the SDK test gate.
- Synchronize lifecycle negotiation test recorder reads with its existing mutex. This removes a race in test bookkeeping without changing plugin lifecycle behavior.

## [0.4.0-rc.11] - 2026-09-13

### Added

- Re-exports the common SDK's asynchronous, paged `cmd.desktop-applications.v2.scan` contract, generated browser declarations and structural validator. The v1 scan contract remains available unchanged (gateway TM-331).

### Changed

- Pins `bytedesk-sdk-dependencies` `v0.4.0-rc.10`.

## [0.4.0-rc.10] - 2026-09-13

### Added

- Re-exports Applications catalog metadata from `bytedesk-sdk-dependencies` `v0.4.0-rc.9`: optional launcher path, RFC3339 install time, estimated-time marker, and icon URL in both Go and generated browser contracts (gateway TM-331).
- Re-exports the common SDK's typed `desktop-applications` host-service commands, payloads and generated TypeScript contract for contained Applications consumers (gateway TM-326).

- `ProfileCommand` (`GET /cmd.profile.v1.cpu?seconds=N`, 1–60): every plugin served by `ServePlugin` now answers a host request to capture a CPU profile of its own process. A spawned plugin has its own Go runtime, so this is the only way the gateway can profile it (gateway TM-285). The route is a plain path check in front of the plugin's handler, so no other routing changes, and a plugin without a handler keeps `/healthz`. The `cmd.` prefix marks it host-only; the gateway refuses browser requests to `/p/<id>/cmd.*`. Untagged: iterated on a pseudo-version and released with the single EP-019 SDK tag.
- Re-exports of the common SDK's lifecycle hook constants and `DeclaredHooks`.
- Re-exports of the common SDK's plugin Kit: `Kit`, `NewKitFrom`, `GrantKind`, `GrantPublish`, `GrantSubscribe` and `GrantRequest`. `NewKitFrom` does not negotiate, and `Can` reads the cached grants while the host still enforces every call.
- Re-exports of the common SDK's optional interfaces, which sit beside `Host` and `Plugin` rather than extending either: `ObservableRegistrar` (`SubscribeErr`, `EveryErr`) makes a refused registration visible instead of silent, `Draining` (`OnDrain`) runs while the bus, timers and state dir still work and may not veto teardown, and `DataVersioned` plus `DataVersion` record which version last wrote this plugin's state dir. Type-assert for what you need; an older host does not implement them and the plugin keeps running.

### Changed

- **Breaking:** UI contribution slots name a role, following the common SDK (gateway ADR 0026 D1, TM-255). `SlotToolbar`, `SlotOverlay` and `SlotBadge` are removed. `SlotMainNavigation`, `SlotSubNavigation`, `SlotPrimaryAction`, `SlotSecondaryActions`, `SlotStatusIndicator`, `SlotObjectActions` and `SlotLauncher` are re-exported. To migrate, replace `toolbar` with `primary-action` and `badge` with `status-indicator`. `overlay` has no replacement.
- **Breaking:** settings section commands are `cmd.host.settings.section.v1.snapshot` and `cmd.host.settings.section.v1.patch`, following the common SDK's renamed `SettingsSectionPoint`.
- `ServePlugin` advertises the lifecycle hooks a plugin implements and serves `POST /cmd.lifecycle.v1.hook` for those the host acknowledges, without running them locally. Against an older host that rejects the hook set, it retries negotiation without hooks and runs them locally as before.
- Regenerated `ui/contracts.d.ts` from the combined common SDK (typed settings field schema and lifecycle hook fields).
- Pins the released `bytedesk-sdk-dependencies` `v0.4.0-rc.9` contract.

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
