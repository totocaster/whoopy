package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/toto/whoopy/internal/api"
	"github.com/toto/whoopy/internal/workouts"
)

func TestWorkoutsListJSONOutput(t *testing.T) {
	orig := workoutsListFn
	defer func() { workoutsListFn = orig }()
	workoutsListFn = func(ctx context.Context, opts *api.ListOptions) (*workouts.ListResult, error) {
		require.NotNil(t, opts)
		return &workouts.ListResult{
			Workouts: []workouts.Workout{
				{
					ID:         "w1",
					SportName:  "Running",
					Start:      time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC),
					End:        time.Date(2026, 3, 4, 1, 0, 0, 0, time.UTC),
					ScoreState: "SCORED",
					Score: workouts.Score{
						Strain:           10.5,
						AverageHeartRate: intPtr(150),
					},
				},
			},
			NextToken: "cursor",
		}, nil
	}

	output := runCLICommand(t, []string{"workouts", "list"}, "")
	require.Contains(t, output, "\"id\": \"w1\"")
	require.Contains(t, output, "\"next_token\": \"cursor\"")
}

func TestWorkoutsListTextOutput(t *testing.T) {
	orig := workoutsListFn
	defer func() { workoutsListFn = orig }()
	workoutsListFn = func(ctx context.Context, opts *api.ListOptions) (*workouts.ListResult, error) {
		return &workouts.ListResult{
			Workouts: []workouts.Workout{
				{
					ID:        "w2",
					SportName: "Cycling",
					Start:     time.Date(2026, 3, 4, 2, 0, 0, 0, time.UTC),
					End:       time.Date(2026, 3, 4, 3, 30, 0, 0, time.UTC),
					Score: workouts.Score{
						Strain:           8.2,
						AverageHeartRate: intPtr(140),
					},
				},
			},
		}, nil
	}

	output := runCLICommand(t, []string{"workouts", "list", "--text"}, "")
	require.Contains(t, output, "Cycling")
	require.Contains(t, output, "8.2")
	require.Contains(t, output, "w2")
}

func intPtr(v int) *int { return &v }
