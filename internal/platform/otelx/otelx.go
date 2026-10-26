// Package otelx is the tracing port. Phase 0 ships a no-op provider.
package otelx

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
)

func Init(cfg config.OTel) (trace.TracerProvider, func(context.Context) error) {
	provider := noop.NewTracerProvider()
	otel.SetTracerProvider(provider)
	_ = cfg
	return provider, func(context.Context) error { return nil }
}

func Tracer() trace.Tracer {
	return otel.Tracer("treatment-plus")
}
