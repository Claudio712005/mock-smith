package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/Claudio712005/mock-smith/internal/faker"
	"github.com/Claudio712005/mock-smith/internal/scenario"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server encapsula um roteador chi configurado a partir dos endpoints OpenAPI.
type Server struct {
	addr   string
	router chi.Router
}

var defaultScenario = scenario.Always(scenario.Result{Kind: scenario.KindSuccess})

// New cria um Server no endereço addr (ex.: ":8080"), registrando uma rota por
// endpoint. Os templates OpenAPI ("/users/{id}") já casam com a sintaxe do chi.
// O scenario decide o comportamento de cada requisição; nil equivale a happy
// (sempre sucesso). Não inicia o servidor; use ListenAndServe.
func New(addr string, endpoints []domain.Endpoint, scen scenario.Scenario) *Server {
	if scen == nil {
		scen = defaultScenario
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	registerAdmin(r, endpoints)

	for _, ep := range endpoints {
		r.MethodFunc(ep.Method, ep.Path, makeHandler(ep, scen))
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

func makeHandler(ep domain.Endpoint, scen scenario.Scenario) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		res := scen.Resolve(&scenario.RequestContext{
			Method:   ep.Method,
			Path:     ep.Path,
			Endpoint: ep,
		})

		switch res.Kind {
		case scenario.KindBusinessError:
			writeBusinessError(w, ep)
		case scenario.KindServerError:
			writeServerError(w, ep, res.Status)
		case scenario.KindTimeout:
			writeTimeout(w, req, res.Delay)
		case scenario.KindMalformed:
			writeMalformed(w)
		case scenario.KindDisconnect:
			disconnect(w, ep)
		default:
			writeSuccess(w, ep)
		}
	}
}

func writeSuccess(w http.ResponseWriter, ep domain.Endpoint) {
	spec := ep.SuccessResponse
	if spec == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeSpec(w, ep, *spec)
}

func writeBusinessError(w http.ResponseWriter, ep domain.Endpoint) {
	for _, e := range ep.ErrorResponses {
		if e.StatusCode >= 400 && e.StatusCode < 500 {
			writeSpec(w, ep, e)
			return
		}
	}
	writeGenericError(w, http.StatusBadRequest)
}

func writeServerError(w http.ResponseWriter, ep domain.Endpoint, status int) {
	if status == 0 {
		status = http.StatusInternalServerError
	}
	for _, e := range ep.ErrorResponses {
		if e.StatusCode == status {
			writeSpec(w, ep, e)
			return
		}
	}
	writeGenericError(w, status)
}

func writeTimeout(w http.ResponseWriter, req *http.Request, delay time.Duration) {
	select {
	case <-time.After(delay):
		writeGenericError(w, http.StatusGatewayTimeout)
	case <-req.Context().Done():
	}
}

func writeMalformed(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"id": 1, "name": `))
}

func disconnect(w http.ResponseWriter, ep domain.Endpoint) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		writeGenericError(w, http.StatusInternalServerError)
		return
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		fmt.Printf("mocksmith: hijack error for %s %s: %v\n", ep.Method, ep.Path, err)
		return
	}
	_ = conn.Close()
}

func writeSpec(w http.ResponseWriter, ep domain.Endpoint, spec domain.ResponseSpec) {
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

func writeGenericError(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{
		"code":    status,
		"message": http.StatusText(status),
	})
}
