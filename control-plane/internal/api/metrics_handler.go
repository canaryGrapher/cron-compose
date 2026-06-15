package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/croncompose/croncompose/control-plane/internal/metrics"
)

// metricsHandler serves the Prometheus exposition format at /metrics. It is mounted
// unauthenticated at the app root; firewall to your monitoring network in prod.
func metricsHandler() fiber.Handler {
	h := promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{})
	return func(c fiber.Ctx) error {
		// Fiber v3 exposes the underlying fasthttp ctx via fasthttpadaptor or by
		// writing the response ourselves. For simplicity we call promhttp via an
		// adapter wrapper.
		adaptor := fasthttpHTTPHandler(h)
		adaptor(c.RequestCtx())
		return nil
	}
}
