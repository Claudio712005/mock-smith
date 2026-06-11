package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Claudio712005/mock-smith/internal/app"
	"github.com/Claudio712005/mock-smith/internal/config"
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
		configPath  string
	)

	cmd := &cobra.Command{
		Use:   "run [spec]",
		Short: "Start a mock server from an OpenAPI spec",
		Args:  cobra.MaximumNArgs(1),
		Example: "  mocksmith run openapi.yaml\n" +
			"  mocksmith run openapi.yaml --addr :9090\n" +
			"  mocksmith run openapi.yaml --slow /payments=2s --fail /auth=503\n" +
			"  mocksmith run openapi.yaml --sequence /jobs=202,202,200\n" +
			"  mocksmith run --config mocksmith.yaml\n" +
			"  mocksmith run --config examples/mocksmith.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			var opts app.Options
			if cmd.Flags().Changed("config") {
				cfg, err := config.Load(configPath)
				if err != nil {
					return err
				}
				servers, err := cfg.ServerList()
				if err != nil {
					return err
				}
				// Vários servers: só via config, sem flags de comportamento.
				if len(servers) > 1 {
					return runMultiServer(servers, configPath)
				}
				opts, err = servers[0].ToOptions()
				if err != nil {
					return err
				}
			}

			if cmd.Flags().Changed("addr") || opts.Addr == "" {
				opts.Addr = addr
			}
			if cmd.Flags().Changed("profile") || opts.Profile == "" {
				opts.Profile = profile
			}
			if cmd.Flags().Changed("force-status") {
				opts.ForceStatus = forceStatus
			}

			// Spec do config resolve relativo ao diretório do arquivo de config;
			// spec posicional resolve relativo ao diretório atual.
			spec := resolveSpec(opts.SpecPath, configPath)
			if len(args) > 0 {
				spec = args[0]
			}
			if spec == "" {
				return fmt.Errorf("no spec given: pass it as an argument or set 'spec' in the config file")
			}
			opts.SpecPath = spec

			opts.Slow = append(opts.Slow, slow...)
			opts.Fail = append(opts.Fail, fail...)
			opts.Timeout = append(opts.Timeout, timeout...)
			opts.Corrupt = append(opts.Corrupt, corrupt...)
			opts.Sequence = append(opts.Sequence, sequence...)

			rt, err := app.New(opts)
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
	cmd.Flags().StringVar(&configPath, "config", "",
		"path to a YAML config file, e.g. "+config.DefaultFile)
	return cmd
}

// resolveSpec resolve o caminho da spec do config relativo ao diretório do
// arquivo de config (caminhos absolutos e spec vazia passam direto).
func resolveSpec(spec, configPath string) string {
	if spec == "" || filepath.IsAbs(spec) || configPath == "" {
		return spec
	}
	return filepath.Join(filepath.Dir(configPath), spec)
}

// runMultiServer sobe um server por entrada do bloco "servers". As flags de
// comportamento não se aplicam aqui — tudo vem do config.
func runMultiServer(servers []config.ServerConfig, configPath string) error {
	list := make([]app.Options, 0, len(servers))
	for i, sc := range servers {
		opts, err := sc.ToOptions()
		if err != nil {
			return err
		}
		if opts.SpecPath == "" {
			return fmt.Errorf("server %d: missing 'spec'", i+1)
		}
		opts.SpecPath = resolveSpec(opts.SpecPath, configPath)
		if opts.Addr == "" {
			return fmt.Errorf("server %q: missing 'addr'", sc.Spec)
		}
		if opts.Profile == "" {
			opts.Profile = "happy"
		}
		list = append(list, opts)
	}
	return app.RunMany(list)
}
