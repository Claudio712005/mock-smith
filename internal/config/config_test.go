package config

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

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
	_ = cfg.ToOptions()
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
	opts := cfg.ToOptions()

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
	opts := cfg.ToOptions()
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
	opts := cfg.ToOptions()
	if opts.Slow != nil || opts.Fail != nil || opts.Overrides != nil {
		t.Fatalf("empty config produced non-nil lists: %+v", opts)
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
