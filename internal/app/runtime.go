package app

import (
	"fmt"
	"time"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/Claudio712005/mock-smith/internal/interceptor"
	"github.com/Claudio712005/mock-smith/internal/openapi"
	"github.com/Claudio712005/mock-smith/internal/scenario"
	httptransport "github.com/Claudio712005/mock-smith/internal/transport/http"
)

// Options reúne as configurações de um Runtime: caminho da spec, endereço de
// escuta, profile de execução e um status forçado opcional.
type Options struct {
	SpecPath    string
	Addr        string
	Profile     string
	ForceStatus int
	Slow        time.Duration
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

	if opts.Slow < 0 {
		return nil, fmt.Errorf("invalid --slow %s (must be >= 0)", opts.Slow)
	}

	var chain interceptor.Chain
	if opts.Slow > 0 {
		chain = append(chain, interceptor.LatencyInterceptor{Delay: opts.Slow})
	}

	return &Runtime{opts: opts, endpoints: endpoints, scenario: scen, chain: chain}, nil
}

// Run inicia o servidor HTTP e bloqueia até ele parar.
func (r *Runtime) Run() error {
	srv := httptransport.New(r.opts.Addr, r.endpoints, r.scenario, r.chain)

	fmt.Printf("MockSmith running on %s\n", r.opts.Addr)
	fmt.Printf("Loaded %d endpoints\n", len(r.endpoints))
	fmt.Printf("Profile: %s\n", r.opts.Profile)
	if r.opts.Slow > 0 {
		fmt.Printf("Adding %s latency to every response\n", r.opts.Slow)
	}
	if r.opts.ForceStatus != 0 {
		fmt.Printf("Forcing status %d on endpoints that document it\n", r.opts.ForceStatus)
	}

	return srv.ListenAndServe()
}
