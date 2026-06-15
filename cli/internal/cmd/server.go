package cmd

import (
	"fmt"
	"net/http"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/croncompose/croncompose/cli/internal/render"
)

func init() {
	srv := &cobra.Command{
		Use:     "server",
		Aliases: []string{"servers"},
		Short:   "Manage servers",
	}
	Root.AddCommand(srv)

	srv.AddCommand(&cobra.Command{
		Use:   "ls",
		Short: "List servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			requireLogin()
			var resp struct {
				Items []struct {
					ID, Name, Status, OS, Arch string
					LastSeenAt                 *string `json:"last_seen_at"`
				} `json:"items"`
			}
			if err := API().JSON(http.MethodGet, "/servers", nil, &resp); err != nil {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tSTATUS\tOS/ARCH\tLAST SEEN")
			for _, s := range resp.Items {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s/%s\t%s\n",
					render.ShortID(s.ID), s.Name, s.Status, s.OS, s.Arch, render.Time(s.LastSeenAt))
			}
			return w.Flush()
		},
	})

	var addName, addDescription string
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Create a server and print the install command",
		RunE: func(cmd *cobra.Command, args []string) error {
			requireLogin()
			var out struct {
				Server         struct{ ID, Name string } `json:"server"`
				InstallCommand string                    `json:"install_command"`
				Enrollment     struct{ Token string }    `json:"enrollment"`
			}
			if err := API().JSON(http.MethodPost, "/servers", map[string]string{
				"name":        addName,
				"description": addDescription,
			}, &out); err != nil {
				return err
			}
			fmt.Printf("created server %s (%s)\n", out.Server.ID, out.Server.Name)
			fmt.Println("\nInstall on the target server:")
			fmt.Println(out.InstallCommand)
			return nil
		},
	}
	addCmd.Flags().StringVarP(&addName, "name", "n", "", "server name (required)")
	addCmd.Flags().StringVarP(&addDescription, "description", "d", "", "description")
	_ = addCmd.MarkFlagRequired("name")
	srv.AddCommand(addCmd)

	srv.AddCommand(&cobra.Command{
		Use:   "rm <server_id>",
		Short: "Delete a server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireLogin()
			res, err := API().Do(http.MethodDelete, "/servers/"+args[0], nil)
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
