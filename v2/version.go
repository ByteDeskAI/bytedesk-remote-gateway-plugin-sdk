package pluginsdk

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionFile string

// Version is this module's SemVer, read from the VERSION file at build time.
//
// v1's CLI returned a hardcoded string that had drifted four releases from the
// VERSION file beside it. Embedding the file is how the two cannot disagree.
func Version() string { return strings.TrimSpace(versionFile) }
