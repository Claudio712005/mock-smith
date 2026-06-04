package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Claudio712005/mock-smith/internal/domain"
)

func runtimeServer() *Server {
	ep := domain.Endpoint{Method: "POST", Path: "/payments", SuccessResponse: objectSpec(201)}
	return New(":0", []domain.Endpoint{ep}, nil, nil)
}

func postJSON(t *testing.T, srv *Server, target string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", target, bytes.NewReader(b))
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)
	return rec.Result()
}

func TestRuntime_SetAppliesOverride(t *testing.T) {
	srv := runtimeServer()

	resp := postJSON(t, srv, adminPrefix+"/runtime", map[string]any{"endpoint": "/payments", "status": 503})
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("set status = %d, want 200", resp.StatusCode)
	}

	out := do(t, srv, "POST", "/payments")
	out.Body.Close()
	if out.StatusCode != 503 {
		t.Fatalf("after override /payments = %d, want 503", out.StatusCode)
	}
}

func TestRuntime_DeleteRestoresBehavior(t *testing.T) {
	srv := runtimeServer()
	postJSON(t, srv, adminPrefix+"/runtime", map[string]any{"endpoint": "/payments", "status": 503}).Body.Close()

	req := httptest.NewRequest("DELETE", adminPrefix+"/runtime?endpoint=/payments", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)
	if rec.Result().StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d, want 200", rec.Result().StatusCode)
	}

	out := do(t, srv, "POST", "/payments")
	out.Body.Close()
	if out.StatusCode != 201 {
		t.Fatalf("after delete /payments = %d, want 201 (restored)", out.StatusCode)
	}
}

func TestRuntime_ClearAll(t *testing.T) {
	srv := runtimeServer()
	postJSON(t, srv, adminPrefix+"/runtime", map[string]any{"endpoint": "/payments", "latencyMs": 5}).Body.Close()

	req := httptest.NewRequest("DELETE", adminPrefix+"/runtime", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)
	var body struct {
		Cleared int `json:"cleared"`
	}
	json.NewDecoder(rec.Result().Body).Decode(&body)
	if body.Cleared != 1 {
		t.Fatalf("cleared = %d, want 1", body.Cleared)
	}
}

func TestRuntime_List(t *testing.T) {
	srv := runtimeServer()
	postJSON(t, srv, adminPrefix+"/runtime", map[string]any{"endpoint": "/payments", "status": 503}).Body.Close()

	resp := do(t, srv, "GET", adminPrefix+"/runtime")
	defer resp.Body.Close()
	var out struct {
		Count     int `json:"count"`
		Overrides map[string]struct {
			Status int `json:"status"`
		} `json:"overrides"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	if out.Count != 1 || out.Overrides["/payments"].Status != 503 {
		t.Fatalf("list = %+v, want 1 override 503", out)
	}
}

func TestRuntime_Validation(t *testing.T) {
	tests := []struct {
		name string
		body map[string]any
		want int
	}{
		{"missing endpoint", map[string]any{"status": 503}, http.StatusBadRequest},
		{"unknown endpoint", map[string]any{"endpoint": "/nope", "status": 503}, http.StatusNotFound},
		{"bad status", map[string]any{"endpoint": "/payments", "status": 42}, http.StatusBadRequest},
		{"negative latency", map[string]any{"endpoint": "/payments", "latencyMs": -1}, http.StatusBadRequest},
		{"rate out of range", map[string]any{"endpoint": "/payments", "status": 503, "rate": 2.0}, http.StatusBadRequest},
		{"nothing set", map[string]any{"endpoint": "/payments"}, http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := runtimeServer()
			resp := postJSON(t, srv, adminPrefix+"/runtime", tc.body)
			resp.Body.Close()
			if resp.StatusCode != tc.want {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.want)
			}
		})
	}
}

func TestRuntime_DeleteMissingReturns404(t *testing.T) {
	srv := runtimeServer()
	req := httptest.NewRequest("DELETE", adminPrefix+"/runtime?endpoint=/payments", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)
	if rec.Result().StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Result().StatusCode)
	}
}

func TestRuntime_BadJSON(t *testing.T) {
	srv := runtimeServer()
	req := httptest.NewRequest("POST", adminPrefix+"/runtime", bytes.NewReader([]byte("{not json")))
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)
	if rec.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Result().StatusCode)
	}
}
