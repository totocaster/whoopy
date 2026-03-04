package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/toto/whoopy/internal/stats"
)

func TestStatsDailyJSONOutput(t *testing.T) {
	orig := statsDailyFn
	defer func() { statsDailyFn = orig }()
	statsDailyFn = func(ctx context.Context, date time.Time) (*stats.DailyReport, error) {
		return &stats.DailyReport{
			Date:  "2026-03-04",
			Start: date,
			End:   date.Add(24 * time.Hour),
			Summary: stats.Summary{
				CycleStrain:        floatPtr(14.2),
				RecoveryScore:      floatPtr(81),
				SleepPerformance:   floatPtr(92),
				TotalSleepHours:    7.5,
				WorkoutCount:       2,
				TotalWorkoutStrain: 18.7,
			},
		}, nil
	}

	output := runCLICommand(t, []string{"stats", "daily", "--date", "2026-03-04", "--text=false"}, "")
	require.Contains(t, output, "\"date\": \"2026-03-04\"")
	require.Contains(t, output, "\"workout_count\": 2")
}

func TestStatsDailyTextOutput(t *testing.T) {
	orig := statsDailyFn
	defer func() { statsDailyFn = orig }()
	statsDailyFn = func(ctx context.Context, date time.Time) (*stats.DailyReport, error) {
		return &stats.DailyReport{
			Date: "2026-03-05",
			Summary: stats.Summary{
				CycleStrain:        floatPtr(13.1),
				RecoveryScore:      floatPtr(82),
				SleepPerformance:   floatPtr(90),
				TotalSleepHours:    7.9,
				WorkoutCount:       1,
				TotalWorkoutStrain: 10.5,
			},
		}, nil
	}

	output := runCLICommand(t, []string{"stats", "daily", "--date", "2026-03-05", "--text"}, "")
	require.Contains(t, output, "WHOOP Stats – 2026-03-05")
	require.Contains(t, output, "Workouts: 1 (total strain 10.5)")
}
