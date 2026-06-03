package interceptor

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/Claudio712005/mock-smith/internal/scenario"
)

func newCtx() *Context {
	res := scenario.Result{Kind: scenario.KindSuccess}
	return &Context{
		Request: httptest.NewRequest("GET", "/x", nil),
		Result:  &res,
	}
}

type recorder struct {
	called int
	err    error
}

func (r *recorder) Apply(*Context) error {
	r.called++
	return r.err
}

func TestChain_AppliesInOrder(t *testing.T) {
	a, b := &recorder{}, &recorder{}
	if err := (Chain{a, b}).Apply(newCtx()); err != nil {
		t.Fatalf("Apply error = %v", err)
	}
	if a.called != 1 || b.called != 1 {
		t.Fatalf("called a=%d b=%d, want 1/1", a.called, b.called)
	}
}

func TestChain_StopsOnError(t *testing.T) {
	boom := errors.New("boom")
	a := &recorder{err: boom}
	b := &recorder{}
	if err := (Chain{a, b}).Apply(newCtx()); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
	if b.called != 0 {
		t.Fatalf("b called %d times, want 0 (chain should stop)", b.called)
	}
}

func TestChain_Empty(t *testing.T) {
	if err := (Chain{}).Apply(newCtx()); err != nil {
		t.Fatalf("empty chain error = %v", err)
	}
}
