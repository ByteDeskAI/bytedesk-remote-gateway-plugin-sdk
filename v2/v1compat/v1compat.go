// Package v1compat lets a v1 plugin run unchanged on the v2 bus.
//
// The implementation lives in bytedesk-sdk-dependencies/v2/plugin/v1compat,
// beside the v1 Host interface it reconstructs. This package is the gateway
// SDK's re-export of it, so a plugin author who imports one module during a
// migration keeps importing one module.
//
// It is a migration aid with a planned end. Nothing new should be written
// against it, and it is expected to be deleted when the last v1 plugin is
// converted.
package v1compat

import (
	v1plugin "github.com/ByteDeskAI/bytedesk-sdk-dependencies/plugin"
	v2plugin "github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/plugin"
	v1compat "github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/plugin/v1compat"
)

// Host builds the v1 Host facade over a bound v2 Base.
func Host(b *v2plugin.Base) v1plugin.Host { return v1compat.Host(b) }
