package perf

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestExportToFile_WritesJSONAndNormalizesPaths(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	tempDir := t.TempDir()
	baseDir := filepath.Join(tempDir, "cfg")
	outDir := filepath.Join(tempDir, "out")

	absConfig := filepath.Join(baseDir, "modlist.json")
	absJar := filepath.Join(baseDir, "mods", "a.jar")

	ctx, span := StartSpan(context.Background(), "example",
		WithAttributes(
			attribute.String("config_path", absConfig),
			attribute.String("path", absJar),
		),
	)
	_, child := StartSpan(ctx, "child", WithAttributes(attribute.Int("status", 200)))
	child.End()
	span.End()

	written, err := ExportToFile(outDir, baseDir)
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(outDir, defaultExportFilename), written)

	raw, err := os.ReadFile(written) // #nosec G304 -- test reads temp file path.
	assert.NoError(t, err)

	var decoded []*ExportSpan
	assert.NoError(t, json.Unmarshal(raw, &decoded))
	all := flattenExportSpanTree(decoded)
	assert.GreaterOrEqual(t, len(all), 2)

	var foundExample *ExportSpan
	for _, span := range all {
		if span.Name == "example" {
			foundExample = span
			break
		}
	}
	assert.NotNil(t, foundExample)
	assert.Equal(t, "modlist.json", foundExample.Attributes["config_path"])
	assert.Equal(t, "mods/a.jar", foundExample.Attributes["path"])
	assert.NotEmpty(t, foundExample.TraceID)
	assert.NotEmpty(t, foundExample.SpanID)
	assert.GreaterOrEqual(t, foundExample.DurationNS, int64(0))
}

func TestExportToFile_ReturnsErrorWhenPerfDisabled(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: false}))

	_, err := ExportToFile(t.TempDir(), t.TempDir())
	assert.Error(t, err)
}

func TestExportToFile_DefaultOutDirWritesToWorkingDirectory(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	assert.NoError(t, err)
	assert.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() { assert.NoError(t, os.Chdir(oldWD)) })

	_, span := StartSpan(context.Background(), "wd-span")
	span.End()

	written, err := ExportToFile("", tempDir)
	assert.NoError(t, err)
	assert.Equal(t, defaultExportFilename, written)
	_, err = os.Stat(written)
	assert.NoError(t, err)
}

func TestExportToFile_ReturnsErrorWhenOutDirIsFile(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "out-file")
	assert.NoError(t, os.WriteFile(outDir, []byte("x"), 0644))

	_, span := StartSpan(context.Background(), "span")
	span.End()

	_, err := ExportToFile(outDir, tempDir)
	assert.Error(t, err)
}

func TestExportToFile_ReturnsErrorWhenTargetPathIsDirectory(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	outDir := t.TempDir()
	assert.NoError(t, os.Mkdir(filepath.Join(outDir, defaultExportFilename), 0755))

	_, span := StartSpan(context.Background(), "span")
	span.End()

	_, err := ExportToFile(outDir, t.TempDir())
	assert.Error(t, err)
}

func TestGetExportTree_ReturnsHierarchicalTree(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	rootCtx, root := StartSpan(context.Background(), "root")
	_, child := StartSpan(rootCtx, "child")
	child.End()
	root.End()

	tree, err := GetExportTree("")
	assert.NoError(t, err)
	assert.Len(t, tree, 1)
	assert.Equal(t, "root", tree[0].Name)
	assert.Len(t, tree[0].Children, 1)
	assert.Equal(t, "child", tree[0].Children[0].Name)
}

func TestExportToFile_ReturnsErrorWhenJSONMarshalFails(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	_, span := StartSpan(context.Background(), "bad-float", WithAttributes(attribute.Float64("nan", math.NaN())))
	span.End()

	_, err := ExportToFile(t.TempDir(), t.TempDir())
	assert.Error(t, err)
}

func TestExportToFile_ExportsEventsLinksStatusAndAttributeTypes(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	ctxA, spanA := StartSpan(context.Background(), "spanA", WithAttributes(
		attribute.Bool("b", true),
		attribute.Int64("i", 7),
		attribute.Float64("f", 1.25),
		attribute.String("s", "x"),
		attribute.StringSlice("ss", []string{"a", "b"}),
	))
	spanA.span.SetStatus(codes.Ok, "")
	spanA.End()

	link, err := LinkFromContext(ctxA)
	assert.NoError(t, err)

	_, spanB := StartSpan(context.Background(), "spanB", WithLinks(link))
	spanB.AddEvent("evt", WithEventAttributes(attribute.Int("attempt", 1)))
	spanB.span.SetStatus(codes.Error, "bad")
	spanB.End()

	outDir := t.TempDir()
	baseDir := t.TempDir()
	written, err := ExportToFile(outDir, baseDir)
	assert.NoError(t, err)

	raw, err := os.ReadFile(written) // #nosec G304 -- test reads temp file path.
	assert.NoError(t, err)

	var decoded []*ExportSpan
	assert.NoError(t, json.Unmarshal(raw, &decoded))
	all := flattenExportSpanTree(decoded)

	var foundB *ExportSpan
	for _, span := range all {
		if span.Name == "spanB" {
			foundB = span
			break
		}
	}
	assert.NotNil(t, foundB)
	assert.Len(t, foundB.Events, 1)
	assert.Equal(t, "evt", foundB.Events[0].Name)
	assert.EqualValues(t, 1, foundB.Events[0].Attributes["attempt"])
	assert.Len(t, foundB.Links, 1)
	assert.NotNil(t, foundB.Status)
	assert.Equal(t, "error", foundB.Status.Code)
	assert.Equal(t, "bad", foundB.Status.Description)
}

