package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/Claudio712005/mock-smith/internal/faker"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server encapsula um roteador chi configurado a partir dos endpoints OpenAPI.
type Server struct {
	addr   string
	router chi.Router
}

// New cria um Server no endereço addr (ex.: ":8080"), registrando uma rota por
// endpoint. Os templates OpenAPI ("/users/{id}") já casam com a sintaxe do chi.
// Não inicia o servidor; use ListenAndServe.
func New(addr string, endpoints []domain.Endpoint) *Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	registerAdmin(r, endpoints)

	for _, ep := range endpoints {
		r.MethodFunc(ep.Method, ep.Path, makeHandler(ep))
	}

	return &Server{addr: addr, router: r}
}

// Addr devolve o endereço de escuta.
func (s *Server) Addr() string { return s.addr }

// Handler expõe o roteador configurado como http.Handler, permitindo servir os
// endpoints sem iniciar um listener (útil para testes e composição).
func (s *Server) Handler() http.Handler { return s.router }

// ListenAndServe inicia o servidor HTTP e bloqueia.
func (s *Server) ListenAndServe() error {
	return http.ListenAndServe(s.addr, s.router)
}

func makeHandler(ep domain.Endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		spec := ep.SuccessResponse
		if spec == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if !spec.HasBody() {
			w.WriteHeader(spec.StatusCode)
			return
		}

		body := spec.Example
		if body == nil {
			body = faker.Generate(spec.Schema)
		}

		w.Header().Set("Content-Type", spec.ContentType)
		w.WriteHeader(spec.StatusCode)
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(body); err != nil {
			fmt.Printf("mocksmith: encode error for %s %s: %v\n", ep.Method, ep.Path, err)
		}
	}
}
