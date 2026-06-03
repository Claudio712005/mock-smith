package scenario

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"time"
)

const defaultTimeout = 30 * time.Second

type weighted struct {
	weight float64
	result Result
}

// Profile é um Scenario que sorteia o Result conforme pesos. A soma dos pesos
// não precisa ser 1; é normalizada no sorteio.
type Profile struct {
	name    string
	entries []weighted
	total   float64
	rand    func() float64
}

// Resolve sorteia um Result conforme os pesos do profile.
func (p *Profile) Resolve(*RequestContext) Result {
	if len(p.entries) == 0 {
		return Result{Kind: KindSuccess}
	}
	pick := p.rand() * p.total
	var acc float64
	for _, e := range p.entries {
		acc += e.weight
		if pick < acc {
			return e.result
		}
	}
	return p.entries[len(p.entries)-1].result
}

// Name devolve o identificador do profile (ex.: "chaos").
func (p *Profile) Name() string { return p.name }

func newProfile(name string, entries []weighted) *Profile {
	var total float64
	for _, e := range entries {
		total += e.weight
	}
	return &Profile{name: name, entries: entries, total: total, rand: rand.Float64}
}

var profiles = map[string]*Profile{
	"happy": newProfile("happy", []weighted{
		{1.0, Result{Kind: KindSuccess}},
	}),
	"sad": newProfile("sad", []weighted{
		{0.70, Result{Kind: KindSuccess}},
		{0.30, Result{Kind: KindBusinessError}},
	}),
	"resilience": newProfile("resilience", []weighted{
		{0.90, Result{Kind: KindSuccess}},
		{0.05, Result{Kind: KindTimeout, Delay: defaultTimeout}},
		{0.05, Result{Kind: KindServerError, Status: 503}},
	}),
	"chaos": newProfile("chaos", []weighted{
		{0.80, Result{Kind: KindSuccess}},
		{0.05, Result{Kind: KindMalformed}},
		{0.05, Result{Kind: KindTimeout, Delay: defaultTimeout}},
		{0.05, Result{Kind: KindDisconnect}},
		{0.05, Result{Kind: KindServerError, Status: 500}},
	}),
}

// ForProfile devolve o Scenario do profile nomeado. Retorna erro com a lista de
// profiles válidos quando o nome não existe.
func ForProfile(name string) (Scenario, error) {
	p, ok := profiles[name]
	if !ok {
		return nil, fmt.Errorf("unknown profile %q (available: %s)", name, available())
	}
	return p, nil
}

// Available lista os nomes de profile suportados, em ordem alfabética.
func Available() []string {
	out := make([]string, 0, len(profiles))
	for name := range profiles {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func available() string {
	names := Available()
	s := ""
	for i, n := range names {
		if i > 0 {
			s += ", "
		}
		s += n
	}
	return s
}
