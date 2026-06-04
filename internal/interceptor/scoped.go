package interceptor

import "strings"

// ScopedInterceptor aplica Inner apenas às requisições cujo endpoint casa com
// Method e Path. Path vazio = qualquer path; Method vazio = qualquer método. O
// casamento de path é exato contra o template OpenAPI (ex.: "/pets/{petId}") e o
// de método é case-insensitive.
type ScopedInterceptor struct {
	Method string
	Path   string
	Inner  Interceptor
}

// Apply roda Inner se método e path casarem com o endpoint da requisição.
func (s ScopedInterceptor) Apply(ctx *Context) error {
	if s.Method != "" && !strings.EqualFold(s.Method, ctx.Endpoint.Method) {
		return nil
	}
	if s.Path != "" && s.Path != ctx.Endpoint.Path {
		return nil
	}
	return s.Inner.Apply(ctx)
}

// SplitTarget separa um alvo "METHOD path" em método (em maiúsculas) e path. Sem
// espaço, o método fica vazio (qualquer método). Ex.: "POST /pets" → ("POST",
// "/pets"); "/pets" → ("", "/pets").
func SplitTarget(s string) (method, path string) {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, ' '); i >= 0 {
		return strings.ToUpper(strings.TrimSpace(s[:i])), strings.TrimSpace(s[i+1:])
	}
	return "", s
}
