// Package e2e exercises MockSmith end-to-end: a real OpenAPI spec is loaded,
// discovered, and served over a live HTTP listener, then probed with a client.
package e2e

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/Claudio712005/mock-smith/internal/interceptor"
	"github.com/Claudio712005/mock-smith/internal/openapi"
	"github.com/Claudio712005/mock-smith/internal/scenario"
	httptransport "github.com/Claudio712005/mock-smith/internal/transport/http"
)

func loadEndpoints(t *testing.T) []domain.Endpoint {
	t.Helper()
	doc, err := openapi.Load(filepath.Join("..", "examples", "petstore.yaml"))
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	eps := openapi.Discover(doc)
	if len(eps) == 0 {
		t.Fatal("no endpoints discovered")
	}
	return eps
}

func startServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptransport.New(":0", loadEndpoints(t), nil, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func startServerScenario(t *testing.T, scen scenario.Scenario) *httptest.Server {
	t.Helper()
	srv := httptransport.New(":0", loadEndpoints(t), scen, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func getJSON(t *testing.T, ts *httptest.Server, path string, out any) *http.Response {
	t.Helper()
	resp, err := ts.Client().Get(ts.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
	}
	return resp
}

func TestE2E_ListPets_GeneratesArray(t *testing.T) {
	ts := startServer(t)

	var pets []map[string]any
	resp := getJSON(t, ts, "/pets", &pets)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if len(pets) < 3 {
		t.Fatalf("got %d pets, want >= 3 (minItems)", len(pets))
	}
	first := pets[0]
	for _, k := range []string{"id", "name", "status"} {
		if _, ok := first[k]; !ok {
			t.Errorf("pet missing required field %q: %v", k, first)
		}
	}
	if first["status"] != "available" {
		t.Errorf("status = %v, want available (first enum)", first["status"])
	}
	if first["name"] != "Rex" {
		t.Errorf("name = %v, want Rex (schema example)", first["name"])
	}
}

func TestE2E_GetUser_UsesMediaExample(t *testing.T) {
	ts := startServer(t)

	var user map[string]any
	resp := getJSON(t, ts, "/users/anything", &user)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if user["name"] != "Ada Lovelace" {
		t.Errorf("name = %v, want Ada Lovelace (media example)", user["name"])
	}
	if user["email"] != "ada@example.com" {
		t.Errorf("email = %v, want example value", user["email"])
	}
}

func TestE2E_CreatePet_201(t *testing.T) {
	ts := startServer(t)

	resp, err := ts.Client().Post(ts.URL+"/pets", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /pets: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var pet map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&pet); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := pet["id"]; !ok {
		t.Errorf("created pet missing id: %v", pet)
	}
}

func TestE2E_DeletePet_204NoBody(t *testing.T) {
	ts := startServer(t)

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/pets/7", nil)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	if len(b) != 0 {
		t.Fatalf("204 body not empty: %q", b)
	}
}

func TestE2E_Admin_ListEndpoints(t *testing.T) {
	ts := startServer(t)

	var out struct {
		Count     int `json:"count"`
		Endpoints []struct {
			Method   string `json:"method"`
			Path     string `json:"path"`
			Statuses []int  `json:"statuses"`
		} `json:"endpoints"`
	}
	resp := getJSON(t, ts, "/__mocksmith/endpoints", &out)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if out.Count == 0 || out.Count != len(out.Endpoints) {
		t.Fatalf("count mismatch: count=%d len=%d", out.Count, len(out.Endpoints))
	}
	found := false
	for _, e := range out.Endpoints {
		if e.Method == "GET" && e.Path == "/pets" {
			found = true
		}
	}
	if !found {
		t.Error("admin list missing GET /pets")
	}
}

func TestE2E_Admin_EndpointDetail(t *testing.T) {
	ts := startServer(t)

	var d struct {
		Method  string `json:"method"`
		Path    string `json:"path"`
		Success *struct {
			Status  int  `json:"status"`
			HasBody bool `json:"hasBody"`
		} `json:"success"`
		Errors []struct {
			Status int `json:"status"`
		} `json:"errors"`
	}
	resp := getJSON(t, ts, "/__mocksmith/endpoint?method=post&path=/payments", &d)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if d.Success == nil || d.Success.Status != 201 {
		t.Fatalf("success wrong: %+v", d.Success)
	}
	has503 := false
	for _, e := range d.Errors {
		if e.Status == 503 {
			has503 = true
		}
	}
	if !has503 {
		t.Errorf("expected 503 in errors: %+v", d.Errors)
	}
}

func TestE2E_ForceStatus_OnlyMappedEndpoints(t *testing.T) {
	base, err := scenario.ForProfile("happy")
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	ts := startServerScenario(t, scenario.ForceStatus(503, base))

	resp, err := ts.Client().Post(ts.URL+"/payments", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /payments: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 503 {
		t.Fatalf("/payments status = %d, want 503 (mapped → forced)", resp.StatusCode)
	}

	resp2 := getJSON(t, ts, "/pets", nil)
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("/pets status = %d, want 200 (503 not mapped → happy)", resp2.StatusCode)
	}
}

func TestE2E_SlowLatency_DelaysResponse(t *testing.T) {
	chain := interceptor.Chain{interceptor.LatencyInterceptor{Delay: 40 * time.Millisecond}}
	srv := httptransport.New(":0", loadEndpoints(t), nil, chain)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	start := time.Now()
	resp := getJSON(t, ts, "/pets", nil)
	resp.Body.Close()
	if elapsed := time.Since(start); elapsed < 40*time.Millisecond {
		t.Fatalf("responded in %v, want >= 40ms (--slow latency)", elapsed)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestE2E_NotFound(t *testing.T) {
	ts := startServer(t)
	resp := getJSON(t, ts, "/does-not-exist", nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

// --- SpringBoot legacy spec (string examples on integer fields) ---

func loadSpringBootLegacyEndpoints(t *testing.T) []domain.Endpoint {
	t.Helper()
	doc, err := openapi.Load(filepath.Join("..", "examples", "springboot-legacy.json"))
	if err != nil {
		t.Fatalf("load springboot-legacy spec: %v", err)
	}
	eps := openapi.Discover(doc)
	if len(eps) == 0 {
		t.Fatal("no endpoints discovered in springboot-legacy spec")
	}
	return eps
}

func startSpringBootLegacyServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptransport.New(":0", loadSpringBootLegacyEndpoints(t), nil, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// TestE2E_SpringBootLegacy_SpecLoads verifica que specs geradas pelo Spring Boot
// com exemplos string em campos integer são carregadas sem erro de validação.
func TestE2E_SpringBootLegacy_SpecLoads(t *testing.T) {
	// loadSpringBootLegacyEndpoints chama openapi.Load internamente; se o fix de
	// DisableExamplesValidation não estiver presente, o teste falha aqui com:
	// "invalid example: validation failed due to: at '': got string, want integer"
	eps := loadSpringBootLegacyEndpoints(t)
	paths := make(map[string]bool, len(eps))
	for _, ep := range eps {
		paths[ep.Path] = true
	}
	for _, want := range []string{"/api/v1/pets", "/api/v2/pets"} {
		if !paths[want] {
			t.Errorf("endpoint %s not discovered", want)
		}
	}
}

// TestE2E_SpringBootLegacy_V1_Returns200 exercita o endpoint v1 com campos integer
// cujo exemplo é uma string ("0001", "05") — padrão Spring Boot codegen.
func TestE2E_SpringBootLegacy_V1_Returns200(t *testing.T) {
	ts := startSpringBootLegacyServer(t)

	resp, err := ts.Client().Post(ts.URL+"/api/v1/pets", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/v1/pets: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

// TestE2E_SpringBootLegacy_V2_Returns200 exercita o endpoint v2 cujo schema
// usa campos string com pattern alfanumérico (sem exemplos inválidos).
func TestE2E_SpringBootLegacy_V2_Returns200(t *testing.T) {
	ts := startSpringBootLegacyServer(t)

	resp, err := ts.Client().Post(ts.URL+"/api/v2/pets", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/v2/pets: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

// TestE2E_SpringBootLegacy_AdminListsEndpoints confirma que os dois endpoints
// do spec springboot-legacy aparecem na listagem administrativa do MockSmith.
func TestE2E_SpringBootLegacy_AdminListsEndpoints(t *testing.T) {
	ts := startSpringBootLegacyServer(t)

	var out struct {
		Count     int `json:"count"`
		Endpoints []struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		} `json:"endpoints"`
	}
	resp := getJSON(t, ts, "/__mocksmith/endpoints", &out)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	want := map[string]bool{
		"POST /api/v1/pets": false,
		"POST /api/v2/pets": false,
	}
	for _, ep := range out.Endpoints {
		key := ep.Method + " " + ep.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for key, found := range want {
		if !found {
			t.Errorf("admin list missing endpoint %s", key)
		}
	}
}
