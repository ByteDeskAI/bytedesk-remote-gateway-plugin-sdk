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

Vault plugins use `bytedesk-vault-sdk` (same inherited Manifest, `VAULT_PLUGIN_*`).
