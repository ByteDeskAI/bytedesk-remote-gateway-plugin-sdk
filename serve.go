package pluginsdk

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Env vars set by the gateway when spawning a process plugin.
const (
	EnvSocket = "GATEWAY_PLUGIN_SOCKET"
	EnvID     = "GATEWAY_PLUGIN_ID"
)

// Config is how a plugin process starts.
type Config struct {
	ID      string       // default: GATEWAY_PLUGIN_ID or "plugin"
	Socket  string       // default: GATEWAY_PLUGIN_SOCKET or "plugin.sock"
	Handler http.Handler // if nil, healthz-only mux
}

// Serve listens on the unix socket and blocks until SIGINT/SIGTERM or ctx done.
func Serve(cfg Config) error {
	return ServeContext(context.Background(), cfg)
}

// ServeContext is Serve with a caller-owned context.
func ServeContext(ctx context.Context, cfg Config) error {
	id := strings.TrimSpace(cfg.ID)
	if id == "" {
		id = strings.TrimSpace(os.Getenv(EnvID))
	}
	if id == "" {
		id = "plugin"
	}
	sock := strings.TrimSpace(cfg.Socket)
	if sock == "" {
		sock = strings.TrimSpace(os.Getenv(EnvSocket))
	}
	if sock == "" {
		sock = "plugin.sock"
	}
	h := cfg.Handler
	if h == nil {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("ok"))
		})
		h = mux
	}
	_ = os.Remove(sock)
	ln, err := net.Listen("unix", sock)
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: h}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()
	select {
	case <-ctx.Done():
		shctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shctx)
		_ = ln.Close()
		_ = os.Remove(sock)
		return nil
	case err := <-errCh:
		_ = ln.Close()
		_ = os.Remove(sock)
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}
