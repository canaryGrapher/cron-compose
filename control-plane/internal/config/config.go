// Package config loads runtime config from environment variables. Keep this small;
// production deployments can layer a file-based loader on top of these defaults.
package config

import (
	"fmt"
	"os"
)

// Config is the resolved set of options the control plane runs with.
type Config struct {
	HTTPAddr       string // Fiber REST listener, e.g. :8080
	GRPCAddr       string // gRPC agent listener, e.g. :9090
	DatabaseURL    string // Postgres connection string
	Env            string // dev | prod
	LogLevel       string // debug | info | warn | error
	EnrollTokenTTL string // e.g. "30m"
	TLSDir         string // where ca.crt / ca.key / server.crt / server.key live
	TLSHosts       []string // SANs the server cert should cover

	SessionSecret      string // HMAC secret for session cookies
	SeedAdminEmail     string // optional: created/updated on every boot
	SeedAdminPassword  string

	// PublicHTTPURL is the externally reachable URL for the REST API (used in the
	// install command shown after creating a server). PublicGRPCAddr is the same for
	// the mTLS gRPC endpoint. InstallScriptURL is the agent installer URL.
	PublicHTTPURL    string
	PublicGRPCAddr   string
	InstallScriptURL string

	SecretsMasterKey string // 32 bytes hex

	// OIDC SSO. Empty issuer URL disables the SSO path.
	OIDCIssuerURL    string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string
	OIDCDefaultRole  string
}

// Load reads config from the environment with sensible dev defaults.
func Load() (Config, error) {
	c := Config{
		HTTPAddr:       env("HTTP_ADDR", ":8080"),
		GRPCAddr:       env("GRPC_ADDR", ":9090"),
		DatabaseURL:    env("DATABASE_URL", "postgres://croncompose:croncompose@localhost:5432/croncompose?sslmode=disable"),
		Env:            env("APP_ENV", "dev"),
		LogLevel:       env("LOG_LEVEL", "info"),
		EnrollTokenTTL: env("ENROLL_TOKEN_TTL", "30m"),
		TLSDir:         env("TLS_DIR", "./tls"),
		TLSHosts:       splitCSV(env("TLS_HOSTS", "localhost,127.0.0.1")),

		SessionSecret:     env("SESSION_SECRET", "dev-only-do-not-use-in-prod"),
		SeedAdminEmail:    env("SEED_ADMIN_EMAIL", ""),
		SeedAdminPassword: env("SEED_ADMIN_PASSWORD", ""),

		PublicHTTPURL:    env("PUBLIC_HTTP_URL", "http://localhost:8080/api/v1"),
		PublicGRPCAddr:   env("PUBLIC_GRPC_ADDR", "localhost:9090"),
		InstallScriptURL: env("INSTALL_SCRIPT_URL", "https://raw.githubusercontent.com/croncompose/croncompose/main/scripts/install-agent.sh"),

		// 32-byte hex. Default is a clearly-marked dev key so local dev works; prod
		// MUST set this to a real value generated with `openssl rand -hex 32`.
		SecretsMasterKey: env("SECRETS_MASTER_KEY", "0000000000000000000000000000000000000000000000000000000000000000"),

		OIDCIssuerURL:    env("OIDC_ISSUER_URL", ""),
		OIDCClientID:     env("OIDC_CLIENT_ID", ""),
		OIDCClientSecret: env("OIDC_CLIENT_SECRET", ""),
		OIDCRedirectURL:  env("OIDC_REDIRECT_URL", ""),
		OIDCDefaultRole:  env("OIDC_DEFAULT_ROLE", "viewer"),
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	if len(c.SessionSecret) < 16 {
		return c, fmt.Errorf("SESSION_SECRET must be at least 16 chars")
	}
	return c, nil
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	out := []string{}
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
