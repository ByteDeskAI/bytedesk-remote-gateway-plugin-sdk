package pluginsdk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAndPack(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(`{
		"id":"example","version":"0.1.0","spawn":true,"binary":"example",
		"minCoreVersion":"0.1.0"
	}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "example"), []byte("#!/bin/true\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateDir(dir); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	res, err := PackDir(dir, out)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(res.Archive); err != nil {
		t.Fatal(err)
	}
	if res.ID != "example" || !res.Unsigned {
		t.Fatalf("got %+v", res)
	}
}

func TestValidateRejectsMissingBinary(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(`{"id":"x","version":"1.0.0","spawn":true,"binary":"x"}`), 0o644)
	if _, err := ValidateDir(dir); err == nil {
		t.Fatal("expected missing binary")
	}
}
