package cli

import (
	"strings"

	"github.com/Claudio712005/mock-smith/internal/app"
	"github.com/Claudio712005/mock-smith/internal/scenario"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var (
		addr        string
		profile     string
		forceStatus int
		slow        []string
		fail        []string
		timeout     []string
		corrupt     []string
		sequence    []string
	)

	cmd := &cobra.Command{
		Use:   "run <spec>",
		Short: "Start a mock server from an OpenAPI spec",
		Args:  cobra.ExactArgs(1),
		Example: "  mocksmith run openapi.yaml\n" +
			"  mocksmith run openapi.yaml --addr :9090\n" +
			"  mocksmith run openapi.yaml --slow /payments=2s --fail /auth=503\n" +
			"  mocksmith run openapi.yaml --timeout /jobs=20% --corrupt /users=10%\n" +
			"  mocksmith run openapi.yaml --sequence /jobs=202,202,200",
		RunE: func(_ *cobra.Command, args []string) error {
			rt, err := app.New(app.Options{
				SpecPath:    args[0],
				Addr:        addr,
				Profile:     profile,
				ForceStatus: forceStatus,
				Slow:        slow,
				Fail:        fail,
				Timeout:     timeout,
				Corrupt:     corrupt,
				Sequence:    sequence,
			})
			if err != nil {
				return err
			}
			return rt.Run()
		},
	}

	cmd.Flags().StringVar(&addr, "addr", ":8080", "address the mock server listens on")
	cmd.Flags().StringVar(&profile, "profile", "happy",
		"runtime profile ("+strings.Join(scenario.Available(), ", ")+")")
	cmd.Flags().IntVar(&forceStatus, "force-status", 0,
		"force this HTTP status on endpoints that document it; others follow the profile (0 = off)")
	cmd.Flags().StringArrayVar(&slow, "slow", nil,
		"add latency, [path=]duration, e.g. /pets=2s or 2s for all (repeatable)")
	cmd.Flags().StringArrayVar(&fail, "fail", nil,
		"force a status, [path=]status, e.g. /payments=503 (repeatable)")
	cmd.Flags().StringArrayVar(&timeout, "timeout", nil,
		"inject timeouts (504) at a rate, [path=]rate, e.g. /auth=20% (repeatable)")
	cmd.Flags().StringArrayVar(&corrupt, "corrupt", nil,
		"corrupt the JSON body at a rate, [path=]rate, e.g. /users=10% (repeatable)")
	cmd.Flags().StringArrayVar(&sequence, "sequence", nil,
		"return statuses in order per request, [path=]s1,s2,..., e.g. /jobs=202,202,200 (repeatable)")
	return cmd
}
