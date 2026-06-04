package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Claudio712005/mock-smith/internal/interceptor"
)

const defaultTimeoutDelay = 30 * time.Second

func buildChain(opts Options) (interceptor.Chain, error) {
	var chain interceptor.Chain

	for _, entry := range opts.Slow {
		path, value := splitTarget(entry)
		dur, err := time.ParseDuration(value)
		if err != nil || dur < 0 {
			return nil, fmt.Errorf("invalid --slow %q (want [path=]duration, e.g. /pets=2s)", entry)
		}
		chain = append(chain, interceptor.ScopedInterceptor{
			Path:  path,
			Inner: interceptor.LatencyInterceptor{Delay: dur},
		})
	}

	for _, entry := range opts.Fail {
		path, value := splitTarget(entry)
		status, err := strconv.Atoi(value)
		if err != nil || status < 100 || status > 599 {
			return nil, fmt.Errorf("invalid --fail %q (want [path=]status, e.g. /payments=503)", entry)
		}
		chain = append(chain, interceptor.ScopedInterceptor{
			Path:  path,
			Inner: interceptor.FailureInterceptor{Status: status, Rate: 1},
		})
	}

	for _, entry := range opts.Timeout {
		path, value := splitTarget(entry)
		rate, err := parseRate(value)
		if err != nil {
			return nil, fmt.Errorf("invalid --timeout %q (want [path=]rate, e.g. /auth=20%%)", entry)
		}
		chain = append(chain, interceptor.ScopedInterceptor{
			Path:  path,
			Inner: interceptor.TimeoutInterceptor{Delay: defaultTimeoutDelay, Rate: rate},
		})
	}

	for _, entry := range opts.Corrupt {
		path, value := splitTarget(entry)
		rate, err := parseRate(value)
		if err != nil {
			return nil, fmt.Errorf("invalid --corrupt %q (want [path=]rate, e.g. /users=10%%)", entry)
		}
		chain = append(chain, interceptor.ScopedInterceptor{
			Path:  path,
			Inner: interceptor.CorruptionInterceptor{Rate: rate},
		})
	}

	return chain, nil
}

func splitTarget(s string) (path, value string) {
	if i := strings.Index(s, "="); i >= 0 {
		return s[:i], s[i+1:]
	}
	return "", s
}

func parseRate(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		if err != nil {
			return 0, err
		}
		if n < 0 || n > 100 {
			return 0, fmt.Errorf("rate out of range")
		}
		return n / 100, nil
	}
	if s == "" {
		return 0, fmt.Errorf("empty rate")
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if n < 0 || n > 1 {
		return 0, fmt.Errorf("rate out of range")
	}
	return n, nil
}
