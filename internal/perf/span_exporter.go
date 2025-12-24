package perf

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/sdk/trace"
)

type spanExporter struct {
	mu    sync.Mutex
	spans []trace.ReadOnlySpan
}

func newSpanExporter() *spanExporter {
	return &spanExporter{
		spans: make([]trace.ReadOnlySpan, 0),
	}
}

func (exporter *spanExporter) ExportSpans(_ context.Context, spans []trace.ReadOnlySpan) error {
	exporter.mu.Lock()
	defer exporter.mu.Unlock()
	exporter.spans = append(exporter.spans, spans...)
	return nil
}

func (exporter *spanExporter) Shutdown(context.Context) error {
	return nil
}

func (exporter *spanExporter) Reset() {
	exporter.mu.Lock()
	defer exporter.mu.Unlock()
	exporter.spans = exporter.spans[:0]
}

func (exporter *spanExporter) Snapshot() []trace.ReadOnlySpan {
	exporter.mu.Lock()
	defer exporter.mu.Unlock()

	out := make([]trace.ReadOnlySpan, len(exporter.spans))
	copy(out, exporter.spans)
	return out
}
