package interceptor

import (
	"math/rand/v2"
	"sync"
	"time"
)

// Override descreve um comportamento aplicado a um endpoint em runtime, definido
// pela Admin API. Status (quando > 0) sobrescreve a resposta; LatencyMs (quando
// > 0) adiciona atraso; Rate (0..1) limita a fração de requisições afetadas pelo
// Status — 0 ou >= 1 significa sempre.
type Override struct {
	Status    int     `json:"status,omitempty"`
	LatencyMs int     `json:"latencyMs,omitempty"`
	Rate      float64 `json:"rate,omitempty"`
}

// Overrides guarda, de forma segura para concorrência, os overrides ativos por
// endpoint (chave = path do template OpenAPI). Leituras vêm do hot path das
// requisições; escritas, da Admin API.
type Overrides struct {
	mu sync.RWMutex
	m  map[string]Override
}

// NewOverrides cria um store vazio.
func NewOverrides() *Overrides {
	return &Overrides{m: make(map[string]Override)}
}

// Set define (ou substitui) o override de um endpoint.
func (o *Overrides) Set(path string, ov Override) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.m[path] = ov
}

// Get devolve o override de um endpoint, se houver.
func (o *Overrides) Get(path string) (Override, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	ov, ok := o.m[path]
	return ov, ok
}

// Delete remove o override de um endpoint, devolvendo se existia.
func (o *Overrides) Delete(path string) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	_, ok := o.m[path]
	delete(o.m, path)
	return ok
}

// Clear remove todos os overrides e devolve quantos havia.
func (o *Overrides) Clear() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	n := len(o.m)
	o.m = make(map[string]Override)
	return n
}

// Snapshot devolve uma cópia dos overrides ativos.
func (o *Overrides) Snapshot() map[string]Override {
	o.mu.RLock()
	defer o.mu.RUnlock()
	out := make(map[string]Override, len(o.m))
	for k, v := range o.m {
		out[k] = v
	}
	return out
}

// OverrideInterceptor aplica os overrides ativos do store à requisição. Mantém-se
// como ponteiro porque o store é mutável em runtime.
type OverrideInterceptor struct {
	store *Overrides
}

// NewOverrideInterceptor cria o interceptor que consulta store a cada requisição.
func NewOverrideInterceptor(store *Overrides) *OverrideInterceptor {
	return &OverrideInterceptor{store: store}
}

// Apply consulta o override do endpoint e, se houver, adiciona latência e/ou
// sobrescreve o status.
func (i *OverrideInterceptor) Apply(ctx *Context) error {
	ov, ok := i.store.Get(ctx.Endpoint.Path)
	if !ok {
		return nil
	}

	if ov.LatencyMs > 0 {
		t := time.NewTimer(time.Duration(ov.LatencyMs) * time.Millisecond)
		defer t.Stop()
		reqCtx := ctx.Request.Context()
		select {
		case <-t.C:
		case <-reqCtx.Done():
			return reqCtx.Err()
		}
	}

	if ov.Status > 0 && (ov.Rate <= 0 || ov.Rate >= 1 || rand.Float64() < ov.Rate) {
		*ctx.Result = statusToResult(ctx.Endpoint, ov.Status)
	}
	return nil
}
