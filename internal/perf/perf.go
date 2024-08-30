package perf

import (
	"context"
	"fmt"
	"runtime/trace"
	"slices"
	"time"
)

type EntryType string

const (
	MarkType    EntryType = "MarkType"
	MeasureType EntryType = "MeasureType"
)

type Entry struct {
	Name      string              `json:"name"`
	Type      EntryType           `json:"type"`
	StartTime time.Time           `json:"start_time,omitempty"`
	Duration  time.Duration       `json:"duration,omitempty"`
	Details   *PerformanceDetails `json:"details,omitempty"`
}

type PerformanceLog []Entry

type PerformanceDetails map[string]interface{}

var perfLog = make(PerformanceLog, 0)

type PerformanceRegion struct {
	Region *trace.Region
	Marker *Entry
}

func (r *PerformanceRegion) End() {
	r.EndWithDetails(nil)
}

func (r *PerformanceRegion) EndWithDetails(details *PerformanceDetails) {
	r.Region.End()
	startName := r.Marker.Name
	endName := fmt.Sprintf("%s-end", r.Marker.Name)
	Mark(endName, details)
	Measure(fmt.Sprintf("%s-duration", r.Marker.Name), startName, endName, r.Marker.Details)
}

func ClearPerformanceLog() {
	perfLog = make(PerformanceLog, 0)
}

func GetPerformanceLog() PerformanceLog {
	return perfLog
}

func filter(entries []Entry, predicate func(Entry) bool) []Entry {
	var result []Entry
	for _, entry := range entries {
		if predicate(entry) {
			result = append(result, entry)
		}
	}
	return result
}

func GetAllMeasurements() PerformanceLog {
	return filter(perfLog, func(e Entry) bool {
		return e.Type == MeasureType
	})
}

func StartRegion(marker string) *PerformanceRegion {
	return StartRegionWithDetils(marker, nil)
}

func StartRegionWithDetils(marker string, details *PerformanceDetails) *PerformanceRegion {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "details", details)

	region := trace.StartRegion(ctx, marker)
	markerEntry := Mark(marker, details)

	return &PerformanceRegion{
		Region: region,
		Marker: markerEntry,
	}
}

func Mark(marker string, details *PerformanceDetails) *Entry {
	entry := Entry{
		Name:      marker,
		Type:      MarkType,
		StartTime: time.Now(),
		Details:   details,
	}
	perfLog = append(perfLog, entry)

	return &entry
}

func Measure(marker string, fromMarker string, toMarker string, details *PerformanceDetails) {
	idx := slices.IndexFunc(perfLog, func(e Entry) bool {
		return e.Name == fromMarker
	})

	if idx == -1 {
		return
	}

	from := perfLog[idx].StartTime

	idx = slices.IndexFunc(perfLog, func(e Entry) bool {
		return e.Name == toMarker
	})

	if idx == -1 {
		return
	}

	to := perfLog[idx].StartTime

	perfLog = append(perfLog, Entry{
		Name:      marker,
		Type:      MeasureType,
		StartTime: from,
		Duration:  to.Sub(from),
		Details:   details,
	})

}
