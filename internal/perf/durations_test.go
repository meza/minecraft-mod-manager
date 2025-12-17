package perf

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetSessionDurations_ReturnsErrorWhenPerfDisabled(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	_, err := GetSessionDurations()
	assert.Error(t, err)
}

func TestGetSessionDurations_ReturnsDurationsWhenEnabled(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	assert.NoError(t, Init(Config{Enabled: true}))

	_, span := StartSpan(nil, "app.lifecycle")
	span.End()

	durations, err := GetSessionDurations()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, durations.Total, time.Duration(0))
}

func TestMergeIntervals_ReturnsZeroWhenEmpty(t *testing.T) {
	assert.Equal(t, time.Duration(0), mergeIntervals(nil))
}

func TestMergeIntervals_MergesAdjacentIntervalsAndSkipsInvalid(t *testing.T) {
	base := time.Now()
	invalid := timeInterval{Start: base.Add(-2 * time.Second), End: base.Add(-3 * time.Second)}

	merged := mergeIntervals([]timeInterval{
		{Start: base.Add(2 * time.Second), End: base.Add(3 * time.Second)},
		invalid,
		{Start: time.Time{}, End: time.Time{}},
		{Start: base, End: base.Add(1 * time.Second)},
		{Start: base.Add(1 * time.Second), End: base.Add(2 * time.Second)},
	})

	assert.Equal(t, 3*time.Second, merged)
}

