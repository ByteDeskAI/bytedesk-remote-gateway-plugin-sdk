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

func TestTerminalPresentationVectorsComeFromPinnedCommonSDK(t *testing.T) {
	moduleDir, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/ByteDeskAI/bytedesk-sdk-dependencies").Output()
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join(strings.TrimSpace(string(moduleDir)), "plugin", "testdata", "terminal_presentation.json"))
	if err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile("ui/testdata/terminal_presentation.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatal("browser terminal presentation vectors drifted from the pinned common SDK")
	}
}

func TestTerminalPresentationHelpersReexportCommonContract(t *testing.T) {
	request := presentationRequest()
	if err := ValidatePresentationRequest(request); err != nil {
		t.Fatal(err)
	}
	if TerminalPresentationInterface != "terminal.presentation.v1" || TerminalPresentationCommand != "terminal.presentation.project.v1" {
		t.Fatal("terminal presentation identifiers drifted")
	}
}

func TestDesktopApplicationsContractComesFromPinnedCommonSDK(t *testing.T) {
	if DesktopApplicationsService != "desktop-applications" || DesktopApplicationsContractRevision != 1 {
		t.Fatalf("desktop applications identity = %q revision %d", DesktopApplicationsService, DesktopApplicationsContractRevision)
	}
	if CmdDesktopApplicationsScan.Name() != DesktopApplicationsScanCommand || CmdDesktopApplicationsOpen.Name() != DesktopApplicationsOpenCommand {
		t.Fatal("desktop applications descriptors drifted from common SDK")
	}
	if CmdDesktopApplicationsScanV2.Name() != "cmd.desktop-applications.v2.scan" || CmdDesktopApplicationsScanV2.Name() != DesktopApplicationsScanV2Command {
		t.Fatal("desktop applications v2 scan descriptor drifted from common SDK")
	}
	app := DesktopApplication{
		ID:                   "claude-desktop",
		Name:                 "Claude Desktop",
		Kind:                 DesktopApplicationKindDesktop,
		Status:               DesktopApplicationReady,
		IconURL:              "/api/plugins/applications/icons/claude-desktop",
		LauncherPath:         "/usr/share/applications/claude.desktop",
		InstalledAt:          "2026-09-12T18:30:00Z",
		InstalledAtEstimated: true,
	}
	if err := app.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (DesktopApplication{
		ID: "claude-desktop", Name: "Claude Desktop", Kind: DesktopApplicationKindDesktop,
		Status: DesktopApplicationReady, InstalledAtEstimated: true,
	}).Validate(); err == nil {
		t.Fatal("gateway SDK alias accepted estimated install metadata without an install time")
	}
	if err := (DesktopApplicationsScanV2Request{ScanID: "scan-1", Cursor: "page-2", Limit: DesktopApplicationsScanV2DefaultLimit}).Validate(); err != nil {
		t.Fatalf("gateway SDK v2 request alias: %v", err)
	}
	if err := desktopApplicationsScanV2ResultForTest(app).Validate(); err != nil {
		t.Fatalf("gateway SDK v2 result alias: %v", err)
	}
}

func TestTmuxReadContractComesFromPinnedCommonSDK(t *testing.T) {
	if TmuxService != "tmux" || TmuxContractRevision != 1 {
		t.Fatalf("tmux identity = %q revision %d", TmuxService, TmuxContractRevision)
	}
	if CmdTmuxAvailability.Name() != TmuxAvailabilityCommand ||
		CmdTmuxSessions.Name() != TmuxSessionsCommand ||
		CmdTmuxWindows.Name() != TmuxWindowsCommand ||
		CmdTmuxPanes.Name() != TmuxPanesCommand {
		t.Fatal("tmux descriptors drifted from common SDK")
	}
	if err := (TmuxAvailabilityResult{Tmux: TmuxAvailability{State: TmuxStateOK, Version: "3.5a", Message: "available"}}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (TmuxPanesResult{Output: "session\t0\t0\tzsh\t/work\t123\t1\t0\t0\ttitle"}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func desktopApplicationsScanV2ResultForTest(app DesktopApplication) DesktopApplicationsScanV2Result {
	return DesktopApplicationsScanV2Result{
		ScanID: "scan-1", State: DesktopApplicationsScanV2Complete,
		Revision: "revision-1", ScannedAt: "2026-09-13T16:00:00Z",
		Total: 1, Applications: []DesktopApplication{app},
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

func TestBrowserJavaScriptComesFromPinnedCommonSDK(t *testing.T) {
	generated, err := exec.Command("go", "run", "github.com/ByteDeskAI/bytedesk-sdk-dependencies/cmd/plugin-typescript", "-emit", "js").Output()
	if err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile("ui/contracts.js")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, generated) {
		t.Fatal("ui/contracts.js drifted from the pinned common SDK")
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
			socket := shortUnixSocketPath(t)
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
