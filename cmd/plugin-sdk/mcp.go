package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	pluginsdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk"
)

// Minimal JSON-RPC stdio MCP for authoring (not the host ABI).
type rpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func runAuthoringMCP() error {
	in := bufio.NewReader(os.Stdin)
	enc := json.NewEncoder(os.Stdout)
	for {
		line, err := in.ReadBytes('\n')
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		var req rpcReq
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}
		switch req.Method {
		case "initialize":
			_ = enc.Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]any{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]any{"tools": map[string]any{}},
					"serverInfo":      map[string]any{"name": "plugin-sdk", "version": readVersion()},
				},
			})
		case "notifications/initialized", "initialized":
			// no reply
		case "tools/list":
			_ = enc.Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]any{
					"tools": []map[string]any{
						{"name": "plugin_validate", "description": "Validate plugin.json + spawn binary in --dir", "inputSchema": map[string]any{
							"type": "object", "properties": map[string]any{"dir": map[string]any{"type": "string"}},
						}},
						{"name": "plugin_pack", "description": "Pack a plugin dir to <id>-<version>.tar.gz", "inputSchema": map[string]any{
							"type": "object", "properties": map[string]any{
								"dir": map[string]any{"type": "string"},
								"out": map[string]any{"type": "string"},
							},
						}},
					},
				},
			})
		case "tools/call":
			var p struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			}
			_ = json.Unmarshal(req.Params, &p)
			dir := "."
			out := "dist"
			if p.Arguments != nil {
				if v, ok := p.Arguments["dir"].(string); ok && v != "" {
					dir = v
				}
				if v, ok := p.Arguments["out"].(string); ok && v != "" {
					out = v
				}
			}
			text := ""
			isErr := false
			switch p.Name {
			case "plugin_validate":
				m, err := pluginsdk.ValidateDir(dir)
				if err != nil {
					text, isErr = err.Error(), true
				} else {
					text = fmt.Sprintf("ok id=%s version=%s", m.ID, m.Version)
				}
			case "plugin_pack":
				res, err := pluginsdk.PackDir(dir, out)
				if err != nil {
					text, isErr = err.Error(), true
				} else {
					text = fmt.Sprintf("wrote %s", res.Archive)
				}
			default:
				text, isErr = "unknown tool", true
			}
			_ = enc.Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]any{
					"content":           []map[string]any{{"type": "text", "text": text}},
					"isError":           isErr,
					"structuredContent": map[string]any{"text": text},
				},
			})
		default:
			if req.ID != nil {
				_ = enc.Encode(map[string]any{
					"jsonrpc": "2.0",
					"id":      req.ID,
					"error":   map[string]any{"code": -32601, "message": "method not found: " + req.Method},
				})
			}
		}
	}
}
