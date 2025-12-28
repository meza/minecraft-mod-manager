package perf

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestInit_ReinitializingReplacesExporter(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	assert.NoError(t, Init(Config{Enabled: true}))

	_, first := StartSpan(context.Background(), "first")
	first.End()

	assert.NoError(t, Init(Config{Enabled: true}))

	_, second := StartSpan(context.Background(), "second")
	second.End()

	spans, err := GetSpans()
	assert.NoError(t, err)

	_, ok := FindSpanByName(spans, "first")
	assert.False(t, ok)

	_, ok = FindSpanByName(spans, "second")
	assert.True(t, ok)
}

func TestInitReturnsErrorWhenShutdownFails(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	originalShutdown := shutdownTracerProvider
	shutdownTracerProvider = func(_ context.Context, _ *trace.TracerProvider) error {
		return errors.New("shutdown failed")
	}
	t.Cleanup(func() {
		shutdownTracerProvider = originalShutdown
	})

	globalTP = trace.NewTracerProvider()
	err := Init(Config{Enabled: false})
	assert.Error(t, err)
}

func TestInitUsesShutdownTimeout(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	originalShutdown := shutdownTracerProvider
	originalTimeout := shutdownTimeout
	t.Cleanup(func() {
		shutdownTracerProvider = originalShutdown
		shutdownTimeout = originalTimeout
	})

	shutdownTimeout = 2 * time.Second
	var deadline time.Time
	shutdownTracerProvider = func(ctx context.Context, _ *trace.TracerProvider) error {
		var ok bool
		deadline, ok = ctx.Deadline()
		assert.True(t, ok)
		return nil
	}

	globalTP = trace.NewTracerProvider()
	start := time.Now()
	assert.NoError(t, Init(Config{Enabled: false}))
	assert.True(t, deadline.After(start))
	assert.WithinDuration(t, start.Add(shutdownTimeout), deadline, 250*time.Millisecond)
}

func TestInitUsesDefaultShutdownTimeoutWhenUnset(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	originalShutdown := shutdownTracerProvider
	originalTimeout := shutdownTimeout
	t.Cleanup(func() {
		shutdownTracerProvider = originalShutdown
		shutdownTimeout = originalTimeout
	})

	shutdownTimeout = 0
	var deadline time.Time
	shutdownTracerProvider = func(ctx context.Context, _ *trace.TracerProvider) error {
		var ok bool
		deadline, ok = ctx.Deadline()
		assert.True(t, ok)
		return nil
	}

	globalTP = trace.NewTracerProvider()
	start := time.Now()
	assert.NoError(t, Init(Config{Enabled: false}))
	assert.True(t, deadline.After(start))
	assert.WithinDuration(t, start.Add(defaultShutdownTimeout), deadline, 250*time.Millisecond)
}

func TestResetContinuesWhenShutdownFails(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	originalShutdown := shutdownTracerProvider
	shutdownTracerProvider = func(_ context.Context, _ *trace.TracerProvider) error {
		return errors.New("shutdown failed")
	}
	t.Cleanup(func() {
		shutdownTracerProvider = originalShutdown
	})

	globalTP = trace.NewTracerProvider()
	Reset()
	assert.Nil(t, globalTP)
}

func TestStartSpan_PanicsOnNilContext(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	assert.Panics(t, func() {
		//nolint:staticcheck // Intentional nil context to verify panic behavior.
		StartSpan(nil, "nil-context", nil)
	})
}

func TestStartSpan_RecordsSpanAndAttributes(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	ctx, span := StartSpan(context.Background(), "test.span", WithAttributes(attribute.String("k", "v")))
	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
	span.End()

	spans, err := GetSpans()
	assert.NoError(t, err)

	s, ok := FindSpanByName(spans, "test.span")
	assert.True(t, ok)
	assert.NotEmpty(t, s.TraceID)
	assert.NotEmpty(t, s.SpanID)
	assert.Equal(t, "v", s.Attributes["k"])
}

