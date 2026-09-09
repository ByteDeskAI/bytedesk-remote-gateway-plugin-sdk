package pluginsdk

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentVectorsComeFromPinnedCommonSDK(t *testing.T) {
	moduleDir, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/ByteDeskAI/bytedesk-sdk-dependencies").Output()
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(moduleDir)), "plugin", "testdata", "document_paths.json"))
	if err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile("ui/testdata/document_paths.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatal("browser document vectors drifted from the pinned common SDK")
	}
}

func TestDocumentHelpersReexportCommonContract(t *testing.T) {
	if err := ValidateDocumentPath("/files/*path"); err != nil {
		t.Fatal(err)
	}
	params, ok := MatchDocumentPath("/files/*path", "/files/a/b")
	if !ok || params["path"] != "a/b" {
		t.Fatalf("document match = %v, %v", params, ok)
	}
	if overlap, err := DocumentPathsOverlap("/files/*path", "/files/:id"); err != nil || !overlap {
		t.Fatalf("document overlap = %v, %v", overlap, err)
	}
}

func TestBrowserDeclarationsComeFromPinnedCommonSDK(t *testing.T) {
	generated, err := exec.Command("go", "run", "github.com/ByteDeskAI/bytedesk-sdk-dependencies/cmd/plugin-typescript").Output()
	if err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile("ui/contracts.d.ts")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, generated) {
		t.Fatal("ui/contracts.d.ts drifted from the pinned common SDK")
	}
}

func TestRPCNegotiationDoesNotGrantUnsupportedFeatures(t *testing.T) {
	for _, tc := range []struct {
		name     string
		major    uint32
		features []string
		status   int
		ok       bool
	}{
		{"supported", 1, []string{FeatureRuntimeSnapshot}, 200, true},
		{"wrong major", 2, []string{FeatureRuntimeSnapshot}, 200, false},
		{"missing feature", 1, nil, 200, false},
		{"legacy host", 0, nil, 404, false},
		{"denied", 1, nil, 403, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			socket := filepath.Join(t.TempDir(), "host.sock")
			listener, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/negotiate" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				var need ProtocolRequirements
				if err := json.NewDecoder(r.Body).Decode(&need); err != nil {
					t.Error(err)
				}
				if need.Major != 1 {
					t.Errorf("major = %d", need.Major)
				}
				w.WriteHeader(tc.status)
				_ = json.NewEncoder(w).Encode(HostCapabilities{Major: tc.major, Features: tc.features, PluginID: "example", Generation: "generation1"})
			})}
			go func() { _ = srv.Serve(listener) }()
			t.Cleanup(func() { _ = srv.Close() })
			host := NewHost(socket).(*rpcHost)
			t.Cleanup(host.Close)
			got, err := host.Negotiate(context.Background(), ProtocolRequirements{Major: 1, Required: []string{FeatureRuntimeSnapshot}})
			if (err == nil) != tc.ok {
				t.Fatalf("error = %v, want success %v", err, tc.ok)
			}
			if err != nil && got.PluginID != "" {
				t.Fatal("failed negotiation leaked a usable identity")
			}
		})
	}
}
