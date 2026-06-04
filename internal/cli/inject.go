package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Claudio712005/mock-smith/internal/interceptor"
	"github.com/spf13/cobra"
)

func newInjectCmd() *cobra.Command {
	var (
		addr      string
		latencyMs int
		list      bool
		clear     bool
		remove    string
	)

	cmd := &cobra.Command{
		Use:   "inject [\"[METHOD ]path:status(rate%)\"]",
		Short: "Change a running server's behavior via its Admin API",
		Example: "  mocksmith inject \"/payments:503(20%)\"\n" +
			"  mocksmith inject \"POST /payments:503\"      # only the POST verb\n" +
			"  mocksmith inject \"GET /payments\" --latency-ms 2000\n" +
			"  mocksmith inject --list\n" +
			"  mocksmith inject --remove \"POST /payments\"\n" +
			"  mocksmith inject --clear",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			base := "http://" + strings.TrimPrefix(addr, "http://") + adminRuntimePath

			switch {
			case list:
				return injectList(base)
			case clear:
				return injectDelete(base, "", "")
			case remove != "":
				method, path := interceptor.SplitTarget(remove)
				return injectDelete(base, method, path)
			}

			if len(args) == 0 {
				return fmt.Errorf("nothing to do: pass \"[METHOD ]path:status(rate%%)\" or use --list/--remove/--clear")
			}

			method, path, status, rate, err := parseInject(args[0])
			if err != nil {
				return err
			}
			if status == 0 && latencyMs == 0 {
				return fmt.Errorf("set a status (e.g. /payments:503) or --latency-ms")
			}
			return injectSet(base, method, path, status, latencyMs, rate)
		},
	}

	cmd.Flags().StringVar(&addr, "addr", ":8080", "address of the running mock server")
	cmd.Flags().IntVar(&latencyMs, "latency-ms", 0, "latency in milliseconds to inject")
	cmd.Flags().BoolVar(&list, "list", false, "list active runtime overrides")
	cmd.Flags().BoolVar(&clear, "clear", false, "clear all runtime overrides")
	cmd.Flags().StringVar(&remove, "remove", "", "remove an override, [METHOD ]path, e.g. \"POST /payments\"")
	return cmd
}

const adminRuntimePath = "/__mocksmith/runtime"

func parseInject(s string) (method, path string, status int, rate float64, err error) {
	if i := strings.Index(s, "("); i >= 0 {
		if !strings.HasSuffix(s, "%)") {
			return "", "", 0, 0, fmt.Errorf("invalid rate in %q (want \"(NN%%)\")", s)
		}
		pct := s[i+1 : len(s)-2]
		n, perr := strconv.ParseFloat(pct, 64)
		if perr != nil || n < 0 || n > 100 {
			return "", "", 0, 0, fmt.Errorf("invalid rate %q (want 0-100)", pct)
		}
		rate = n / 100
		s = s[:i]
	}

	target := s
	if i := strings.LastIndex(s, ":"); i >= 0 {
		target = s[:i]
		status, err = strconv.Atoi(s[i+1:])
		if err != nil || status < 100 || status > 599 {
			return "", "", 0, 0, fmt.Errorf("invalid status in %q (want [METHOD ]path:status, 100-599)", s)
		}
	}

	method, path = interceptor.SplitTarget(target)
	if path == "" {
		return "", "", 0, 0, fmt.Errorf("missing endpoint path")
	}
	return method, path, status, rate, nil
}

func injectSet(base, method, path string, status, latencyMs int, rate float64) error {
	body, _ := json.Marshal(map[string]any{
		"endpoint":  path,
		"method":    method,
		"status":    status,
		"latencyMs": latencyMs,
		"rate":      rate,
	})
	resp, err := http.Post(base, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("calling admin API: %w", err)
	}
	return printResponse(resp)
}

func injectList(base string) error {
	resp, err := http.Get(base)
	if err != nil {
		return fmt.Errorf("calling admin API: %w", err)
	}
	return printResponse(resp)
}

func injectDelete(base, method, path string) error {
	target := base
	if path != "" {
		target += "?endpoint=" + url.QueryEscape(path)
		if method != "" {
			target += "&method=" + url.QueryEscape(method)
		}
	}
	req, _ := http.NewRequest(http.MethodDelete, target, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("calling admin API: %w", err)
	}
	return printResponse(resp)
}

func printResponse(resp *http.Response) error {
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	fmt.Println(strings.TrimSpace(string(b)))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("admin API returned %d", resp.StatusCode)
	}
	return nil
}