func TestExportToFile_ExportsEventWithNoAttributes(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	_, span := StartSpan(context.Background(), "eventless")
	span.AddEvent("evt")
	span.End()

	written, err := ExportToFile(t.TempDir(), t.TempDir())
	assert.NoError(t, err)

	raw, err := os.ReadFile(written) // #nosec G304 -- test reads temp file path.
	assert.NoError(t, err)

	var decoded []*ExportSpan
	assert.NoError(t, json.Unmarshal(raw, &decoded))
	all := flattenExportSpanTree(decoded)

	var found *ExportSpan
	for _, span := range all {
		if span.Name == "eventless" {
			found = span
			break
		}
	}
	assert.NotNil(t, found)
	assert.Len(t, found.Events, 1)
	assert.Nil(t, found.Events[0].Attributes)
}

func TestExportToFile_SkipsInvalidLinks(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	_, span := StartSpan(context.Background(), "invalid-link", WithLinks(oteltrace.Link{}))
	span.End()

	written, err := ExportToFile(t.TempDir(), t.TempDir())
	assert.NoError(t, err)

	raw, err := os.ReadFile(written) // #nosec G304 -- test reads temp file path.
	assert.NoError(t, err)

	var decoded []*ExportSpan
	assert.NoError(t, json.Unmarshal(raw, &decoded))
	all := flattenExportSpanTree(decoded)

	var found *ExportSpan
	for _, span := range all {
		if span.Name == "invalid-link" {
			found = span
			break
		}
	}
	assert.NotNil(t, found)
	assert.Nil(t, found.Links)
}

func TestExportLinks_SkipsInvalidSpanContexts(t *testing.T) {
	traceID := oteltrace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	spanID := oteltrace.SpanID{8, 7, 6, 5, 4, 3, 2, 1}
	valid := trace.Link{
		SpanContext: oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: oteltrace.FlagsSampled,
		}),
	}

	links := exportLinks([]trace.Link{{}, valid}, "")
	assert.Len(t, links, 1)
	assert.Equal(t, traceID.String(), links[0].TraceID)
	assert.Equal(t, spanID.String(), links[0].SpanID)
	assert.Nil(t, links[0].Attributes)
}

func TestExportSpanTree_EmptyReturnsNil(t *testing.T) {
	assert.Nil(t, exportSpanTree(nil, ""))
	assert.Nil(t, exportSpanTree([]trace.ReadOnlySpan{}, ""))
}

func TestAttributeValueToInterface_CoversAllTypes(t *testing.T) {
	assert.Equal(t, true, attributeValueToInterface(attribute.Bool("b", true).Value))
	assert.Equal(t, int64(7), attributeValueToInterface(attribute.Int64("i", 7).Value))
	assert.Equal(t, 1.25, attributeValueToInterface(attribute.Float64("f", 1.25).Value))
	assert.Equal(t, "x", attributeValueToInterface(attribute.String("s", "x").Value))
	assert.Equal(t, []bool{true, false}, attributeValueToInterface(attribute.BoolSlice("bs", []bool{true, false}).Value))
	assert.Equal(t, []int64{1, 2}, attributeValueToInterface(attribute.Int64Slice("is", []int64{1, 2}).Value))
	assert.Equal(t, []float64{1.0, 2.5}, attributeValueToInterface(attribute.Float64Slice("fs", []float64{1.0, 2.5}).Value))
	assert.Equal(t, []string{"a", "b"}, attributeValueToInterface(attribute.StringSlice("ss", []string{"a", "b"}).Value))
	assert.Equal(t, attribute.Value{}.Emit(), attributeValueToInterface(attribute.Value{}))
}

