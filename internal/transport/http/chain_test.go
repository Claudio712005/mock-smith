package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/Claudio712005/mock-smith/internal/interceptor"
	"github.com/Claudio712005/mock-smith/internal/scenario"
)

func TestChain_LatencyDelaysResponse(t *testing.T) {
	ep := domain.Endpoint{Method: "GET", Path: "/x", SuccessResponse: objectSpec(200)}
	chain := interceptor.Chain{interceptor.LatencyInterceptor{Delay: 30 * time.Millisecond}}
	srv := New(":0", []domain.Endpoint{ep}, nil, chain, nil)

	req := httptest.NewRequest("GET", "/x", nil)
	rec := httptest.NewRecorder()
	start := time.Now()
	srv.router.ServeHTTP(rec, req)

	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("served in %v, want >= 30ms (latency)", elapsed)
	}
	if rec.Result().StatusCode != 200 {
		t.Fatalf("status = %d, want 200", rec.Result().StatusCode)
	}
}

func TestChain_FailureOverridesResultOnHappy(t *testing.T) {
	ep := domain.Endpoint{Method: "GET", Path: "/x", SuccessResponse: objectSpec(200)}
	chain := interceptor.Chain{interceptor.FailureInterceptor{
		Status: 500, Rate: 1, Rand: func() float64 { return 0 },
	}}
	srv := New(":0", []domain.Endpoint{ep}, nil, chain, nil)

	req := httptest.NewRequest("GET", "/x", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)

	if rec.Result().StatusCode != 500 {
		t.Fatalf("status = %d, want 500 (failure interceptor override)", rec.Result().StatusCode)
	}
}

func TestChain_CorruptionOverridesBody(t *testing.T) {
	ep := domain.Endpoint{Method: "GET", Path: "/x", SuccessResponse: objectSpec(200)}
	chain := interceptor.Chain{interceptor.CorruptionInterceptor{
		Rate: 1, Rand: func() float64 { return 0 },
	}}
	srv := New(":0", []domain.Endpoint{ep}, nil, chain, nil)

	req := httptest.NewRequest("GET", "/x", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)
	resp := rec.Result()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	if len(b) == 0 || b[len(b)-1] == '}' {
		t.Fatalf("body %q looks like valid JSON, want malformed", b)
	}
}

func TestChain_NilIsNoop(t *testing.T) {
	ep := domain.Endpoint{Method: "GET", Path: "/x", SuccessResponse: objectSpec(200)}
	srv := New(":0", []domain.Endpoint{ep}, scenario.Always(scenario.Result{Kind: scenario.KindSuccess}), nil, nil)

	req := httptest.NewRequest("GET", "/x", nil)
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)
	if rec.Result().StatusCode != 200 {
		t.Fatalf("status = %d, want 200", rec.Result().StatusCode)
	}
}
