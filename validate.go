package pluginsdk

import "github.com/ByteDeskAI/bytedesk-sdk-dependencies/plugin"

// ValidateDir loads plugin.json via the common contract and requires
// targets to include gateway (authoring/pack: version required).
func ValidateDir(dir string) (plugin.Manifest, error) {
	return plugin.LoadDirForHost(dir, plugin.TargetGateway, true)
}

// ValidateDirDiscover is the gateway enable/scan gate (version optional).
func ValidateDirDiscover(dir string) (plugin.Manifest, error) {
	return plugin.LoadDirForHost(dir, plugin.TargetGateway, false)
}
