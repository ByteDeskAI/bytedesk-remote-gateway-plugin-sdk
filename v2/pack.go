package pluginsdk

import (
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/pack"
	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/plugin"
)

// PackResult is the written archive. v2 adds the grants digest, which is what
// an operator's consent is keyed by.
type PackResult = pack.Result

// PackDir validates this plugin for gateway, then builds the common tar.gz.
func PackDir(dir, outDir string) (PackResult, error) {
	m, err := ValidateDir(dir)
	if err != nil {
		return PackResult{}, err
	}
	if err := plugin.Validate(m); err != nil {
		return PackResult{}, err
	}
	return pack.Dir(dir, outDir)
}
