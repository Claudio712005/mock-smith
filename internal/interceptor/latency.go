package interceptor

import "time"

// LatencyInterceptor atrasa toda requisição por Delay, simulando um serviço
// lento. Respeita o cancelamento do cliente: se a conexão cair durante a espera,
// aborta o pipeline.
type LatencyInterceptor struct {
	Delay time.Duration
}

// Apply dorme por Delay ou retorna o erro do contexto se o cliente desistir.
func (l LatencyInterceptor) Apply(ctx *Context) error {
	if l.Delay <= 0 {
		return nil
	}
	t := time.NewTimer(l.Delay)
	defer t.Stop()

	reqCtx := ctx.Request.Context()
	select {
	case <-t.C:
		return nil
	case <-reqCtx.Done():
		return reqCtx.Err()
	}
}
