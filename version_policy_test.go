package pluginsdk

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestGoModPinsDependenciesIndependently records the policy: this SDK's
// VERSION is not required to equal the required sdk-dependencies version.
func TestGoModPinsDependenciesIndependently(t *testing.T) {
	mod, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mod), "\nreplace ") || strings.HasPrefix(string(mod), "replace ") {
		t.Fatal("go.mod must not replace sdk-dependencies with a local path")
	}
	re := regexp.MustCompile(`(?m)^\s*(?:require\s+)?github.com/ByteDeskAI/bytedesk-sdk-dependencies\s+(v[0-9]+\.[0-9]+\.[0-9]+)`)
	m := re.FindSubmatch(mod)
	if m == nil {
		t.Fatal("go.mod must require github.com/ByteDeskAI/bytedesk-sdk-dependencies vX.Y.Z")
	}
	deps := string(m[1])
	raw, err := os.ReadFile("VERSION")
	if err != nil {
		t.Fatal(err)
	}
	sdk := strings.TrimSpace(string(raw))
	if sdk == "" {
		t.Fatal("VERSION empty")
	}
	// Equality is allowed but not required. Do not add a lockstep assertion.
	t.Logf("sdk VERSION=%s requires sdk-dependencies %s", sdk, deps)
}
