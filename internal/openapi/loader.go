package openapi

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

// Load lê e valida um documento OpenAPI 3 (YAML ou JSON) do caminho informado,
// resolvendo as referências $ref. Retorna o documento pronto para descoberta.
func Load(path string) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading spec %q: %w", path, err)
	}

	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("validating spec %q: %w", path, err)
	}

	return doc, nil
}
