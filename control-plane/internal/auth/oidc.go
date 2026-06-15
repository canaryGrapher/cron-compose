package auth

import (
	"context"
	"errors"
	"log/slog"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCConfig is the minimum needed to bring up the SSO path.
type OIDCConfig struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	DefaultRole  string // role assigned to auto-provisioned users
}

// Enabled reports whether SSO is configured.
func (c OIDCConfig) Enabled() bool {
	return c.IssuerURL != "" && c.ClientID != "" && c.RedirectURL != ""
}

// OIDC bundles a configured provider + verifier + oauth2 config. nil when SSO is off.
type OIDC struct {
	Cfg      OIDCConfig
	Provider *oidc.Provider
	Verifier *oidc.IDTokenVerifier
	OAuth    *oauth2.Config
}

// NewOIDC initialises the provider by hitting issuer/.well-known/openid-configuration.
// Returns nil, nil when SSO is not configured.
func NewOIDC(ctx context.Context, cfg OIDCConfig, log *slog.Logger) (*OIDC, error) {
	if !cfg.Enabled() {
		log.Info("oidc disabled (set OIDC_ISSUER_URL + OIDC_CLIENT_ID + OIDC_REDIRECT_URL to enable)")
		return nil, nil
	}
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, err
	}
	if cfg.DefaultRole == "" {
		cfg.DefaultRole = "viewer"
	}
	if !validRole(cfg.DefaultRole) {
		return nil, errors.New("OIDC_DEFAULT_ROLE must be viewer|operator|admin|owner")
	}
	o := &OIDC{
		Cfg:      cfg,
		Provider: provider,
		Verifier: provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		OAuth: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
		},
	}
	log.Info("oidc enabled", "issuer", cfg.IssuerURL)
	return o, nil
}

func validRole(r string) bool {
	switch r {
	case "viewer", "operator", "admin", "owner":
		return true
	}
	return false
}
