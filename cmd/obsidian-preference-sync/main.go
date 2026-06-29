package main

import (
	"context"
	"fmt"
	"os"

	"obsidian-preference-sync/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, version); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(cli.ExitCode(err))
	}
}
