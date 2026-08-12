package pluginsdk

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/plugin"
)

// ValidateDir loads plugin.json from dir and checks layout + spawn binary
// (authoring/pack: version required).
func ValidateDir(dir string) (plugin.Manifest, error) {
	return validateDir(dir, true)
}

// ValidateDirDiscover is the host enable/scan gate (version optional;
// publisher may be a legacy string).
func ValidateDirDiscover(dir string) (plugin.Manifest, error) {
	return validateDir(dir, false)
}

func validateDir(dir string, requireVersion bool) (plugin.Manifest, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "plugin.json"))
	if err != nil {
		return plugin.Manifest{}, fmt.Errorf("plugin.json: %w", err)
	}
	m, err := plugin.ParseManifest(raw)
	if err != nil {
		return plugin.Manifest{}, fmt.Errorf("plugin.json: %w", err)
	}
	if requireVersion {
		if err := m.Validate(); err != nil {
			return m, err
		}
	} else if err := m.ValidateDiscover(); err != nil {
		return m, err
	}
	if m.Spawn {
		bin := filepath.Join(dir, filepath.Base(m.Binary))
		if _, err := os.Stat(bin); err != nil {
			return m, fmt.Errorf("spawn binary missing: %s", m.Binary)
		}
	}
	return m, nil
}
