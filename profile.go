package pluginsdk

import (
	"bytes"
	"net/http"
	"runtime/pprof"
	"strconv"
	"time"
)

// ProfileCommand is the host-only route on a spawned plugin's socket that
// captures a CPU profile of the plugin's own process.
//
// A spawned plugin has its own Go runtime, so the gateway cannot attribute its
// CPU with pprof labels the way it does for in-process plugins: the plugin has
// to profile itself. ServePlugin answers this route in front of the plugin's own
// handler, so every plugin built on ServePlugin supports it with no code of its
// own. The cmd. prefix marks it host-only; the gateway refuses browser requests
// to /p/<id>/cmd.* paths.
//
//	GET /cmd.profile.v1.cpu?seconds=N   (1 to ProfileMaxSeconds, default 10)
//
// The response is a pprof CPU profile. A second capture while one is running is
// refused with 409, because a Go process has one CPU profiler.
const ProfileCommand = "/cmd.profile.v1.cpu"

// ProfileMaxSeconds bounds one capture.
const ProfileMaxSeconds = 60

// withProfileCommand serves ProfileCommand and passes every other request to
// next unchanged. It is a plain path check rather than a mux, so the plugin's
// own routing, path cleaning and redirects behave exactly as before.
func withProfileCommand(next http.Handler) http.Handler {
	if next == nil {
		// Keep the /healthz that the socket server provides when a plugin has no
		// handler of its own.
		next = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte("ok"))
				return
			}
			http.NotFound(w, r)
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == ProfileCommand {
			serveCPUProfile(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func serveCPUProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	seconds := 10
	if raw := r.URL.Query().Get("seconds"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > ProfileMaxSeconds {
			http.Error(w, "seconds must be between 1 and 60", http.StatusBadRequest)
			return
		}
		seconds = n
	}
	var buf bytes.Buffer
	if err := pprof.StartCPUProfile(&buf); err != nil {
		http.Error(w, "a CPU profile is already being captured", http.StatusConflict)
		return
	}
	timer := time.NewTimer(time.Duration(seconds) * time.Second)
	select {
	case <-timer.C:
	case <-r.Context().Done():
		timer.Stop()
	}
	pprof.StopCPUProfile()
	if r.Context().Err() != nil {
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(buf.Bytes())
}
