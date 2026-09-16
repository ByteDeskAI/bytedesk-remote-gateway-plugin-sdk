package pluginsdk

import (
	"net/http"
	"strings"

	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/plugin"
)

// HeaderCorrelationID carries the host-minted id for one HTTP request. The
// host must strip client copies before stamping this private transport header.
// It is diagnostic metadata and never grants authority.
const HeaderCorrelationID = "X-Bytedesk-Correlation-ID"

// CorrelationIDLogKey is the canonical structured logging key from the common
// plugin contract.
const CorrelationIDLogKey = plugin.CorrelationIDLogKey

// CorrelationID reads one canonical 128-bit lowercase hexadecimal id from the
// authenticated private host-to-plugin transport. Missing, malformed, or
// ambiguous values are rejected.
func CorrelationID(r *http.Request) (string, bool) {
	if r == nil {
		return "", false
	}
	var values []string
	for key, entries := range r.Header {
		if strings.EqualFold(key, HeaderCorrelationID) {
			values = append(values, entries...)
		}
	}
	if len(values) != 1 || len(values[0]) != 32 {
		return "", false
	}
	for _, c := range values[0] {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return "", false
		}
	}
	return values[0], true
}

// LoggerForRequest returns the plugin logger bound to the host-minted request
// correlation id. It preserves the ordinary logger when an older host omits
// the header or the value is invalid.
func LoggerForRequest(host Host, r *http.Request) Logger {
	if host == nil {
		return nil
	}
	logger := host.Logger()
	if correlationID, ok := CorrelationID(r); ok {
		return plugin.LoggerWithCorrelationID(logger, correlationID)
	}
	return logger
}
