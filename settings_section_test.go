package pluginsdk

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

type fakeSection struct {
	vals     map[string]any
	readOnly bool
}

func (f *fakeSection) SectionSnapshot(context.Context) (map[string]any, error) { return f.vals, nil }
func (f *fakeSection) SectionPatch(_ context.Context, body map[string]any) (bool, error) {
	if f.readOnly {
		return false, ErrSettingsReadOnly
	}
	for k, v := range body {
		f.vals[k] = v
	}
	return true, nil
}

func TestSettingsSectionHandlerServesBothCommands(t *testing.T) {
	sec := &fakeSection{vals: map[string]any{"mode": "compact"}}
	mux := http.NewServeMux()
	MountSettingsSection(mux, sec)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/"+SettingsSectionSnapshotCommand, nil))
	var snap map[string]any
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &snap) != nil || snap["mode"] != "compact" {
		t.Fatalf("snapshot: %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/"+SettingsSectionPatchCommand, bytes.NewBufferString(`{"mode":"wide"}`)))
	var reply map[string]bool
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &reply) != nil || !reply["restart"] || sec.vals["mode"] != "wide" {
		t.Fatalf("patch: %d %s vals=%v", rec.Code, rec.Body.String(), sec.vals)
	}
}

func TestSettingsSectionHandlerRefusesWhatItMust(t *testing.T) {
	sec := &fakeSection{vals: map[string]any{}, readOnly: true}
	h := SettingsSectionHTTPHandler(sec)
	cases := []struct {
		name, method, path, body string
		want                     int
	}{
		{"get is not an operation", http.MethodGet, "/" + SettingsSectionSnapshotCommand, "", http.StatusMethodNotAllowed},
		{"read-only section", http.MethodPost, "/" + SettingsSectionPatchCommand, `{"a":1}`, http.StatusMethodNotAllowed},
		{"patch must be an object", http.MethodPost, "/" + SettingsSectionPatchCommand, `[1,2]`, http.StatusBadRequest},
		{"unknown command", http.MethodPost, "/cmd.host.settings.section.v1.delete", "", http.StatusNotFound},
		{"legacy bare command path", http.MethodPost, "/cmd.settings.section.v1.snapshot", "", http.StatusNotFound},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, bytes.NewBufferString(c.body)))
		if rec.Code != c.want {
			t.Errorf("%s: got %d want %d", c.name, rec.Code, c.want)
		}
	}
}

// The wire names must match what the gateway bridge posts to:
// "cmd." + point + ".v1." + op, with point host.settings.section.
func TestSettingsSectionCommandNamesMatchTheBridge(t *testing.T) {
	if SettingsSectionSnapshotCommand != "cmd.host.settings.section.v1.snapshot" || SettingsSectionPatchCommand != "cmd.host.settings.section.v1.patch" {
		t.Fatalf("command names drifted: %s %s", SettingsSectionSnapshotCommand, SettingsSectionPatchCommand)
	}
}

func TestRPCHostRegisterExtensionPostsTheRegistration(t *testing.T) {
	dir, err := os.MkdirTemp("", "pr")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sock := filepath.Join(dir, "h.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	mux := http.NewServeMux()
	mux.HandleFunc("/registerExtension", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ext:host.settings.section:tmux","contributionId":"tmux"}`))
	})
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	reg, ok := NewHost(sock).(ExtensionRegistrar)
	if !ok {
		t.Fatal("the spawned-plugin host does not implement ExtensionRegistrar")
	}
	handle, err := reg.RegisterExtension(context.Background(), SettingsSectionPoint, "tmux", "")
	if err != nil || handle != "ext:host.settings.section:tmux" {
		t.Fatalf("handle=%q err=%v", handle, err)
	}
	if got["point"] != "host.settings.section" || got["id"] != "tmux" {
		t.Fatalf("registration body: %v", got)
	}
}
