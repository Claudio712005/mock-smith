package interceptor

import (
	"sync/atomic"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/Claudio712005/mock-smith/internal/scenario"
)

// SequenceInterceptor devolve os Statuses em ordem, um por requisição: a 1ª
// resposta usa Statuses[0], a 2ª Statuses[1], e assim por diante. Esgotada a
// lista, fixa no último (útil para polling, ex.: 202,202,200). É stateful e
// seguro para uso concorrente; use sempre como ponteiro para preservar o estado.
type SequenceInterceptor struct {
	Statuses []int
	n        atomic.Uint64
}

// Apply avança a sequência e sobrescreve o Result com o status da vez.
func (s *SequenceInterceptor) Apply(ctx *Context) error {
	if len(s.Statuses) == 0 {
		return nil
	}
	i := int(s.n.Add(1) - 1)
	if i >= len(s.Statuses) {
		i = len(s.Statuses) - 1
	}
	*ctx.Result = statusToResult(ctx.Endpoint, s.Statuses[i])
	return nil
}

func statusToResult(ep domain.Endpoint, status int) scenario.Result {
	if ep.SuccessResponse != nil && ep.SuccessResponse.StatusCode == status {
		return scenario.Result{Kind: scenario.KindSuccess}
	}
	return scenario.Result{Kind: scenario.KindServerError, Status: status}
}
