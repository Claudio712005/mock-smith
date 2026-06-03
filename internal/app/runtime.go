package app

import (
	"fmt"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/Claudio712005/mock-smith/internal/openapi"
	httptransport "github.com/Claudio712005/mock-smith/internal/transport/http"
)

// Options reúne as configurações de um Runtime: caminho da spec, endereço de
// escuta e profile de execução (no MVP, apenas "happy").
type Options struct {
	SpecPath string
	Addr     string
	Profile  string
}

// Runtime guarda os endpoints carregados e os serve via HTTP.
type Runtime struct {
	opts      Options
	endpoints []domain.Endpoint
}

// New carrega e valida a spec, descobre os endpoints e devolve um Runtime
// pronto (sem iniciar o servidor). Retorna erro se a spec for inválida ou não
// tiver endpoints.
func New(opts Options) (*Runtime, error) {
	doc, err := openapi.Load(opts.SpecPath)
	if err != nil {
		return nil, err
	}

	endpoints := openapi.Discover(doc)
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints found in %q", opts.SpecPath)
	}

	return &Runtime{opts: opts, endpoints: endpoints}, nil
}

// Run inicia o servidor HTTP e bloqueia até ele parar.
func (r *Runtime) Run() error {
	srv := httptransport.New(r.opts.Addr, r.endpoints)

	fmt.Printf("MockSmith running on %s\n", r.opts.Addr)
	fmt.Printf("Loaded %d endpoints\n", len(r.endpoints))
	fmt.Printf("Profile: %s\n", r.opts.Profile)

	return srv.ListenAndServe()
}
