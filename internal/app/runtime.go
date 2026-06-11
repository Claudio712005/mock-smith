package app

import (
	"fmt"
	"strings"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/Claudio712005/mock-smith/internal/faker"
	"github.com/Claudio712005/mock-smith/internal/interceptor"
	"github.com/Claudio712005/mock-smith/internal/openapi"
	"github.com/Claudio712005/mock-smith/internal/scenario"
	httptransport "github.com/Claudio712005/mock-smith/internal/transport/http"
)

// Options reúne as configurações de um Runtime: caminho da spec, endereço de
// escuta, profile de execução, status forçado e comportamentos por endpoint
// (latência, falha, timeout, corrupção). Cada campo por endpoint aceita várias
// entradas no formato "path=valor"; sem "=", a entrada vale para todos.
type Options struct {
	SpecPath    string
	Addr        string
	Profile     string
	ForceStatus int
	Slow        []string
	Fail        []string
	Timeout     []string
	Corrupt     []string
	Sequence    []string
	Overrides   map[string]interceptor.Override
	// GlobalRules aplica regras de valor a todos os endpoints do server.
	GlobalRules ValueRuleSet
	// EndpointRules são regras de valor por endpoint, com chave "[MÉTODO ]path".
	EndpointRules map[string]ValueRuleSet
	// ValueRules é a forma resolvida (por endpoint, chave "MÉTODO path"),
	// montada em New a partir de GlobalRules/EndpointRules e dos endpoints.
	ValueRules map[string]*faker.Rules
}

// ValueRuleSet descreve sobrescritas de geração: valores literais (Values) e
// tamanhos de lista (Counts), keyed pelo matcher (campo, caminho ou "$"). Cada
// item de Counts é um int ou nil (null no lugar da lista).
type ValueRuleSet struct {
	Values map[string][]any
	Counts map[string][]any
}

func (s ValueRuleSet) empty() bool { return len(s.Values) == 0 && len(s.Counts) == 0 }

// Runtime guarda os endpoints carregados e os serve via HTTP.
type Runtime struct {
	opts      Options
	endpoints []domain.Endpoint
	scenario  scenario.Scenario
	chain     interceptor.Chain
}

// New carrega e valida a spec, descobre os endpoints, resolve o profile e
// devolve um Runtime pronto (sem iniciar o servidor). Retorna erro se a spec for
// inválida, não tiver endpoints ou o profile for desconhecido.
func New(opts Options) (*Runtime, error) {
	scen, err := scenario.ForProfile(opts.Profile)
	if err != nil {
		return nil, err
	}
	if opts.ForceStatus != 0 {
		if opts.ForceStatus < 100 || opts.ForceStatus > 599 {
			return nil, fmt.Errorf("invalid --force-status %d (must be a valid HTTP status, 100-599)", opts.ForceStatus)
		}
		scen = scenario.ForceStatus(opts.ForceStatus, scen)
	}

	doc, err := openapi.Load(opts.SpecPath)
	if err != nil {
		return nil, err
	}

	endpoints := openapi.Discover(doc)
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints found in %q", opts.SpecPath)
	}

	chain, err := buildChain(opts)
	if err != nil {
		return nil, err
	}

	opts.ValueRules = buildValueRules(endpoints, opts.GlobalRules, opts.EndpointRules)

	return &Runtime{opts: opts, endpoints: endpoints, scenario: scen, chain: chain}, nil
}

// buildValueRules resolve, por endpoint descoberto, as regras de valor: começa
// das globais e sobrepõe as do endpoint cuja chave casa (path exato e método, ou
// path só). A chave do resultado é canônica "MÉTODO path".
func buildValueRules(endpoints []domain.Endpoint, global ValueRuleSet, byEndpoint map[string]ValueRuleSet) map[string]*faker.Rules {
	if global.empty() && len(byEndpoint) == 0 {
		return nil
	}
	out := make(map[string]*faker.Rules)
	for _, ep := range endpoints {
		r := faker.NewRules()
		applyRuleSet(r, global)
		for cfgKey, rs := range byEndpoint {
			method, path := interceptor.SplitTarget(cfgKey)
			if path == ep.Path && (method == "" || strings.EqualFold(method, ep.Method)) {
				applyRuleSet(r, rs)
			}
		}
		if !r.Empty() {
			out[ep.Method+" "+ep.Path] = r
		}
	}
	return out
}

func applyRuleSet(r *faker.Rules, rs ValueRuleSet) {
	for matcher, items := range rs.Values {
		r.SetValues(matcher, items)
	}
	for matcher, items := range rs.Counts {
		r.SetCount(matcher, items)
	}
}

// Run inicia o servidor HTTP e bloqueia até ele parar.
func (r *Runtime) Run() error {
	srv := httptransport.New(r.opts.Addr, r.endpoints, r.scenario, r.chain, r.opts.Overrides, r.opts.ValueRules)

	fmt.Printf("MockSmith running on %s\n", r.opts.Addr)
	fmt.Printf("Loaded %d endpoints\n", len(r.endpoints))
	fmt.Printf("Profile: %s\n", r.opts.Profile)
	if r.opts.ForceStatus != 0 {
		fmt.Printf("Forcing status %d on endpoints that document it\n", r.opts.ForceStatus)
	}
	if n := len(r.chain); n > 0 {
		fmt.Printf("Active interceptors: %d\n", n)
	}
	if n := len(r.opts.Overrides); n > 0 {
		fmt.Printf("Seeded %d runtime override(s)\n", n)
	}

	return srv.ListenAndServe()
}

// RunMany sobe vários servers concorrentemente, um por Options (cada um na sua
// porta). Valida e carrega todos antes de escutar — falha rápido se qualquer
// spec/profile for inválido. Bloqueia; devolve no primeiro server que sair ou
// falhar.
func RunMany(optsList []Options) error {
	if len(optsList) == 0 {
		return fmt.Errorf("no servers to run")
	}

	runtimes := make([]*Runtime, 0, len(optsList))
	for _, opts := range optsList {
		rt, err := New(opts)
		if err != nil {
			return fmt.Errorf("server %s: %w", opts.Addr, err)
		}
		runtimes = append(runtimes, rt)
	}

	errc := make(chan error, len(runtimes))
	for _, rt := range runtimes {
		rt := rt
		go func() { errc <- rt.Run() }()
	}
	return <-errc
}
