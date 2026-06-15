package cmd

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/croncompose/croncompose/cli/internal/render"
)

func init() {
	job := &cobra.Command{
		Use:     "job",
		Aliases: []string{"jobs"},
		Short:   "Manage jobs",
	}
	Root.AddCommand(job)

	var listServer string
	listCmd := &cobra.Command{
		Use:   "ls",
		Short: "List jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			requireLogin()
			path := "/jobs"
			if listServer != "" {
				path += "?server=" + listServer
			}
			var resp struct {
				Items []struct {
					ID, Name, TargetKind, ScheduleCron, Timezone string
					Enabled                                      bool
				} `json:"items"`
			}
			if err := API().JSON(http.MethodGet, path, nil, &resp); err != nil {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tTARGET\tCRON\tTZ\tENABLED")
			for _, j := range resp.Items {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%v\n",
					render.ShortID(j.ID), j.Name, j.TargetKind, j.ScheduleCron, j.Timezone, j.Enabled)
			}
			return w.Flush()
		},
	}
	listCmd.Flags().StringVar(&listServer, "server", "", "filter by server id")
	job.AddCommand(listCmd)

	var (
		addServer, addLabels       string
		addName, addCron, addTZ    string
		addInterp, addScriptFile   string
		addTimeout, addCPU, addMem int
		addSecretRefs              string
	)
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Create a job (script via --script-file, or - for stdin)",
		RunE: func(cmd *cobra.Command, args []string) error {
			requireLogin()
			if addServer == "" && addLabels == "" {
				return errors.New("either --server or --labels is required")
			}
			script, err := readScript(addScriptFile)
			if err != nil {
				return err
			}
			body := map[string]any{
				"name":              addName,
				"schedule_cron":     addCron,
				"timezone":          addTZ,
				"interpreter":       addInterp,
				"script_body":       script,
				"timeout_seconds":   addTimeout,
				"cpu_quota_percent": addCPU,
				"memory_max_mb":     addMem,
			}
			if addServer != "" {
				body["target_kind"] = "server"
				body["server_id"] = addServer
			} else {
				body["target_kind"] = "labels"
				body["target_labels"] = parseKV(addLabels)
			}
			if addSecretRefs != "" {
				refs := []string{}
				for _, r := range strings.Split(addSecretRefs, ",") {
					if r = strings.TrimSpace(r); r != "" {
						refs = append(refs, r)
					}
				}
				body["secret_refs"] = refs
			}
			var out struct{ ID, Name string }
			if err := API().JSON(http.MethodPost, "/jobs", body, &out); err != nil {
				return err
			}
			fmt.Printf("created %s (%s)\n", out.ID, out.Name)
			return nil
		},
	}
	addCmd.Flags().StringVar(&addServer, "server", "", "single-server target id (mutually exclusive with --labels)")
	addCmd.Flags().StringVar(&addLabels, "labels", "", "label selector, e.g. env=prod,role=worker")
	addCmd.Flags().StringVar(&addName, "name", "", "job name (required)")
	addCmd.Flags().StringVar(&addCron, "cron", "", "cron schedule, e.g. \"0 */6 * * *\" (required)")
	addCmd.Flags().StringVar(&addTZ, "tz", "UTC", "IANA timezone")
	addCmd.Flags().StringVar(&addInterp, "interp", "bash", "interpreter")
	addCmd.Flags().StringVar(&addScriptFile, "script-file", "", "path to script file (- for stdin)")
	addCmd.Flags().IntVar(&addTimeout, "timeout", 3600, "timeout in seconds")
	addCmd.Flags().IntVar(&addCPU, "cpu-quota", 0, "CPU quota %% (0 = unlimited)")
	addCmd.Flags().IntVar(&addMem, "memory-mb", 0, "memory cap in MB (0 = unlimited)")
	addCmd.Flags().StringVar(&addSecretRefs, "secrets", "", "comma-separated secret names")
	_ = addCmd.MarkFlagRequired("name")
	_ = addCmd.MarkFlagRequired("cron")
	_ = addCmd.MarkFlagRequired("script-file")
	job.AddCommand(addCmd)

	for _, action := range []string{"enable", "disable"} {
		a := action
		job.AddCommand(&cobra.Command{
			Use:   a + " <job_id>",
			Short: strings.Title(a) + " a job",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				requireLogin()
				_, err := API().Do(http.MethodPost, "/jobs/"+args[0]+"/"+a, nil)
				return err
			},
		})
	}

	job.AddCommand(&cobra.Command{
		Use:   "run <job_id>",
		Short: "Manually run a job now (fans out to all matching servers)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireLogin()
			var out struct {
				Runs []struct {
					ServerID string `json:"server_id"`
					RunID    string `json:"run_id"`
					Status   string `json:"status"`
				} `json:"runs"`
			}
			if err := API().JSON(http.MethodPost, "/jobs/"+args[0]+"/run", nil, &out); err != nil {
				return err
			}
			for _, r := range out.Runs {
				fmt.Printf("%s\t%s\t%s\n", render.ShortID(r.RunID), render.ShortID(r.ServerID), r.Status)
			}
			return nil
		},
	})

	job.AddCommand(&cobra.Command{
		Use:   "rm <job_id>",
		Short: "Delete a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			requireLogin()
			res, err := API().Do(http.MethodDelete, "/jobs/"+args[0], nil)
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

func readScript(path string) (string, error) {
	if path == "-" {
		b, err := io.ReadAll(os.Stdin)
		return string(b), err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func parseKV(in string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(in, ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		eq := strings.Index(p, "=")
		if eq < 0 {
			continue
		}
		out[strings.TrimSpace(p[:eq])] = strings.TrimSpace(p[eq+1:])
	}
	return out
}
