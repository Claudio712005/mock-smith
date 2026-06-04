package interceptor

// ScopedInterceptor aplica Inner apenas às requisições cujo endpoint casa com
// Path. Path vazio significa global (todas as requisições). O casamento é exato
// contra o template OpenAPI do endpoint (ex.: "/pets/{petId}").
type ScopedInterceptor struct {
	Path  string
	Inner Interceptor
}

// Apply roda Inner se o escopo casar com o endpoint da requisição.
func (s ScopedInterceptor) Apply(ctx *Context) error {
	if s.Path != "" && s.Path != ctx.Endpoint.Path {
		return nil
	}
	return s.Inner.Apply(ctx)
}
