// sync-ui-v2 copies only canonical browser artifacts from the released common
// module pinned by v2/go.mod. It never generates contracts from Gateway types.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func artifactPaths() []string {
	paths := []string{"typescript/contracts.d.ts", "typescript/validators.js", "typescript/descriptors.js", "typescript/schemas.json"}
	for _, pkg := range []string{"webapps", "aidecision", "payloads", "provideraccess", "codingsessions", "hostsettings"} {
		for _, file := range []string{"contracts.d.ts", "validators.js", "descriptors.js"} {
			paths = append(paths, filepath.Join(pkg, "typescript", file))
		}
	}
	return paths
}
func syncArtifacts(source, root string) error {
	for _, path := range artifactPaths() {
		data, err := os.ReadFile(filepath.Join(source, path))
		if err != nil {
			return err
		}
		parts := strings.Split(filepath.ToSlash(path), "/")
		relative := parts[len(parts)-1]
		if len(parts) == 3 {
			relative = filepath.Join(parts[0], relative)
		}
		target := filepath.Join(root, "ui", "v2", relative)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0644); err != nil {
			return err
		}
	}
	return nil
}
func main() {
	root, err := os.Getwd()
	if err != nil {
		fail(err)
	}
	cmd := exec.Command("go", "list", "-m", "-f", "{{if .Replace}}REPLACED{{else}}{{.Dir}}{{end}}", "github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2")
	cmd.Dir = filepath.Join(root, "v2")
	output, err := cmd.Output()
	if err != nil {
		fail(fmt.Errorf("resolve pinned common module: %w", err))
	}
	source := strings.TrimSpace(string(output))
	if source == "" || source == "REPLACED" {
		fail(fmt.Errorf("a released, non-replaced common module is required"))
	}
	if err := syncArtifacts(source, root); err != nil {
		fail(err)
	}
	fmt.Printf("Synchronized %d canonical browser artifacts from the pinned common SDK.\n", len(artifactPaths()))
}
func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
