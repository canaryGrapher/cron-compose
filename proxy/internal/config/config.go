// Package config loads the proxy's runtime configuration from the environment.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config is the proxy's runtime configuration.
type Config struct {
	// ListenAddr is the single public entry point. Browser HTTP(S) and agent
	// gRPC all arrive here and are demultiplexed by the mux.
	ListenAddr string

	// Upstreams (internal addresses, reachable only from the proxy).
	WebUpstream  string // e.g. http://web:3000          (Next.js UI)
	APIUpstream  string // e.g. http://control-plane:8080 (Fiber REST)
	GRPCUpstream string // host:port, e.g. control-plane:9090 (agent mTLS gRPC)

	// Path mapping for the REST API. Requests whose path starts with APIPrefix
	// go to the control plane with APIPrefix rewritten to APIUpstreamPrefix.
	APIPrefix         string // public prefix, default "/api"
	APIUpstreamPrefix string // control-plane prefix, default "/api/v1"

	// WebPrefix is the path the web UI is served under. It must match the
	// Next.js basePath. Requests under it go to the web upstream unchanged;
	// anything that is neither WebPrefix nor APIPrefix (e.g. bare "/") is
	// redirected into it. Set to "" to serve the UI at the root instead.
	WebPrefix string // default "/app"

	// TLS termination for browser HTTPS at the proxy (optional). When set, the
	// mux terminates TLS for browser traffic and routes agent TLS (matched by
	// AgentSNI) straight through to gRPC without decryption.
	TLSCertFile string
	TLSKeyFile  string
	AgentSNI    string // SNI agents present, e.g. "agents.example.com"

	// Timeouts.
	ReadHeaderTimeout time.Duration
	IdleTimeout       time.Duration
	DialTimeout       time.Duration
}

// TLSEnabled reports whether the proxy terminates browser TLS itself.
func (c Config) TLSEnabled() bool { return c.TLSCertFile != "" }

// Load reads configuration from the environment, applies defaults, and validates.
func Load() (Config, error) {
	c := Config{
		ListenAddr:        env("PROXY_LISTEN_ADDR", ":8000"),
		WebUpstream:       env("WEB_UPSTREAM", "http://localhost:3000"),
		APIUpstream:       env("API_UPSTREAM", "http://localhost:8080"),
		GRPCUpstream:      env("GRPC_UPSTREAM", "localhost:9090"),
		APIPrefix:         env("API_PREFIX", "/api"),
		APIUpstreamPrefix: env("API_UPSTREAM_PREFIX", "/api/v1"),
		WebPrefix:         strings.TrimSuffix(env("WEB_PREFIX", "/app"), "/"),
		TLSCertFile:       os.Getenv("TLS_CERT_FILE"),
		TLSKeyFile:        os.Getenv("TLS_KEY_FILE"),
		AgentSNI:          os.Getenv("AGENT_SNI"),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		DialTimeout:       10 * time.Second,
	}

	if !strings.HasPrefix(c.APIPrefix, "/") {
		return c, fmt.Errorf("API_PREFIX must start with '/': %q", c.APIPrefix)
	}
	if c.WebPrefix != "" && !strings.HasPrefix(c.WebPrefix, "/") {
		return c, fmt.Errorf("WEB_PREFIX must start with '/' (or be '/' for root): %q", c.WebPrefix)
	}
	if (c.TLSCertFile == "") != (c.TLSKeyFile == "") {
		return c, fmt.Errorf("TLS_CERT_FILE and TLS_KEY_FILE must be set together")
	}
	// With a cert, agent TLS and browser HTTPS both arrive as TLS; SNI is the
	// only way to tell them apart, so it becomes required.
	if c.TLSCertFile != "" && c.AgentSNI == "" {
		return c, fmt.Errorf("AGENT_SNI is required when TLS termination is enabled")
	}
	return c, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
