package pluginsdk

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/serve"
)

// Config is how a plugin process serves its HTTP routes. This half of the
// contract is deliberately UNCHANGED from v1: the bus replaced the host RPC
// wire, not the plugin's own HTTP surface, and changing both at once would have
// made every route regression look like a bus regression.
type Config struct {
	ID      string
	Socket  string
	Handler http.Handler
}

// Serve listens on GATEWAY_PLUGIN_SOCKET (or cfg.Socket) until SIGINT/SIGTERM.
func Serve(cfg Config) error { return ServeContext(context.Background(), cfg) }

// ServeContext is Serve with a caller-owned context.
func ServeContext(ctx context.Context, cfg Config) error {
	id := strings.TrimSpace(cfg.ID)
	if id == "" {
		id = strings.TrimSpace(os.Getenv(EnvID))
	}
	sock := strings.TrimSpace(cfg.Socket)
	if sock == "" {
		sock = strings.TrimSpace(os.Getenv(EnvSocket))
	}
	return serve.Serve(ctx, serve.Config{ID: id, Socket: sock, Handler: cfg.Handler})
}

// StderrLogger is the default logger for a spawned v2 plugin.
//
// v1 forwarded a plugin's log records to the host over the RPC wire. v2 has no
// such wire, and routing them over the bus would mean a log line vanishes
// whenever nothing happens to be subscribed. The host already captures a
// spawned plugin's stdout and stderr into its own output, so stderr is the
// channel that cannot silently drop anything.
func StderrLogger(id string) Logger {
	return slogLogger{l: slog.New(slog.NewTextHandler(os.Stderr, nil)).With("plugin", id)}
}

type slogLogger struct{ l *slog.Logger }

func (s slogLogger) Info(msg string, args ...any)  { s.l.Info(msg, args...) }
func (s slogLogger) Warn(msg string, args ...any)  { s.l.Warn(msg, args...) }
func (s slogLogger) Error(msg string, args ...any) { s.l.Error(msg, args...) }
