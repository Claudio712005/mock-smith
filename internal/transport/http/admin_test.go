package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/getkin/kin-openapi/openapi3"
)

func sampleEndpoints() []domain.Endpoint {
	schema := openapi3.NewObjectSchema()
	schema.WithProperty("name", openapi3.NewStringSchema())
	return []domain.Endpoint{
		{
			Method:      "GET",
			Path:        "/pets",
			OperationID: "listPets",
			Summary:     "List pets",
			SuccessResponse: &domain.ResponseSpec{
				StatusCode: 200, ContentType: "application/json", Schema: schema.NewRef(),
			},
			ErrorResponses: []domain.ResponseSpec{
				{StatusCode: 500, ContentType: "application/json", Schema: schema.NewRef()},
			},
		},
		{
			Method:          "DELETE",
			Path:            "/pets/{id}",
			SuccessResponse: &domain.ResponseSpec{StatusCode: 204},
		},
	}
}

func TestAdmin_ListEndpoints(t *testing.T) {
	srv := New(":0", sampleEndpoints(), nil, nil, nil)
	resp := do(t, srv, "GET", adminPrefix+"/endpoints")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out struct {
		Count     int               `json:"count"`
		Endpoints []endpointSummary `json:"endpoints"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Count != 2 || len(out.Endpoints) != 2 {
		t.Fatalf("count = %d, endpoints = %d, want 2/2", out.Count, len(out.Endpoints))
	}
	first := out.Endpoints[0]
	if first.Method != "GET" || first.Path != "/pets" || first.OperationID != "listPets" {
		t.Errorf("summary wrong: %+v", first)
	}
	if len(first.Statuses) != 2 || first.Statuses[0] != 200 || first.Statuses[1] != 500 {
		t.Errorf("statuses = %v, want [200 500]", first.Statuses)
	}
}

func TestAdmin_GetEndpoint_Found(t *testing.T) {
	srv := New(":0", sampleEndpoints(), nil, nil, nil)
	resp := do(t, srv, "GET", adminPrefix+"/endpoint?method=get&path=/pets")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var d endpointDetail
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if d.Method != "GET" || d.Path != "/pets" {
		t.Errorf("detail wrong: %+v", d)
	}
	if d.Success == nil || d.Success.Status != 200 || !d.Success.HasBody {
		t.Errorf("success info wrong: %+v", d.Success)
	}
	if len(d.Errors) != 1 || d.Errors[0].Status != 500 {
		t.Errorf("errors wrong: %+v", d.Errors)
	}
}

func TestAdmin_GetEndpoint_MissingParams(t *testing.T) {
	srv := New(":0", sampleEndpoints(), nil, nil, nil)
	resp := do(t, srv, "GET", adminPrefix+"/endpoint?method=get")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestAdmin_GetEndpoint_NotFound(t *testing.T) {
	srv := New(":0", sampleEndpoints(), nil, nil, nil)
	resp := do(t, srv, "GET", adminPrefix+"/endpoint?method=get&path=/nope")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestAdmin_GetEndpoint_MethodCaseInsensitive(t *testing.T) {
	srv := New(":0", sampleEndpoints(), nil, nil, nil)
	resp := do(t, srv, "GET", adminPrefix+"/endpoint?method=delete&path=/pets/{id}")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}
