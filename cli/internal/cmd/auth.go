package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/croncompose/croncompose/cli/internal/config"
)

var loginEmail string

func init() {
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Sign in and save the session",
		RunE: func(cmd *cobra.Command, args []string) error {
			email := loginEmail
			if email == "" {
				fmt.Print("Email: ")
				_, _ = fmt.Scanln(&email)
				email = strings.TrimSpace(email)
			}
			fmt.Print("Password: ")
			pw, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println()
			if err != nil {
				return err
			}
			c := API()
			session, err := c.Login(email, string(pw))
			if err != nil {
				return err
			}
			Cfg.SessionID = session
			if err := config.Save(Cfg); err != nil {
				return err
			}
			fmt.Printf("signed in as %s\n", email)
			return nil
		},
	}
	loginCmd.Flags().StringVarP(&loginEmail, "email", "e", "", "email (prompted if omitted)")
	Root.AddCommand(loginCmd)

	Root.AddCommand(&cobra.Command{
		Use:   "logout",
		Short: "Clear the saved session",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := API()
			if Cfg.SessionID != "" {
				_, _ = c.Do(http.MethodPost, "/auth/logout", nil)
			}
			Cfg.SessionID = ""
			return config.Save(Cfg)
		},
	})

	Root.AddCommand(&cobra.Command{
		Use:   "whoami",
		Short: "Print the signed-in user",
		RunE: func(cmd *cobra.Command, args []string) error {
			requireLogin()
			var me struct {
				Email string `json:"email"`
				Role  string `json:"role"`
			}
			if err := API().JSON(http.MethodGet, "/me", nil, &me); err != nil {
				return err
			}
			fmt.Printf("%s (%s)\n", me.Email, me.Role)
			return nil
		},
	})
}
