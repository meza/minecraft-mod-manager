package perf

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type Config struct {
	Enabled bool
}

var (
	globalMu      sync.Mutex
	globalEnabled bool
	globalTP      *trace.TracerProvider
	globalExp     *spanExporter
)

func Init(cfg Config) error {
	globalMu.Lock()
	defer globalMu.Unlock()

	if globalTP != nil {
		_ = globalTP.Shutdown(context.Background())
		globalTP = nil
	}

	globalEnabled = cfg.Enabled
	if !cfg.Enabled {
		globalExp = nil
		otel.SetTracerProvider(oteltrace.NewNoopTracerProvider())
		return nil
	}

	globalExp = newSpanExporter()
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithSpanProcessor(trace.NewSimpleSpanProcessor(globalExp)),
	)
	globalTP = tp
	otel.SetTracerProvider(tp)
	return nil
}

func Shutdown(ctx context.Context) error {
	globalMu.Lock()
	tp := globalTP
	globalMu.Unlock()

	if tp == nil {
		return nil
	}
	return tp.Shutdown(ctx)
}

func Reset() {
	globalMu.Lock()
	defer globalMu.Unlock()

	globalEnabled = false
	globalExp = nil
	if globalTP != nil {
		_ = globalTP.Shutdown(context.Background())
		globalTP = nil
	}
	otel.SetTracerProvider(oteltrace.NewNoopTracerProvider())
}

func Enabled() bool {
	globalMu.Lock()
	defer globalMu.Unlock()
	return globalEnabled
}

type SpanOption func(*spanOptions)

type spanOptions struct {
	attributes []attribute.KeyValue
	links      []oteltrace.Link
}

func WithAttributes(attrs ...attribute.KeyValue) SpanOption {
	return func(o *spanOptions) {
		o.attributes = append(o.attributes, attrs...)
	}
}

func WithLinks(links ...oteltrace.Link) SpanOption {
	return func(o *spanOptions) {
		o.links = append(o.links, links...)
	}
}

func StartSpan(ctx context.Context, name string, opts ...SpanOption) (context.Context, *Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	spanOpts := spanOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&spanOpts)
		}
	}

	startOpts := make([]oteltrace.SpanStartOption, 0, 3)
	if len(spanOpts.attributes) > 0 {
		startOpts = append(startOpts, oteltrace.WithAttributes(spanOpts.attributes...))
	}
	if len(spanOpts.links) > 0 {
		startOpts = append(startOpts, oteltrace.WithLinks(spanOpts.links...))
	}

	tracer := otel.Tracer("github.com/meza/minecraft-mod-manager/internal/perf")
	ctx, span := tracer.Start(ctx, name, startOpts...)
	return ctx, &Span{span: span}
}

type Span struct {
	span oteltrace.Span
}

func (s *Span) End() {
	if s == nil || s.span == nil {
		return
	}
	s.span.End()
}

func (s *Span) SetAttributes(attrs ...attribute.KeyValue) {
	if s == nil || s.span == nil {
		return
	}
	s.span.SetAttributes(attrs...)
}

type EventOption func(*eventOptions)

type eventOptions struct {
	attributes []attribute.KeyValue
}

func WithEventAttributes(attrs ...attribute.KeyValue) EventOption {
	return func(o *eventOptions) {
		o.attributes = append(o.attributes, attrs...)
	}
}

func (s *Span) AddEvent(name string, opts ...EventOption) {
	if s == nil || s.span == nil {
		return
	}

	o := eventOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}

	eventOpts := make([]oteltrace.EventOption, 0, 1)
	if len(o.attributes) > 0 {
		eventOpts = append(eventOpts, oteltrace.WithAttributes(o.attributes...))
	}

	s.span.AddEvent(name, eventOpts...)
}

func SpanFromContext(ctx context.Context) *Span {
	if ctx == nil {
		return &Span{span: oteltrace.SpanFromContext(context.Background())}
	}
	return &Span{span: oteltrace.SpanFromContext(ctx)}
}

func LinkFromContext(ctx context.Context) (oteltrace.Link, error) {
	if ctx == nil {
		return oteltrace.Link{}, errors.New("nil context")
	}

	sc := oteltrace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return oteltrace.Link{}, errors.New("no span in context")
	}
	return oteltrace.Link{SpanContext: sc}, nil
}
