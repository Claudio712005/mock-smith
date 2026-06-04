package app

import (
	"fmt"

	"github.com/Claudio712005/mock-smith/internal/domain"
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
}

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

	return &Runtime{opts: opts, endpoints: endpoints, scenario: scen, chain: chain}, nil
}

// Run inicia o servidor HTTP e bloqueia até ele parar.
func (r *Runtime) Run() error {
	srv := httptransport.New(r.opts.Addr, r.endpoints, r.scenario, r.chain)

	fmt.Printf("MockSmith running on %s\n", r.opts.Addr)
	fmt.Printf("Loaded %d endpoints\n", len(r.endpoints))
	fmt.Printf("Profile: %s\n", r.opts.Profile)
	if r.opts.ForceStatus != 0 {
		fmt.Printf("Forcing status %d on endpoints that document it\n", r.opts.ForceStatus)
	}
	if n := len(r.chain); n > 0 {
		fmt.Printf("Active interceptors: %d\n", n)
	}

	return srv.ListenAndServe()
}
