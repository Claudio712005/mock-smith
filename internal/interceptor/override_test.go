package interceptor

import (
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/Claudio712005/mock-smith/internal/scenario"
)

func ovCtx(path string) *Context {
	res := scenario.Result{Kind: scenario.KindSuccess}
	return &Context{
		Request:  httptest.NewRequest("GET", "/x", nil),
		Endpoint: domain.Endpoint{Path: path, SuccessResponse: &domain.ResponseSpec{StatusCode: 200}},
		Result:   &res,
	}
}

func TestOverrides_SetGetDeleteClear(t *testing.T) {
	o := NewOverrides()
	if _, ok := o.Get("/a"); ok {
		t.Fatal("empty store returned a value")
	}
	o.Set("/a", Override{Status: 500})
	if ov, ok := o.Get("/a"); !ok || ov.Status != 500 {
		t.Fatalf("Get = %+v,%v", ov, ok)
	}
	if !o.Delete("/a") {
		t.Fatal("Delete existing returned false")
	}
	if o.Delete("/a") {
		t.Fatal("Delete missing returned true")
	}
	o.Set("/a", Override{Status: 500})
	o.Set("/b", Override{LatencyMs: 10})
	if n := o.Clear(); n != 2 {
		t.Fatalf("Clear = %d, want 2", n)
	}
}

func TestOverrides_SnapshotIsCopy(t *testing.T) {
	o := NewOverrides()
	o.Set("/a", Override{Status: 503})
	snap := o.Snapshot()
	snap["/a"] = Override{Status: 200}
	if ov, _ := o.Get("/a"); ov.Status != 503 {
		t.Fatalf("snapshot mutation leaked into store: %+v", ov)
	}
}

func TestOverrideInterceptor_NoOverrideIsNoop(t *testing.T) {
	o := NewOverrides()
	ctx := ovCtx("/x")
	NewOverrideInterceptor(o).Apply(ctx)
	if ctx.Result.Kind != scenario.KindSuccess {
		t.Fatalf("noop mutated result: %+v", *ctx.Result)
	}
}

func TestOverrideInterceptor_StatusOverride(t *testing.T) {
	o := NewOverrides()
	o.Set("/x", Override{Status: 503})
	ctx := ovCtx("/x")
	NewOverrideInterceptor(o).Apply(ctx)
	if ctx.Result.Kind != scenario.KindServerError || ctx.Result.Status != 503 {
		t.Fatalf("got %+v, want 503 server error", *ctx.Result)
	}
}

func TestOverrideInterceptor_StatusEqualsSuccess(t *testing.T) {
	o := NewOverrides()
	o.Set("/x", Override{Status: 200})
	ctx := ovCtx("/x")
	NewOverrideInterceptor(o).Apply(ctx)
	if ctx.Result.Kind != scenario.KindSuccess {
		t.Fatalf("got %+v, want success (200 == documented success)", *ctx.Result)
	}
}

func TestOverrideInterceptor_LatencyOnly(t *testing.T) {
	o := NewOverrides()
	o.Set("/x", Override{LatencyMs: 20})
	ctx := ovCtx("/x")
	if err := NewOverrideInterceptor(o).Apply(ctx); err != nil {
		t.Fatalf("Apply error = %v", err)
	}
	if ctx.Result.Kind != scenario.KindSuccess {
		t.Fatalf("latency-only mutated status: %+v", *ctx.Result)
	}
}

func TestOverrideInterceptor_LiveUpdate(t *testing.T) {
	o := NewOverrides()
	it := NewOverrideInterceptor(o)

	ctx1 := ovCtx("/x")
	it.Apply(ctx1)
	if ctx1.Result.Kind != scenario.KindSuccess {
		t.Fatal("expected success before override set")
	}

	o.Set("/x", Override{Status: 503})
	ctx2 := ovCtx("/x")
	it.Apply(ctx2)
	if ctx2.Result.Status != 503 {
		t.Fatal("override not picked up live")
	}

	o.Delete("/x")
	ctx3 := ovCtx("/x")
	it.Apply(ctx3)
	if ctx3.Result.Kind != scenario.KindSuccess {
		t.Fatal("override not removed live")
	}
}

func TestOverrides_ConcurrentSafe(t *testing.T) {
	o := NewOverrides()
	it := NewOverrideInterceptor(o)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); o.Set("/x", Override{Status: 503}) }()
		go func() { defer wg.Done(); it.Apply(ovCtx("/x")) }()
	}
	wg.Wait()
}
