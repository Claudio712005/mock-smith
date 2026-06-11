package config

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/Claudio712005/mock-smith/internal/app"
)

func toOpts(t *testing.T, cfg *Config) app.Options {
	t.Helper()
	o, err := cfg.ServerConfig.ToOptions()
	if err != nil {
		t.Fatalf("ToOptions error = %v", err)
	}
	return o
}

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mocksmith.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

const full = `
spec: petstore.yaml
addr: ":9000"
profile: resilience
forceStatus: 503
endpoints:
  /payments:
    fail: 503
    slow: 800ms
  /pets:
    timeout: 20%
    corrupt: 10%
  /jobs/{id}:
    sequence: [202, 202, 200]
overrides:
  /users/{id}:
    status: 503
    latencyMs: 1000
    rate: 0.5
`

func TestLoad_FullConfig(t *testing.T) {
	cfg, err := Load(write(t, full))
	if err != nil {
		t.Fatalf("Load error = %v", err)
	}
	if cfg.Spec != "petstore.yaml" || cfg.Addr != ":9000" || cfg.Profile != "resilience" || cfg.ForceStatus != 503 {
		t.Fatalf("scalars wrong: %+v", cfg)
	}
	if cfg.Endpoints["/payments"].Fail != 503 || cfg.Endpoints["/payments"].Slow != "800ms" {
		t.Fatalf("payments endpoint wrong: %+v", cfg.Endpoints["/payments"])
	}
	if got := cfg.Endpoints["/jobs/{id}"].Sequence; len(got) != 3 || got[2] != 200 {
		t.Fatalf("sequence wrong: %v", got)
	}
	if cfg.Overrides["/users/{id}"].Status != 503 {
		t.Fatalf("override wrong: %+v", cfg.Overrides["/users/{id}"])
	}
}

func TestLoad_BundledExample(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "examples", "mocksmith.yaml"))
	if err != nil {
		t.Fatalf("Load(examples/mocksmith.yaml) error = %v", err)
	}
	if cfg.Spec == "" || len(cfg.Endpoints) == 0 || len(cfg.Overrides) == 0 {
		t.Fatalf("bundled example incomplete: %+v", cfg)
	}
	_ = toOpts(t, cfg)
}

func TestLoad_MissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("Load(missing) expected error, got nil")
	}
}

