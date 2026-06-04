package interceptor

import (
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/Claudio712005/mock-smith/internal/scenario"
)

func seqCtx(ep domain.Endpoint) *Context {
	res := scenario.Result{Kind: scenario.KindSuccess}
	return &Context{Request: httptest.NewRequest("GET", "/x", nil), Endpoint: ep, Result: &res}
}

func TestSequence_AdvancesPerRequest(t *testing.T) {
	ep := domain.Endpoint{SuccessResponse: &domain.ResponseSpec{StatusCode: 200}}
	s := &SequenceInterceptor{Statuses: []int{202, 202, 200}}

	want := []struct {
		kind   scenario.Kind
		status int
	}{
		{scenario.KindServerError, 202},
		{scenario.KindServerError, 202},
		{scenario.KindSuccess, 0},
	}
	for i, w := range want {
		ctx := seqCtx(ep)
		s.Apply(ctx)
		if ctx.Result.Kind != w.kind || (w.kind == scenario.KindServerError && ctx.Result.Status != w.status) {
			t.Fatalf("req %d → %+v, want kind=%v status=%d", i+1, *ctx.Result, w.kind, w.status)
		}
	}
}

func TestSequence_StickyLast(t *testing.T) {
	ep := domain.Endpoint{SuccessResponse: &domain.ResponseSpec{StatusCode: 200}}
	s := &SequenceInterceptor{Statuses: []int{500, 200}}
	for i := 0; i < 5; i++ {
		s.Apply(seqCtx(ep))
	}
	ctx := seqCtx(ep)
	s.Apply(ctx)
	if ctx.Result.Kind != scenario.KindSuccess {
		t.Fatalf("after exhaustion got %+v, want sticky last (200 success)", *ctx.Result)
	}
}

func TestSequence_SuccessStatusUsesSuccess(t *testing.T) {
	ep := domain.Endpoint{SuccessResponse: &domain.ResponseSpec{StatusCode: 201}}
	s := &SequenceInterceptor{Statuses: []int{201}}
	ctx := seqCtx(ep)
	s.Apply(ctx)
	if ctx.Result.Kind != scenario.KindSuccess {
		t.Fatalf("got %+v, want success (201 == documented success)", *ctx.Result)
	}
}

func TestSequence_EmptyIsNoop(t *testing.T) {
	ep := domain.Endpoint{SuccessResponse: &domain.ResponseSpec{StatusCode: 200}}
	s := &SequenceInterceptor{}
	ctx := seqCtx(ep)
	s.Apply(ctx)
	if ctx.Result.Kind != scenario.KindSuccess {
		t.Fatalf("empty sequence mutated result: %+v", *ctx.Result)
	}
}

func TestSequence_ConcurrentAdvancesExactlyOncePerCall(t *testing.T) {
	ep := domain.Endpoint{SuccessResponse: &domain.ResponseSpec{StatusCode: 200}}
	s := &SequenceInterceptor{Statuses: []int{500, 500, 500, 200}}

	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			s.Apply(seqCtx(ep))
		}()
	}
	wg.Wait()

	if got := s.n.Load(); got != n {
		t.Fatalf("counter = %d after %d concurrent calls, want %d", got, n, n)
	}
}
