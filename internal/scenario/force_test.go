package scenario

import (
	"testing"

	"github.com/Claudio712005/mock-smith/internal/domain"
)

func epWith(success int, errs ...int) domain.Endpoint {
	ep := domain.Endpoint{Method: "GET", Path: "/x"}
	if success != 0 {
		ep.SuccessResponse = &domain.ResponseSpec{StatusCode: success}
	}
	for _, c := range errs {
		ep.ErrorResponses = append(ep.ErrorResponses, domain.ResponseSpec{StatusCode: c})
	}
	return ep
}

func ctxFor(ep domain.Endpoint) *RequestContext {
	return &RequestContext{Method: ep.Method, Path: ep.Path, Endpoint: ep}
}

func TestForceStatus_DocumentedError_Forced(t *testing.T) {
	s := ForceStatus(500, Always(Result{Kind: KindSuccess}))
	got := s.Resolve(ctxFor(epWith(200, 500, 503)))
	if got.Kind != KindServerError || got.Status != 500 {
		t.Fatalf("got %+v, want forced 500 server error", got)
	}
}

func TestForceStatus_NotDocumented_FallsToBase(t *testing.T) {
	base := Always(Result{Kind: KindBusinessError})
	s := ForceStatus(500, base)
	got := s.Resolve(ctxFor(epWith(200, 404)))
	if got.Kind != KindBusinessError {
		t.Fatalf("got %+v, want base business error (500 not documented)", got)
	}
}

func TestForceStatus_MatchesSuccessStatus(t *testing.T) {
	s := ForceStatus(201, Always(Result{Kind: KindBusinessError}))
	got := s.Resolve(ctxFor(epWith(201, 400)))
	if got.Kind != KindSuccess {
		t.Fatalf("got %+v, want success (forced status == documented success)", got)
	}
}

func TestForceStatus_4xxAlsoForceable(t *testing.T) {
	s := ForceStatus(404, Always(Result{Kind: KindSuccess}))
	got := s.Resolve(ctxFor(epWith(200, 404)))
	if got.Kind != KindServerError || got.Status != 404 {
		t.Fatalf("got %+v, want forced 404", got)
	}
}

func TestForceStatus_NoSuccessResponse(t *testing.T) {
	s := ForceStatus(503, Always(Result{Kind: KindSuccess}))
	got := s.Resolve(ctxFor(epWith(0, 503)))
	if got.Kind != KindServerError || got.Status != 503 {
		t.Fatalf("got %+v, want forced 503", got)
	}
}
