package faker

import (
	"net"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

func strSchema(format string) *openapi3.SchemaRef {
	s := openapi3.NewStringSchema()
	s.Format = format
	return s.NewRef()
}

func TestGenerate_NilAndEmpty(t *testing.T) {
	if got := Generate(nil); got != nil {
		t.Fatalf("Generate(nil) = %v, want nil", got)
	}
	if got := Generate(&openapi3.SchemaRef{}); got != nil {
		t.Fatalf("Generate(empty ref) = %v, want nil", got)
	}
	if got := Generate(openapi3.NewSchema().NewRef()); got != nil {
		t.Fatalf("Generate(no type) = %v, want nil", got)
	}
}

func TestGenerate_ExampleWins(t *testing.T) {
	s := openapi3.NewStringSchema()
	s.Example = "fixed-example"
	s.Default = "ignored-default"
	s.Enum = []any{"ignored-enum"}
	if got := Generate(s.NewRef()); got != "fixed-example" {
		t.Fatalf("example not prioritized: got %v", got)
	}
}

func TestGenerate_DefaultBeforeEnum(t *testing.T) {
	s := openapi3.NewStringSchema()
	s.Default = "the-default"
	s.Enum = []any{"the-enum"}
	if got := Generate(s.NewRef()); got != "the-default" {
		t.Fatalf("default not prioritized over enum: got %v", got)
	}
}

func TestGenerate_EnumFirstValue(t *testing.T) {
	s := openapi3.NewStringSchema()
	s.Enum = []any{"first", "second"}
	if got := Generate(s.NewRef()); got != "first" {
		t.Fatalf("enum first value not used: got %v", got)
	}
}

func TestGenerate_StringFormats(t *testing.T) {
	tests := []struct {
		format string
		check  func(t *testing.T, v string)
	}{
		{"email", func(t *testing.T, v string) {
			if !strings.Contains(v, "@") {
				t.Errorf("email %q missing @", v)
			}
		}},
		{"uuid", func(t *testing.T, v string) {
			if len(v) != 36 || strings.Count(v, "-") != 4 {
				t.Errorf("uuid %q malformed", v)
			}
		}},
		{"uri", func(t *testing.T, v string) {
			if _, err := url.ParseRequestURI(v); err != nil {
				t.Errorf("uri %q invalid: %v", v, err)
			}
		}},
		{"url", func(t *testing.T, v string) {
			if _, err := url.ParseRequestURI(v); err != nil {
				t.Errorf("url %q invalid: %v", v, err)
			}
		}},
		{"hostname", func(t *testing.T, v string) {
			if !strings.Contains(v, ".") {
				t.Errorf("hostname %q missing dot", v)
			}
		}},
		{"ipv4", func(t *testing.T, v string) {
			ip := net.ParseIP(v)
			if ip == nil || ip.To4() == nil {
				t.Errorf("ipv4 %q invalid", v)
			}
		}},
		{"ipv6", func(t *testing.T, v string) {
			if net.ParseIP(v) == nil {
				t.Errorf("ipv6 %q invalid", v)
			}
		}},
		{"date", func(t *testing.T, v string) {
			if _, err := time.Parse("2006-01-02", v); err != nil {
				t.Errorf("date %q invalid: %v", v, err)
			}
		}},
		{"date-time", func(t *testing.T, v string) {
			if _, err := time.Parse(time.RFC3339, v); err != nil {
				t.Errorf("date-time %q invalid: %v", v, err)
			}
		}},
		{"byte", func(t *testing.T, v string) {
			if len(v) != 16 {
				t.Errorf("byte %q len = %d, want 16", v, len(v))
			}
		}},
		{"password", func(t *testing.T, v string) {
			if len(v) != 12 {
				t.Errorf("password %q len = %d, want 12", v, len(v))
			}
		}},
	}

	for _, tc := range tests {
		t.Run(tc.format, func(t *testing.T) {
			got := Generate(strSchema(tc.format))
			s, ok := got.(string)
			if !ok {
				t.Fatalf("format %s: got %T, want string", tc.format, got)
			}
			tc.check(t, s)
		})
	}
}

func TestGenerate_StringPlain(t *testing.T) {
	got := Generate(openapi3.NewStringSchema().NewRef())
	if _, ok := got.(string); !ok {
		t.Fatalf("plain string: got %T, want string", got)
	}
}

func TestGenerate_StringBoundedLength(t *testing.T) {
	s := openapi3.NewStringSchema()
	s.MinLength = 5
	max := uint64(5)
	s.MaxLength = &max
	got := Generate(s.NewRef()).(string)
	if len(got) != 5 {
		t.Fatalf("bounded len = %d, want 5", len(got))
	}
}

func TestGenerate_StringMaxBelowDefaultMin(t *testing.T) {
	s := openapi3.NewStringSchema()
	max := uint64(3)
	s.MaxLength = &max
	got := Generate(s.NewRef()).(string)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
}

func TestGenerate_Integer(t *testing.T) {
	s := openapi3.NewIntegerSchema()
	min, max := 10.0, 20.0
	s.Min = &min
	s.Max = &max
	for i := 0; i < 50; i++ {
		v, ok := Generate(s.NewRef()).(int)
		if !ok {
			t.Fatalf("integer: got %T, want int", Generate(s.NewRef()))
		}
		if v < 10 || v > 20 {
			t.Fatalf("integer %d out of [10,20]", v)
		}
	}
}

func TestGenerate_IntegerMinGreaterThanMax(t *testing.T) {
	s := openapi3.NewIntegerSchema()
	min, max := 50.0, 10.0
	s.Min = &min
	s.Max = &max
	v := Generate(s.NewRef()).(int)
	if v != 50 {
		t.Fatalf("min>max collapse: got %d, want 50", v)
	}
}

func TestGenerate_Number(t *testing.T) {
	s := openapi3.NewFloat64Schema()
	min, max := 1.5, 2.5
	s.Min = &min
	s.Max = &max
	for i := 0; i < 50; i++ {
		v, ok := Generate(s.NewRef()).(float64)
		if !ok {
			t.Fatalf("number: got %T, want float64", Generate(s.NewRef()))
		}
		if v < 1.5 || v > 2.5 {
			t.Fatalf("number %v out of [1.5,2.5]", v)
		}
	}
}

func TestGenerate_NumberMinGreaterThanMax(t *testing.T) {
	s := openapi3.NewFloat64Schema()
	min, max := 9.0, 1.0
	s.Min = &min
	s.Max = &max
	v := Generate(s.NewRef()).(float64)
	if v != 9.0 {
		t.Fatalf("min>max collapse: got %v, want 9", v)
	}
}

func TestGenerate_Boolean(t *testing.T) {
	got := Generate(openapi3.NewBoolSchema().NewRef())
	if _, ok := got.(bool); !ok {
		t.Fatalf("boolean: got %T, want bool", got)
	}
}

func TestGenerate_Object(t *testing.T) {
	s := openapi3.NewObjectSchema()
	s.WithProperty("name", openapi3.NewStringSchema())
	s.WithProperty("age", openapi3.NewIntegerSchema())

	got, ok := Generate(s.NewRef()).(map[string]any)
	if !ok {
		t.Fatalf("object: got %T, want map", Generate(s.NewRef()))
	}
	if _, ok := got["name"].(string); !ok {
		t.Errorf("object.name not string: %T", got["name"])
	}
	if _, ok := got["age"].(int); !ok {
		t.Errorf("object.age not int: %T", got["age"])
	}
}

func TestGenerate_ObjectByPropertiesNoType(t *testing.T) {
	s := openapi3.NewSchema()
	s.WithProperty("x", openapi3.NewStringSchema())
	got, ok := Generate(s.NewRef()).(map[string]any)
	if !ok {
		t.Fatalf("got %T, want map", Generate(s.NewRef()))
	}
	if _, ok := got["x"]; !ok {
		t.Errorf("missing property x")
	}
}

func TestGenerate_Array(t *testing.T) {
	s := openapi3.NewArraySchema()
	s.Items = openapi3.NewStringSchema().NewRef()
	s.MinItems = 3
	got, ok := Generate(s.NewRef()).([]any)
	if !ok {
		t.Fatalf("array: got %T, want slice", Generate(s.NewRef()))
	}
	if len(got) != 3 {
		t.Fatalf("array len = %d, want 3 (minItems)", len(got))
	}
}

func TestGenerate_ArrayMaxItemsCaps(t *testing.T) {
	s := openapi3.NewArraySchema()
	s.Items = openapi3.NewStringSchema().NewRef()
	s.MinItems = 5
	max := uint64(2)
	s.MaxItems = &max
	got := Generate(s.NewRef()).([]any)
	if len(got) != 2 {
		t.Fatalf("array len = %d, want 2 (maxItems cap)", len(got))
	}
}

func TestGenerate_ArrayNoItems(t *testing.T) {
	s := openapi3.NewArraySchema()
	got := Generate(s.NewRef()).([]any)
	if len(got) != 0 {
		t.Fatalf("array without items len = %d, want 0", len(got))
	}
}

func TestGenerate_AllOf(t *testing.T) {
	inner := openapi3.NewStringSchema()
	inner.Example = "from-allof"
	s := openapi3.NewSchema()
	s.AllOf = openapi3.SchemaRefs{inner.NewRef()}
	if got := Generate(s.NewRef()); got != "from-allof" {
		t.Fatalf("allOf: got %v", got)
	}
}

func TestGenerate_OneOf(t *testing.T) {
	inner := openapi3.NewStringSchema()
	inner.Example = "from-oneof"
	s := openapi3.NewSchema()
	s.OneOf = openapi3.SchemaRefs{inner.NewRef()}
	if got := Generate(s.NewRef()); got != "from-oneof" {
		t.Fatalf("oneOf: got %v", got)
	}
}

func TestGenerate_AnyOf(t *testing.T) {
	inner := openapi3.NewStringSchema()
	inner.Example = "from-anyof"
	s := openapi3.NewSchema()
	s.AnyOf = openapi3.SchemaRefs{inner.NewRef()}
	if got := Generate(s.NewRef()); got != "from-anyof" {
		t.Fatalf("anyOf: got %v", got)
	}
}

func TestGenerate_DepthGuard(t *testing.T) {
	s := openapi3.NewObjectSchema()
	ref := s.NewRef()
	s.WithPropertyRef("self", ref)

	done := make(chan any, 1)
	go func() { done <- Generate(ref) }()

	select {
	case got := <-done:
		if _, ok := got.(map[string]any); !ok {
			t.Fatalf("recursive object: got %T, want map", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Generate did not terminate on recursive schema")
	}
}
