package pluginsdk

import (
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/bus"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/plugin"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/semver"
)

// Common types inherited from sdk-dependencies. Plugin authors should
// import this module only; do not redefine Manifest.
type (
	Manifest     = plugin.Manifest
	NavItem      = plugin.NavItem
	PanelSpec    = plugin.PanelSpec
	LauncherSpec = plugin.LauncherSpec
	Pricing      = plugin.Pricing
	Publisher    = plugin.Publisher
	Envelope     = bus.Envelope
)

const (
	TargetGateway = plugin.TargetGateway
	TargetVault   = plugin.TargetVault
)

// ParseManifest decodes plugin.json using the common contract.
func ParseManifest(raw []byte) (Manifest, error) {
	return plugin.ParseManifest(raw)
}

// CoreVersionAtLeast is the shared minCoreVersion compare.
func CoreVersionAtLeast(have, need string) bool {
	return semver.AtLeast(have, need)
}
