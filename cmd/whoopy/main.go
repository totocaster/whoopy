package main

import (
	"context"
	"fmt"
	"os"

	"github.com/toto/whoopy/internal/cli"
)

func main() {
	ctx := context.Background()
	if err := cli.Execute(ctx); err != nil {
		if cliErr, ok := err.(interface{ ExitCode() int }); ok {
			os.Exit(cliErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
