package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Claudio712005/mock-smith/internal/domain"
	"github.com/Claudio712005/mock-smith/internal/interceptor"
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

func registerAdmin(r chi.Router, endpoints []domain.Endpoint, overrides *interceptor.Overrides) {
	r.Get(adminPrefix+"/endpoints", listEndpointsHandler(endpoints))
	r.Get(adminPrefix+"/endpoint", getEndpointHandler(endpoints))
	r.Get(adminPrefix+"/runtime", listRuntimeHandler(overrides))
	r.Post(adminPrefix+"/runtime", setRuntimeHandler(endpoints, overrides))
	r.Delete(adminPrefix+"/runtime", deleteRuntimeHandler(overrides))
}

type runtimeRequest struct {
	Endpoint  string  `json:"endpoint"`
	Status    int     `json:"status"`
	LatencyMs int     `json:"latencyMs"`
	Rate      float64 `json:"rate"`
}

func listRuntimeHandler(overrides *interceptor.Overrides) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		snap := overrides.Snapshot()
		writeJSON(w, http.StatusOK, map[string]any{
			"count":     len(snap),
			"overrides": snap,
		})
	}
}

func setRuntimeHandler(endpoints []domain.Endpoint, overrides *interceptor.Overrides) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var body runtimeRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		if body.Endpoint == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "field 'endpoint' is required"})
			return
		}
		if !pathExists(endpoints, body.Endpoint) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no endpoint with path " + body.Endpoint})
			return
		}
		if body.Status != 0 && (body.Status < 100 || body.Status > 599) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status must be 0 or a valid HTTP status (100-599)"})
			return
		}
		if body.LatencyMs < 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "latencyMs must be >= 0"})
			return
		}
		if body.Rate < 0 || body.Rate > 1 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "rate must be between 0 and 1"})
			return
		}
		if body.Status == 0 && body.LatencyMs == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "set at least one of 'status' or 'latencyMs'"})
			return
		}

		ov := interceptor.Override{Status: body.Status, LatencyMs: body.LatencyMs, Rate: body.Rate}
		overrides.Set(body.Endpoint, ov)
		writeJSON(w, http.StatusOK, map[string]any{
			"endpoint": body.Endpoint,
			"override": ov,
		})
	}
}

func deleteRuntimeHandler(overrides *interceptor.Overrides) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Query().Get("endpoint")
		if path == "" {
			n := overrides.Clear()
			writeJSON(w, http.StatusOK, map[string]any{"cleared": n})
			return
		}
		if !overrides.Delete(path) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no override for " + path})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": path})
	}
}

func pathExists(endpoints []domain.Endpoint, path string) bool {
	for _, ep := range endpoints {
		if ep.Path == path {
			return true
		}
	}
	return false
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
