package pluginsdk

import "github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/plugin"

// ValidateDir loads plugin.json through the v2 contract and requires targets to
// include gateway.
func ValidateDir(dir string) (Manifest, error) {
	return plugin.LoadDirForHost(dir, plugin.TargetGateway)
}

// Validate checks an in-memory manifest with the authoring rules (version
// required).
func Validate(m Manifest) error { return plugin.Validate(m) }

// ValidateDiscover is the host's enable/scan gate: the same rules with version
// optional, because a plugin discovered in a source tree has not been packed.
func ValidateDiscover(m Manifest) error { return plugin.ValidateDiscover(m) }
