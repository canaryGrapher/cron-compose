// Package metrics owns the Prometheus registry and the metric vectors the rest of the
// control plane writes to. Kept tiny so feature packages just call e.g.
// metrics.RunsTotal.WithLabelValues("succeeded").Inc() without seeing any plumbing.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Registry is the local registry. We avoid the default registry so test imports and
// shutdowns are clean.
var Registry = prometheus.NewRegistry()

// HTTPRequestsTotal counts REST requests by method + path template + status.
var HTTPRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "cc_http_requests_total",
	Help: "Total REST requests handled by the control plane.",
}, []string{"method", "path", "status"})

// HTTPRequestDuration measures REST handler latency.
var HTTPRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "cc_http_request_duration_seconds",
	Help:    "REST handler latency in seconds.",
	Buckets: prometheus.DefBuckets,
}, []string{"method", "path"})

// AgentsConnected is the number of agents currently holding an open AgentStream.
var AgentsConnected = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "cc_agents_connected",
	Help: "Number of agents with an active AgentStream.",
})

// RunsTotal counts run terminations by status (succeeded / failed / timed_out /
// canceled / skipped).
var RunsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "cc_runs_total",
	Help: "Total runs by terminal status.",
}, []string{"status"})

// LogSubscribers tracks how many SSE subscribers the broker has, summed across runs.
var LogSubscribers = prometheus.NewGauge(prometheus.GaugeOpts{
	Name: "cc_log_subscribers",
	Help: "Current SSE subscribers across all runs.",
})

func init() {
	Registry.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
		AgentsConnected,
		RunsTotal,
		LogSubscribers,
		// Process + Go runtime collectors for free.
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
}
