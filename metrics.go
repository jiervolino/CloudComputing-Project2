package main

import (
    "net/http"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // Count total weather API requests.
    weatherRequests = promauto.NewCounter(
        prometheus.CounterOpts{
            Name: "weather_requests_total",
            Help: "Total number of weather API requests.",
        },
    )

    // Measure how long weather requests take.
    weatherRequestDuration = promauto.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "weather_request_duration_seconds",
            Help:    "Duration of weather API requests in seconds.",
            Buckets: prometheus.DefBuckets,
        },
    )

    // Track currently active weather requests.
    weatherRequestsInProgress = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "weather_requests_in_progress",
            Help: "Number of weather requests currently being processed.",
        },
    )
)

// Record metrics for each weather API request.
func metricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        weatherRequests.Inc()
        weatherRequestsInProgress.Inc()

        defer func() {
            weatherRequestDuration.Observe(time.Since(start).Seconds())
            weatherRequestsInProgress.Dec()
        }()

        next(w, r)
    }
}
