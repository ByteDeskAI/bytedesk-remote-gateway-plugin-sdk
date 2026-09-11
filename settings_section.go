package pluginsdk

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/plugin"
)

// Settings sections over the extension bridge.
//
// A spawned plugin contributes a settings section by implementing
// SettingsSectionPoint: it names the section in Manifest.Implements (and labels
// it in Manifest.Config), serves the two commands below on its own socket, and
// the host bridges the gateway's settings API onto them. The plugin owns the
// section's values; the host never hands it the gateway.
const (
	SettingsSectionPoint = plugin.SettingsSectionPoint

	// SettingsSectionSnapshotCommand returns the section body as a JSON object.
	SettingsSectionSnapshotCommand = "cmd." + plugin.SettingsSectionPoint + ".v1.snapshot"
	// SettingsSectionPatchCommand applies a JSON object and answers
	// {"restart": bool}, true when the change needs a gateway restart.
	SettingsSectionPatchCommand = "cmd." + plugin.SettingsSectionPoint + ".v1.patch"
)

// settingsSectionMaxBytes bounds a PATCH body. It matches the host's per
// operation budget so a plugin rejects what the host would never send.
const settingsSectionMaxBytes = 64 << 10

// SettingsSection is what a plugin implements to serve one section.
type SettingsSection interface {
	SectionSnapshot(ctx context.Context) (map[string]any, error)
	SectionPatch(ctx context.Context, body map[string]any) (restart bool, err error)
}

// ErrSettingsReadOnly is returned by SectionPatch for a section with nothing to
// write. The handler answers 405, which the host reports as read-only.
var ErrSettingsReadOnly = errors.New("settings section is read-only")

// SettingsSectionHTTPHandler serves both commands for one section. Mount it at
// both command paths, or use MountSettingsSection.
func SettingsSectionHTTPHandler(section SettingsSection) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if section == nil {
			http.Error(w, "settings section not implemented", http.StatusNotImplemented)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		switch r.URL.Path {
		case "/" + SettingsSectionSnapshotCommand:
			snap, err := section.SectionSnapshot(r.Context())
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			if snap == nil {
				snap = map[string]any{}
			}
			writeSettingsJSON(w, snap)
		case "/" + SettingsSectionPatchCommand:
			var body map[string]any
			dec := json.NewDecoder(io.LimitReader(r.Body, settingsSectionMaxBytes+1))
			if err := dec.Decode(&body); err != nil {
				http.Error(w, "settings patch must be a JSON object", http.StatusBadRequest)
				return
			}
			restart, err := section.SectionPatch(r.Context(), body)
			switch {
			case errors.Is(err, ErrSettingsReadOnly):
				http.Error(w, err.Error(), http.StatusMethodNotAllowed)
			case err != nil:
				http.Error(w, err.Error(), http.StatusBadRequest)
			default:
				writeSettingsJSON(w, map[string]bool{"restart": restart})
			}
		default:
			http.NotFound(w, r)
		}
	})
}

// MountSettingsSection registers one section's commands on mux, so a plugin
// cannot mount one path and forget the other.
func MountSettingsSection(mux *http.ServeMux, section SettingsSection) {
	h := SettingsSectionHTTPHandler(section)
	mux.Handle("/"+SettingsSectionSnapshotCommand, h)
	mux.Handle("/"+SettingsSectionPatchCommand, h)
}

func writeSettingsJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
