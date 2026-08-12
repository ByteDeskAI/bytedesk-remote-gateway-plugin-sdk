# bytedesk-remote-gateway-plugin-sdk

Serve, validate, and pack **gateway process plugins**. Host ABI is unix-socket
HTTP (`GATEWAY_PLUGIN_SOCKET`). Authoring MCP is optional (`plugin-sdk mcp`).

```go
import pluginsdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk"

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/ui", ui)
    log.Fatal(pluginsdk.Serve(pluginsdk.Config{Handler: mux}))
}
```

```bash
plugin-sdk validate --dir .
plugin-sdk pack --dir . --out dist
plugin-sdk mcp   # stdio authoring tools
```

Depends on `bytedesk-sdk-dependencies`. See gateway ADR 0014.
