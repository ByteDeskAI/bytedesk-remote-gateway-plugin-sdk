package pluginsdk

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/serve"
)

// Env vars the gateway host sets when spawning a process plugin.
const (
	EnvSocket = "GATEWAY_PLUGIN_SOCKET"
	EnvID     = "GATEWAY_PLUGIN_ID"
)

// Config is how a gateway plugin process starts.
type Config struct {
	ID      string
	Socket  string
	Handler http.Handler
}

// Serve listens on GATEWAY_PLUGIN_SOCKET (or cfg.Socket) until SIGINT/SIGTERM.
func Serve(cfg Config) error {
	return ServeContext(context.Background(), cfg)
}

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
