package cmd

import (
	"fmt"
	"net/http"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func init() {
	Root.AddCommand(&cobra.Command{
		Use:   "audit",
		Short: "Print recent audit log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			requireLogin()
			var resp struct {
				Items []struct {
					Action, TargetType, TargetID string
					ActorUserID                  *string `json:"actor_user_id"`
					Ts                           string  `json:"ts"`
				} `json:"items"`
			}
			if err := API().JSON(http.MethodGet, "/audit?limit=50", nil, &resp); err != nil {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "WHEN\tACTOR\tACTION\tTARGET")
			for _, e := range resp.Items {
				actor := "system"
				if e.ActorUserID != nil && *e.ActorUserID != "" {
					actor = (*e.ActorUserID)[:8]
				}
				target := ""
				if e.TargetType != "" {
					target = e.TargetType + "/" + first8(e.TargetID)
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Ts, actor, e.Action, target)
			}
			return w.Flush()
		},
	})
}

func first8(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:8]
}
