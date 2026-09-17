package main

import (
	"fmt"
	"os"

	pluginsdk "github.com/ByteDeskAI/bytedesk-remote-gateway-plugin-sdk/v2"
)

const usage = `usage: plugin-sdk validate|pack|digest|mcp|version [flags]

  validate --dir DIR
  pack --dir DIR --out DIR   builds the archive and prints the grants digest
  digest --dir DIR           prints the grants digest without packing
  mcp                        stdio authoring MCP (validate + pack tools)
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
		m, err := pluginsdk.ValidateDir(flagVal(args, "--dir", "."))
		if err != nil {
			die(err)
		}
		fmt.Printf("ok id=%s version=%s\n", m.ID, m.Version)
	case "digest":
		m, err := pluginsdk.ValidateDir(flagVal(args, "--dir", "."))
		if err != nil {
			die(err)
		}
		fmt.Println(pluginsdk.GrantsDigest(m))
	case "pack":
		dir := flagVal(args, "--dir", ".")
		out := flagVal(args, "--out", "dist")
		res, err := pluginsdk.PackDir(dir, out)
		if err != nil {
			die(err)
		}
		// The digest is printed by the PACKER on purpose: an operator's
		// consent is keyed by it, and a digest printed by anything other than
		// the tool that built the archive is a digest of something else.
		fmt.Printf("wrote %s id=%s version=%s unsigned=%v\n", res.Archive, res.ID, res.Version, res.Unsigned)
		fmt.Printf("grants-digest %s\n", res.GrantsDigest)
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

// readVersion reports the SDK version the CLI was built from. It reads the
// embedded VERSION file rather than a constant, because v1's constant had
// drifted four releases from the file beside it.
func readVersion() string { return pluginsdk.Version() }
