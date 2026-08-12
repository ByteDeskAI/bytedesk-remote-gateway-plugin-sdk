package pluginsdk

import "github.com/ByteDeskAI/bytedesk-sdk-dependencies/pack"

// PackResult is the written archive (inherited from sdk-dependencies/pack).
type PackResult = pack.Result

// PackDir validates this plugin for gateway, then builds the common tar.gz.
func PackDir(dir, outDir string) (PackResult, error) {
	if _, err := ValidateDir(dir); err != nil {
		return PackResult{}, err
	}
	return pack.Dir(dir, outDir)
}
