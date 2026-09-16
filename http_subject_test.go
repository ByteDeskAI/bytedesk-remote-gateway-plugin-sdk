package pluginsdk

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testSubjectLease = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestSubjectLeaseContextDistinguishesMissingInvalidAndCanonical(t *testing.T) {
	tests := []struct {
		name    string
		header  http.Header
		want    string
		wantErr bool
	}{
		{name: "missing", header: http.Header{}},
		{name: "canonical", header: http.Header{HeaderSubjectLease: []string{testSubjectLease}}, want: testSubjectLease},
		{name: "empty", header: http.Header{HeaderSubjectLease: []string{""}}, wantErr: true},
		{name: "uppercase", header: http.Header{HeaderSubjectLease: []string{"0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF"}}, wantErr: true},
		{name: "short", header: http.Header{HeaderSubjectLease: []string{"0123456789abcdef"}}, wantErr: true},
		{name: "comma joined", header: http.Header{HeaderSubjectLease: []string{testSubjectLease + "," + testSubjectLease}}, wantErr: true},
		{name: "duplicate values", header: http.Header{HeaderSubjectLease: []string{testSubjectLease, testSubjectLease}}, wantErr: true},
		{name: "duplicate casing", header: http.Header{HeaderSubjectLease: []string{testSubjectLease}, "x-bytedesk-subject-lease": []string{testSubjectLease}}, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://plugin/", nil)
			req.Header = tc.header
			got, err := SubjectLeaseFromContext(ContextForRequest(req))
			if got != tc.want || (err != nil) != tc.wantErr {
				t.Fatalf("SubjectLeaseFromContext = %q, %v; want %q, error=%v", got, err, tc.want, tc.wantErr)
			}
		})
	}
	if got, err := SubjectLeaseFromContext(context.Background()); got != "" || err != nil {
		t.Fatalf("plain context = %q, %v", got, err)
	}
	if got, err := SubjectLeaseFromContext(ContextForRequest(nil)); got != "" || err != nil {
		t.Fatalf("nil request = %q, %v", got, err)
	}
}

func TestSubjectLeaseHandlerPropagatesRequestContext(t *testing.T) {
	var got string
	h := withSubjectLeaseContext(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		var err error
		got, err = SubjectLeaseFromContext(r.Context())
		if err != nil {
			t.Errorf("lease: %v", err)
		}
	}))
	req := httptest.NewRequest(http.MethodGet, "http://plugin/", nil)
	req.Header.Set(HeaderSubjectLease, testSubjectLease)
	h.ServeHTTP(httptest.NewRecorder(), req)
	if got != testSubjectLease {
		t.Fatalf("handler lease = %q", got)
	}
}

func TestContextForRequestPreservesCallerCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "http://plugin/", nil).WithContext(parent)
	req.Header.Set(HeaderSubjectLease, testSubjectLease)
	ctx := ContextForRequest(req)
	cancel()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("context error = %v", ctx.Err())
	}
}

func TestSubjectLeaseErrorIsStable(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://plugin/", nil)
	req.Header[HeaderSubjectLease] = []string{"invalid"}
	_, err := SubjectLeaseFromContext(ContextForRequest(req))
	if !errors.Is(err, ErrInvalidSubjectLease) {
		t.Fatalf("error = %v", err)
	}
}
