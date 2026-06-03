package domain

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestResponseSpec_HasBody(t *testing.T) {
	schema := openapi3.NewStringSchema().NewRef()

	tests := []struct {
		name string
		spec ResponseSpec
		want bool
	}{
		{"no content type", ResponseSpec{StatusCode: 204}, false},
		{"content type only", ResponseSpec{ContentType: "application/json"}, false},
		{"content type + schema", ResponseSpec{ContentType: "application/json", Schema: schema}, true},
		{"content type + example", ResponseSpec{ContentType: "application/json", Example: map[string]any{"x": 1}}, true},
		{"schema but no content type", ResponseSpec{Schema: schema}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.spec.HasBody(); got != tc.want {
				t.Fatalf("HasBody() = %v, want %v", got, tc.want)
			}
		})
	}
}
