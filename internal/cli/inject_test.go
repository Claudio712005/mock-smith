package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseInject_Valid(t *testing.T) {
	tests := []struct {
		in     string
		path   string
		status int
		rate   float64
	}{
		{"/payments:503(20%)", "/payments", 503, 0.2},
		{"/payments:503", "/payments", 503, 0},
		{"/payments", "/payments", 0, 0},
		{"/a/b/{id}:500(100%)", "/a/b/{id}", 500, 1},
		{"/x:404(0%)", "/x", 404, 0},
	}
	for _, tc := range tests {
		path, status, rate, err := parseInject(tc.in)
		if err != nil {
			t.Errorf("parseInject(%q) error = %v", tc.in, err)
			continue
		}
		if path != tc.path || status != tc.status || rate != tc.rate {
			t.Errorf("parseInject(%q) = (%q,%d,%v), want (%q,%d,%v)", tc.in, path, status, rate, tc.path, tc.status, tc.rate)
		}
	}
}

func TestParseInject_Invalid(t *testing.T) {
	for _, bad := range []string{
		"",
		":503",
		"/x:99",
		"/x:700",
		"/x:abc",
		"/x:503(abc%)",
		"/x:503(150%)",
		"/x:503(20)",
	} {
		if _, _, _, err := parseInject(bad); err == nil {
			t.Errorf("parseInject(%q) expected error", bad)
		}
	}
}

func TestInjectClient_SetListDelete(t *testing.T) {
	var gotMethod, gotBody, gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotQuery = r.URL.RawQuery
		if r.Body != nil {
			buf := make([]byte, r.ContentLength)
			if r.ContentLength > 0 {
				r.Body.Read(buf)
			}
			gotBody = string(buf)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()
	base := ts.URL + adminRuntimePath

	if err := injectSet(base, "/payments", 503, 200, 0.2); err != nil {
		t.Fatalf("injectSet error = %v", err)
	}
	if gotMethod != "POST" || !strings.Contains(gotBody, `"status":503`) || !strings.Contains(gotBody, `"latencyMs":200`) {
		t.Fatalf("set request wrong: method=%s body=%s", gotMethod, gotBody)
	}

	if err := injectList(base); err != nil {
		t.Fatalf("injectList error = %v", err)
	}
	if gotMethod != "GET" {
		t.Fatalf("list method = %s, want GET", gotMethod)
	}

	if err := injectDelete(base, "/payments"); err != nil {
		t.Fatalf("injectDelete error = %v", err)
	}
	if gotMethod != "DELETE" || !strings.Contains(gotQuery, "endpoint=") {
		t.Fatalf("delete request wrong: method=%s query=%s", gotMethod, gotQuery)
	}
}

func TestInjectClient_ErrorStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"nope"}`))
	}))
	defer ts.Close()

	if err := injectList(ts.URL + adminRuntimePath); err == nil {
		t.Fatal("expected error on 404 response, got nil")
	}
}
