package queue

import (
	"context"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"stg/pkg/connections/rediscli"
	"stg/pkg/telemetry"
)

// Check ownership and ACK/renew atomically. A late callback must not ACK another consumer's delivery.
var ownershipScript = redis.NewScript(`
local pending = redis.call('XPENDING', KEYS[1], ARGV[1], ARGV[3], ARGV[3], 1)
if #pending == 0 or pending[1][2] ~= ARGV[2] then return 0 end
if ARGV[4] == 'ack' then
    return redis.call('XACK', KEYS[1], ARGV[1], ARGV[3])
end
local ids = redis.call('XCLAIM', KEYS[1], ARGV[1], ARGV[2], 0, ARGV[3], 'JUSTID')
return #ids
`)

func (cg *consumer) handle(ctx context.Context, stream, consumer string, entry redis.XMessage, callback ConsumerCallback) {
	if ctx.Err() != nil {
		return
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, streamCarrier(entry.Values))
	ctx, span := otel.Tracer("queue").Start(ctx, "queue.process",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "redis"),
			attribute.String("messaging.destination.name", stream),
			attribute.String("messaging.operation.name", "process"),
			attribute.String("messaging.operation.type", "process"),
			attribute.String("messaging.consumer.group.name", cg.group),
			attribute.String("messaging.client.id", consumer),
			attribute.String("messaging.message.id", entry.ID),
			attribute.String("queue.ack_mode", string(cg.config.AckMode)),
		),
	)
	defer span.End()
	msg, err := decodeMessage(stream, entry)
	if err != nil {
		recordSpanError(span, err)
		cg.report("decode", stream, err)

		return
	}
	if cg.config.AckMode == AckBeforeHandle {
		if err = cg.ownership(ctx, stream, consumer, entry.ID, "ack"); err != nil {
			recordSpanError(span, err)
			cg.report("ack", stream, err)

			return
		}
	}
	err, contextErr := cg.runCallback(ctx, consumer, msg, callback)
	recordSpanError(span, err)
	if err != nil {
		cg.report("handler", stream, fmt.Errorf("message %s: %w", entry.ID, err))
	}
	if ctx.Err() != nil || contextErr != nil {
		return
	}
	if cg.shouldAck(err) {
		if err = cg.ownership(ctx, stream, consumer, entry.ID, "ack"); err != nil {
			recordSpanError(span, err)
			cg.report("ack", stream, err)
		}
	}
}

func (cg *consumer) shouldAck(err error) bool {
	return cg.config.AckMode == AckAfterHandle || (cg.config.AckMode == AckOnSuccess && err == nil)
}

func (cg *consumer) runCallback(ctx context.Context, name string, msg Message, callback ConsumerCallback) (error, error) {
	handlerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if cg.config.HandlerTimeout > 0 {
		var timeoutCancel context.CancelFunc
		handlerCtx, timeoutCancel = context.WithTimeout(handlerCtx, cg.config.HandlerTimeout)
		defer timeoutCancel()
	}
	done := make(chan struct{})
	if cg.config.AckMode != AckBeforeHandle {
		go func() {
			defer close(done)
			cg.heartbeat(handlerCtx, cancel, msg.Stream, name, msg.ID)
		}()
	} else {
		close(done)
	}
	start := time.Now()
	messageLag.WithLabelValues(msg.Stream, cg.group).Observe(time.Since(msg.Timestamp).Seconds())
	err := cg.invoke(handlerCtx, msg, callback)
	contextErr := handlerCtx.Err()
	cancel()
	<-done
	if err == nil {
		err = contextErr
	}
	processedTotal.WithLabelValues(msg.Stream, cg.group, telemetry.ErrLabelValue(err)).Inc()
	handlerDuration.WithLabelValues(msg.Stream, cg.group).Observe(time.Since(start).Seconds())

	return err, contextErr
}

func (cg *consumer) invoke(ctx context.Context, msg Message, callback ConsumerCallback) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("callback panic: %v", recovered)
			cg.logger.Error("queue callback panic", zap.String("id", msg.ID), zap.ByteString("stack", debug.Stack()))
		}
	}()

	return callback(ctx, msg)
}

func (cg *consumer) heartbeat(ctx context.Context, cancel context.CancelFunc, stream, consumer, id string) {
	ticker := time.NewTicker(cg.config.ReclaimIdle / 3)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := cg.ownership(ctx, stream, consumer, id, "renew"); err != nil {
				if ctx.Err() == nil {
					recordSpanError(trace.SpanFromContext(ctx), err)
					cg.report("renew", stream, err)
				}
				cancel()

				return
			}
		}
	}
}

func (cg *consumer) ownership(ctx context.Context, stream, consumer, id, operation string) error {
	return cg.redis.Call(ctx, "queue_"+operation, func(ctx context.Context, cli redis.UniversalClient, kf rediscli.KeyFormatter) error {
		key := streamKey(kf, cg.prefix, stream)
		count, err := ownershipScript.Run(ctx, cli, []string{key}, cg.group, consumer, id, operation).Int64()
		if err != nil {
			return err
		}
		if count == 0 {
			return ErrOwnershipLost
		}

		return nil
	})
}

func decodeMessage(stream string, entry redis.XMessage) (Message, error) {
	var payload []byte
	switch value := entry.Values["payload"].(type) {
	case string:
		payload = []byte(value)
	case []byte:
		payload = append([]byte(nil), value...)
	default:
		return Message{}, fmt.Errorf("message %s has no binary payload", entry.ID)
	}
	ms, _, _ := strings.Cut(entry.ID, "-")
	timestamp, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return Message{}, fmt.Errorf("invalid stream ID %q: %w", entry.ID, err)
	}

	return Message{ID: entry.ID, Stream: stream, Payload: payload, Timestamp: time.UnixMilli(timestamp)}, nil
}
