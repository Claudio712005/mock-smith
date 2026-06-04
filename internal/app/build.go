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
		method, path, value := splitTarget(entry)
		dur, err := time.ParseDuration(value)
		if err != nil || dur < 0 {
			return nil, fmt.Errorf("invalid --slow %q (want [[METHOD ]path=]duration, e.g. /pets=2s or \"POST /pets=2s\")", entry)
		}
		chain = append(chain, interceptor.ScopedInterceptor{
			Method: method, Path: path,
			Inner: interceptor.LatencyInterceptor{Delay: dur},
		})
	}

	for _, entry := range opts.Fail {
		method, path, value := splitTarget(entry)
		status, err := strconv.Atoi(value)
		if err != nil || status < 100 || status > 599 {
			return nil, fmt.Errorf("invalid --fail %q (want [[METHOD ]path=]status, e.g. /payments=503 or \"POST /payments=503\")", entry)
		}
		chain = append(chain, interceptor.ScopedInterceptor{
			Method: method, Path: path,
			Inner: interceptor.FailureInterceptor{Status: status, Rate: 1},
		})
	}

	for _, entry := range opts.Timeout {
		method, path, value := splitTarget(entry)
		rate, err := parseRate(value)
		if err != nil {
			return nil, fmt.Errorf("invalid --timeout %q (want [[METHOD ]path=]rate, e.g. /auth=20%%)", entry)
		}
		chain = append(chain, interceptor.ScopedInterceptor{
			Method: method, Path: path,
			Inner: interceptor.TimeoutInterceptor{Delay: defaultTimeoutDelay, Rate: rate},
		})
	}

	for _, entry := range opts.Corrupt {
		method, path, value := splitTarget(entry)
		rate, err := parseRate(value)
		if err != nil {
			return nil, fmt.Errorf("invalid --corrupt %q (want [[METHOD ]path=]rate, e.g. /users=10%%)", entry)
		}
		chain = append(chain, interceptor.ScopedInterceptor{
			Method: method, Path: path,
			Inner: interceptor.CorruptionInterceptor{Rate: rate},
		})
	}

	for _, entry := range opts.Sequence {
		method, path, value := splitTarget(entry)
		statuses, err := parseStatuses(value)
		if err != nil {
			return nil, fmt.Errorf("invalid --sequence %q (want [[METHOD ]path=]s1,s2,..., e.g. /jobs=202,202,200)", entry)
		}
		chain = append(chain, interceptor.ScopedInterceptor{
			Method: method, Path: path,
			Inner: &interceptor.SequenceInterceptor{Statuses: statuses},
		})
	}

	return chain, nil
}

func parseStatuses(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		status, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil || status < 100 || status > 599 {
			return nil, fmt.Errorf("invalid status %q", p)
		}
		out = append(out, status)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty sequence")
	}
	return out, nil
}

// splitTarget separa "[[METHOD ]path=]value" em método, path e valor. Sem "=",
// tudo é valor (alvo global). O método/path saem de interceptor.SplitTarget.
func splitTarget(s string) (method, path, value string) {
	target := ""
	if i := strings.Index(s, "="); i >= 0 {
		target, value = s[:i], s[i+1:]
	} else {
		value = s
	}
	method, path = interceptor.SplitTarget(target)
	return method, path, value
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
