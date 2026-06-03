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
	)

	cmd := &cobra.Command{
		Use:   "run <spec>",
		Short: "Start a mock server from an OpenAPI spec",
		Args:  cobra.ExactArgs(1),
		Example: "  mocksmith run openapi.yaml\n" +
			"  mocksmith run openapi.yaml --addr :9090",
		RunE: func(_ *cobra.Command, args []string) error {
			rt, err := app.New(app.Options{
				SpecPath:    args[0],
				Addr:        addr,
				Profile:     profile,
				ForceStatus: forceStatus,
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
	return cmd
}
