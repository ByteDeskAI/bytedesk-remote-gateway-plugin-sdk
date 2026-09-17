package pluginsdk_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The release gates, carried over from v1 and widened for v2's two pins.
//
// v2 requires BOTH sdk-dependencies modules: /v2 for the contract, and v1 for
// the pieces v2 did not re-home (serve) and for v1compat. Each is pinned
// independently of this module's own VERSION, and neither may be papered over
// with a replace.

var (
	depV2 = regexp.MustCompile(`(?m)^\s*(?:require\s+)?github\.com/ByteDeskAI/bytedesk-sdk-dependencies/v2\s+(\S+)`)
	depV1 = regexp.MustCompile(`(?m)^\s*(?:require\s+)?github\.com/ByteDeskAI/bytedesk-sdk-dependencies\s+(\S+)`)
	// goPseudoVersion matches the suffix Go appends when a module is required
	// by commit rather than by tag. The separator before the timestamp is NOT
	// always a dash: off a pre-release base Go appends to the existing
	// pre-release segment and writes a dot. Every pin this repo has used is
	// the second shape, so a dash-only pattern matches none of them and the
	// gate passes silently forever.
	goPseudoVersion = regexp.MustCompile(`[-.][0-9]{14}-[0-9a-f]{12}$`)
)

func goMod(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(moduleRoot(t), "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestGoModPinsDependenciesIndependently records the policy: this SDK's VERSION
// is not required to equal either required sdk-dependencies version.
func TestGoModPinsDependenciesIndependently(t *testing.T) {
	mod := goMod(t)
	if strings.Contains(mod, "\nreplace ") || strings.HasPrefix(mod, "replace ") {
		t.Fatal("go.mod must not replace sdk-dependencies with a local path")
	}
	v2 := depV2.FindStringSubmatch(mod)
	if v2 == nil {
		t.Fatal("go.mod must require github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2")
	}
	v1 := depV1.FindStringSubmatch(mod)
	if v1 == nil {
		t.Fatal("go.mod must require github.com/ByteDeskAI/bytedesk-sdk-dependencies (v1, for serve and v1compat)")
	}
	raw, err := os.ReadFile(filepath.Join(moduleRoot(t), "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	sdk := strings.TrimSpace(string(raw))
	if sdk == "" {
		t.Fatal("VERSION empty")
	}
	if !strings.HasPrefix(sdk, "2.") {
		t.Fatalf("VERSION is %q; a /v2 module's SemVer major must be 2", sdk)
	}
	// Equality is allowed but not required. Do not add a lockstep assertion.
	t.Logf("sdk VERSION=%s requires sdk-dependencies/v2 %s and sdk-dependencies %s", sdk, v2[1], v1[1])
}

// TestNoPseudoVersionReachesATag is the standing half of gateway TM-254.
//
// Iterating the contract untagged on pseudo-versions is normal and this test
// says nothing about it. What must never happen is a pseudo-version surviving
// into a RELEASE: a tag whose go.mod requires one hands every consumer a
// dependency they cannot reach from any tag.
func TestNoPseudoVersionReachesATag(t *testing.T) {
	mod := goMod(t)
	tag := releaseTag()
	for _, dep := range []struct {
		name string
		re   *regexp.Regexp
	}{
		{"bytedesk-sdk-dependencies/v2", depV2},
		{"bytedesk-sdk-dependencies", depV1},
	} {
		m := dep.re.FindStringSubmatch(mod)
		if m == nil {
			t.Fatalf("go.mod must require %s", dep.name)
		}
		pin := m[1]
		if !goPseudoVersion.MatchString(pin) {
			t.Logf("%s %s is a released version; nothing to gate", dep.name, pin)
			continue
		}
		if tag != "" {
			t.Fatalf("this commit is tagged %s but requires %s %s, a pseudo-version.\n"+
				"Tag %s first, pin the tag here, then tag this module.", tag, dep.name, pin, dep.name)
		}
		t.Logf("iterating untagged on %s %s — tag it and pin the tag before tagging this module", dep.name, pin)
	}
}

// TestGoPseudoVersionRecognisesBothShapes exists because the first version of
// the pattern above matched neither shape this repo actually uses, and the gate
// passed happily on a pseudo-version while reporting it as released. A gate
// whose predicate is wrong is worse than no gate: it reports success.
func TestGoPseudoVersionRecognisesBothShapes(t *testing.T) {
	for _, tc := range []struct {
		version string
		pseudo  bool
	}{
		{"v0.4.0-rc.8.0.20260912011527-0af154410449", true},
		{"v0.0.0-20260912011527-0af154410449", true},
		{"v1.2.3-0.20260101000000-abcdef123456", true},
		{"v2.0.0-rc.1.0.20260917120000-0af154410449", true},
		{"v2.0.0-rc.1", false},
		{"v2.0.0", false},
		{"v0.4.0-rc.17", false},
		{"v0.4.0-rc.8.0.2026091201152-0af154410449", false},  // 13-digit timestamp
		{"v0.4.0-rc.8.0.20260912011527-0af15441044", false},  // 11-character hash
		{"v0.4.0-rc.8.0.20260912011527-0af15441044g", false}, // not hex
	} {
		if got := goPseudoVersion.MatchString(tc.version); got != tc.pseudo {
			t.Errorf("%s: pseudo = %v, want %v", tc.version, got, tc.pseudo)
		}
	}
}

// TestVersionIsEmbedded proves the CLI and the VERSION file cannot disagree.
// v1's CLI returned a constant that had drifted four releases from the file
// beside it, and nothing failed.
func TestVersionIsEmbedded(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(moduleRoot(t), "VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := pluginsdkVersion(), strings.TrimSpace(string(raw)); got != want {
		t.Fatalf("Version() = %q, VERSION file = %q", got, want)
	}
}

// releaseTag reports the tag on the current commit, or "" when there is none.
//
// RELEASE_TAG short-circuits it so a release pipeline can assert the check
// without depending on tags being fetched: a shallow clone has no tags, and a
// gate that silently cannot see them is not a gate.
func releaseTag() string {
	if tag := strings.TrimSpace(os.Getenv("RELEASE_TAG")); tag != "" {
		return tag
	}
	out, err := exec.Command("git", "describe", "--exact-match", "--tags", "HEAD").Output()
	if err != nil {
		return "" // not tagged, or no git here: the ordinary iterating case
	}
	return strings.TrimSpace(string(out))
}