func TestLoad_UnknownField(t *testing.T) {
	if _, err := Load(write(t, "spec: x.yaml\nbogus: 1\n")); err == nil {
		t.Fatal("Load(unknown field) expected error (KnownFields), got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	if _, err := Load(write(t, "spec: [unterminated")); err == nil {
		t.Fatal("Load(bad yaml) expected error, got nil")
	}
}

func TestToOptions_TranslatesEndpoints(t *testing.T) {
	cfg, err := Load(write(t, full))
	if err != nil {
		t.Fatalf("Load error = %v", err)
	}
	opts := toOpts(t, cfg)

	if opts.SpecPath != "petstore.yaml" || opts.Addr != ":9000" || opts.Profile != "resilience" || opts.ForceStatus != 503 {
		t.Fatalf("options scalars wrong: %+v", opts)
	}
	assertContains(t, opts.Slow, "/payments=800ms")
	assertContains(t, opts.Fail, "/payments=503")
	assertContains(t, opts.Timeout, "/pets=20%")
	assertContains(t, opts.Corrupt, "/pets=10%")
	assertContains(t, opts.Sequence, "/jobs/{id}=202,202,200")

	ov, ok := opts.Overrides["/users/{id}"]
	if !ok || ov.Status != 503 || ov.LatencyMs != 1000 || ov.Rate != 0.5 {
		t.Fatalf("override translation wrong: %+v", opts.Overrides)
	}
}

func TestToOptions_StableOrder(t *testing.T) {
	const spec = `
spec: x.yaml
endpoints:
  /c: { fail: 500 }
  /a: { fail: 500 }
  /b: { fail: 500 }
`
	cfg, _ := Load(write(t, spec))
	opts := toOpts(t, cfg)
	got := append([]string(nil), opts.Fail...)
	want := []string{"/a=500", "/b=500", "/c=500"}
	if !sort.StringsAreSorted(got) || len(got) != 3 {
		t.Fatalf("fail order not stable/sorted: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestToOptions_EmptyEndpoints(t *testing.T) {
	cfg, _ := Load(write(t, "spec: x.yaml\n"))
	opts := toOpts(t, cfg)
	if opts.Slow != nil || opts.Fail != nil || opts.Overrides != nil {
		t.Fatalf("empty config produced non-nil lists: %+v", opts)
	}
}

func TestServerList_SingleInline(t *testing.T) {
	cfg, _ := Load(write(t, "spec: a.yaml\naddr: \":8080\"\n"))
	servers, err := cfg.ServerList()
	if err != nil {
		t.Fatalf("ServerList error = %v", err)
	}
	if len(servers) != 1 || servers[0].Spec != "a.yaml" {
		t.Fatalf("got %+v, want 1 inline server", servers)
	}
}

func TestServerList_Multi(t *testing.T) {
	const spec = `
servers:
  - spec: a.yaml
    addr: ":8080"
  - spec: b.yaml
    addr: ":8081"
    profile: chaos
`
	cfg, _ := Load(write(t, spec))
	servers, err := cfg.ServerList()
	if err != nil {
		t.Fatalf("ServerList error = %v", err)
	}
	if len(servers) != 2 || servers[1].Addr != ":8081" || servers[1].Profile != "chaos" {
		t.Fatalf("got %+v, want 2 servers", servers)
	}
}

func TestServerList_ConflictErrors(t *testing.T) {
	const spec = `
spec: top.yaml
servers:
  - spec: a.yaml
    addr: ":8080"
`
	cfg, _ := Load(write(t, spec))
	if _, err := cfg.ServerList(); err == nil {
		t.Fatal("ServerList with both inline spec and servers expected error, got nil")
	}
}

func TestToOptions_ValueRules(t *testing.T) {
	const spec = `
spec: a.yaml
values:
  cpf: [null, "111"]
count:
  enderecos: 2
endpoints:
  GET /pessoas:
    count:
      $: [1, 3, 6, 7]
    values:
      pessoa.cpf: ["222"]
`
	cfg, _ := Load(write(t, spec))
	opts := toOpts(t, cfg)

	if got := opts.GlobalRules.Values["cpf"]; len(got) != 2 || got[0] != nil || got[1] != "111" {
		t.Fatalf("global cpf = %v, want [nil 111]", got)
	}
	if got := opts.GlobalRules.Counts["enderecos"]; len(got) != 1 || got[0] != 2 {
		t.Fatalf("global enderecos count = %v, want [2]", got)
	}
	er, ok := opts.EndpointRules["GET /pessoas"]
	if !ok {
		t.Fatal("missing endpoint rules for GET /pessoas")
	}
	if got := er.Counts["$"]; len(got) != 4 || got[0] != 1 || got[3] != 7 {
		t.Fatalf("$ count = %v, want [1 3 6 7]", got)
	}
	if got := er.Values["pessoa.cpf"]; len(got) != 1 || got[0] != "222" {
		t.Fatalf("pessoa.cpf = %v, want [222]", got)
	}
}

func TestToOptions_CountNullAndInvalid(t *testing.T) {
	cfg, _ := Load(write(t, "spec: a.yaml\ncount:\n  $: [3, null, 1]\n"))
	opts := toOpts(t, cfg)
	got := opts.GlobalRules.Counts["$"]
	if len(got) != 3 || got[0] != 3 || got[1] != nil || got[2] != 1 {
		t.Fatalf("count = %v, want [3 nil 1]", got)
	}

	bad, _ := Load(write(t, "spec: a.yaml\ncount:\n  $: [\"three\"]\n"))
	if _, err := bad.ServerConfig.ToOptions(); err == nil {
		t.Fatal("count with non-int item expected error, got nil")
	}
}

func assertContains(t *testing.T, list []string, want string) {
	t.Helper()
	for _, v := range list {
		if v == want {
			return
		}
	}
	t.Errorf("list %v missing %q", list, want)
}
