// Package scenario decide, em runtime, qual comportamento uma requisição recebe:
// sucesso, erro de negócio, erro de servidor, timeout, corpo malformado ou
// desconexão. A escolha é feita por um Scenario (tipicamente um Profile com
// pesos), deixando o transport apenas executar o Result.
package scenario

import (
	"time"

	"github.com/Claudio712005/mock-smith/internal/domain"
)

// Kind enumera os comportamentos possíveis de uma resposta mockada.
type Kind int

const (
	// KindSuccess devolve a resposta de sucesso documentada do endpoint.
	KindSuccess Kind = iota
	// KindBusinessError devolve um erro de negócio documentado (4xx) do endpoint.
	KindBusinessError
	// KindServerError devolve um status de servidor fixo (ex.: 500, 503).
	KindServerError
	// KindTimeout dorme por Delay e então responde 504, simulando serviço travado.
	KindTimeout
	// KindMalformed responde 200 com um corpo JSON inválido.
	KindMalformed
	// KindDisconnect derruba a conexão sem responder.
	KindDisconnect
)

// Result descreve a decisão do scenario para uma requisição.
type Result struct {
	Kind   Kind
	Status int
	Delay  time.Duration
}

// RequestContext carrega o que o scenario pode inspecionar para decidir. No MVP
// a decisão é só probabilística, mas o endpoint já fica disponível para regras
// futuras (por path, método, etc.).
type RequestContext struct {
	Method   string
	Path     string
	Endpoint domain.Endpoint
}

// Scenario resolve o comportamento de uma requisição.
type Scenario interface {
	Resolve(*RequestContext) Result
}

// Always devolve um Scenario que sempre produz o mesmo Result. Útil para o
// profile happy, para testes e para overrides por endpoint no futuro.
func Always(r Result) Scenario { return alwaysScenario{r} }

type alwaysScenario struct{ result Result }

func (a alwaysScenario) Resolve(*RequestContext) Result { return a.result }
