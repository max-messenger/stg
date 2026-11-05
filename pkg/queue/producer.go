package queue

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"stg/pkg/connections/rediscli"
	"stg/pkg/telemetry"
)

// Send publishes the payload without serialization or automatic stream trimming.
func (p *Queue) Send(ctx context.Context, stream string, payload []byte) error {
	_, err := p.Publish(ctx, stream, payload)

	return err
}

// Publish publishes the payload and returns its Redis stream ID.
func (p *Queue) Publish(ctx context.Context, stream string, payload []byte) (string, error) {
	if stream == "" {
		return "", fmt.Errorf("stream is required")
	}
	ctx, span := otel.Tracer("queue").Start(ctx, "queue.publish",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "redis"),
			attribute.String("messaging.destination.name", stream),
			attribute.String("messaging.operation.name", "publish"),
			attribute.String("messaging.operation.type", "send"),
			attribute.Int("messaging.message.body.size", len(payload)),
		),
	)
	defer span.End()
	values := streamCarrier{"payload": string(payload)}
	otel.GetTextMapPropagator().Inject(ctx, values)
	var id string
	err := p.redis.Call(ctx, "queue_publish", func(ctx context.Context, cli redis.UniversalClient, kf rediscli.KeyFormatter) error {
		var err error
		id, err = cli.XAdd(ctx, &redis.XAddArgs{
			Stream: streamKey(kf, p.config.Redis.Prefix, stream),
			Values: map[string]any(values),
		}).Result()

		return err
	})
	publishedTotal.WithLabelValues(stream, telemetry.ErrLabelValue(err)).Inc()
	if err != nil {
		recordSpanError(span, err)

		return "", fmt.Errorf("publish to %s: %w", stream, err)
	}
	span.SetAttributes(attribute.String("messaging.message.id", id))
	p.logger.Debug("message published", zap.String("stream", stream), zap.String("id", id))

	return id, nil
}
