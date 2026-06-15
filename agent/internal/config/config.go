// Package config loads agent settings from environment + a small config file.
package config

import (
	"fmt"
	"os"
)

// Config is everything the agent needs at runtime.
type Config struct {
	ControlPlaneAddr     string // host:port of the gRPC endpoint
	ControlPlaneHTTPBase string // base URL for REST calls (enrollment)
	ControlPlaneSNI      string // server name to verify against in TLS
	DataDir              string // where the local store, cert, and key live
	AgentVersion         string // injected at build time or hard-coded
}

// Load reads env vars with dev-friendly defaults.
func Load() (Config, error) {
	c := Config{
		ControlPlaneAddr:     env("CONTROL_PLANE_ADDR", "localhost:9090"),
		ControlPlaneHTTPBase: env("CONTROL_PLANE_HTTP", "http://localhost:8080/api/v1"),
		ControlPlaneSNI:      env("CONTROL_PLANE_SNI", "localhost"),
		DataDir:              env("DATA_DIR", "/var/lib/croncompose"),
		AgentVersion:         env("AGENT_VERSION", "0.1.0-dev"),
	}
	if c.ControlPlaneAddr == "" {
		return c, fmt.Errorf("CONTROL_PLANE_ADDR is required")
	}
	return c, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
