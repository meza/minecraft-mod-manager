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

func (e *spanExporter) ExportSpans(_ context.Context, spans []trace.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spans = append(e.spans, spans...)
	return nil
}

func (e *spanExporter) Shutdown(context.Context) error {
	return nil
}

func (e *spanExporter) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.spans = e.spans[:0]
}

func (e *spanExporter) Snapshot() []trace.ReadOnlySpan {
	e.mu.Lock()
	defer e.mu.Unlock()

	out := make([]trace.ReadOnlySpan, len(e.spans))
	copy(out, e.spans)
	return out
}
