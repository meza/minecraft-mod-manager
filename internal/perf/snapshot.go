package perf

import (
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
)

type SpanSnapshot struct {
	Name         string
	TraceID      string
	SpanID       string
	ParentSpanID string
	StartTime    time.Time
	EndTime      time.Time
	Attributes   map[string]interface{}
	Events       []EventSnapshot
	Links        []LinkSnapshot
}

type EventSnapshot struct {
	Name       string
	Timestamp  time.Time
	Attributes map[string]interface{}
}

type LinkSnapshot struct {
	TraceID    string
	SpanID     string
	Attributes map[string]interface{}
}

func GetSpans() ([]SpanSnapshot, error) {
	spans, err := SnapshotSpans()
	if err != nil {
		return nil, err
	}

	out := make([]SpanSnapshot, 0, len(spans))
	for _, span := range spans {
		out = append(out, snapshotSpan(span))
	}
	return out, nil
}

func MustGetSpans() []SpanSnapshot {
	spans, err := GetSpans()
	if err != nil {
		return nil
	}
	return spans
}

func FindSpanByName(spans []SpanSnapshot, name string) (SpanSnapshot, bool) {
	for _, span := range spans {
		if span.Name == name {
			return span, true
		}
	}
	return SpanSnapshot{}, false
}

func snapshotSpan(span trace.ReadOnlySpan) SpanSnapshot {
	sc := span.SpanContext()
	psc := span.Parent()

	out := SpanSnapshot{
		Name:      span.Name(),
		TraceID:   sc.TraceID().String(),
		SpanID:    sc.SpanID().String(),
		StartTime: span.StartTime(),
		EndTime:   span.EndTime(),
		Attributes: func() map[string]interface{} {
			return attributesToMap(span.Attributes())
		}(),
	}
	if psc.IsValid() {
		out.ParentSpanID = psc.SpanID().String()
	}

	evs := span.Events()
	if len(evs) > 0 {
		out.Events = make([]EventSnapshot, 0, len(evs))
		for _, e := range evs {
			out.Events = append(out.Events, EventSnapshot{
				Name:       e.Name,
				Timestamp:  e.Time,
				Attributes: attributesToMap(e.Attributes),
			})
		}
	}

	links := span.Links()
	if len(links) > 0 {
		out.Links = snapshotLinks(links)
	}

	return out
}

func snapshotLinks(links []trace.Link) []LinkSnapshot {
	if len(links) == 0 {
		return nil
	}

	out := make([]LinkSnapshot, 0, len(links))
	for _, link := range links {
		if !link.SpanContext.IsValid() {
			continue
		}
		out = append(out, LinkSnapshot{
			TraceID:    link.SpanContext.TraceID().String(),
			SpanID:     link.SpanContext.SpanID().String(),
			Attributes: attributesToMap(link.Attributes),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func AttributesFromStrings(attrs map[string]string) []attribute.KeyValue {
	if len(attrs) == 0 {
		return nil
	}
	out := make([]attribute.KeyValue, 0, len(attrs))
	for k, v := range attrs {
		out = append(out, attribute.String(k, v))
	}
	return out
}
