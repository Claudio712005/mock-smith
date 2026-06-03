package scenario

import "testing"

func TestAlways(t *testing.T) {
	want := Result{Kind: KindServerError, Status: 503}
	s := Always(want)
	got := s.Resolve(&RequestContext{Method: "GET", Path: "/x"})
	if got != want {
		t.Fatalf("Always.Resolve() = %+v, want %+v", got, want)
	}
}
