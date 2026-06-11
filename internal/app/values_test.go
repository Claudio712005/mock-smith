package app

import (
	"testing"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/Claudio712005/mock-smith/internal/faker"
	"github.com/getkin/kin-openapi/openapi3"
)

func objWithX() *openapi3.SchemaRef {
	s := openapi3.NewObjectSchema()
	s.WithProperty("x", openapi3.NewStringSchema())
	return s.NewRef()
}

func TestBuildValueRules_GlobalAndEndpointMerge(t *testing.T) {
	endpoints := []domain.Endpoint{
		{Method: "GET", Path: "/a"},
		{Method: "POST", Path: "/a"},
		{Method: "GET", Path: "/b"},
	}
	global := ValueRuleSet{Values: map[string][]any{"x": {"G"}}}
	byEndpoint := map[string]ValueRuleSet{
		"GET /a": {Values: map[string][]any{"x": {"A"}}}, // method-specific
		"/b":     {Values: map[string][]any{"x": {"B"}}}, // path-only, any method
	}

	rules := buildValueRules(endpoints, global, byEndpoint)
	for _, k := range []string{"GET /a", "POST /a", "GET /b"} {
		if rules[k] == nil {
			t.Fatalf("missing rules for %q", k)
		}
	}

	gen := func(key string) string {
		return faker.GenerateWith(objWithX(), rules[key]).(map[string]any)["x"].(string)
	}
	if got := gen("GET /a"); got != "A" {
		t.Errorf("GET /a x = %q, want A (endpoint wins)", got)
	}
	if got := gen("POST /a"); got != "G" {
		t.Errorf("POST /a x = %q, want G (only global)", got)
	}
	if got := gen("GET /b"); got != "B" {
		t.Errorf("GET /b x = %q, want B (path-only match)", got)
	}
}

func TestBuildValueRules_NoneIsNil(t *testing.T) {
	endpoints := []domain.Endpoint{{Method: "GET", Path: "/a"}}
	if got := buildValueRules(endpoints, ValueRuleSet{}, nil); got != nil {
		t.Fatalf("expected nil rules map, got %v", got)
	}
}

func TestBuildValueRules_EndpointOnlyAffectsMatch(t *testing.T) {
	endpoints := []domain.Endpoint{
		{Method: "GET", Path: "/a"},
		{Method: "GET", Path: "/b"},
	}
	byEndpoint := map[string]ValueRuleSet{
		"GET /a": {Counts: map[string][]any{"$": {3}}},
	}
	rules := buildValueRules(endpoints, ValueRuleSet{}, byEndpoint)
	if rules["GET /a"] == nil {
		t.Fatal("GET /a should have rules")
	}
	if rules["GET /b"] != nil {
		t.Fatalf("GET /b should have no rules, got %v", rules["GET /b"])
	}
}

func TestRunMany_ValidationFailsFast(t *testing.T) {
	// Invalid profile is rejected by New before any listener binds.
	err := RunMany([]Options{{SpecPath: "irrelevant", Profile: "bogus"}})
	if err == nil {
		t.Fatal("RunMany with bad profile expected error, got nil")
	}
}

func TestRunMany_Empty(t *testing.T) {
	if err := RunMany(nil); err == nil {
		t.Fatal("RunMany(nil) expected error, got nil")
	}
}
