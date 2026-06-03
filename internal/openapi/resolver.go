package openapi

import (
	"sort"
	"strconv"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/getkin/kin-openapi/openapi3"
)

const jsonMime = "application/json"

// Discover percorre um documento OpenAPI resolvido e produz a lista
// normalizada de endpoints servidos pelo MockSmith, em ordem estável
// (path, depois método).
func Discover(doc *openapi3.T) []domain.Endpoint {
	if doc == nil || doc.Paths == nil {
		return nil
	}

	var endpoints []domain.Endpoint
	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			endpoints = append(endpoints, buildEndpoint(method, path, op))
		}
	}

	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Path != endpoints[j].Path {
			return endpoints[i].Path < endpoints[j].Path
		}
		return endpoints[i].Method < endpoints[j].Method
	})

	return endpoints
}

func buildEndpoint(method, path string, op *openapi3.Operation) domain.Endpoint {
	ep := domain.Endpoint{
		Method:      method,
		Path:        path,
		OperationID: op.OperationID,
		Summary:     op.Summary,
	}

	if op.Responses == nil {
		return ep
	}

	type coded struct {
		code int
		spec domain.ResponseSpec
	}
	var successes, errors []coded

	for status, ref := range op.Responses.Map() {
		if ref == nil || ref.Value == nil {
			continue
		}
		code, ok := parseStatus(status)
		if !ok {
			continue
		}
		spec := buildResponseSpec(code, ref.Value)
		switch {
		case code >= 200 && code < 300:
			successes = append(successes, coded{code, spec})
		case code >= 400:
			errors = append(errors, coded{code, spec})
		}
	}

	if len(successes) > 0 {
		sort.Slice(successes, func(i, j int) bool { return successes[i].code < successes[j].code })
		ep.SuccessResponse = &successes[0].spec
	}

	sort.Slice(errors, func(i, j int) bool { return errors[i].code < errors[j].code })
	for _, e := range errors {
		ep.ErrorResponses = append(ep.ErrorResponses, e.spec)
	}

	return ep
}

func buildResponseSpec(code int, resp *openapi3.Response) domain.ResponseSpec {
	spec := domain.ResponseSpec{StatusCode: code}

	media := resp.Content.Get(jsonMime)
	if media == nil {
		return spec
	}

	spec.ContentType = jsonMime
	spec.Schema = media.Schema
	spec.Example = pickExample(media)
	return spec
}

func pickExample(media *openapi3.MediaType) any {
	if media.Example != nil {
		return media.Example
	}
	for _, ex := range media.Examples {
		if ex != nil && ex.Value != nil && ex.Value.Value != nil {
			return ex.Value.Value
		}
	}
	return nil
}

func parseStatus(s string) (int, bool) {
	code, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return code, true
}
