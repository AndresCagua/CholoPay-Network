package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

type metrics struct {
	startedAt    time.Time
	httpRequests atomic.Uint64
}

func newMetrics() *metrics { return &metrics{startedAt: time.Now()} }

func (m *metrics) handler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "# HELP cholonet_process_info Process information for the local synthetic service.\n# TYPE cholonet_process_info gauge\ncholonet_process_info{service=\"network\"} 1\n")
	fmt.Fprintf(w, "# HELP cholonet_process_uptime_seconds Process uptime in seconds.\n# TYPE cholonet_process_uptime_seconds gauge\ncholonet_process_uptime_seconds %.0f\n", time.Since(m.startedAt).Seconds())
	fmt.Fprintf(w, "# HELP cholonet_http_requests_total HTTP requests observed by the local service.\n# TYPE cholonet_http_requests_total counter\ncholonet_http_requests_total{service=\"network\"} %d\n", m.httpRequests.Load())
	fmt.Fprint(w, "# HELP cholonet_synthetic_acceptance_info Scope marker for local synthetic acceptance only.\n# TYPE cholonet_synthetic_acceptance_info gauge\ncholonet_synthetic_acceptance_info{scope=\"local\"} 1\n")
}

func (m *metrics) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.httpRequests.Add(1)
		next.ServeHTTP(w, r)
	})
}
