package perf

import (
	"errors"
	"sort"
	"strings"
	"time"
)

type SessionDurations struct {
	Total    time.Duration
	Thinking time.Duration
	Work     time.Duration
}

func GetSessionDurations() (SessionDurations, error) {
	spans, err := GetSpans()
	if err != nil {
		return SessionDurations{}, err
	}
	return sessionDurationsFromSpans(spans)
}

func totalDurationFromLifecycle(spans []SpanSnapshot) (time.Duration, bool) {
	bestStart := time.Time{}
	bestEnd := time.Time{}

	for _, span := range spans {
		if span.Name != "app.lifecycle" {
			continue
		}
		if span.StartTime.IsZero() || span.EndTime.IsZero() {
			continue
		}
		if span.EndTime.Before(span.StartTime) {
			continue
		}

		if bestEnd.IsZero() || bestEnd.Before(span.EndTime) {
			bestStart = span.StartTime
			bestEnd = span.EndTime
		}
	}

	if bestStart.IsZero() || bestEnd.IsZero() {
		return 0, false
	}
	return bestEnd.Sub(bestStart), true
}

func totalDurationFromSpanBounds(spans []SpanSnapshot) (time.Duration, error) {
	var minStart time.Time
	var maxEnd time.Time

	for _, span := range spans {
		if span.StartTime.IsZero() || span.EndTime.IsZero() {
			continue
		}
		if span.EndTime.Before(span.StartTime) {
			continue
		}

		if minStart.IsZero() || span.StartTime.Before(minStart) {
			minStart = span.StartTime
		}
		if maxEnd.IsZero() || maxEnd.Before(span.EndTime) {
			maxEnd = span.EndTime
		}
	}

	if minStart.IsZero() || maxEnd.IsZero() {
		return 0, errors.New("no spans with valid timestamps")
	}

	return maxEnd.Sub(minStart), nil
}

func thinkingDurationFromSpans(spans []SpanSnapshot) time.Duration {
	if len(spans) == 0 {
		return 0
	}

	intervals := make([]timeInterval, 0, len(spans))
	for _, span := range spans {
		if span.StartTime.IsZero() || span.EndTime.IsZero() {
			continue
		}
		if span.EndTime.Before(span.StartTime) {
			continue
		}

		if !isThinkingSpanName(span.Name) {
			continue
		}

		intervals = append(intervals, timeInterval{
			Start: span.StartTime,
			End:   span.EndTime,
		})
	}

	return mergeIntervals(intervals)
}

func isThinkingSpanName(name string) bool {
	name = strings.TrimSpace(name)
	return strings.HasPrefix(name, "tui.") && strings.Contains(name, ".wait.")
}

type timeInterval struct {
	Start time.Time
	End   time.Time
}

func mergeIntervals(intervals []timeInterval) time.Duration {
	if len(intervals) == 0 {
		return 0
	}

	sort.Slice(intervals, func(i, j int) bool {
		if intervals[i].Start.Before(intervals[j].Start) {
			return true
		}
		if intervals[j].Start.Before(intervals[i].Start) {
			return false
		}
		return intervals[i].End.Before(intervals[j].End)
	})

	currentStart := intervals[0].Start
	currentEnd := intervals[0].End
	total := time.Duration(0)

	flush := func() {
		if currentStart.IsZero() || currentEnd.IsZero() {
			return
		}
		if currentEnd.Before(currentStart) {
			return
		}
		total += currentEnd.Sub(currentStart)
	}

	for _, interval := range intervals[1:] {
		if interval.Start.After(currentEnd) {
			flush()
			currentStart = interval.Start
			currentEnd = interval.End
			continue
		}

		if currentEnd.Before(interval.End) {
			currentEnd = interval.End
		}
	}

	flush()
	return total
}

func sessionDurationsFromSpans(spans []SpanSnapshot) (SessionDurations, error) {
	total, ok := totalDurationFromLifecycle(spans)
	if !ok {
		var totalErr error
		total, totalErr = totalDurationFromSpanBounds(spans)
		if totalErr != nil {
			return SessionDurations{}, totalErr
		}
	}

	thinking := thinkingDurationFromSpans(spans)
	work := total - thinking
	if work < 0 {
		work = 0
	}

	return SessionDurations{
		Total:    total,
		Thinking: thinking,
		Work:     work,
	}, nil
}
