package faker

import (
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/getkin/kin-openapi/openapi3"
)

const maxDepth = 12

// Generate gera um valor para o schema informado. Resolve, nesta ordem:
// exemplo do schema, default, primeiro valor do enum e, por fim, dado falso
// conforme tipo/formato. Retorna nil para schema vazio ou tipo não suportado.
func Generate(ref *openapi3.SchemaRef) any {
	return generate(ref, 0)
}

func generate(ref *openapi3.SchemaRef, depth int) any {
	if ref == nil || ref.Value == nil || depth > maxDepth {
		return nil
	}
	schema := ref.Value

	if schema.Example != nil {
		return schema.Example
	}
	if schema.Default != nil {
		return schema.Default
	}
	if len(schema.Enum) > 0 {
		return schema.Enum[0]
	}

	switch {
	case schema.Type.Is(openapi3.TypeObject) || len(schema.Properties) > 0:
		return generateObject(schema, depth)
	case schema.Type.Is(openapi3.TypeArray):
		return generateArray(schema, depth)
	case schema.Type.Is(openapi3.TypeString):
		return generateString(schema)
	case schema.Type.Is(openapi3.TypeInteger):
		return generateInteger(schema)
	case schema.Type.Is(openapi3.TypeNumber):
		return generateNumber(schema)
	case schema.Type.Is(openapi3.TypeBoolean):
		return gofakeit.Bool()
	}

	if len(schema.AllOf) > 0 {
		return generate(schema.AllOf[0], depth+1)
	}
	if len(schema.OneOf) > 0 {
		return generate(schema.OneOf[0], depth+1)
	}
	if len(schema.AnyOf) > 0 {
		return generate(schema.AnyOf[0], depth+1)
	}

	return nil
}

func generateObject(schema *openapi3.Schema, depth int) map[string]any {
	out := make(map[string]any, len(schema.Properties))
	for name, prop := range schema.Properties {
		out[name] = generate(prop, depth+1)
	}
	return out
}

func generateArray(schema *openapi3.Schema, depth int) []any {
	if schema.Items == nil {
		return []any{}
	}
	n := 1
	if schema.MinItems > 0 {
		n = int(schema.MinItems)
	}
	if schema.MaxItems != nil && uint64(n) > *schema.MaxItems {
		n = int(*schema.MaxItems)
	}
	out := make([]any, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, generate(schema.Items, depth+1))
	}
	return out
}

func generateString(schema *openapi3.Schema) any {
	switch schema.Format {
	case "email":
		return gofakeit.Email()
	case "uuid":
		return gofakeit.UUID()
	case "uri", "url":
		return gofakeit.URL()
	case "hostname":
		return gofakeit.DomainName()
	case "ipv4":
		return gofakeit.IPv4Address()
	case "ipv6":
		return gofakeit.IPv6Address()
	case "date":
		return gofakeit.Date().Format("2006-01-02")
	case "date-time":
		return gofakeit.Date().Format(time.RFC3339)
	case "byte":
		return gofakeit.LetterN(16)
	case "password":
		return gofakeit.Password(true, true, true, false, false, 12)
	}

	if schema.MinLength > 0 || schema.MaxLength != nil {
		return gofakeit.LetterN(boundedLen(schema))
	}
	return gofakeit.Word()
}

func boundedLen(schema *openapi3.Schema) uint {
	n := uint(8)
	if schema.MinLength > 0 {
		n = uint(schema.MinLength)
	}
	if schema.MaxLength != nil && uint64(n) > *schema.MaxLength {
		n = uint(*schema.MaxLength)
	}
	if n == 0 {
		n = 1
	}
	return n
}

func generateInteger(schema *openapi3.Schema) int {
	min, max := 1, 1000
	if schema.Min != nil {
		min = int(*schema.Min)
	}
	if schema.Max != nil {
		max = int(*schema.Max)
	}
	if min > max {
		max = min
	}
	return gofakeit.Number(min, max)
}

func generateNumber(schema *openapi3.Schema) float64 {
	min, max := 0.0, 1000.0
	if schema.Min != nil {
		min = *schema.Min
	}
	if schema.Max != nil {
		max = *schema.Max
	}
	if min > max {
		max = min
	}
	return gofakeit.Float64Range(min, max)
}
