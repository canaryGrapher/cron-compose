package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/croncompose/croncompose/cli/internal/render"
)

func init() {
	secret := &cobra.Command{
		Use:     "secret",
		Aliases: []string{"secrets"},
		Short:   "Manage secrets (admin)",
	}
	Root.AddCommand(secret)

	secret.AddCommand(&cobra.Command{
		Use:   "ls",
		Short: "List secret names (values are never returned)",
		RunE: func(cmd *cobra.Command, args []string) error {
			requireLogin()
			var resp struct {
				Items []struct {
					ID        string `json:"id"`
					Name      string `json:"name"`
					Scope     string `json:"scope"`
					ScopeID   string `json:"scope_id,omitempty"`
					CreatedAt string `json:"created_at"`
				} `json:"items"`
			}
			if err := API().JSON(http.MethodGet, "/secrets", nil, &resp); err != nil {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tSCOPE\tSCOPE_ID\tCREATED")
			for _, s := range resp.Items {
				ts := s.CreatedAt
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					render.ShortID(s.ID), s.Name, s.Scope, render.ShortID(s.ScopeID), ts)
			}
			return w.Flush()
		},
	})

	var addName, addScope, addScopeID string
	var addFromStdin bool
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Create a secret. Value read from stdin (or prompted).",
		RunE: func(cmd *cobra.Command, args []string) error {
			requireLogin()
			if addName == "" {
				return errors.New("--name is required")
			}
			var value string
			if addFromStdin {
				b := make([]byte, 0, 4096)
				buf := make([]byte, 1024)
				for {
					n, err := os.Stdin.Read(buf)
					if n > 0 {
						b = append(b, buf[:n]...)
					}
					if err != nil {
						break
					}
				}
				value = strings.TrimRight(string(b), "\r\n")
			} else {
				fmt.Print("Value: ")
				pw, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Println()
				if err != nil {
					return err
				}
				value = string(pw)
			}
			body := map[string]string{
				"name":  addName,
				"value": value,
				"scope": addScope,
			}
			if addScopeID != "" {
				body["scope_id"] = addScopeID
			}
			var out struct{ ID, Name string }
			if err := API().JSON(http.MethodPost, "/secrets", body, &out); err != nil {
				return err
			}
			fmt.Printf("created %s (%s)\n", out.ID, out.Name)
			return nil
		},
	}
	addCmd.Flags().StringVar(&addName, "name", "", "env var name (required)")
	addCmd.Flags().StringVar(&addScope, "scope", "global", "scope: global|server|job")
	addCmd.Flags().StringVar(&addScopeID, "scope-id", "", "server_id or job_id when scope != global")
	addCmd.Flags().BoolVar(&addFromStdin, "stdin", false, "read value from stdin instead of prompting")
	secret.AddCommand(addCmd)

	secret.AddCommand(&cobra.Command{
		Use:   "rm <secret_id>",
		Short: "Delete a secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireLogin()
			res, err := API().Do(http.MethodDelete, "/secrets/"+args[0], nil)
			if err != nil {
				return err
			}
			res.Body.Close()
			if res.StatusCode/100 != 2 {
				return fmt.Errorf("delete: %d", res.StatusCode)
			}
			fmt.Println("deleted")
			return nil
		},
	})
}
