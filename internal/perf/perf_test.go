package perf

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestMark(t *testing.T) {
	ClearPerformanceLog()
	details := &PerformanceDetails{"key": "value"}
	entry := Mark("test-marker", details)
	perfLog := GetPerformanceLog()

	assert.Equal(t, "test-marker", entry.Name)
	assert.Equal(t, MarkType, entry.Type)
	assert.Equal(t, details, entry.Details)
	assert.Len(t, perfLog, 1)
}

func TestMeasure(t *testing.T) {
	ClearPerformanceLog()
	startDetails := &PerformanceDetails{"key": "start"}
	endDetails := &PerformanceDetails{"key": "end"}

	startEntry := Mark("start-marker", startDetails)
	time.Sleep(10 * time.Millisecond)
	endEntry := Mark("end-marker", endDetails)

	Measure("test-MeasureType", "start-marker", "end-marker", nil)

	perfLog := GetPerformanceLog()

	measureEntry := perfLog[len(perfLog)-1]
	assert.Equal(t, "test-MeasureType", measureEntry.Name)
	assert.Equal(t, MeasureType, measureEntry.Type)

	expectedDuration := endEntry.StartTime.Sub(startEntry.StartTime)
	assert.Equal(t, expectedDuration, measureEntry.Duration)
}

func TestStartRegionAndEnd(t *testing.T) {
	ClearPerformanceLog()
	details := &PerformanceDetails{"key": "value"}
	region := StartRegionWithDetails("test-region", details)
	time.Sleep(10 * time.Millisecond)
	region.End()

	perfLog := GetPerformanceLog()

	startEntry := perfLog[len(perfLog)-3]
	endEntry := perfLog[len(perfLog)-2]
	measureEntry := perfLog[len(perfLog)-1]

	assert.Equal(t, "test-region", startEntry.Name)
	assert.Equal(t, MarkType, startEntry.Type)

	assert.Equal(t, "test-region-end", endEntry.Name)
	assert.Equal(t, MarkType, endEntry.Type)

	assert.Equal(t, "test-region-duration", measureEntry.Name)
	assert.Equal(t, MeasureType, measureEntry.Type)
}

func TestStartRegionWithoutDetailsAndEnd(t *testing.T) {
	ClearPerformanceLog()
	region := StartRegion("test-region")
	time.Sleep(10 * time.Millisecond)
	region.End()

	perfLog := GetPerformanceLog()

	startEntry := perfLog[len(perfLog)-3]
	endEntry := perfLog[len(perfLog)-2]
	measureEntry := perfLog[len(perfLog)-1]

	assert.Equal(t, "test-region", startEntry.Name)
	assert.Equal(t, MarkType, startEntry.Type)

	assert.Equal(t, "test-region-end", endEntry.Name)
	assert.Equal(t, MarkType, endEntry.Type)

	assert.Equal(t, "test-region-duration", measureEntry.Name)
	assert.Equal(t, MeasureType, measureEntry.Type)
}

func TestGetAllMeasurements(t *testing.T) {
	ClearPerformanceLog()
	// Add entries using Mark and Measure functions
	Mark("start-marker1", nil)
	time.Sleep(10 * time.Millisecond)
	Mark("end-marker1", nil)
	Measure("measure1", "start-marker1", "end-marker1", nil)

	Mark("start-marker2", nil)
	time.Sleep(10 * time.Millisecond)
	Mark("end-marker2", nil)
	Measure("measure2", "start-marker2", "end-marker2", nil)

	measurements := GetAllMeasurements()

	assert.Len(t, measurements, 2)

	for _, entry := range measurements {
		assert.Equal(t, MeasureType, entry.Type)
	}
}

func TestMeasureMissingMarkers(t *testing.T) {
	ClearPerformanceLog()

	// Scenario 1: fromMarker does not exist
	Mark("end-marker", nil)
	Measure("measure1", "non-existent-start-marker", "end-marker", nil)
	perfLog := GetPerformanceLog()
	assert.Len(t, perfLog, 1) // Only the end-marker should be present

	// Scenario 2: toMarker does not exist
	ClearPerformanceLog()
	Mark("start-marker", nil)
	Measure("measure2", "start-marker", "non-existent-end-marker", nil)
	perfLog = GetPerformanceLog()
	assert.Len(t, perfLog, 1) // Only the start-marker should be present

	// Scenario 3: both fromMarker and toMarker do not exist
	ClearPerformanceLog()
	Measure("measure3", "non-existent-start-marker", "non-existent-end-marker", nil)
	perfLog = GetPerformanceLog()
	assert.Len(t, perfLog, 0) // No markers should be present
}
