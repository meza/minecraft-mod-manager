package perf

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
)

const defaultExportFilename = "mmm-perf.json"

type ExportSpan struct {
	Name         string                 `json:"name"`
	TraceID      string                 `json:"trace_id"`
	SpanID       string                 `json:"span_id"`
	ParentSpanID string                 `json:"parent_span_id,omitempty"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	DurationNS   int64                  `json:"duration_ns"`
	Attributes   map[string]interface{} `json:"attributes,omitempty"`
	Events       []ExportEvent          `json:"events,omitempty"`
	Links        []ExportLink           `json:"links,omitempty"`
	Status       *ExportStatus          `json:"status,omitempty"`
	Children     []*ExportSpan          `json:"children,omitempty"`
}

type ExportEvent struct {
	Name       string                 `json:"name"`
	Timestamp  time.Time              `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type ExportLink struct {
	TraceID    string                 `json:"trace_id"`
	SpanID     string                 `json:"span_id"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type ExportStatus struct {
	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
}

// ExportToFile writes all spans collected by the active tracer provider as JSON to
// <outDir>/mmm-perf.json. Any absolute filesystem paths in known attribute keys
// are rewritten to be relative to baseDir so output stays portable.
//
// This is intended to be used as a best-effort diagnostic artifact; callers
// should treat any returned error as non-fatal.
func ExportToFile(outDir string, baseDir string) (string, error) {
	if outDir == "" {
		outDir = "."
	}

	tree, err := GetExportTree(baseDir)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}

	path := filepath.Join(outDir, defaultExportFilename)
	data, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return "", err
	}

	return path, os.WriteFile(path, data, 0644)
}

// GetExportTree returns a hierarchical span tree suitable for exporting in
// `mmm-perf.json` and for attaching to telemetry events. The returned slice
// contains only root spans; child spans are nested under `children[]`.
func GetExportTree(baseDir string) ([]*ExportSpan, error) {
	spans, err := SnapshotSpans()
	if err != nil {
		return nil, err
	}
	return exportSpanTree(spans, baseDir), nil
}

func SnapshotSpans() ([]trace.ReadOnlySpan, error) {
	globalMu.Lock()
	exp := globalExp
	enabled := globalEnabled
	globalMu.Unlock()

	if !enabled {
		return nil, errors.New("perf disabled")
	}
	if exp == nil {
		return nil, errors.New("no span exporter configured")
	}
	return exp.Snapshot(), nil
}

func exportSpanTree(spans []trace.ReadOnlySpan, baseDir string) []*ExportSpan {
	if len(spans) == 0 {
		return nil
	}

	nodes := make(map[string]*ExportSpan, len(spans))
	for _, span := range spans {
		exported := exportSpan(span, baseDir)
		nodes[spanKey(exported.TraceID, exported.SpanID)] = &exported
	}

	roots := make([]*ExportSpan, 0, len(nodes))
	for _, node := range nodes {
		if node.ParentSpanID == "" {
			roots = append(roots, node)
			continue
		}

		parent, ok := nodes[spanKey(node.TraceID, node.ParentSpanID)]
		if !ok {
			roots = append(roots, node)
			continue
		}

		parent.Children = append(parent.Children, node)
	}

	sortExportSpans(roots)
	return roots
}

func exportSpan(span trace.ReadOnlySpan, baseDir string) ExportSpan {
	sc := span.SpanContext()
	psc := span.Parent()

	out := ExportSpan{
		Name:       span.Name(),
		TraceID:    sc.TraceID().String(),
		SpanID:     sc.SpanID().String(),
		StartTime:  span.StartTime(),
		EndTime:    span.EndTime(),
		DurationNS: span.EndTime().Sub(span.StartTime()).Nanoseconds(),
	}
	if psc.IsValid() {
		out.ParentSpanID = psc.SpanID().String()
	}

	out.Attributes = normalizeAttributes(attributesToMap(span.Attributes()), baseDir)
	out.Events = exportEvents(span.Events(), baseDir)
	out.Links = exportLinks(span.Links(), baseDir)
	out.Status = exportStatus(span.Status())

	if len(out.Attributes) == 0 {
		out.Attributes = nil
	}

	return out
}

func spanKey(traceID string, spanID string) string {
	return traceID + ":" + spanID
}

func sortExportSpans(spans []*ExportSpan) {
	if len(spans) == 0 {
		return
	}

	sort.Slice(spans, func(i, j int) bool {
		return exportSpanLess(spans[i], spans[j])
	})

	for i := range spans {
		sortExportSpans(spans[i].Children)
	}
}

func exportSpanLess(left *ExportSpan, right *ExportSpan) bool {
	if left.StartTime.Before(right.StartTime) {
		return true
	}
	if right.StartTime.Before(left.StartTime) {
		return false
	}
	if left.Name != right.Name {
		return left.Name < right.Name
	}
	return left.SpanID < right.SpanID
}

func exportStatus(status trace.Status) *ExportStatus {
	if status.Code == codes.Unset {
		return nil
	}
	code := "unset"
	switch status.Code {
	case codes.Ok:
		code = "ok"
	case codes.Error:
		code = "error"
	}
	return &ExportStatus{
		Code:        code,
		Description: status.Description,
	}
}

func exportEvents(events []trace.Event, baseDir string) []ExportEvent {
	if len(events) == 0 {
		return nil
	}

	out := make([]ExportEvent, 0, len(events))
	for _, e := range events {
		attrs := normalizeAttributes(attributesToMap(e.Attributes), baseDir)
		if len(attrs) == 0 {
			attrs = nil
		}
		out = append(out, ExportEvent{
			Name:       e.Name,
			Timestamp:  e.Time,
			Attributes: attrs,
		})
	}
	return out
}

func exportLinks(links []trace.Link, baseDir string) []ExportLink {
	if len(links) == 0 {
		return nil
	}

	out := make([]ExportLink, 0, len(links))
	for _, l := range links {
		sc := l.SpanContext
		if !sc.IsValid() {
			continue
		}
		attrs := normalizeAttributes(attributesToMap(l.Attributes), baseDir)
		if len(attrs) == 0 {
			attrs = nil
		}
		out = append(out, ExportLink{
			TraceID:    sc.TraceID().String(),
			SpanID:     sc.SpanID().String(),
			Attributes: attrs,
		})
	}
	return out
}

func attributesToMap(attrs []attribute.KeyValue) map[string]interface{} {
	if len(attrs) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(attrs))
	for _, kv := range attrs {
		out[string(kv.Key)] = attributeValueToInterface(kv.Value)
	}
	return out
}

func attributeValueToInterface(v attribute.Value) interface{} {
	switch v.Type() {
	case attribute.BOOL:
		return v.AsBool()
	case attribute.INT64:
		return v.AsInt64()
	case attribute.FLOAT64:
		return v.AsFloat64()
	case attribute.STRING:
		return v.AsString()
	case attribute.BOOLSLICE:
		return v.AsBoolSlice()
	case attribute.INT64SLICE:
		return v.AsInt64Slice()
	case attribute.FLOAT64SLICE:
		return v.AsFloat64Slice()
	case attribute.STRINGSLICE:
		return v.AsStringSlice()
	default:
		return v.Emit()
	}
}

func normalizeAttributes(attrs map[string]interface{}, baseDir string) map[string]interface{} {
	if len(attrs) == 0 {
		return attrs
	}

	normalized := make(map[string]interface{}, len(attrs))
	for key, value := range attrs {
		normalized[key] = normalizeValue(key, value, baseDir)
	}
	return normalized
}

func normalizeValue(key string, value interface{}, baseDir string) interface{} {
	stringValue, ok := value.(string)
	if !ok {
		return value
	}

	if key == "config_path" {
		if baseDir != "" && filepath.IsAbs(stringValue) {
			rel, err := filepath.Rel(baseDir, stringValue)
			if err == nil {
				return exportPath(rel)
			}
		}
		return exportPath(stringValue)
	}

	if !looksLikePathKey(key) {
		return value
	}

	if baseDir == "" {
		return exportPath(stringValue)
	}

	if filepath.IsAbs(stringValue) {
		rel, err := filepath.Rel(baseDir, stringValue)
		if err == nil {
			return exportPath(rel)
		}
	}

	return exportPath(stringValue)
}

func looksLikePathKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	return key == "path" || strings.HasSuffix(key, "_path") || strings.HasSuffix(key, "path")
}

func trimLeadingDot(value string) string {
	if value == "." {
		return value
	}
	value = strings.TrimPrefix(value, "./")
	return value
}

func exportPath(value string) string {
	cleaned := trimLeadingDot(filepath.Clean(value))
	if cleaned == "." {
		return cleaned
	}
	return filepath.ToSlash(cleaned)
}
