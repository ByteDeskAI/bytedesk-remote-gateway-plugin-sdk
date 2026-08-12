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

func TestValidateRejectsVaultOnly(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(`{"id":"v","version":"1.0.0","targets":["vault"]}`), 0o644)
	if _, err := ValidateDir(dir); err == nil {
		t.Fatal("expected vault-only plugin rejected on gateway SDK")
	}
}

func TestValidateDirDiscoverStringPublisherAndNoVersion(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plugin.json"), []byte(`{"id":"example","spawn":false,"publisher":"bytedesk"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := ValidateDirDiscover(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "example" || m.Publisher == nil || m.Publisher.ID != "bytedesk" {
		t.Fatalf("got %+v", m)
	}
}
