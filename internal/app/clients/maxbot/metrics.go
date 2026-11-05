package maxbot

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"stg/pkg/telemetry"
)

var (
	metricRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "maxbot_requests_total",
			Help: "Total number of maxbot requests.",
		},
		[]string{"method", telemetry.ErrLabel},
	)

	metricRequestsDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "maxbot_requests_duration_seconds",
			Help:    "maxbot requests latencies in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)
)
