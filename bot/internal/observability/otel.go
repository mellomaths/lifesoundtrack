// Package observability configures OpenTelemetry and structured logging primitives.
package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"
)

// TracerName is the OpenTelemetry tracer scope name for this service.
const TracerName = "github.com/mellomaths/lifesoundtrack/bot"

// InitTracing installs a tracer provider (noop until an exporter-backed SDK is added).
func InitTracing() {
	otel.SetTracerProvider(noop.NewTracerProvider())
}
