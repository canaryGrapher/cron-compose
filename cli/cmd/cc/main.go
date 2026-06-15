// Command cc is the CronCompose CLI. See `cc help` for subcommands.
package main

import (
	"fmt"
	"os"

	"github.com/croncompose/croncompose/cli/internal/cmd"
)

func main() {
	if err := cmd.Root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
