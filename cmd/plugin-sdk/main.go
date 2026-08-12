package main

import (
	"fmt"
	"os"

	pluginsdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk"
)

const usage = `usage: plugin-sdk validate|pack|mcp|version [flags]

  validate --dir DIR
  pack --dir DIR --out DIR
  mcp                 stdio authoring MCP (validate + pack tools)
  version
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	switch cmd {
	case "version", "-v", "--version":
		fmt.Println(readVersion())
	case "validate":
		dir := flagVal(args, "--dir", ".")
		m, err := pluginsdk.ValidateDir(dir)
		if err != nil {
			die(err)
		}
		fmt.Printf("ok id=%s version=%s\n", m.ID, m.Version)
	case "pack":
		dir := flagVal(args, "--dir", ".")
		out := flagVal(args, "--out", "dist")
		res, err := pluginsdk.PackDir(dir, out)
		if err != nil {
			die(err)
		}
		fmt.Printf("wrote %s id=%s version=%s unsigned=%v\n", res.Archive, res.ID, res.Version, res.Unsigned)
	case "mcp":
		if err := runAuthoringMCP(); err != nil {
			die(err)
		}
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
}

func flagVal(args []string, name, def string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return def
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "plugin-sdk: %v\n", err)
	os.Exit(1)
}

func readVersion() string {
	return "0.1.0"
}
