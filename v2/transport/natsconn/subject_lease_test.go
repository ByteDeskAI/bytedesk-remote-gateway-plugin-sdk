package natsconn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testSubjectLease = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// Mirrors the v1 SDK's TestSubjectLeaseContextDistinguishesMissingInvalidAndCanonical
// exactly: the shape of the problem (an opaque lease riding one HTTP header
// into a request context) is identical between the two transports.
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
