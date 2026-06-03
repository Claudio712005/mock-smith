package interceptor

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Claudio712005/mock-smith/internal/scenario"
)

func ctxWith(kind scenario.Kind) *Context {
	res := scenario.Result{Kind: kind}
	return &Context{Request: httptest.NewRequest("GET", "/x", nil), Result: &res}
}

func always() func() float64 { return func() float64 { return 0.0 } }
func never() func() float64  { return func() float64 { return 0.99 } }

func TestFailure_FiresWhenRateHits(t *testing.T) {
	ctx := ctxWith(scenario.KindSuccess)
	(FailureInterceptor{Status: 503, Rate: 0.5, Rand: always()}).Apply(ctx)
	if ctx.Result.Kind != scenario.KindServerError || ctx.Result.Status != 503 {
		t.Fatalf("got %+v, want 503 server error", *ctx.Result)
	}
}

func TestFailure_SkipsWhenRateMisses(t *testing.T) {
	ctx := ctxWith(scenario.KindSuccess)
	(FailureInterceptor{Status: 503, Rate: 0.5, Rand: never()}).Apply(ctx)
	if ctx.Result.Kind != scenario.KindSuccess {
		t.Fatalf("got %+v, want untouched success", *ctx.Result)
	}
}

func TestFailure_DefaultStatus500(t *testing.T) {
	ctx := ctxWith(scenario.KindSuccess)
	(FailureInterceptor{Rate: 1, Rand: always()}).Apply(ctx)
	if ctx.Result.Status != 500 {
		t.Fatalf("status = %d, want 500 default", ctx.Result.Status)
	}
}

func TestTimeout_Fires(t *testing.T) {
	ctx := ctxWith(scenario.KindSuccess)
	(TimeoutInterceptor{Delay: 2 * time.Second, Rate: 1, Rand: always()}).Apply(ctx)
	if ctx.Result.Kind != scenario.KindTimeout || ctx.Result.Delay != 2*time.Second {
		t.Fatalf("got %+v, want 2s timeout", *ctx.Result)
	}
}

func TestCorruption_Fires(t *testing.T) {
	ctx := ctxWith(scenario.KindSuccess)
	(CorruptionInterceptor{Rate: 1, Rand: always()}).Apply(ctx)
	if ctx.Result.Kind != scenario.KindMalformed {
		t.Fatalf("got %+v, want malformed", *ctx.Result)
	}
}

func TestFires_Boundaries(t *testing.T) {
	if fires(always(), 0) {
		t.Error("rate 0 must never fire")
	}
	if !fires(never(), 1) {
		t.Error("rate 1 must always fire")
	}
	if !fires(always(), 0.5) {
		t.Error("rand 0.0 < 0.5 must fire")
	}
	if fires(never(), 0.5) {
		t.Error("rand 0.99 >= 0.5 must not fire")
	}
}