func TestNormalizeValue_CleansRelativePaths(t *testing.T) {
	assert.Equal(t, ".", trimLeadingDot("."))
	assert.Equal(t, "mods/a.jar", normalizeValue("path", "./mods/a.jar", "").(string))
	assert.Equal(t, "mods/a.jar", normalizeValue("path", "mods/a.jar", "/base").(string))
	assert.Equal(t, "modlist.json", normalizeValue("config_path", "./modlist.json", "").(string))
	assert.Equal(t, ".", normalizeValue("path", ".", "").(string))
	assert.Equal(t, "not-a-path", normalizeValue("url", "not-a-path", "/base").(string))
}

func TestLooksLikePathKey(t *testing.T) {
	assert.True(t, looksLikePathKey("path"))
	assert.True(t, looksLikePathKey("config_path"))
	assert.True(t, looksLikePathKey("outputPath"))
	assert.False(t, looksLikePathKey("url"))
}

func TestExportSpanTree_MissingParentBecomesRoot(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	rootCtx, root := StartSpan(context.Background(), "root")
	_, child := StartSpan(rootCtx, "child")
	child.End()
	root.End()

	spans, err := SnapshotSpans()
	assert.NoError(t, err)

	var onlyChild []trace.ReadOnlySpan
	for _, span := range spans {
		if span.Name() == "child" {
			onlyChild = append(onlyChild, span)
		}
	}
	assert.Len(t, onlyChild, 1)

	tree := exportSpanTree(onlyChild, "")
	assert.Len(t, tree, 1)
	assert.Equal(t, "child", tree[0].Name)
	assert.Nil(t, tree[0].Children)
}

func TestSortExportSpans_SortsRootsAndChildren(t *testing.T) {
	sortExportSpans(nil)
	sortExportSpans([]*ExportSpan{})

	now := time.Now()
	rootA := &ExportSpan{Name: "a", SpanID: "1", StartTime: now}
	rootB := &ExportSpan{Name: "b", SpanID: "2", StartTime: now.Add(time.Second)}

	childA2 := &ExportSpan{Name: "a2", SpanID: "c2", StartTime: now.Add(2 * time.Second)}
	childA1 := &ExportSpan{Name: "a1", SpanID: "c1", StartTime: now.Add(time.Second)}
	rootA.Children = []*ExportSpan{childA2, childA1}

	spans := []*ExportSpan{rootB, rootA}
	sortExportSpans(spans)

	assert.Equal(t, "a", spans[0].Name)
	assert.Equal(t, "b", spans[1].Name)
	assert.Equal(t, "a1", spans[0].Children[0].Name)
	assert.Equal(t, "a2", spans[0].Children[1].Name)
}

func TestSortExportSpans_TieBreakers(t *testing.T) {
	now := time.Now()

	byNameB := &ExportSpan{Name: "b", SpanID: "1", StartTime: now}
	byNameA := &ExportSpan{Name: "a", SpanID: "2", StartTime: now}

	spans := []*ExportSpan{byNameB, byNameA}
	sortExportSpans(spans)
	assert.Equal(t, "a", spans[0].Name)
	assert.Equal(t, "b", spans[1].Name)

	byID2 := &ExportSpan{Name: "same", SpanID: "2", StartTime: now}
	byID1 := &ExportSpan{Name: "same", SpanID: "1", StartTime: now}
	spans = []*ExportSpan{byID2, byID1}
	sortExportSpans(spans)
	assert.Equal(t, "1", spans[0].SpanID)
	assert.Equal(t, "2", spans[1].SpanID)
}

func TestExportSpanLess_OrdersByTimeThenNameThenID(t *testing.T) {
	now := time.Now()
	earlier := &ExportSpan{Name: "z", SpanID: "2", StartTime: now}
	later := &ExportSpan{Name: "a", SpanID: "1", StartTime: now.Add(time.Second)}
	assert.True(t, exportSpanLess(earlier, later))
	assert.False(t, exportSpanLess(later, earlier))

	sameTimeB := &ExportSpan{Name: "b", SpanID: "2", StartTime: now}
	sameTimeA := &ExportSpan{Name: "a", SpanID: "3", StartTime: now}
	assert.True(t, exportSpanLess(sameTimeA, sameTimeB))
	assert.False(t, exportSpanLess(sameTimeB, sameTimeA))

	sameName2 := &ExportSpan{Name: "same", SpanID: "2", StartTime: now}
	sameName1 := &ExportSpan{Name: "same", SpanID: "1", StartTime: now}
	assert.True(t, exportSpanLess(sameName1, sameName2))
	assert.False(t, exportSpanLess(sameName2, sameName1))
}

func flattenExportSpanTree(spans []*ExportSpan) []*ExportSpan {
	if len(spans) == 0 {
		return nil
	}

	out := make([]*ExportSpan, 0, len(spans))
	var walk func(*ExportSpan)
	walk = func(span *ExportSpan) {
		out = append(out, span)
		for _, child := range span.Children {
			walk(child)
		}
	}

	for _, span := range spans {
		walk(span)
	}

	return out
}
