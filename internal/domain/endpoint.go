package domain

// Endpoint representa, em memória, uma operação OpenAPI: um método HTTP ligado
// a um path, com suas respostas de sucesso e de erro documentadas.
type Endpoint struct {
	Method          string
	Path            string
	OperationID     string
	Summary         string
	SuccessResponse *ResponseSpec
	ErrorResponses  []ResponseSpec
}
