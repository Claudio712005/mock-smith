package domain

import "github.com/getkin/kin-openapi/openapi3"

// ResponseSpec descreve, de forma normalizada, uma resposta documentada de um
// endpoint, guardando o schema resolvido para gerar o corpo sob demanda.
type ResponseSpec struct {
	StatusCode  int
	ContentType string
	Schema      *openapi3.SchemaRef
	Example     any
}

// HasBody indica se a resposta deve carregar um corpo.
func (r ResponseSpec) HasBody() bool {
	return r.ContentType != "" && (r.Schema != nil || r.Example != nil)
}
