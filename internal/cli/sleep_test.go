package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/toto/whoopy/internal/api"
	"github.com/toto/whoopy/internal/sleep"
)

func TestSleepListJSONOutput(t *testing.T) {
	orig := sleepListFn
	defer func() { sleepListFn = orig }()
	sleepListFn = func(ctx context.Context, opts *api.ListOptions) (*sleep.ListResult, error) {
		return &sleep.ListResult{
			Sleeps: []sleep.Session{
				{
					ID:    "sleep-1",
					Start: time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2026, 3, 4, 6, 30, 0, 0, time.UTC),
					Score: sleep.Score{SleepPerformancePercentage: intPtr(95), RespiratoryRate: floatPtr(14.2)},
				},
			},
		}, nil
	}

	output := runCLICommand(t, []string{"sleep", "list", "--text=false"}, "")
	require.Contains(t, output, "\"sleep-1\"")
	require.Contains(t, output, "\"sleep_performance_percentage\": 95")
}

func TestSleepViewTextOutput(t *testing.T) {
	orig := sleepViewFn
	defer func() { sleepViewFn = orig }()
	sleepViewFn = func(ctx context.Context, id string) (*sleep.Session, error) {
		return &sleep.Session{
			ID:         "sleep-2",
			Start:      time.Date(2026, 3, 5, 1, 0, 0, 0, time.UTC),
			End:        time.Date(2026, 3, 5, 8, 15, 0, 0, time.UTC),
			ScoreState: "SCORED",
			Score: sleep.Score{
				SleepPerformancePercentage: intPtr(88),
				SleepConsistencyPercentage: intPtr(92),
				RespiratoryRate:            floatPtr(13.7),
				StageSummary: sleep.StageSummary{
					TotalInBedTimeMilli:      int64Ptr(28000000),
					TotalLightSleepTimeMilli: int64Ptr(12000000),
				},
			},
		}, nil
	}

	output := runCLICommand(t, []string{"sleep", "view", "sleep-2", "--text"}, "")
	require.Contains(t, output, "ID: sleep-2")
	require.Contains(t, output, "Performance: 88")
	require.Contains(t, output, "In Bed:")
}
