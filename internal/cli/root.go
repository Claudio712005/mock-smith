package cli

import "github.com/spf13/cobra"

// NewRootCmd monta o comando raiz `mocksmith` e registra os subcomandos. A
// impressão de erro/uso do cobra é silenciada para que o erro suba até Execute
// e seja exibido uma única vez.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "mocksmith",
		Short: "Runtime API mocking engine driven by OpenAPI specs",
		Long: "MockSmith turns an OpenAPI/Swagger specification into a live HTTP mock\n" +
			"server at runtime — no static mock files, zero config by default.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newRunCmd())
	root.AddCommand(newInjectCmd())
	return root
}

// Execute monta e roda o comando raiz, devolvendo eventual erro ao chamador
// sem imprimi-lo. É o ponto de entrada usado pela main.
func Execute() error {
	return NewRootCmd().Execute()
}
