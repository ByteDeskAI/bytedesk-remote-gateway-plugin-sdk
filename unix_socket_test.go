package pluginsdk

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// shortUnixSocketPath avoids t.TempDir's test-name suffix and long CI TMPDIRs.
// Unix socket paths have a small platform limit independent of filesystem limits.
// The socket lives in a fresh mode-0700 directory, never a predictable shared path.
// Register listener cleanup after this call so it runs before directory cleanup.
func shortUnixSocketPath(t *testing.T) string {
	t.Helper()
	base := os.TempDir()
	if runtime.GOOS != "windows" {
		base = "/tmp"
	}
	dir, err := os.MkdirTemp(base, "bds-")
	if err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(dir, "s")
	t.Cleanup(func() {
		// Remove only our exact socket and private directory, never recursively.
		if err := os.Remove(socket); err != nil && !os.IsNotExist(err) {
			t.Errorf("remove test socket: %v", err)
		}
		if err := os.Remove(dir); err != nil {
			t.Errorf("remove test socket directory: %v", err)
		}
	})
	if len(socket) > 100 {
		t.Fatalf("temporary socket path exceeds conservative Unix limit: %d bytes", len(socket))
	}
	return socket
}

func TestShortUnixSocketPathIgnoresLongTempAndCleansUp(t *testing.T) {
	var socket string
	t.Run("private fixture", func(t *testing.T) {
		longTemp := filepath.Join(t.TempDir(), strings.Repeat("long-", 24))
		if err := os.Mkdir(longTemp, 0700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("TMPDIR", longTemp)
		socket = shortUnixSocketPath(t)
		info, err := os.Stat(filepath.Dir(socket))
		if err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm() != 0700 {
			t.Fatalf("socket directory permissions = %o", info.Mode().Perm())
		}
		listener, err := net.Listen("unix", socket)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = listener.Close() })
	})
	if socket == "" {
		t.Fatal("fixture did not allocate a socket path")
	}
	if _, err := os.Stat(filepath.Dir(socket)); !os.IsNotExist(err) {
		t.Fatalf("private socket directory remains after cleanup: %v", err)
	}
}
