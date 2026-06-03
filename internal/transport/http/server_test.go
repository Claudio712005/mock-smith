package http

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/getkin/kin-openapi/openapi3"
)

func objectSpec(code int) *domain.ResponseSpec {
	schema := openapi3.NewObjectSchema()
	schema.WithProperty("name", openapi3.NewStringSchema())
	return &domain.ResponseSpec{
		StatusCode:  code,
		ContentType: "application/json",
		Schema:      schema.NewRef(),
	}
}

func do(t *testing.T, srv *Server, method, target string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)
	return rec.Result()
}

func TestServer_Addr(t *testing.T) {
	srv := New(":9999", nil, nil)
	if srv.Addr() != ":9999" {
		t.Fatalf("Addr() = %q, want :9999", srv.Addr())
	}
}

func TestHandler_GeneratesBody(t *testing.T) {
	ep := domain.Endpoint{Method: "GET", Path: "/things", SuccessResponse: objectSpec(200)}
	srv := New(":0", []domain.Endpoint{ep}, nil)

	resp := do(t, srv, "GET", "/things")
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q", ct)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if _, ok := body["name"]; !ok {
		t.Errorf("generated body missing 'name': %v", body)
	}
}

func TestHandler_UsesExample(t *testing.T) {
	spec := objectSpec(200)
	spec.Example = map[string]any{"name": "fixed"}
	ep := domain.Endpoint{Method: "GET", Path: "/ex", SuccessResponse: spec}
	srv := New(":0", []domain.Endpoint{ep}, nil)

	resp := do(t, srv, "GET", "/ex")
	defer resp.Body.Close()

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["name"] != "fixed" {
		t.Fatalf("example not used, body = %v", body)
	}
}

func TestHandler_NoSuccessResponse(t *testing.T) {
	ep := domain.Endpoint{Method: "DELETE", Path: "/gone"}
	srv := New(":0", []domain.Endpoint{ep}, nil)

	resp := do(t, srv, "DELETE", "/gone")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	if len(b) != 0 {
		t.Fatalf("expected empty body, got %q", b)
	}
}

func TestHandler_SuccessWithoutBody(t *testing.T) {
	ep := domain.Endpoint{
		Method:          "DELETE",
		Path:            "/items/{id}",
		SuccessResponse: &domain.ResponseSpec{StatusCode: 204},
	}
	srv := New(":0", []domain.Endpoint{ep}, nil)

	resp := do(t, srv, "DELETE", "/items/42")
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	if len(b) != 0 {
		t.Fatalf("expected empty body, got %q", b)
	}
}

func TestHandler_PathParamRoutes(t *testing.T) {
	ep := domain.Endpoint{Method: "GET", Path: "/users/{id}", SuccessResponse: objectSpec(200)}
	srv := New(":0", []domain.Endpoint{ep}, nil)

	resp := do(t, srv, "GET", "/users/abc123")
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("templated path status = %d, want 200", resp.StatusCode)
	}
}
