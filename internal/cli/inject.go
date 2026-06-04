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
		Use:   "inject [\"/path:status(rate%)\"]",
		Short: "Change a running server's behavior via its Admin API",
		Example: "  mocksmith inject \"/payments:503(20%)\"\n" +
			"  mocksmith inject /payments:503\n" +
			"  mocksmith inject /payments --latency-ms 2000\n" +
			"  mocksmith inject --list\n" +
			"  mocksmith inject --remove /payments\n" +
			"  mocksmith inject --clear",
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			base := "http://" + strings.TrimPrefix(addr, "http://") + adminRuntimePath

			switch {
			case list:
				return injectList(base)
			case clear:
				return injectDelete(base, "")
			case remove != "":
				return injectDelete(base, remove)
			}

			if len(args) == 0 {
				return fmt.Errorf("nothing to do: pass \"/path:status(rate%%)\" or use --list/--remove/--clear")
			}

			path, status, rate, err := parseInject(args[0])
			if err != nil {
				return err
			}
			if status == 0 && latencyMs == 0 {
				return fmt.Errorf("set a status (e.g. /payments:503) or --latency-ms")
			}
			return injectSet(base, path, status, latencyMs, rate)
		},
	}

	cmd.Flags().StringVar(&addr, "addr", ":8080", "address of the running mock server")
	cmd.Flags().IntVar(&latencyMs, "latency-ms", 0, "latency in milliseconds to inject")
	cmd.Flags().BoolVar(&list, "list", false, "list active runtime overrides")
	cmd.Flags().BoolVar(&clear, "clear", false, "clear all runtime overrides")
	cmd.Flags().StringVar(&remove, "remove", "", "remove the override of a specific endpoint path")
	return cmd
}

const adminRuntimePath = "/__mocksmith/runtime"

func parseInject(s string) (path string, status int, rate float64, err error) {
	if i := strings.Index(s, "("); i >= 0 {
		if !strings.HasSuffix(s, "%)") {
			return "", 0, 0, fmt.Errorf("invalid rate in %q (want \"(NN%%)\")", s)
		}
		pct := s[i+1 : len(s)-2]
		n, perr := strconv.ParseFloat(pct, 64)
		if perr != nil || n < 0 || n > 100 {
			return "", 0, 0, fmt.Errorf("invalid rate %q (want 0-100)", pct)
		}
		rate = n / 100
		s = s[:i]
	}

	path = s
	if i := strings.LastIndex(s, ":"); i >= 0 {
		path = s[:i]
		status, err = strconv.Atoi(s[i+1:])
		if err != nil || status < 100 || status > 599 {
			return "", 0, 0, fmt.Errorf("invalid status in %q (want path:status, 100-599)", s)
		}
	}
	if path == "" {
		return "", 0, 0, fmt.Errorf("missing endpoint path")
	}
	return path, status, rate, nil
}

func injectSet(base, path string, status, latencyMs int, rate float64) error {
	body, _ := json.Marshal(map[string]any{
		"endpoint":  path,
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

func injectDelete(base, path string) error {
	target := base
	if path != "" {
		target += "?endpoint=" + url.QueryEscape(path)
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
