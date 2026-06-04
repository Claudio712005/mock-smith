package openapi

import (
	"os"
	"path/filepath"
	"testing"
)

const validSpec = `
openapi: 3.0.3
info:
  title: Test
  version: 1.0.0
paths:
  /ping:
    get:
      responses:
        '200':
          description: ok
`

func writeSpec(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	return path
}

func TestLoad_Valid(t *testing.T) {
	doc, err := Load(writeSpec(t, "spec.yaml", validSpec))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if doc == nil || doc.Paths == nil {
		t.Fatal("Load() returned empty doc")
	}
	if doc.Paths.Find("/ping") == nil {
		t.Fatal("Load() missing /ping path")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("Load(missing) expected error, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	if _, err := Load(writeSpec(t, "bad.yaml", "::: not yaml :::")); err == nil {
		t.Fatal("Load(invalid yaml) expected error, got nil")
	}
}

func TestLoad_InvalidSpec(t *testing.T) {
	const badSpec = `
openapi: 3.0.3
info:
  title: Broken
`
	if _, err := Load(writeSpec(t, "broken.yaml", badSpec)); err == nil {
		t.Fatal("Load(invalid spec) expected validation error, got nil")
	}
}

func TestLoad_ExampleSpec(t *testing.T) {
	doc, err := Load(filepath.Join("..", "..", "examples", "petstore.yaml"))
	if err != nil {
		t.Fatalf("Load(petstore) error = %v", err)
	}
	if doc.Paths.Find("/pets") == nil {
		t.Fatal("petstore missing /pets")
	}
}

func TestLoad_SpringBootLegacyJSON(t *testing.T) {
	doc, err := Load(filepath.Join("..", "..", "examples", "springboot-legacy.json"))
	if err != nil {
		t.Fatalf("Load(springboot-legacy) error = %v", err)
	}
	if doc.Paths == nil {
		t.Fatal("Load(springboot-legacy) returned doc with no paths")
	}
	if doc.Paths.Find("/api/v1/pets") == nil {
		t.Fatal("springboot-legacy missing /api/v1/pets")
	}
	if doc.Paths.Find("/api/v2/pets") == nil {
		t.Fatal("springboot-legacy missing /api/v2/pets")
	}
}

func TestLoad_StringExampleOnIntegerField(t *testing.T) {
	const spec = `
openapi: "3.1.0"
info:
  title: SpringGenerated
  version: 1.0.0
paths:
  /resource:
    post:
      operationId: create
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                code:
                  type: integer
                  format: int32
                  example: "0001"
                branch:
                  type: integer
                  format: int32
                  example: "05"
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  result:
                    type: string
`
	doc, err := Load(writeSpec(t, "spring.yaml", spec))
	if err != nil {
		t.Fatalf("Load(string-example-on-integer) error = %v; want nil (examples validation must be lenient)", err)
	}
	if doc.Paths.Find("/resource") == nil {
		t.Fatal("missing /resource path")
	}
}
