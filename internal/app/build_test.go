package app

import (
	"testing"
	"time"

	"github.com/Claudio712005/mock-smith/internal/interceptor"
)

func TestBuildChain_Empty(t *testing.T) {
	chain, err := buildChain(Options{})
	if err != nil {
		t.Fatalf("buildChain error = %v", err)
	}
	if len(chain) != 0 {
		t.Fatalf("chain len = %d, want 0", len(chain))
	}
}

func TestBuildChain_AllFlags(t *testing.T) {
	chain, err := buildChain(Options{
		Slow:    []string{"/pets=2s", "100ms"},
		Fail:    []string{"/payments=503"},
		Timeout: []string{"/auth=20%"},
		Corrupt: []string{"/users=10%"},
	})
	if err != nil {
		t.Fatalf("buildChain error = %v", err)
	}
	if len(chain) != 5 {
		t.Fatalf("chain len = %d, want 5", len(chain))
	}
}

func TestBuildChain_ScopedAndGlobal(t *testing.T) {
	chain, err := buildChain(Options{Slow: []string{"/pets=2s", "1s"}})
	if err != nil {
		t.Fatalf("buildChain error = %v", err)
	}
	scoped, ok := chain[0].(interceptor.ScopedInterceptor)
	if !ok || scoped.Path != "/pets" {
		t.Fatalf("entry 0 = %+v, want scoped /pets", chain[0])
	}
	global := chain[1].(interceptor.ScopedInterceptor)
	if global.Path != "" {
		t.Fatalf("entry 1 path = %q, want global (empty)", global.Path)
	}
	lat, ok := scoped.Inner.(interceptor.LatencyInterceptor)
	if !ok || lat.Delay != 2*time.Second {
		t.Fatalf("inner = %+v, want latency 2s", scoped.Inner)
	}
}

func TestBuildChain_FailWrapsFailureRate1(t *testing.T) {
	chain, err := buildChain(Options{Fail: []string{"/x=500"}})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	fi := chain[0].(interceptor.ScopedInterceptor).Inner.(interceptor.FailureInterceptor)
	if fi.Status != 500 || fi.Rate != 1 {
		t.Fatalf("failure = %+v, want status 500 rate 1", fi)
	}
}

func TestBuildChain_Errors(t *testing.T) {
	tests := []struct {
		name string
		opts Options
	}{
		{"slow bad dur", Options{Slow: []string{"/x=abc"}}},
		{"slow negative", Options{Slow: []string{"/x=-2s"}}},
		{"fail non-numeric", Options{Fail: []string{"/x=abc"}}},
		{"fail out of range", Options{Fail: []string{"/x=42"}}},
		{"timeout bad", Options{Timeout: []string{"/x=abc"}}},
		{"timeout over 100%", Options{Timeout: []string{"/x=120%"}}},
		{"corrupt over 1", Options{Corrupt: []string{"/x=2"}}},
		{"corrupt empty", Options{Corrupt: []string{"/x="}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := buildChain(tc.opts); err == nil {
				t.Fatalf("buildChain(%+v) expected error, got nil", tc.opts)
			}
		})
	}
}

func TestSplitTarget(t *testing.T) {
	tests := []struct {
		in        string
		path, val string
	}{
		{"/pets=2s", "/pets", "2s"},
		{"2s", "", "2s"},
		{"/a/b=503", "/a/b", "503"},
		{"=5", "", "5"},
	}
	for _, tc := range tests {
		p, v := splitTarget(tc.in)
		if p != tc.path || v != tc.val {
			t.Errorf("splitTarget(%q) = (%q,%q), want (%q,%q)", tc.in, p, v, tc.path, tc.val)
		}
	}
}

func TestParseRate(t *testing.T) {
	ok := []struct {
		in   string
		want float64
	}{
		{"20%", 0.2},
		{"0%", 0},
		{"100%", 1},
		{"0.5", 0.5},
		{"1", 1},
	}
	for _, tc := range ok {
		got, err := parseRate(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("parseRate(%q) = (%v,%v), want %v", tc.in, got, err, tc.want)
		}
	}
	for _, bad := range []string{"", "abc", "120%", "-5%", "2", "-0.1"} {
		if _, err := parseRate(bad); err == nil {
			t.Errorf("parseRate(%q) expected error", bad)
		}
	}
}
