package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/toto/whoopy/internal/api"
	"github.com/toto/whoopy/internal/cycles"
	"github.com/toto/whoopy/internal/profile"
	"github.com/toto/whoopy/internal/recovery"
	"github.com/toto/whoopy/internal/sleep"
	"github.com/toto/whoopy/internal/workouts"
)

func TestWriteSleepSessionHPXOutput(t *testing.T) {
	output := encodeHPXRecords(t, func(w *hpxWriter) error {
		return w.writeSleepSession(sleep.Session{
			ID:             "sleep-1",
			CycleID:        42,
			ScoreState:     "SCORED",
			Start:          time.Date(2026, 3, 5, 23, 30, 0, 0, time.UTC),
			End:            time.Date(2026, 3, 6, 7, 15, 0, 0, time.UTC),
			TimezoneOffset: "-05:00",
			Nap:            true,
			Score: sleep.Score{
				SleepPerformancePercentage: floatPtr(91),
				SleepConsistencyPercentage: floatPtr(87),
				RespiratoryRate:            floatPtr(13.4),
				SleepEfficiencyPercentage:  floatPtr(94),
				StageSummary: sleep.StageSummary{
					TotalInBedTimeMilli:         int64Ptr(27900000),
					TotalAwakeTimeMilli:         int64Ptr(1200000),
					TotalLightSleepTimeMilli:    int64Ptr(15000000),
					TotalSlowWaveSleepTimeMilli: int64Ptr(5400000),
					TotalRemSleepTimeMilli:      int64Ptr(5700000),
				},
			},
		})
	})
	records := parseNDJSONRecords(t, output)
	require.Len(t, records, 9)

	start := findRecord(t, records, "table", "signpost", "kind", "sleep", "edge", "start")
	require.Equal(t, "whoop", start["source"])
	require.Equal(t, "sleep-sleep-1-start", start["id"])
	data := start["data"].(map[string]any)
	require.Equal(t, true, data["is_nap"])
	require.Equal(t, "-05:00", data["timezone_offset"])

	duration := findRecord(t, records, "table", "metric", "key", "sleep.duration_ms")
	require.Equal(t, "2026-03-06", duration["date"])
	require.Equal(t, "sleep-sleep-1-start", duration["signpost_id"])
	require.Equal(t, "2026-03-06T07:15:00Z", duration["ts"])
}

func TestWriteWorkoutHPXOutput(t *testing.T) {
	output := encodeHPXRecords(t, func(w *hpxWriter) error {
		return w.writeWorkout(workouts.Workout{
			ID:             "w9",
			SportName:      "Rowing",
			SportID:        intPtr(16),
			ScoreState:     "SCORED",
			Start:          time.Date(2026, 3, 6, 8, 0, 0, 0, time.UTC),
			End:            time.Date(2026, 3, 6, 9, 5, 0, 0, time.UTC),
			TimezoneOffset: "+00:00",
			Score: workouts.Score{
				Strain:           10.2,
				AverageHeartRate: intPtr(148),
				MaxHeartRate:     intPtr(172),
				Kilojoule:        floatPtr(600),
			},
		})
	})
	records := parseNDJSONRecords(t, output)

	start := findRecord(t, records, "table", "signpost", "kind", "workout", "edge", "start")
	require.Equal(t, "workout-w9-start", start["id"])
	data := start["data"].(map[string]any)
	require.Equal(t, "Rowing", data["sport"])

	kj := findRecord(t, records, "table", "metric", "key", "workout.kilojoules")
	require.Equal(t, "workout-w9-start", kj["signpost_id"])
	require.Equal(t, "2026-03-06T08:00:00Z", kj["ts"])
	calories := findRecord(t, records, "table", "metric", "key", "workout.calories_kcal")
	require.Equal(t, "whoop-workout-w9-calories", calories["origin_id"])
	require.Equal(t, "2026-03-06T08:00:00Z", calories["ts"])
}

func TestWriteRecoveryHPXUsesCycleContext(t *testing.T) {
	output := encodeHPXRecords(t, func(w *hpxWriter) error {
		return w.writeRecovery(recovery.Recovery{
			CycleID:    12,
			SleepID:    "sleep-12",
			ScoreState: "SCORED",
			CreatedAt:  time.Date(2026, 3, 6, 9, 0, 0, 0, time.UTC),
			UpdatedAt:  time.Date(2026, 3, 6, 9, 30, 0, 0, time.UTC),
			Score: recovery.RecoveryScore{
				RecoveryScore:    floatPtr(82),
				RestingHeartRate: floatPtr(46),
				HRVRMSSDMilli:    floatPtr(105.5),
				RespiratoryRate:  floatPtr(13.1),
			},
		}, &cycles.Cycle{
			ID:             12,
			Start:          time.Date(2026, 3, 5, 22, 0, 0, 0, time.UTC),
			End:            time.Date(2026, 3, 6, 9, 0, 0, 0, time.UTC),
			TimezoneOffset: "-05:00",
			ScoreState:     "SCORED",
		})
	})
	records := parseNDJSONRecords(t, output)

	start := findRecord(t, records, "table", "signpost", "kind", "recovery_window", "edge", "start")
	require.Equal(t, "recovery-window-12-start", start["id"])

	score := findRecord(t, records, "table", "metric", "key", "recovery.score_pct")
	require.Equal(t, "recovery-window-12-start", score["signpost_id"])
	require.Equal(t, "2026-03-06", score["date"])
	require.Equal(t, "2026-03-06T09:00:00Z", score["ts"])
}

