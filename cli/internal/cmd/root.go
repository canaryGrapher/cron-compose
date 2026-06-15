// Package cmd registers the root cobra command and feature subcommands.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/croncompose/croncompose/cli/internal/client"
	"github.com/croncompose/croncompose/cli/internal/config"
)

// Root is the top-level `cc` command. Subcommands attach via their own files.
var Root = &cobra.Command{
	Use:   "cc",
	Short: "CronCompose CLI",
	Long: "Power-user CLI for the CronCompose control plane. " +
		"Run `cc login` first; everything else uses the saved session.",
	SilenceUsage: true,
}

// Cfg is the loaded config, available to subcommands.
var Cfg config.Config

// API returns a Client configured from the loaded config.
func API() *client.Client {
	return client.New(Cfg.APIBase, Cfg.SessionID)
}

// requireLogin prints a hint and exits if no session is saved.
func requireLogin() {
	if Cfg.SessionID == "" {
		fmt.Fprintln(os.Stderr, "not logged in. Run: cc login")
		os.Exit(2)
	}
}

// init loads the config once and seeds Cfg for every subcommand. cobra calls
// PersistentPreRun before running any subcommand.
func init() {
	Root.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		c, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, "config:", err)
			os.Exit(1)
		}
		Cfg = c
	}
}
