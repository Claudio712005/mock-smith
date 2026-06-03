package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/go-chi/chi/v5"
)

const adminPrefix = "/__mocksmith"

type responseInfo struct {
	Status      int    `json:"status"`
	ContentType string `json:"contentType,omitempty"`
	HasBody     bool   `json:"hasBody"`
	HasExample  bool   `json:"hasExample"`
}

type endpointSummary struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	OperationID string `json:"operationId,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Statuses    []int  `json:"statuses"`
}

type endpointDetail struct {
	Method      string         `json:"method"`
	Path        string         `json:"path"`
	OperationID string         `json:"operationId,omitempty"`
	Summary     string         `json:"summary,omitempty"`
	Success     *responseInfo  `json:"success,omitempty"`
	Errors      []responseInfo `json:"errors,omitempty"`
}

func registerAdmin(r chi.Router, endpoints []domain.Endpoint) {
	r.Get(adminPrefix+"/endpoints", listEndpointsHandler(endpoints))
	r.Get(adminPrefix+"/endpoint", getEndpointHandler(endpoints))
}

func listEndpointsHandler(endpoints []domain.Endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		out := make([]endpointSummary, 0, len(endpoints))
		for _, ep := range endpoints {
			out = append(out, endpointSummary{
				Method:      ep.Method,
				Path:        ep.Path,
				OperationID: ep.OperationID,
				Summary:     ep.Summary,
				Statuses:    statusesOf(ep),
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"count":     len(out),
			"endpoints": out,
		})
	}
}

func getEndpointHandler(endpoints []domain.Endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		method := strings.ToUpper(req.URL.Query().Get("method"))
		path := req.URL.Query().Get("path")
		if method == "" || path == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "query params 'method' and 'path' are required",
			})
			return
		}

		for _, ep := range endpoints {
			if ep.Method == method && ep.Path == path {
				writeJSON(w, http.StatusOK, detailOf(ep))
				return
			}
		}

		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "endpoint not found: " + method + " " + path,
		})
	}
}

func detailOf(ep domain.Endpoint) endpointDetail {
	d := endpointDetail{
		Method:      ep.Method,
		Path:        ep.Path,
		OperationID: ep.OperationID,
		Summary:     ep.Summary,
	}
	if ep.SuccessResponse != nil {
		info := responseInfoOf(*ep.SuccessResponse)
		d.Success = &info
	}
	for _, e := range ep.ErrorResponses {
		d.Errors = append(d.Errors, responseInfoOf(e))
	}
	return d
}

func responseInfoOf(r domain.ResponseSpec) responseInfo {
	return responseInfo{
		Status:      r.StatusCode,
		ContentType: r.ContentType,
		HasBody:     r.HasBody(),
		HasExample:  r.Example != nil,
	}
}

func statusesOf(ep domain.Endpoint) []int {
	var out []int
	if ep.SuccessResponse != nil {
		out = append(out, ep.SuccessResponse.StatusCode)
	}
	for _, e := range ep.ErrorResponses {
		out = append(out, e.StatusCode)
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(body)
}
