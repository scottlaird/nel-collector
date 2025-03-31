package collector

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RunMetricsServer creates an HTTP server that listens on the supplied
// `addr` and serves Prometheus metrics on `/metrics`.  Under normal
// circumstances, this will not return until server shutdown.
func RunMetricsServer(addr string) error {
	metricMux := http.NewServeMux()
	metricMux.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(addr, metricMux)
}
