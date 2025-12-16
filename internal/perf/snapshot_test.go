package perf

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestMustGetSpans_ReturnsNilWhenDisabled(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: false}))

	assert.Nil(t, MustGetSpans())
}

func TestAttributesFromStrings(t *testing.T) {
	attrs := AttributesFromStrings(map[string]string{
		"a": "1",
		"b": "2",
	})
	assert.Len(t, attrs, 2)

	got := map[string]string{}
	for _, kv := range attrs {
		got[string(kv.Key)] = kv.Value.AsString()
	}
	assert.Equal(t, map[string]string{"a": "1", "b": "2"}, got)
}

func TestAttributesFromStrings_EmptyReturnsNil(t *testing.T) {
	assert.Nil(t, AttributesFromStrings(map[string]string{}))
}

func TestMustGetSpans_ReturnsSpansWhenEnabled(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	_, span := StartSpan(context.Background(), "x")
	span.End()

	spans := MustGetSpans()
	assert.NotEmpty(t, spans)
}

func TestFindSpanByName_ReturnsFalseWhenMissing(t *testing.T) {
	s, ok := FindSpanByName([]SpanSnapshot{}, "x")
	assert.False(t, ok)
	assert.Equal(t, SpanSnapshot{}, s)
}

func TestGetSpans_SkipsInvalidLinks(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	_, span := StartSpan(context.Background(), "invalid-link", WithLinks(oteltrace.Link{}))
	span.End()

	spans, err := GetSpans()
	assert.NoError(t, err)

	s, ok := FindSpanByName(spans, "invalid-link")
	assert.True(t, ok)
	assert.Empty(t, s.Links)
}

func TestSnapshotLinks_SkipsInvalidSpanContexts(t *testing.T) {
	traceID := oteltrace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	spanID := oteltrace.SpanID{8, 7, 6, 5, 4, 3, 2, 1}
	valid := trace.Link{
		SpanContext: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: oteltrace.FlagsSampled,
		}),
	}

	links := snapshotLinks([]trace.Link{{}, valid})
	assert.Len(t, links, 1)
	assert.Equal(t, traceID.String(), links[0].TraceID)
	assert.Equal(t, spanID.String(), links[0].SpanID)
	assert.Nil(t, links[0].Attributes)
}

func TestSnapshotLinks_EmptyAndAllInvalidReturnNil(t *testing.T) {
	assert.Nil(t, snapshotLinks(nil))
	assert.Nil(t, snapshotLinks([]trace.Link{}))
	assert.Nil(t, snapshotLinks([]trace.Link{{}}))
}

func TestSpanExporterReset(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	_, span := StartSpan(context.Background(), "x")
	span.End()

	spans, err := SnapshotSpans()
	assert.NoError(t, err)
	assert.NotEmpty(t, spans)

	globalMu.Lock()
	exp := globalExp
	globalMu.Unlock()

	assert.NotNil(t, exp)
	exp.Reset()

	spans, err = SnapshotSpans()
	assert.NoError(t, err)
	assert.Empty(t, spans)
}
