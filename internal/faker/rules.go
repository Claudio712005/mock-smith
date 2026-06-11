package faker

import "sync/atomic"

// cycle devolve os elementos de uma lista em ordem, um por chamada, reiniciando
// do começo quando chega ao fim. Seguro para uso concorrente.
type cycle struct {
	items []any
	n     atomic.Uint64
}

func newCycle(items []any) *cycle { return &cycle{items: items} }

func (c *cycle) next() any {
	if len(c.items) == 0 {
		return nil
	}
	i := int((c.n.Add(1) - 1) % uint64(len(c.items)))
	return c.items[i]
}

// Rules guarda sobrescritas de geração por campo: valores literais (values) e
// tamanhos de lista (count). A chave de cada regra é um "matcher": o caminho do
// campo a partir da raiz ("pessoa.cpf"), o nome solto do campo ("cpf") ou "$"
// para a própria resposta raiz. É stateful (os ciclos avançam por chamada) e
// seguro para concorrência.
type Rules struct {
	values map[string]*cycle
	counts map[string]*cycle
}

// NewRules cria um conjunto de regras vazio.
func NewRules() *Rules {
	return &Rules{values: map[string]*cycle{}, counts: map[string]*cycle{}}
}

// SetValues registra o ciclo de valores literais de um matcher (campo/caminho/"$").
func (r *Rules) SetValues(matcher string, items []any) {
	r.values[matcher] = newCycle(items)
}

// SetCount registra o ciclo de tamanhos de lista de um matcher. Cada item deve
// ser um int (quantidade) ou nil (devolve null no lugar da lista).
func (r *Rules) SetCount(matcher string, items []any) {
	r.counts[matcher] = newCycle(items)
}

// Empty indica que não há nenhuma regra (atalho para gerar sem custo).
func (r *Rules) Empty() bool {
	return r == nil || (len(r.values) == 0 && len(r.counts) == 0)
}

// value busca o ciclo de valores para um campo: primeiro pelo caminho exato,
// depois pelo nome solto (o caminho tem prioridade).
func (r *Rules) value(path, name string) (*cycle, bool) {
	if c, ok := r.values[path]; ok {
		return c, true
	}
	if name != "" {
		if c, ok := r.values[name]; ok {
			return c, true
		}
	}
	return nil, false
}

// count busca o ciclo de tamanhos para uma lista, na mesma ordem de prioridade.
func (r *Rules) count(path, name string) (*cycle, bool) {
	if c, ok := r.counts[path]; ok {
		return c, true
	}
	if name != "" {
		if c, ok := r.counts[name]; ok {
			return c, true
		}
	}
	return nil, false
}
