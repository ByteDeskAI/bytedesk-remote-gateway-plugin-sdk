package pluginsdk_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishedBrowserArtifactsComeFromPinnedCommonSDKV2(t *testing.T) {
	moduleDir, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2").Output()
	if err != nil {
		t.Fatal(err)
	}
	common := filepath.Join(strings.TrimSpace(string(moduleDir)), "typescript")
	published := filepath.Join(moduleRoot(t), "..", "ui", "v2")
	for _, name := range []string{"contracts.d.ts", "validators.js", "descriptors.js", "schemas.json"} {
		t.Run(name, func(t *testing.T) {
			want, err := os.ReadFile(filepath.Join(common, name))
			if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(published, name))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("ui/v2/%s drifted from the pinned common SDK v2 generator output", name)
			}
		})
	}
}
