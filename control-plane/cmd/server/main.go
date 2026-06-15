// Command server is the CronCompose control plane: a Fiber REST API for the UI and
// an mTLS gRPC endpoint for agents.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/croncompose/croncompose/control-plane/internal/agentgw"
	"github.com/croncompose/croncompose/control-plane/internal/api"
	"github.com/croncompose/croncompose/control-plane/internal/auth"
	"github.com/croncompose/croncompose/control-plane/internal/config"
	"github.com/croncompose/croncompose/control-plane/internal/cryptobox"
	"github.com/croncompose/croncompose/control-plane/internal/db"
	"github.com/croncompose/croncompose/control-plane/internal/logger"
	"github.com/croncompose/croncompose/control-plane/internal/notify"
	"github.com/croncompose/croncompose/control-plane/internal/pki"
	"github.com/croncompose/croncompose/control-plane/internal/secrets"
)

func main() {
	if err := run(); err != nil {
		os.Stderr.WriteString("fatal: " + err.Error() + "\n")
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel)
	log.Info("starting control plane", "env", cfg.Env)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	auth.SeedAdmin(ctx, log, auth.NewStore(pool), cfg.SeedAdminEmail, cfg.SeedAdminPassword)

	box, err := cryptobox.New(cfg.SecretsMasterKey)
	if err != nil {
		return err
	}

	bundle, err := pki.LoadOrCreate(cfg.TLSDir, cfg.TLSHosts)
	if err != nil {
		return err
	}
	log.Info("pki ready", "dir", cfg.TLSDir, "hosts", cfg.TLSHosts)

	secretStore := secrets.NewStore(pool, box)
	notifier := notify.NewNotifier(notify.NewStore(pool), log)
	gw := agentgw.New(cfg.GRPCAddr, log, pool, bundle, secretStore)
	gw.SetFailedRunHook(notifier)
	if err := gw.Start(ctx); err != nil {
		return err
	}
	defer gw.Stop()

	app := api.New(api.Deps{
		Log:              log,
		Pool:             pool,
		Gateway:          gw,
		PKI:              bundle,
		GRPCAddr:         cfg.GRPCAddr,
		SessionSecret:    []byte(cfg.SessionSecret),
		PublicHTTPURL:    cfg.PublicHTTPURL,
		PublicGRPCAddr:   cfg.PublicGRPCAddr,
		InstallScriptURL: cfg.InstallScriptURL,
		Crypto:           box,
	})

	errCh := make(chan error, 1)
	go func() { errCh <- app.Listen(cfg.HTTPAddr) }()
	log.Info("http listening", "addr", cfg.HTTPAddr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		log.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			log.Error("http stopped", "err", err)
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = app.ShutdownWithContext(shutdownCtx)
	return nil
}
