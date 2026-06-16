// Command proxy is CronCompose's single entry point. It fronts the web UI, the
// REST API, and agent gRPC on one address: browser traffic is path-routed to the
// UI and control plane, while agent mTLS gRPC is passed straight through.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/croncompose/croncompose/proxy/internal/config"
	"github.com/croncompose/croncompose/proxy/internal/httpproxy"
	"github.com/croncompose/croncompose/proxy/internal/mux"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		log.Error("invalid configuration", "err", err)
		os.Exit(1)
	}

	handler, err := httpproxy.New(cfg)
	if err != nil {
		log.Error("build http proxy", "err", err)
		os.Exit(1)
	}

	srv, err := mux.New(cfg, handler, log)
	if err != nil {
		log.Error("build mux", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("croncompose proxy starting",
		"listen", cfg.ListenAddr,
		"web", cfg.WebUpstream,
		"api", cfg.APIUpstream,
		"grpc", cfg.GRPCUpstream,
		"tls_termination", cfg.TLSEnabled(),
		"agent_sni", cfg.AgentSNI,
	)

	if err := srv.Serve(ctx); err != nil {
		log.Error("proxy stopped", "err", err)
		os.Exit(1)
	}
	log.Info("proxy stopped cleanly")
}
