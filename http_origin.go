package pluginsdk

import (
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

// HeaderExternalOrigin carries the external HTTP origin attested by the host.
// The host MUST strip client-supplied copies and stamp a value derived from
// direct TLS or explicitly trusted proxy metadata, never arbitrary forwarded
// headers. This is an HTTP proxy header, not a bus-envelope identity header.
const HeaderExternalOrigin = "X-Bytedesk-External-Origin"

// ExternalOrigin reads one canonical host-attested http/https origin. It never
// falls back to Origin, Referer, Forwarded, or X-Forwarded-* headers.
//
// Call only on the authenticated private host-to-plugin transport. Header
// presence is NOT authentication: a public client can construct this header.
// The caller must still enforce request authentication and compare the browser
// Origin against the returned value for same-origin operations.
func ExternalOrigin(r *http.Request) (string, error) {
	invalid := errors.New("missing or invalid host-attested external origin")
	if r == nil {
		return "", invalid
	}
	var values []string
	for key, entries := range r.Header {
		if strings.EqualFold(key, HeaderExternalOrigin) {
			values = append(values, entries...)
		}
	}
	if len(values) != 1 {
		return "", invalid
	}
	raw := values[0]
	if len(raw) == 0 || len(raw) > 2048 {
		return "", invalid
	}
	for _, c := range raw {
		if c <= 32 || c >= 127 {
			return "", invalid
		}
	}
	if strings.ContainsAny(raw, ",*\\?#") {
		return "", invalid
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Opaque != "" || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return "", invalid
	}
	host := u.Hostname()
	if host == "" || strings.ToLower(host) != host {
		return "", invalid
	}
	authority := host
	if ip, err := netip.ParseAddr(host); err == nil {
		if ip.Zone() != "" || ip.String() != host {
			return "", invalid
		}
		if ip.Is6() {
			authority = "[" + host + "]"
		}
	} else {
		if len(host) > 253 || strings.Trim(host, "0123456789.") == "" {
			return "", invalid
		}
		// A fully qualified DNS name may retain its one trailing root dot.
		for _, label := range strings.Split(strings.TrimSuffix(host, "."), ".") {
			if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' || strings.Trim(label, "abcdefghijklmnopqrstuvwxyz0123456789-") != "" {
				return "", invalid
			}
		}
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 || strconv.Itoa(n) != port || (u.Scheme == "http" && n == 80) || (u.Scheme == "https" && n == 443) {
			return "", invalid
		}
		authority += ":" + port
	}
	if raw != u.Scheme+"://"+authority {
		return "", invalid
	}
	return raw, nil
}
