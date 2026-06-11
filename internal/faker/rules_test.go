package faker

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func personSchema() *openapi3.SchemaRef {
	cpf := openapi3.NewStringSchema()
	inner := openapi3.NewObjectSchema()
	inner.WithPropertyRef("cpf", cpf.NewRef())

	s := openapi3.NewObjectSchema()
	s.WithProperty("cpf", openapi3.NewStringSchema())
	s.WithPropertyRef("pessoa", inner.NewRef())
	return s.NewRef()
}

func TestCycle_WrapsAround(t *testing.T) {
	c := newCycle([]any{1, 2, 3})
	got := []any{c.next(), c.next(), c.next(), c.next(), c.next()}
	want := []any{1, 2, 3, 1, 2}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("cycle[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestRules_ValuesByName(t *testing.T) {
	r := NewRules()
	r.SetValues("cpf", []any{nil, "111"})

	out1 := GenerateWith(personSchema(), r).(map[string]any)
	if out1["cpf"] != nil {
		t.Fatalf("1st cpf = %v, want nil", out1["cpf"])
	}
	out2 := GenerateWith(personSchema(), r).(map[string]any)
	if out2["cpf"] != "111" {
		t.Fatalf("2nd cpf = %v, want 111", out2["cpf"])
	}
	// name matcher also hits the nested pessoa.cpf
	if out2["pessoa"].(map[string]any)["cpf"] != "111" {
		t.Fatalf("nested cpf not overridden by name matcher: %v", out2["pessoa"])
	}
}

func TestRules_PathBeatsName(t *testing.T) {
	r := NewRules()
	r.SetValues("cpf", []any{"name-match"})
	r.SetValues("pessoa.cpf", []any{"path-match"})

	out := GenerateWith(personSchema(), r).(map[string]any)
	if out["cpf"] != "name-match" {
		t.Errorf("top cpf = %v, want name-match", out["cpf"])
	}
	if out["pessoa"].(map[string]any)["cpf"] != "path-match" {
		t.Errorf("pessoa.cpf = %v, want path-match (path wins)", out["pessoa"])
	}
}

func arraySchema() *openapi3.SchemaRef {
	item := openapi3.NewStringSchema()
	arr := openapi3.NewArraySchema()
	arr.Items = item.NewRef()
	arr.MinItems = 1
	return arr.NewRef()
}

func TestRules_RootCount(t *testing.T) {
	r := NewRules()
	r.SetCount("$", []any{3, nil, 1})

	if got := GenerateWith(arraySchema(), r).([]any); len(got) != 3 {
		t.Fatalf("1st len = %d, want 3", len(got))
	}
	if got := GenerateWith(arraySchema(), r); got != nil {
		t.Fatalf("2nd = %v, want nil (count null)", got)
	}
	if got := GenerateWith(arraySchema(), r).([]any); len(got) != 1 {
		t.Fatalf("3rd len = %d, want 1", len(got))
	}
	if got := GenerateWith(arraySchema(), r).([]any); len(got) != 3 {
		t.Fatalf("4th len = %d, want 3 (wraps)", len(got))
	}
}

func TestRules_CountOnNamedField(t *testing.T) {
	item := openapi3.NewStringSchema()
	tags := openapi3.NewArraySchema()
	tags.Items = item.NewRef()
	tags.MinItems = 1
	s := openapi3.NewObjectSchema()
	s.WithPropertyRef("tags", tags.NewRef())

	r := NewRules()
	r.SetCount("tags", []any{4})

	out := GenerateWith(s.NewRef(), r).(map[string]any)
	if got := out["tags"].([]any); len(got) != 4 {
		t.Fatalf("tags len = %d, want 4", len(got))
	}
}

func TestRules_ValueNullForListField(t *testing.T) {
	item := openapi3.NewStringSchema()
	tags := openapi3.NewArraySchema()
	tags.Items = item.NewRef()
	s := openapi3.NewObjectSchema()
	s.WithPropertyRef("tags", tags.NewRef())

	r := NewRules()
	r.SetValues("tags", []any{nil})

	out := GenerateWith(s.NewRef(), r).(map[string]any)
	if out["tags"] != nil {
		t.Fatalf("tags = %v, want nil (value override)", out["tags"])
	}
}

func TestGenerate_NilRulesUnchanged(t *testing.T) {
	// Generate (nil rules) keeps the documented schema-example behaviour.
	s := openapi3.NewStringSchema()
	s.Example = "fixed"
	if got := Generate(s.NewRef()); got != "fixed" {
		t.Fatalf("got %v, want fixed", got)
	}
}

func TestRules_Empty(t *testing.T) {
	if !(*Rules)(nil).Empty() {
		t.Error("nil Rules should be Empty")
	}
	if !NewRules().Empty() {
		t.Error("fresh Rules should be Empty")
	}
	r := NewRules()
	r.SetValues("x", []any{1})
	if r.Empty() {
		t.Error("Rules with a value should not be Empty")
	}
}

func TestChild(t *testing.T) {
	tests := []struct{ parent, field, want string }{
		{"$", "cpf", "cpf"},
		{"", "cpf", "cpf"},
		{"pessoa", "cpf", "pessoa.cpf"},
		{"pessoa.endereco", "cep", "pessoa.endereco.cep"},
	}
	for _, tc := range tests {
		if got := child(tc.parent, tc.field); got != tc.want {
			t.Errorf("child(%q,%q) = %q, want %q", tc.parent, tc.field, got, tc.want)
		}
	}
}
