package queue

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	labelStream = "stream"
	labelGroup  = "group"
)

var (
	publishedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "redis_queue_published_total", Help: "Stream publication attempts.",
	}, []string{labelStream, "error"})
	processedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "redis_queue_processed_total", Help: "Callback attempts by outcome.",
	}, []string{labelStream, labelGroup, "error"})
	consumerErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "redis_queue_errors_total", Help: "Consumer errors by operation.",
	}, []string{labelStream, labelGroup, "operation"})
	reclaimedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "redis_queue_reclaimed_total", Help: "Pending messages reclaimed for redelivery.",
	}, []string{labelStream, labelGroup})
	handlerDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "redis_queue_handler_duration_seconds", Help: "Callback execution time.",
		Buckets: prometheus.DefBuckets,
	}, []string{labelStream, labelGroup})
	messageLag = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "redis_queue_message_lag_seconds", Help: "Message age when the callback starts.",
		Buckets: prometheus.DefBuckets,
	}, []string{labelStream, labelGroup})
)
