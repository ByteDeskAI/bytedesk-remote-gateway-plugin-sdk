package pluginsdk_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishedAIBrowserArtifactsComeFromPinnedCommonSDK(t *testing.T) {
	moduleDir, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range []string{"aidecision", "payloads", "provideraccess", "codingsessions", "hostsettings"} {
		for _, file := range []string{"contracts.d.ts", "validators.js", "descriptors.js"} {
			t.Run(pkg+"/"+file, func(t *testing.T) {
				want, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(moduleDir)), pkg, "typescript", file))
				if err != nil {
					t.Fatal(err)
				}
				got, err := os.ReadFile(filepath.Join(moduleRoot(t), "..", "ui", "v2", pkg, file))
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, want) {
					t.Fatal("browser artifact differs from released common SDK; run npm run sync:v2")
				}
			})
		}
	}
}
