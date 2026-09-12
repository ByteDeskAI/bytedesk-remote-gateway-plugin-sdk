package pluginsdk

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// goPseudoVersion matches the suffix Go appends when a module is required by
// commit rather than by tag: a UTC timestamp and a 12-character hash, which is
// what `go get <mod>@<sha>` produces.
//
// The separator before the timestamp is NOT always a dash. Off a plain base Go
// writes v0.0.0-20260912011527-0af154410449, but off a pre-release base it
// appends to the existing pre-release segment and writes
// v0.4.0-rc.8.0.20260912011527-0af154410449 — a dot. Every pin this repo has
// used is the second shape, so a dash-only pattern matches none of them and the
// gate passes silently forever.
var goPseudoVersion = regexp.MustCompile(`[-.][0-9]{14}-[0-9a-f]{12}$`)

// TestNoPseudoVersionReachesATag is the standing half of gateway TM-254.
//
// Iterating the contract untagged on pseudo-versions is the point of that task,
// so a pseudo-version in go.mod is the normal working state and this test says
// nothing about it. What must never happen is a pseudo-version surviving into a
// RELEASE: a tag whose go.mod requires `v0.4.0-rc.8.0.20260912011527-0af1544104`
// hands every consumer a dependency they cannot reach from any tag, and the
// iteration that was supposed to strand nobody strands everybody.
//
// The existing TestGoModPinsDependenciesIndependently cannot catch this: its
// regex prefix-matches `vX.Y.Z`, and a pseudo-version starts with exactly that.
// That is deliberate — it is what lets iteration pass — so the release check
// lives here instead.
func TestNoPseudoVersionReachesATag(t *testing.T) {
	mod, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`(?m)^\s*(?:require\s+)?github\.com/ByteDeskAI/bytedesk-sdk-dependencies\s+(\S+)`)
	m := re.FindSubmatch(mod)
	if m == nil {
		t.Fatal("go.mod must require github.com/ByteDeskAI/bytedesk-sdk-dependencies")
	}
	pin := string(m[1])
	if !goPseudoVersion.MatchString(pin) {
		t.Logf("sdk-dependencies %s is a released version; nothing to gate", pin)
		return
	}
	if tag := releaseTag(); tag != "" {
		t.Fatalf("this commit is tagged %s but requires sdk-dependencies %s, a pseudo-version.\n"+
			"Tag sdk-dependencies first, pin the tag here, then tag this module.", tag, pin)
	}
	t.Logf("iterating untagged on %s — tag sdk-dependencies and pin the tag before tagging this module", pin)
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
		{"v0.4.0-rc.8.0.20260912011527-0af154410449", true}, // off a pre-release base
		{"v0.0.0-20260912011527-0af154410449", true},        // off a plain base
		{"v1.2.3-0.20260101000000-abcdef123456", true},      // off a release base
		{"v0.4.0-rc.8", false},
		{"v0.4.0", false},
		{"v0.3.0", false},
		{"v0.4.0-rc.8.0.2026091201152-0af154410449", false},  // 13-digit timestamp
		{"v0.4.0-rc.8.0.20260912011527-0af15441044", false},  // 11-character hash
		{"v0.4.0-rc.8.0.20260912011527-0af15441044g", false}, // not hex
	} {
		if got := goPseudoVersion.MatchString(tc.version); got != tc.pseudo {
			t.Errorf("%s: pseudo = %v, want %v", tc.version, got, tc.pseudo)
		}
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
