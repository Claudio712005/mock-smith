package http

import (
	"encoding/json"
	"testing"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/Claudio712005/mock-smith/internal/faker"
	"github.com/getkin/kin-openapi/openapi3"
)

func arrayEndpoint() domain.Endpoint {
	item := openapi3.NewObjectSchema()
	item.WithProperty("name", openapi3.NewStringSchema())
	arr := openapi3.NewArraySchema()
	arr.Items = item.NewRef()
	arr.MinItems = 1
	return domain.Endpoint{
		Method:          "GET",
		Path:            "/xs",
		SuccessResponse: &domain.ResponseSpec{StatusCode: 200, ContentType: "application/json", Schema: arr.NewRef()},
	}
}

func TestHandler_AppliesValueRules(t *testing.T) {
	ep := arrayEndpoint()
	rules := faker.NewRules()
	rules.SetCount("$", []any{2})
	rules.SetValues("name", []any{"FIXO"})

	srv := New(":0", []domain.Endpoint{ep}, nil, nil, nil, map[string]*faker.Rules{
		"GET /xs": rules,
	})

	resp := do(t, srv, "GET", "/xs")
	defer resp.Body.Close()

	var body []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 2 {
		t.Fatalf("len = %d, want 2 (count rule)", len(body))
	}
	if body[0]["name"] != "FIXO" {
		t.Fatalf("name = %v, want FIXO (value rule)", body[0]["name"])
	}
}

func TestHandler_NoRulesUsesFaker(t *testing.T) {
	ep := arrayEndpoint()
	srv := New(":0", []domain.Endpoint{ep}, nil, nil, nil, nil)

	resp := do(t, srv, "GET", "/xs")
	defer resp.Body.Close()
	var body []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 1 {
		t.Fatalf("len = %d, want 1 (minItems default)", len(body))
	}
}
