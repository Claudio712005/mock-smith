package interceptor

import (
	"net/http/httptest"
	"testing"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/Claudio712005/mock-smith/internal/scenario"
)

func scopedCtx(path string) *Context {
	res := scenario.Result{Kind: scenario.KindSuccess}
	return &Context{
		Request:  httptest.NewRequest("GET", "/x", nil),
		Endpoint: domain.Endpoint{Path: path},
		Result:   &res,
	}
}

func TestScoped_AppliesOnMatch(t *testing.T) {
	inner := FailureInterceptor{Status: 503, Rate: 1, Rand: func() float64 { return 0 }}
	ctx := scopedCtx("/payments")
	(ScopedInterceptor{Path: "/payments", Inner: inner}).Apply(ctx)
	if ctx.Result.Kind != scenario.KindServerError {
		t.Fatalf("matched scope did not apply inner: %+v", *ctx.Result)
	}
}

func TestScoped_SkipsOnMismatch(t *testing.T) {
	inner := FailureInterceptor{Status: 503, Rate: 1, Rand: func() float64 { return 0 }}
	ctx := scopedCtx("/pets")
	(ScopedInterceptor{Path: "/payments", Inner: inner}).Apply(ctx)
	if ctx.Result.Kind != scenario.KindSuccess {
		t.Fatalf("mismatched scope applied inner: %+v", *ctx.Result)
	}
}

func TestScoped_EmptyPathIsGlobal(t *testing.T) {
	inner := CorruptionInterceptor{Rate: 1, Rand: func() float64 { return 0 }}
	ctx := scopedCtx("/anything")
	(ScopedInterceptor{Path: "", Inner: inner}).Apply(ctx)
	if ctx.Result.Kind != scenario.KindMalformed {
		t.Fatalf("global scope did not apply: %+v", *ctx.Result)
	}
}
