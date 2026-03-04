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

func TestWorkoutsViewJSONOutput(t *testing.T) {
	orig := workoutsViewFn
	defer func() { workoutsViewFn = orig }()
	workoutsViewFn = func(ctx context.Context, id string) (*workouts.Workout, error) {
		require.Equal(t, "w9", id)
		return &workouts.Workout{
			ID:        "w9",
			SportName: "Rowing",
			Start:     time.Date(2026, 3, 5, 1, 0, 0, 0, time.UTC),
			End:       time.Date(2026, 3, 5, 2, 0, 0, 0, time.UTC),
			Score: workouts.Score{
				Strain:           9.9,
				AverageHeartRate: intPtr(130),
				ZoneDurations:    workouts.ZoneDurations{},
			},
		}, nil
	}

	output := runCLICommand(t, []string{"workouts", "view", "w9"}, "")
	require.Contains(t, output, "\"id\": \"w9\"")
	require.Contains(t, output, "\"sport_name\": \"Rowing\"")
}

func TestWorkoutsViewTextOutput(t *testing.T) {
	orig := workoutsViewFn
	defer func() { workoutsViewFn = orig }()
	workoutsViewFn = func(ctx context.Context, id string) (*workouts.Workout, error) {
		return &workouts.Workout{
			ID:        "w11",
			SportName: "Skiing",
			Start:     time.Date(2026, 3, 6, 5, 0, 0, 0, time.UTC),
			End:       time.Date(2026, 3, 6, 6, 15, 0, 0, time.UTC),
			Score: workouts.Score{
				Strain:           11.2,
				AverageHeartRate: intPtr(145),
				MaxHeartRate:     intPtr(170),
				ZoneDurations:    workouts.ZoneDurations{},
			},
		}, nil
	}

	output := runCLICommand(t, []string{"workouts", "view", "w11", "--text"}, "")
	require.Contains(t, output, "ID: w11")
	require.Contains(t, output, "Sport: Skiing")
	require.Contains(t, output, "Avg HR: 145")
	require.Contains(t, output, "Strain: 11.2")
}

func intPtr(v int) *int { return &v }