func TestSessionDurationsFromSpans_UsesLifecycleSpanAndSubtractsThinking(t *testing.T) {
	base := time.Now()

	durations, err := sessionDurationsFromSpans([]SpanSnapshot{
		{
			Name:      "app.lifecycle",
			StartTime: base,
			EndTime:   base.Add(2 * time.Second),
		},
		{
			Name:      "tui.add.wait.enter_id",
			StartTime: base.Add(1 * time.Second),
			EndTime:   base.Add(1100 * time.Millisecond),
		},
		{
			Name:      "tui.add.wait.confirm",
			StartTime: base.Add(1050 * time.Millisecond),
			EndTime:   base.Add(1200 * time.Millisecond),
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, 2*time.Second, durations.Total)
	assert.Equal(t, 200*time.Millisecond, durations.Thinking)
	assert.Equal(t, 1800*time.Millisecond, durations.Work)
}

func TestSessionDurationsFromSpans_ThinkingUsesUnionWhenSpansOverlap(t *testing.T) {
	base := time.Now()

	durations, err := sessionDurationsFromSpans([]SpanSnapshot{
		{
			Name:      "app.lifecycle",
			StartTime: base,
			EndTime:   base.Add(1 * time.Second),
		},
		{
			Name:      "tui.add.wait.one",
			StartTime: base.Add(200 * time.Millisecond),
			EndTime:   base.Add(400 * time.Millisecond),
		},
		{
			Name:      "tui.add.wait.two",
			StartTime: base.Add(300 * time.Millisecond),
			EndTime:   base.Add(500 * time.Millisecond),
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, 1*time.Second, durations.Total)
	assert.Equal(t, 300*time.Millisecond, durations.Thinking)
	assert.Equal(t, 700*time.Millisecond, durations.Work)
}

func TestSessionDurationsFromSpans_FallsBackToSpanBoundsWhenLifecycleMissing(t *testing.T) {
	base := time.Now()

	durations, err := sessionDurationsFromSpans([]SpanSnapshot{
		{
			Name:      "something",
			StartTime: base.Add(100 * time.Millisecond),
			EndTime:   base.Add(200 * time.Millisecond),
		},
		{
			Name:      "another",
			StartTime: base.Add(300 * time.Millisecond),
			EndTime:   base.Add(900 * time.Millisecond),
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, 800*time.Millisecond, durations.Total)
	assert.Equal(t, time.Duration(0), durations.Thinking)
	assert.Equal(t, 800*time.Millisecond, durations.Work)
}

func TestTotalDurationFromSpanBounds_ReturnsErrorWhenNoValidTimestamps(t *testing.T) {
	_, err := totalDurationFromSpanBounds([]SpanSnapshot{{Name: "x"}})
	assert.Error(t, err)
}

func TestIsThinkingSpanName_MatchesWaitSpans(t *testing.T) {
	assert.True(t, isThinkingSpanName("tui.add.wait.enter_id"))
	assert.False(t, isThinkingSpanName("tui.add.session"))
	assert.False(t, isThinkingSpanName("app.command.add"))
}

func TestTotalDurationFromLifecycle_PicksLatestEndedLifecycleSpan(t *testing.T) {
	base := time.Now()
	duration, ok := totalDurationFromLifecycle([]SpanSnapshot{
		{Name: "app.lifecycle", StartTime: base, EndTime: base.Add(1 * time.Second)},
		{Name: "app.lifecycle", StartTime: base, EndTime: base.Add(2 * time.Second)},
		{Name: "app.lifecycle", StartTime: base, EndTime: base.Add(1500 * time.Millisecond)},
	})

	assert.True(t, ok)
	assert.Equal(t, 2*time.Second, duration)
}

func TestTotalDurationFromLifecycle_ReturnsFalseWhenLifecycleInvalid(t *testing.T) {
	base := time.Now()
	duration, ok := totalDurationFromLifecycle([]SpanSnapshot{
		{Name: "app.lifecycle", StartTime: base.Add(2 * time.Second), EndTime: base.Add(1 * time.Second)},
	})
	assert.False(t, ok)
	assert.Equal(t, time.Duration(0), duration)
}

func TestTotalDurationFromLifecycle_SkipsZeroTimestamps(t *testing.T) {
	base := time.Now()
	duration, ok := totalDurationFromLifecycle([]SpanSnapshot{
		{Name: "app.lifecycle", StartTime: time.Time{}, EndTime: base},
		{Name: "app.lifecycle", StartTime: base, EndTime: time.Time{}},
	})
	assert.False(t, ok)
	assert.Equal(t, time.Duration(0), duration)
}

func TestThinkingDurationFromSpans_IgnoresInvalidAndNonWaitSpans(t *testing.T) {
	base := time.Now()
	thinking := thinkingDurationFromSpans([]SpanSnapshot{
		{Name: "tui.add.wait.one", StartTime: base.Add(2 * time.Second), EndTime: base.Add(1 * time.Second)},
		{Name: "tui.add.session", StartTime: base, EndTime: base.Add(1 * time.Second)},
	})
	assert.Equal(t, time.Duration(0), thinking)
}

func TestSessionDurationsFromSpans_ReturnsErrorWhenNoValidSpans(t *testing.T) {
	_, err := sessionDurationsFromSpans([]SpanSnapshot{{Name: "x"}})
	assert.Error(t, err)
}

func TestSessionDurationsFromSpans_ClampsNegativeWorkToZero(t *testing.T) {
	base := time.Now()
	durations, err := sessionDurationsFromSpans([]SpanSnapshot{
		{Name: "app.lifecycle", StartTime: base, EndTime: base.Add(1 * time.Second)},
		{Name: "tui.add.wait.one", StartTime: base, EndTime: base.Add(2 * time.Second)},
	})
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), durations.Work)
}

func TestThinkingDurationFromSpans_ReturnsZeroWhenEmpty(t *testing.T) {
	assert.Equal(t, time.Duration(0), thinkingDurationFromSpans(nil))
}

func TestThinkingDurationFromSpans_ReturnsZeroWhenNoWaitSpansPresent(t *testing.T) {
	base := time.Now()
	thinking := thinkingDurationFromSpans([]SpanSnapshot{
		{Name: "app.command.list", StartTime: base, EndTime: base.Add(1 * time.Second)},
	})
	assert.Equal(t, time.Duration(0), thinking)
}

func TestThinkingDurationFromSpans_SkipsZeroTimestamps(t *testing.T) {
	base := time.Now()
	thinking := thinkingDurationFromSpans([]SpanSnapshot{
		{Name: "tui.add.wait.one", StartTime: time.Time{}, EndTime: base.Add(1 * time.Second)},
		{Name: "tui.add.wait.two", StartTime: base, EndTime: time.Time{}},
	})
	assert.Equal(t, time.Duration(0), thinking)
}

func TestTotalDurationFromSpanBounds_TracksMinAndMax(t *testing.T) {
	base := time.Now()
	duration, err := totalDurationFromSpanBounds([]SpanSnapshot{
		{Name: "one", StartTime: base.Add(1 * time.Second), EndTime: base.Add(4 * time.Second)},
		{Name: "two", StartTime: base.Add(2 * time.Second), EndTime: base.Add(3 * time.Second)},
	})
	assert.NoError(t, err)
	assert.Equal(t, 3*time.Second, duration)
}

func TestTotalDurationFromSpanBounds_SkipsInvalidSpans(t *testing.T) {
	base := time.Now()
	duration, err := totalDurationFromSpanBounds([]SpanSnapshot{
		{Name: "zero", StartTime: time.Time{}, EndTime: base.Add(1 * time.Second)},
		{Name: "reversed", StartTime: base.Add(2 * time.Second), EndTime: base.Add(1 * time.Second)},
		{Name: "valid", StartTime: base.Add(3 * time.Second), EndTime: base.Add(4 * time.Second)},
	})
	assert.NoError(t, err)
	assert.Equal(t, 1*time.Second, duration)
}

func TestMergeIntervals_SortsWhenStartsEqual(t *testing.T) {
	base := time.Now()
	merged := mergeIntervals([]timeInterval{
		{Start: base, End: base.Add(2 * time.Second)},
		{Start: base, End: base.Add(1 * time.Second)},
	})
	assert.Equal(t, 2*time.Second, merged)
}
