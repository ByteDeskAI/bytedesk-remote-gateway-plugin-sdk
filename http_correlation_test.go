package pluginsdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

const testCorrelationID = "0123456789abcdef0123456789abcdef"

type correlationTestHost struct{ logger Logger }

func (h correlationTestHost) Publish(Envelope) error                  { return nil }
func (h correlationTestHost) Subscribe(string, func(Envelope)) func() { return func() {} }
func (h correlationTestHost) Request(_ context.Context, _ Envelope) (Envelope, error) {
	return Envelope{}, nil
}
func (h correlationTestHost) Logger() Logger                     { return h.logger }
func (h correlationTestHost) StateDir(string) string             { return "" }
func (h correlationTestHost) Every(time.Duration, func()) func() { return func() {} }
func (h correlationTestHost) BumpContributions()                 {}

type correlationTestLogger struct{ args []any }

func (l *correlationTestLogger) Info(_ string, args ...any)  { l.args = append([]any(nil), args...) }
func (l *correlationTestLogger) Warn(_ string, args ...any)  { l.args = append([]any(nil), args...) }
func (l *correlationTestLogger) Error(_ string, args ...any) { l.args = append([]any(nil), args...) }

func TestCorrelationIDRequiresOneCanonicalHostValue(t *testing.T) {
	valid := &http.Request{Header: http.Header{HeaderCorrelationID: []string{testCorrelationID}}}
	if got, ok := CorrelationID(valid); !ok || got != testCorrelationID {
		t.Fatalf("CorrelationID = %q, %v", got, ok)
	}

	for _, headers := range []http.Header{
		{},
		{HeaderCorrelationID: []string{""}},
		{HeaderCorrelationID: []string{"0123456789ABCDEF0123456789ABCDEF"}},
		{HeaderCorrelationID: []string{"0123456789abcdef"}},
		{HeaderCorrelationID: []string{testCorrelationID, testCorrelationID}},
		{HeaderCorrelationID: []string{testCorrelationID}, "x-bytedesk-correlation-id": []string{testCorrelationID}},
	} {
		if got, ok := CorrelationID(&http.Request{Header: headers}); ok {
			t.Errorf("accepted %v as %q", headers, got)
		}
	}
	if got, ok := CorrelationID(nil); ok || got != "" {
		t.Fatalf("nil request = %q, %v", got, ok)
	}
}

func TestLoggerForRequestIsAdditive(t *testing.T) {
	capture := &correlationTestLogger{}
	host := correlationTestHost{logger: capture}
	request := &http.Request{Header: http.Header{HeaderCorrelationID: []string{testCorrelationID}}}
	LoggerForRequest(host, request).Info("finished", "operation", "scan")
	want := []any{"operation", "scan", CorrelationIDLogKey, testCorrelationID}
	if !reflect.DeepEqual(capture.args, want) {
		t.Fatalf("args = %#v, want %#v", capture.args, want)
	}

	legacy := &correlationTestLogger{}
	legacyHost := correlationTestHost{logger: legacy}
	if got := LoggerForRequest(legacyHost, &http.Request{}); got != legacy {
		t.Fatal("missing correlation id did not preserve legacy logger")
	}
}

func TestLoggerForRequestCarriesCorrelationAcrossRPC(t *testing.T) {
	var body struct {
		Level string `json:"level"`
		Msg   string `json:"msg"`
		Args  []any  `json:"args"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/log" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	host := NewHost("").(*rpcHost)
	host.client = &http.Client{Transport: rewriteHostTransport{base: srv.URL, delegate: http.DefaultTransport}}
	request := &http.Request{Header: http.Header{HeaderCorrelationID: []string{testCorrelationID}}}
	LoggerForRequest(host, request).Warn("slow", "elapsed_ms", 14)

	if body.Level != "warn" || body.Msg != "slow" {
		t.Fatalf("log = %#v", body)
	}
	want := []any{"elapsed_ms", float64(14), CorrelationIDLogKey, testCorrelationID}
	if !reflect.DeepEqual(body.Args, want) {
		t.Fatalf("args = %#v, want %#v", body.Args, want)
	}
}
