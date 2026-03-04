package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/toto/whoopy/internal/api"
	"github.com/toto/whoopy/internal/recovery"
)

func TestRecoveryListJSONOutput(t *testing.T) {
	orig := recoveryListFn
	defer func() { recoveryListFn = orig }()
	recoveryListFn = func(ctx context.Context, opts *api.ListOptions) (*recovery.ListResult, error) {
		return &recovery.ListResult{
			Recoveries: []recovery.Recovery{
				{
					CycleID:   1,
					SleepID:   "sleep-1",
					CreatedAt: time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC),
					Score: recovery.RecoveryScore{
						RecoveryScore:    intPtr(82),
						RestingHeartRate: intPtr(46),
					},
				},
			},
			NextToken: "token",
		}, nil
	}

	output := runCLICommand(t, []string{"recovery", "list", "--text=false"}, "")
	require.Contains(t, output, "\"recoveries\"")
	require.Contains(t, output, "\"sleep_id\": \"sleep-1\"")
	require.Contains(t, output, "\"next_token\": \"token\"")
}

func TestRecoveryViewTextOutput(t *testing.T) {
	orig := recoveryViewFn
	defer func() { recoveryViewFn = orig }()
	recoveryViewFn = func(ctx context.Context, cycleID string) (*recovery.Recovery, error) {
		return &recovery.Recovery{
			CycleID:    10,
			SleepID:    "sleep-10",
			ScoreState: "SCORED",
			Score: recovery.RecoveryScore{
				RecoveryScore:    intPtr(90),
				RestingHeartRate: intPtr(44),
				HRVRMSSDMilli:    floatPtr(110.0),
				UserCalibrating:  true,
			},
			CreatedAt: time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC),
		}, nil
	}

	output := runCLICommand(t, []string{"recovery", "view", "10", "--text"}, "")
	require.Contains(t, output, "Cycle ID: 10")
	require.Contains(t, output, "Recovery Score: 90")
	require.Contains(t, output, "User Calibrating: yes")
}
