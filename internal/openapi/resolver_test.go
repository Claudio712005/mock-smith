package openapi

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func loadDoc(t *testing.T, body string) *openapi3.T {
	t.Helper()
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData([]byte(body))
	if err != nil {
		t.Fatalf("load data: %v", err)
	}
	return doc
}

func TestDiscover_NilDoc(t *testing.T) {
	if got := Discover(nil); got != nil {
		t.Fatalf("Discover(nil) = %v, want nil", got)
	}
	if got := Discover(&openapi3.T{}); got != nil {
		t.Fatalf("Discover(no paths) = %v, want nil", got)
	}
}

const multiSpec = `
openapi: 3.0.3
info:
  title: Multi
  version: 1.0.0
paths:
  /b:
    get:
      operationId: getB
      summary: get b
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  x: { type: string }
        '404':
          description: missing
          content:
            application/json:
              schema:
                type: object
                properties:
                  error: { type: string }
        '500':
          description: boom
          content:
            application/json:
              schema:
                type: string
    post:
      responses:
        '204':
          description: no content
  /a:
    get:
      responses:
        '201':
          description: created
          content:
            application/json:
              schema:
                type: string
        '200':
          description: ok lower wins
          content:
            application/json:
              schema:
                type: string
`

func TestDiscover_SortedStable(t *testing.T) {
	eps := Discover(loadDoc(t, multiSpec))
	want := []struct{ method, path string }{
		{"GET", "/a"},
		{"GET", "/b"},
		{"POST", "/b"},
	}
	if len(eps) != len(want) {
		t.Fatalf("got %d endpoints, want %d: %+v", len(eps), len(want), eps)
	}
	for i, w := range want {
		if eps[i].Method != w.method || eps[i].Path != w.path {
			t.Errorf("endpoint[%d] = %s %s, want %s %s", i, eps[i].Method, eps[i].Path, w.method, w.path)
		}
	}
}

func TestDiscover_LowestSuccessWins(t *testing.T) {
	eps := Discover(loadDoc(t, multiSpec))
	for _, ep := range eps {
		if ep.Path == "/a" && ep.Method == "GET" {
			if ep.SuccessResponse == nil {
				t.Fatal("/a GET missing success response")
			}
			if ep.SuccessResponse.StatusCode != 200 {
				t.Fatalf("/a GET success = %d, want 200 (lowest 2xx)", ep.SuccessResponse.StatusCode)
			}
		}
	}
}

func TestDiscover_MetadataAndErrors(t *testing.T) {
	eps := Discover(loadDoc(t, multiSpec))
	for _, ep := range eps {
		if ep.Path == "/b" && ep.Method == "GET" {
			if ep.OperationID != "getB" {
				t.Errorf("operationId = %q, want getB", ep.OperationID)
			}
			if ep.Summary != "get b" {
				t.Errorf("summary = %q, want 'get b'", ep.Summary)
			}
			if ep.SuccessResponse == nil || ep.SuccessResponse.StatusCode != 200 {
				t.Fatalf("success response wrong: %+v", ep.SuccessResponse)
			}
			if ep.SuccessResponse.ContentType != "application/json" {
				t.Errorf("content type = %q", ep.SuccessResponse.ContentType)
			}
			if len(ep.ErrorResponses) != 2 {
				t.Fatalf("got %d errors, want 2", len(ep.ErrorResponses))
			}
			if ep.ErrorResponses[0].StatusCode != 404 || ep.ErrorResponses[1].StatusCode != 500 {
				t.Errorf("errors not sorted: %d, %d", ep.ErrorResponses[0].StatusCode, ep.ErrorResponses[1].StatusCode)
			}
		}
	}
}

func TestDiscover_NoBodyResponse(t *testing.T) {
	eps := Discover(loadDoc(t, multiSpec))
	for _, ep := range eps {
		if ep.Path == "/b" && ep.Method == "POST" {
			if ep.SuccessResponse == nil {
				t.Fatal("/b POST missing 204 success")
			}
			if ep.SuccessResponse.StatusCode != 204 {
				t.Errorf("status = %d, want 204", ep.SuccessResponse.StatusCode)
			}
			if ep.SuccessResponse.HasBody() {
				t.Error("204 should have no body")
			}
		}
	}
}

const exampleSpec = `
openapi: 3.0.3
info:
  title: Ex
  version: 1.0.0
paths:
  /u:
    get:
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  name: { type: string }
              example:
                name: Ada
`

func TestDiscover_PicksExample(t *testing.T) {
	eps := Discover(loadDoc(t, exampleSpec))
	if len(eps) != 1 {
		t.Fatalf("got %d endpoints, want 1", len(eps))
	}
	spec := eps[0].SuccessResponse
	if spec == nil || spec.Example == nil {
		t.Fatalf("example not picked: %+v", spec)
	}
	m, ok := spec.Example.(map[string]any)
	if !ok || m["name"] != "Ada" {
		t.Fatalf("example wrong: %v", spec.Example)
	}
}