func TestWriteProfileHPXOutput(t *testing.T) {
	updatedAt := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	weight := 70.0
	height := 175.0
	output := encodeHPXRecords(t, func(w *hpxWriter) error {
		return w.writeProfile(&profile.Summary{
			UserID:    "user-1",
			Name:      "Ada Lovelace",
			WeightKg:  &weight,
			HeightCm:  &height,
			UpdatedAt: &updatedAt,
		})
	})
	records := parseNDJSONRecords(t, output)
	require.Len(t, records, 2)
	weightMetric := findRecord(t, records, "table", "metric", "key", "body.weight_kg")
	require.Equal(t, "2026-03-04", weightMetric["date"])
	require.Equal(t, "2026-03-04T12:00:00Z", weightMetric["ts"])
	findRecord(t, records, "table", "metric", "key", "body.bmi")
}

func TestLegacyHPXFlagRemoved(t *testing.T) {
	_, err := runCLICommandWithError(t, []string{"version", "--hpx"}, "")
	require.ErrorContains(t, err, "unknown flag: --hpx")
}

func TestHPXExportDedupesRecoveryWindowSignpostsAndAppliesWindow(t *testing.T) {
	origProfile := profileFetchFn
	origSleep := sleepListFn
	origCycles := cyclesListFn
	origRecovery := recoveryListFn
	origWorkouts := workoutsListFn
	origNow := nowFunc
	defer func() {
		profileFetchFn = origProfile
		sleepListFn = origSleep
		cyclesListFn = origCycles
		recoveryListFn = origRecovery
		workoutsListFn = origWorkouts
		nowFunc = origNow
	}()

	nowFunc = func() time.Time {
		return time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	}

	var sleepOpts api.ListOptions
	var cycleOpts api.ListOptions
	var recoveryOpts api.ListOptions
	var workoutOpts api.ListOptions

	profileFetchFn = func(ctx context.Context) (*profile.Summary, error) {
		return nil, nil
	}
	sleepListFn = func(ctx context.Context, opts *api.ListOptions) (*sleep.ListResult, error) {
		require.NotNil(t, opts)
		sleepOpts = *opts
		return &sleep.ListResult{}, nil
	}
	cyclesListFn = func(ctx context.Context, opts *api.ListOptions) (*cycles.ListResult, error) {
		require.NotNil(t, opts)
		cycleOpts = *opts
		return &cycles.ListResult{
			Cycles: []cycles.Cycle{
				{
					ID:             12,
					Start:          time.Date(2026, 3, 5, 22, 0, 0, 0, time.UTC),
					End:            time.Date(2026, 3, 6, 9, 0, 0, 0, time.UTC),
					TimezoneOffset: "-05:00",
					ScoreState:     "SCORED",
					Score: cycles.Score{
						Strain: 14.3,
					},
				},
			},
		}, nil
	}
	recoveryListFn = func(ctx context.Context, opts *api.ListOptions) (*recovery.ListResult, error) {
		require.NotNil(t, opts)
		recoveryOpts = *opts
		return &recovery.ListResult{
			Recoveries: []recovery.Recovery{
				{
					CycleID:    12,
					SleepID:    "sleep-12",
					ScoreState: "SCORED",
					CreatedAt:  time.Date(2026, 3, 6, 9, 0, 0, 0, time.UTC),
					Score: recovery.RecoveryScore{
						RecoveryScore: floatPtr(82),
					},
				},
			},
		}, nil
	}
	workoutsListFn = func(ctx context.Context, opts *api.ListOptions) (*workouts.ListResult, error) {
		require.NotNil(t, opts)
		workoutOpts = *opts
		return &workouts.ListResult{}, nil
	}

	output := runCLICommand(t, []string{"hpx", "export", "--last", "2d"}, "")
	records := parseNDJSONRecords(t, output)
	require.Len(t, records, 4)

	require.NotNil(t, sleepOpts.Start)
	require.NotNil(t, sleepOpts.End)
	require.Equal(t, time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC), sleepOpts.Start.UTC())
	require.Equal(t, time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC), sleepOpts.End.UTC())
	require.Equal(t, sleepOpts, cycleOpts)
	require.Equal(t, sleepOpts, recoveryOpts)
	require.Equal(t, sleepOpts, workoutOpts)

	startCount := 0
	endCount := 0
	for _, record := range records {
		if record["table"] != "signpost" || record["kind"] != "recovery_window" {
			continue
		}
		switch record["edge"] {
		case "start":
			startCount++
		case "end":
			endCount++
		}
	}
	require.Equal(t, 1, startCount)
	require.Equal(t, 1, endCount)

	score := findRecord(t, records, "table", "metric", "key", "recovery.score_pct")
	require.Equal(t, "2026-03-06T09:00:00Z", score["ts"])
}

func parseNDJSONRecords(t *testing.T, output string) []map[string]any {
	t.Helper()
	lines := splitNonEmptyLines(output)
	records := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		var record map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &record))
		records = append(records, record)
	}
	return records
}

func encodeHPXRecords(t *testing.T, writeFn func(*hpxWriter) error) string {
	t.Helper()
	var buf strings.Builder
	require.NoError(t, writeFn(newHPXWriter(&buf)))
	return buf.String()
}

func splitNonEmptyLines(output string) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func findRecord(t *testing.T, records []map[string]any, fields ...string) map[string]any {
	t.Helper()
	require.Equal(t, 0, len(fields)%2, "fields must be key/value pairs")
	for _, record := range records {
		matched := true
		for i := 0; i < len(fields); i += 2 {
			key := fields[i]
			expected := fields[i+1]
			if record[key] != expected {
				matched = false
				break
			}
		}
		if matched {
			return record
		}
	}
	t.Fatalf("record not found for fields %v", fields)
	return nil
}
