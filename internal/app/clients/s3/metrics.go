package s3

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"stg/pkg/telemetry"
)

var (
	metricRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "s3_requests_total",
			Help: "Total number of S3 requests",
		},
		[]string{"action", telemetry.ErrLabel},
	)
	metricRequestsDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "s3_request_duration_seconds",
			Help:    "Duration of S3 requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"action"},
	)
)
