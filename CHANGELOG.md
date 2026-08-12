# Changelog

## [Unreleased]

### Changed

- docs: SDK SemVer is independent of `sdk-dependencies`; `go.mod` `require` is the pin

## [0.1.2] - 2026-08-12

### Changed

- Inherit Manifest, pack, serve, bus, and semver from `bytedesk-sdk-dependencies@v0.1.2`
- `ValidateDir` / `PackDir` require `targets` include `gateway` via `LoadDirForHost`
- `Serve` wraps common unix Serve and reads `GATEWAY_PLUGIN_SOCKET` / `GATEWAY_PLUGIN_ID`

### Added

- Type aliases for the common plugin contract (`Manifest`, `Envelope`, …)
