package interceptor

import (
	"math/rand/v2"
	"time"

	"github.com/Claudio712005/mock-smith/internal/scenario"
)

// FailureInterceptor sobrescreve o Result com um erro de servidor em uma fração
// Rate (0..1) das requisições.
type FailureInterceptor struct {
	Status int
	Rate   float64
	Rand   func() float64
}

// Apply, com probabilidade Rate, força um erro de servidor com Status.
func (f FailureInterceptor) Apply(ctx *Context) error {
	if fires(f.Rand, f.Rate) {
		status := f.Status
		if status == 0 {
			status = 500
		}
		*ctx.Result = scenario.Result{Kind: scenario.KindServerError, Status: status}
	}
	return nil
}

// TimeoutInterceptor sobrescreve o Result com um timeout (504 após Delay) em uma
// fração Rate das requisições.
type TimeoutInterceptor struct {
	Delay time.Duration
	Rate  float64
	Rand  func() float64
}

// Apply, com probabilidade Rate, força um timeout de duração Delay.
func (t TimeoutInterceptor) Apply(ctx *Context) error {
	if fires(t.Rand, t.Rate) {
		*ctx.Result = scenario.Result{Kind: scenario.KindTimeout, Delay: t.Delay}
	}
	return nil
}

// CorruptionInterceptor sobrescreve o Result com um corpo malformado em uma
// fração Rate das requisições.
type CorruptionInterceptor struct {
	Rate float64
	Rand func() float64
}

// Apply, com probabilidade Rate, força um corpo JSON malformado.
func (c CorruptionInterceptor) Apply(ctx *Context) error {
	if fires(c.Rand, c.Rate) {
		*ctx.Result = scenario.Result{Kind: scenario.KindMalformed}
	}
	return nil
}

func fires(r func() float64, rate float64) bool {
	if rate <= 0 {
		return false
	}
	if rate >= 1 {
		return true
	}
	if r == nil {
		r = rand.Float64
	}
	return r() < rate
}
