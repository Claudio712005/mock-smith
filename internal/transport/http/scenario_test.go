package http

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/Claudio712005/mock-smith/internal/scenario"
	"github.com/getkin/kin-openapi/openapi3"
)

func endpointWithErrors() domain.Endpoint {
	schema := openapi3.NewObjectSchema()
	schema.WithProperty("message", openapi3.NewStringSchema())
	return domain.Endpoint{
		Method:          "POST",
		Path:            "/pay",
		SuccessResponse: &domain.ResponseSpec{StatusCode: 201, ContentType: "application/json", Schema: schema.NewRef()},
		ErrorResponses: []domain.ResponseSpec{
			{StatusCode: 400, ContentType: "application/json", Schema: schema.NewRef()},
			{StatusCode: 503, ContentType: "application/json", Schema: schema.NewRef()},
		},
	}
}

func serveWith(ep domain.Endpoint, res scenario.Result) *http.Response {
	srv := New(":0", []domain.Endpoint{ep}, scenario.Always(res))
	req := httptest.NewRequest(ep.Method, ep.Path, nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)
	return rec.Result()
}

func TestScenario_BusinessError_UsesDocumented4xx(t *testing.T) {
	resp := serveWith(endpointWithErrors(), scenario.Result{Kind: scenario.KindBusinessError})
	defer resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Fatalf("status = %d, want 400 (lowest documented 4xx)", resp.StatusCode)
	}
}

func TestScenario_BusinessError_FallbackGeneric(t *testing.T) {
	ep := domain.Endpoint{Method: "GET", Path: "/x", SuccessResponse: objectSpec(200)}
	resp := serveWith(ep, scenario.Result{Kind: scenario.KindBusinessError})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 fallback", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["code"] != float64(400) {
		t.Errorf("generic body code = %v, want 400", body["code"])
	}
}

func TestScenario_ServerError_UsesDocumentedStatus(t *testing.T) {
	resp := serveWith(endpointWithErrors(), scenario.Result{Kind: scenario.KindServerError, Status: 503})
	defer resp.Body.Close()
	if resp.StatusCode != 503 {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
}

func TestScenario_ServerError_GenericWhenUndocumented(t *testing.T) {
	ep := domain.Endpoint{Method: "GET", Path: "/x", SuccessResponse: objectSpec(200)}
	resp := serveWith(ep, scenario.Result{Kind: scenario.KindServerError, Status: 500})
	defer resp.Body.Close()
	if resp.StatusCode != 500 {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}

func TestScenario_ServerError_ZeroStatusDefaults500(t *testing.T) {
	ep := domain.Endpoint{Method: "GET", Path: "/x", SuccessResponse: objectSpec(200)}
	resp := serveWith(ep, scenario.Result{Kind: scenario.KindServerError})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 default", resp.StatusCode)
	}
}

func TestScenario_Malformed_InvalidJSON(t *testing.T) {
	ep := domain.Endpoint{Method: "GET", Path: "/x", SuccessResponse: objectSpec(200)}
	resp := serveWith(ep, scenario.Result{Kind: scenario.KindMalformed})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	var v any
	if err := json.Unmarshal(b, &v); err == nil {
		t.Fatalf("expected invalid JSON, but %q parsed cleanly", b)
	}
}

func TestScenario_Timeout_RespondsAfterDelay(t *testing.T) {
	ep := domain.Endpoint{Method: "GET", Path: "/x", SuccessResponse: objectSpec(200)}
	start := time.Now()
	resp := serveWith(ep, scenario.Result{Kind: scenario.KindTimeout, Delay: 30 * time.Millisecond})
	defer resp.Body.Close()
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("returned after %v, want >= 30ms", elapsed)
	}
	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want 504", resp.StatusCode)
	}
}

func TestScenario_Disconnect_RealServerClosesConn(t *testing.T) {
	ep := domain.Endpoint{Method: "GET", Path: "/x", SuccessResponse: objectSpec(200)}
	srv := New(":0", []domain.Endpoint{ep}, scenario.Always(scenario.Result{Kind: scenario.KindDisconnect}))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	_, err := ts.Client().Get(ts.URL + "/x")
	if err == nil {
		t.Fatal("expected connection error on disconnect, got nil")
	}
}

func TestScenario_Disconnect_RecorderFallback500(t *testing.T) {
	ep := domain.Endpoint{Method: "GET", Path: "/x", SuccessResponse: objectSpec(200)}
	resp := serveWith(ep, scenario.Result{Kind: scenario.KindDisconnect})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 fallback", resp.StatusCode)
	}
}
