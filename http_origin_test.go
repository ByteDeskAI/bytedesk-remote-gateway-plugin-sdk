package pluginsdk

import (
	"net/http"
	"testing"
)

func TestExternalOrigin(t *testing.T) {
	for _, origin := range []string{"http://localhost", "http://localhost:8080", "https://gateway.example", "https://gateway.example.", "https://gateway.example:8443", "http://127.0.0.1:8090", "https://[2001:db8::1]:8443", "http://[::1]", "https://xn--bcher-kva.example"} {
		r := &http.Request{Header: http.Header{HeaderExternalOrigin: []string{origin}}}
		got, err := ExternalOrigin(r)
		if err != nil || got != origin {
			t.Errorf("%q: got %q, %v", origin, got, err)
		}
	}
	for _, origin := range []string{"", " https://example.com", "https://example.com ", "https://example.com,https://other.example", "https://user@example.com", "https://example.com/", "https://example.com?x", "https://example.com?", "https://example.com#x", "https://example.com#", "https:example.com", "ftp://example.com", "https://*.example.com", "https://example.com:abc", "https://example.com:0", "https://example.com:65536", "https://example.com:", "https://example.com:0443", "https://example.com:443", "http://example.com:80", "https://EXAMPLE.com", "HTTPS://example.com", "https://example.com\n", "https://exam\x00ple.com", "https://example.com\x7f", "https://a..com", "https://-a.com", "https://a_.com", "https://bücher.example", "https://[fe80::1%25eth0]", "https://[::1", "https://::1", "https://127.000.0.1", "https://[2001:0db8::1]"} {
		if got, err := ExternalOrigin(&http.Request{Header: http.Header{HeaderExternalOrigin: []string{origin}}}); err == nil {
			t.Errorf("accepted %q => %q", origin, got)
		}
	}
	for _, headers := range []http.Header{
		{},
		{"Origin": []string{"https://example.com"}, "Forwarded": []string{"host=example.com;proto=https"}, "X-Forwarded-Host": []string{"example.com"}, "X-Forwarded-Proto": []string{"https"}, "Referer": []string{"https://example.com/"}},
		{HeaderExternalOrigin: []string{"https://example.com", "https://example.com"}},
		{HeaderExternalOrigin: []string{"https://example.com"}, "x-bytedesk-external-origin": []string{"https://other.example"}},
	} {
		if _, err := ExternalOrigin(&http.Request{Header: headers}); err == nil {
			t.Errorf("accepted missing/ambiguous attestation: %v", headers)
		}
	}
	if _, err := ExternalOrigin(nil); err == nil {
		t.Fatal("accepted nil request")
	}
}
