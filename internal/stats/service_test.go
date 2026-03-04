package stats

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/toto/whoopy/internal/api"
	"github.com/toto/whoopy/internal/cycles"
	"github.com/toto/whoopy/internal/recovery"
	"github.com/toto/whoopy/internal/sleep"
	"github.com/toto/whoopy/internal/workouts"
)

func TestDailyAggregatesData(t *testing.T) {
	start := time.Date(2026, 3, 4, 0, 0, 0, 0, time.Local)
	end := start.Add(24 * time.Hour)
	cycle := cycles.Cycle{ID: 42, Start: start, End: end, Score: cycles.Score{Strain: 14.2}}
	rec := recovery.Recovery{CycleID: 42, Score: recovery.RecoveryScore{RecoveryScore: floatPtr(81), RestingHeartRate: floatPtr(46)}}
	sleepSession := sleep.Session{ID: "sleep-1", Start: start.Add(30 * time.Minute), End: start.Add(8 * time.Hour), Score: sleep.Score{SleepPerformancePercentage: floatPtr(92)}}
	workout := workouts.Workout{ID: "work-1", Start: start.Add(10 * time.Hour), End: start.Add(11 * time.Hour), Score: workouts.Score{Strain: 10.5}}

	svc := &Service{
		cyclesSvc:   &fakeCycles{result: &cycles.ListResult{Cycles: []cycles.Cycle{cycle}}},
		recoverySvc: &fakeRecovery{recovery: &rec},
		sleepSvc:    &fakeSleep{result: &sleep.ListResult{Sleeps: []sleep.Session{sleepSession}}},
		workoutSvc:  &fakeWorkouts{result: &workouts.ListResult{Workouts: []workouts.Workout{workout}}},
	}

	report, err := svc.Daily(context.Background(), start)
	require.NoError(t, err)
	require.NotNil(t, report.Cycle)
	require.NotNil(t, report.Recovery)
	require.Len(t, report.Sleep, 1)
	require.Len(t, report.Workouts, 1)
	require.Equal(t, 1, report.Summary.WorkoutCount)
	require.InDelta(t, 10.5, report.Summary.TotalWorkoutStrain, 0.001)
	require.InDelta(t, 7.5, report.Summary.TotalSleepHours, 0.001)
	require.NotNil(t, report.Summary.CycleStrain)
	require.NotNil(t, report.Summary.RecoveryScore)
	require.NotNil(t, report.Summary.SleepPerformance)
}

// fakes

type fakeCycles struct {
	result *cycles.ListResult
	err    error
}

func (f *fakeCycles) List(ctx context.Context, opts *api.ListOptions) (*cycles.ListResult, error) {
	return f.result, f.err
}

type fakeRecovery struct {
	result   *recovery.ListResult
	recovery *recovery.Recovery
	err      error
}

func (f *fakeRecovery) List(ctx context.Context, opts *api.ListOptions) (*recovery.ListResult, error) {
	if f.result != nil {
		return f.result, nil
	}
	if f.recovery != nil {
		return &recovery.ListResult{Recoveries: []recovery.Recovery{*f.recovery}}, nil
	}
	return nil, f.err
}

func (f *fakeRecovery) GetByCycle(ctx context.Context, cycleID string) (*recovery.Recovery, error) {
	return f.recovery, f.err
}

type fakeSleep struct {
	result *sleep.ListResult
	err    error
}

func (f *fakeSleep) List(ctx context.Context, opts *api.ListOptions) (*sleep.ListResult, error) {
	return f.result, f.err
}

type fakeWorkouts struct {
	result *workouts.ListResult
	err    error
}

func (f *fakeWorkouts) List(ctx context.Context, opts *api.ListOptions) (*workouts.ListResult, error) {
	return f.result, f.err
}

func floatPtr(v float64) *float64 { return &v }
