package scenario

import (
	"testing"
	"time"
)

func TestForProfile_Unknown(t *testing.T) {
	if _, err := ForProfile("nope"); err == nil {
		t.Fatal("ForProfile(nope) expected error, got nil")
	}
}

func TestForProfile_KnownNames(t *testing.T) {
	for _, name := range []string{"happy", "sad", "resilience", "chaos"} {
		if _, err := ForProfile(name); err != nil {
			t.Errorf("ForProfile(%q) error = %v", name, err)
		}
	}
}

func TestAvailable_Sorted(t *testing.T) {
	got := Available()
	want := []string{"chaos", "happy", "resilience", "sad"}
	if len(got) != len(want) {
		t.Fatalf("Available() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Available()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestHappy_AlwaysSuccess(t *testing.T) {
	p := profiles["happy"]
	for i := 0; i < 100; i++ {
		if got := p.Resolve(nil); got.Kind != KindSuccess {
			t.Fatalf("happy yielded %v", got.Kind)
		}
	}
}

// withRand clones a profile with a deterministic rand() for boundary testing.
func withRand(src *Profile, r float64) *Profile {
	return &Profile{
		name:    src.name,
		entries: src.entries,
		total:   src.total,
		rand:    func() float64 { return r },
	}
}

func TestProfile_WeightedBoundaries_Sad(t *testing.T) {
	src := profiles["sad"] // 0.70 success, 0.30 business error
	tests := []struct {
		r    float64
		kind Kind
	}{
		{0.0, KindSuccess},
		{0.69, KindSuccess},
		{0.70, KindBusinessError}, // exactly at the success boundary tips over
		{0.99, KindBusinessError},
	}
	for _, tc := range tests {
		got := withRand(src, tc.r).Resolve(nil)
		if got.Kind != tc.kind {
			t.Errorf("sad rand=%.2f → %v, want %v", tc.r, got.Kind, tc.kind)
		}
	}
}

func TestProfile_Resilience_Distribution(t *testing.T) {
	src := profiles["resilience"] // 0.90 success, 0.05 timeout, 0.05 503
	// success region
	if got := withRand(src, 0.5).Resolve(nil); got.Kind != KindSuccess {
		t.Errorf("rand=0.5 → %v, want success", got.Kind)
	}
	// timeout region [0.90, 0.95)
	got := withRand(src, 0.92).Resolve(nil)
	if got.Kind != KindTimeout {
		t.Fatalf("rand=0.92 → %v, want timeout", got.Kind)
	}
	if got.Delay != defaultTimeout {
		t.Errorf("timeout delay = %v, want %v", got.Delay, defaultTimeout)
	}
	// server-error region [0.95, 1.0)
	got = withRand(src, 0.97).Resolve(nil)
	if got.Kind != KindServerError || got.Status != 503 {
		t.Errorf("rand=0.97 → %+v, want 503 server error", got)
	}
}

func TestProfile_Chaos_AllKindsReachable(t *testing.T) {
	src := profiles["chaos"] // 0.80 ok, .05 malformed, .05 timeout, .05 disconnect, .05 500
	cases := []struct {
		r    float64
		kind Kind
	}{
		{0.10, KindSuccess},
		{0.82, KindMalformed},
		{0.87, KindTimeout},
		{0.92, KindDisconnect},
		{0.97, KindServerError},
	}
	for _, tc := range cases {
		if got := withRand(src, tc.r).Resolve(nil); got.Kind != tc.kind {
			t.Errorf("chaos rand=%.2f → %v, want %v", tc.r, got.Kind, tc.kind)
		}
	}
}

func TestProfile_RandAtUpperBoundFallsToLast(t *testing.T) {
	src := profiles["sad"]
	// rand()*total == total (r=1.0) must not panic and returns the last entry.
	if got := withRand(src, 1.0).Resolve(nil); got.Kind != KindBusinessError {
		t.Fatalf("rand=1.0 → %v, want last entry (business error)", got.Kind)
	}
}

func TestProfile_EmptyEntries(t *testing.T) {
	p := &Profile{name: "x", rand: func() float64 { return 0.5 }}
	if got := p.Resolve(nil); got.Kind != KindSuccess {
		t.Fatalf("empty profile → %v, want success fallback", got.Kind)
	}
}

func TestDefaultTimeoutValue(t *testing.T) {
	if defaultTimeout != 30*time.Second {
		t.Fatalf("defaultTimeout = %v, want 30s", defaultTimeout)
	}
}
