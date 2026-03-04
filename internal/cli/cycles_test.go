package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/toto/whoopy/internal/api"
	"github.com/toto/whoopy/internal/cycles"
)

func TestCyclesListJSONOutput(t *testing.T) {
	orig := cyclesListFn
	defer func() { cyclesListFn = orig }()
	cyclesListFn = func(ctx context.Context, opts *api.ListOptions) (*cycles.ListResult, error) {
		return &cycles.ListResult{
			Cycles: []cycles.Cycle{
				{
					ID:         1,
					Start:      time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
					End:        time.Date(2026, 3, 3, 8, 0, 0, 0, time.UTC),
					Score:      cycles.Score{Strain: 10.5, AverageHeartRate: intPtr(120)},
					ScoreState: "SCORED",
				},
			},
			NextToken: "cursor",
		}, nil
	}

	output := runCLICommand(t, []string{"cycles", "list", "--text=false"}, "")
	require.Contains(t, output, "\"cycles\"")
	require.Contains(t, output, "\"id\": 1")
	require.Contains(t, output, "\"next_token\": \"cursor\"")
}

func TestCyclesListTextOutput(t *testing.T) {
	orig := cyclesListFn
	defer func() { cyclesListFn = orig }()
	cyclesListFn = func(ctx context.Context, opts *api.ListOptions) (*cycles.ListResult, error) {
		return &cycles.ListResult{
			Cycles: []cycles.Cycle{
				{
					ID:    2,
					Start: time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2026, 3, 4, 6, 0, 0, 0, time.UTC),
					Score: cycles.Score{Strain: 8.3, AverageHeartRate: intPtr(110), MaxHeartRate: intPtr(150)},
				},
			},
		}, nil
	}

	output := runCLICommand(t, []string{"cycles", "list", "--text"}, "")
	require.Contains(t, output, "2")
	require.Contains(t, output, "8.3")
	require.Contains(t, output, "110")
}

func TestCyclesViewJSONOutput(t *testing.T) {
	orig := cyclesViewFn
	defer func() { cyclesViewFn = orig }()
	cyclesViewFn = func(ctx context.Context, id string) (*cycles.Cycle, error) {
		require.Equal(t, "7", id)
		return &cycles.Cycle{
			ID:    7,
			Start: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 3, 1, 5, 0, 0, 0, time.UTC),
			Score: cycles.Score{Strain: 9.1},
		}, nil
	}

	output := runCLICommand(t, []string{"cycles", "view", "7", "--text=false"}, "")
	require.Contains(t, output, "\"id\": 7")
	require.Contains(t, output, "\"strain\": 9.1")
}

func TestCyclesViewTextOutput(t *testing.T) {
	orig := cyclesViewFn
	defer func() { cyclesViewFn = orig }()
	cyclesViewFn = func(ctx context.Context, id string) (*cycles.Cycle, error) {
		return &cycles.Cycle{
			ID:    11,
			Start: time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 3, 2, 9, 30, 0, 0, time.UTC),
			Score: cycles.Score{Strain: 11.2, Kilojoule: floatPtr(4000), AverageHeartRate: intPtr(115), MaxHeartRate: intPtr(160)},
		}, nil
	}

	output := runCLICommand(t, []string{"cycles", "view", "11", "--text"}, "")
	require.Contains(t, output, "ID: 11")
	require.Contains(t, output, "Strain: 11.2")
	require.Contains(t, output, "Avg HR: 115")
}
