// Package interceptor implementa um pipeline de comportamento de runtime
// (Chain of Responsibility) aplicado a cada requisição antes da resposta ser
// escrita. Cada Interceptor pode introduzir efeitos (ex.: latência) e/ou
// sobrescrever o Result do scenario (ex.: falha, timeout, corrupção), compondo
// comportamentos sem acoplar o transport a cada um deles.
package interceptor

import (
	"net/http"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/Claudio712005/mock-smith/internal/scenario"
)

// Context carrega o estado mutável de uma requisição ao longo do pipeline. Os
// interceptors leem Request/Endpoint e podem sobrescrever Result.
type Context struct {
	Request  *http.Request
	Endpoint domain.Endpoint
	Result   *scenario.Result
}

// Interceptor aplica um comportamento ao Context. Retornar erro aborta o
// pipeline (ex.: cliente desconectou durante a latência).
type Interceptor interface {
	Apply(*Context) error
}

// Chain é uma sequência de interceptors aplicada em ordem.
type Chain []Interceptor

// Apply roda cada interceptor em ordem, parando no primeiro erro.
func (c Chain) Apply(ctx *Context) error {
	for _, it := range c {
		if err := it.Apply(ctx); err != nil {
			return err
		}
	}
	return nil
}
