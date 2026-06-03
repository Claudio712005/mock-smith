package cli

import (
	"github.com/Claudio712005/mock-smith/internal/app"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var (
		addr    string
		profile string
	)

	cmd := &cobra.Command{
		Use:   "run <spec>",
		Short: "Start a mock server from an OpenAPI spec",
		Args:  cobra.ExactArgs(1),
		Example: "  mocksmith run openapi.yaml\n" +
			"  mocksmith run openapi.yaml --addr :9090",
		RunE: func(_ *cobra.Command, args []string) error {
			rt, err := app.New(app.Options{
				SpecPath: args[0],
				Addr:     addr,
				Profile:  profile,
			})
			if err != nil {
				return err
			}
			return rt.Run()
		},
	}

	cmd.Flags().StringVar(&addr, "addr", ":8080", "address the mock server listens on")
	cmd.Flags().StringVar(&profile, "profile", "happy", "runtime profile (MVP: happy)")
	return cmd
}
