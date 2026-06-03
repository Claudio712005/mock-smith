package interceptor

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Claudio712005/mock-smith/internal/scenario"
)

func TestLatency_Sleeps(t *testing.T) {
	res := scenario.Result{Kind: scenario.KindSuccess}
	ctx := &Context{Request: httptest.NewRequest("GET", "/x", nil), Result: &res}

	start := time.Now()
	if err := (LatencyInterceptor{Delay: 30 * time.Millisecond}).Apply(ctx); err != nil {
		t.Fatalf("Apply error = %v", err)
	}
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("slept %v, want >= 30ms", elapsed)
	}
}

func TestLatency_ZeroIsNoop(t *testing.T) {
	res := scenario.Result{Kind: scenario.KindSuccess}
	ctx := &Context{Request: httptest.NewRequest("GET", "/x", nil), Result: &res}

	start := time.Now()
	if err := (LatencyInterceptor{Delay: 0}).Apply(ctx); err != nil {
		t.Fatalf("Apply error = %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Millisecond {
		t.Fatalf("zero delay took %v, want ~0", elapsed)
	}
}

func TestLatency_AbortsOnClientCancel(t *testing.T) {
	cctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/x", nil).WithContext(cctx)
	res := scenario.Result{Kind: scenario.KindSuccess}
	ctx := &Context{Request: req, Result: &res}

	cancel()
	if err := (LatencyInterceptor{Delay: time.Hour}).Apply(ctx); err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}