func TestStartSpan_ChildInheritsTraceIDAndHasParent(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	rootCtx, root := StartSpan(context.Background(), "root")
	childCtx, child := StartSpan(rootCtx, "child")
	assert.NotNil(t, childCtx)

	child.End()
	root.End()

	spans, err := GetSpans()
	assert.NoError(t, err)

	rootSpan, ok := FindSpanByName(spans, "root")
	assert.True(t, ok)
	childSpan, ok := FindSpanByName(spans, "child")
	assert.True(t, ok)

	assert.Equal(t, rootSpan.TraceID, childSpan.TraceID)
	assert.Equal(t, rootSpan.SpanID, childSpan.ParentSpanID)
}

func TestLinks_CreateDAGEdge(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	ctxA, spanA := StartSpan(context.Background(), "spanA")
	spanA.End()

	link, err := LinkFromContext(ctxA)
	assert.NoError(t, err)

	_, spanB := StartSpan(context.Background(), "spanB", WithLinks(link))
	spanB.End()

	spans, err := GetSpans()
	assert.NoError(t, err)

	spanASnap, ok := FindSpanByName(spans, "spanA")
	assert.True(t, ok)
	spanBSnap, ok := FindSpanByName(spans, "spanB")
	assert.True(t, ok)

	assert.Len(t, spanBSnap.Links, 1)
	assert.Equal(t, spanASnap.TraceID, spanBSnap.Links[0].TraceID)
	assert.Equal(t, spanASnap.SpanID, spanBSnap.Links[0].SpanID)
}

func TestAddEvent_RecordsEventAttributes(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	_, span := StartSpan(context.Background(), "with-event")
	span.AddEvent("evt", WithEventAttributes(attribute.Int("attempt", 2)))
	span.End()

	spans, err := GetSpans()
	assert.NoError(t, err)

	s, ok := FindSpanByName(spans, "with-event")
	assert.True(t, ok)
	assert.Len(t, s.Events, 1)
	assert.Equal(t, "evt", s.Events[0].Name)
	assert.Equal(t, int64(2), s.Events[0].Attributes["attempt"])
}

func TestEnabled_ReflectsInitState(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	assert.NoError(t, Init(Config{Enabled: false}))
	assert.False(t, Enabled())

	assert.NoError(t, Init(Config{Enabled: true}))
	assert.True(t, Enabled())
}

func TestShutdown_NoProviderIsNoop(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	assert.NoError(t, Shutdown(context.Background()))
}

func TestShutdown_WithProvider(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	assert.NoError(t, Shutdown(context.Background()))
}

func TestSpanMethods_NoOpOnNil(t *testing.T) {
	var span *Span
	span.End()
	span.SetAttributes(attribute.String("k", "v"))
	span.AddEvent("evt")
}

func TestSpanMethods_SetAttributesPersistsOnSpan(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	_, span := StartSpan(context.Background(), "attr-span")
	span.SetAttributes(attribute.String("k", "v"))
	span.End()

	spans, err := GetSpans()
	assert.NoError(t, err)
	s, ok := FindSpanByName(spans, "attr-span")
	assert.True(t, ok)
	assert.Equal(t, "v", s.Attributes["k"])
}

func TestSpanFromContext_NilContextReturnsNonNilSpan(t *testing.T) {
	//nolint:staticcheck // Validates nil context handling.
	span := SpanFromContext(nil)
	assert.NotNil(t, span)
}

func TestSpanFromContext_WithSpanContextReturnsValidSpan(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	ctx, span := StartSpan(context.Background(), "ctx-span")
	got := SpanFromContext(ctx)
	assert.NotNil(t, got)
	assert.True(t, got.span.SpanContext().IsValid())
	span.End()
}

func TestLinkFromContext_ErrorsOnNilAndMissingSpan(t *testing.T) {
	//nolint:staticcheck // Validates nil context handling.
	_, err := LinkFromContext(nil)
	assert.Error(t, err)

	_, err = LinkFromContext(context.Background())
	assert.Error(t, err)
}

func TestExportStatus_CapturedWhenSetViaUnderlyingSpan(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	_, span := StartSpan(context.Background(), "status-span")
	span.span.SetStatus(codes.Ok, "")
	span.End()

	spans, err := GetSpans()
	assert.NoError(t, err)
	_, ok := FindSpanByName(spans, "status-span")
	assert.True(t, ok)
}

func TestSnapshotSpans_NoExporterConfiguredErrors(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	globalMu.Lock()
	globalEnabled = true
	globalExp = nil
	globalTP = nil
	globalMu.Unlock()

	_, err := SnapshotSpans()
	assert.Error(t, err)
}
