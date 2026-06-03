package main

import (
	"fmt"
	"os"

	"github.com/Claudio712005/mock-smith/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
