package app

import (
	"os"
	"path/filepath"
	"testing"
)

const pingSpec = `
openapi: 3.0.3
info:
  title: T
  version: 1.0.0
paths:
  /ping:
    get:
      responses:
        '200':
          description: ok
`

func writeSpec(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "spec.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	return path
}

func TestNew_Success(t *testing.T) {
	const spec = `
openapi: 3.0.3
info:
  title: T
  version: 1.0.0
paths:
  /ping:
    get:
      responses:
        '200':
          description: ok
`
	rt, err := New(Options{SpecPath: writeSpec(t, spec), Addr: ":0", Profile: "happy"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if len(rt.endpoints) != 1 {
		t.Fatalf("endpoints = %d, want 1", len(rt.endpoints))
	}
}

func TestNew_BadSpecPath(t *testing.T) {
	if _, err := New(Options{SpecPath: "/no/such/file.yaml", Profile: "happy"}); err == nil {
		t.Fatal("New(bad path) expected error, got nil")
	}
}

func TestNew_NoEndpoints(t *testing.T) {
	const spec = `
openapi: 3.0.3
info:
  title: Empty
  version: 1.0.0
paths: {}
`
	_, err := New(Options{SpecPath: writeSpec(t, spec), Profile: "happy"})
	if err == nil {
		t.Fatal("New(no endpoints) expected error, got nil")
	}
}

func TestNew_ForceStatus_Valid(t *testing.T) {
	const spec = `
openapi: 3.0.3
info:
  title: T
  version: 1.0.0
paths:
  /ping:
    get:
      responses:
        '200':
          description: ok
`
	if _, err := New(Options{SpecPath: writeSpec(t, spec), Profile: "happy", ForceStatus: 500}); err != nil {
		t.Fatalf("New(force-status 500) error = %v", err)
	}
}

func TestNew_ForceStatus_Invalid(t *testing.T) {
	if _, err := New(Options{SpecPath: "irrelevant", Profile: "happy", ForceStatus: 42}); err == nil {
		t.Fatal("New(force-status 42) expected error, got nil")
	}
}

func TestNew_UnknownProfile(t *testing.T) {
	_, err := New(Options{SpecPath: writeSpec(t, pingSpec), Profile: "bogus"})
	if err == nil {
		t.Fatal("New(unknown profile) expected error, got nil")
	}
}

func TestNew_Slow_Valid(t *testing.T) {
	if _, err := New(Options{SpecPath: writeSpec(t, pingSpec), Profile: "happy", Slow: []string{"/ping=2s", "1s"}}); err != nil {
		t.Fatalf("New(slow) error = %v", err)
	}
}

func TestNew_Slow_Invalid(t *testing.T) {
	if _, err := New(Options{SpecPath: writeSpec(t, pingSpec), Profile: "happy", Slow: []string{"/ping=nope"}}); err == nil {
		t.Fatal("New(slow bad duration) expected error, got nil")
	}
}

func TestNew_Fail_Valid(t *testing.T) {
	if _, err := New(Options{SpecPath: writeSpec(t, pingSpec), Profile: "happy", Fail: []string{"/ping=503"}}); err != nil {
		t.Fatalf("New(fail) error = %v", err)
	}
}

func TestNew_Fail_Invalid(t *testing.T) {
	if _, err := New(Options{SpecPath: writeSpec(t, pingSpec), Profile: "happy", Fail: []string{"/ping=99"}}); err == nil {
		t.Fatal("New(fail bad status) expected error, got nil")
	}
}

func TestNew_Timeout_Valid(t *testing.T) {
	if _, err := New(Options{SpecPath: writeSpec(t, pingSpec), Profile: "happy", Timeout: []string{"/ping=20%"}}); err != nil {
		t.Fatalf("New(timeout) error = %v", err)
	}
}

func TestNew_Corrupt_Invalid(t *testing.T) {
	if _, err := New(Options{SpecPath: writeSpec(t, pingSpec), Profile: "happy", Corrupt: []string{"/ping=200%"}}); err == nil {
		t.Fatal("New(corrupt rate>100%) expected error, got nil")
	}
}
