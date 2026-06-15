// Package api wires the Fiber HTTP layer. This file holds shared middleware so the
// router stays focused on routing.
package api

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/croncompose/croncompose/control-plane/internal/metrics"
)

// requestLogger logs each request with method, path, status, and duration.
func requestLogger(log *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		log.Info("http",
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"dur_ms", time.Since(start).Milliseconds(),
		)
		return err
	}
}

// requestMetrics records request counts + latency to Prometheus. We use the route's
// registered pattern (e.g. "/api/v1/jobs/:id") so cardinality stays bounded.
func requestMetrics() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		pattern := c.Route().Path
		if pattern == "" {
			pattern = c.Path()
		}
		status := strconv.Itoa(c.Response().StatusCode())
		metrics.HTTPRequestsTotal.WithLabelValues(c.Method(), pattern, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(c.Method(), pattern).Observe(time.Since(start).Seconds())
		return err
	}
}
