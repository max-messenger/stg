package queue

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// streamCarrier stores propagation fields alongside the payload in a Redis stream entry.
type streamCarrier map[string]any

func (c streamCarrier) Get(key string) string {
	value, ok := c[key].(string)
	if !ok {
		return ""
	}

	return value
}

func (c streamCarrier) Set(key, value string) { c[key] = value }

func (c streamCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}

	return keys
}

func recordSpanError(span trace.Span, err error) {
	if err == nil || errors.Is(err, context.Canceled) {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
