package stats

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/toto/whoopy/internal/api"
	"github.com/toto/whoopy/internal/cycles"
	"github.com/toto/whoopy/internal/recovery"
	"github.com/toto/whoopy/internal/sleep"
	"github.com/toto/whoopy/internal/workouts"
)

// Service aggregates WHOOP resources into higher-level reports.
type Service struct {
	cyclesSvc   cyclesFetcher
	recoverySvc recoveryFetcher
	sleepSvc    sleepFetcher
	workoutSvc  workoutFetcher
}

// NewService wires the stats service to shared API-backed services.
func NewService(client *api.Client) *Service {
	return &Service{
		cyclesSvc:   cycles.NewService(client),
		recoverySvc: recovery.NewService(client),
		sleepSvc:    sleep.NewService(client),
		workoutSvc:  workouts.NewService(client),
	}
}

// DailyReport summarizes all activity for a single calendar day.
type DailyReport struct {
	Date     string             `json:"date"`
	Start    time.Time          `json:"start"`
	End      time.Time          `json:"end"`
	Cycle    *cycles.Cycle      `json:"cycle,omitempty"`
	Recovery *recovery.Recovery `json:"recovery,omitempty"`
	Sleep    []sleep.Session    `json:"sleep_sessions,omitempty"`
	Workouts []workouts.Workout `json:"workouts,omitempty"`
	Summary  Summary            `json:"summary"`
}

// Summary contains derived statistics for a daily report.
type Summary struct {
	CycleStrain        *float64 `json:"cycle_strain,omitempty"`
	RecoveryScore      *float64 `json:"recovery_score,omitempty"`
	SleepPerformance   *float64 `json:"sleep_performance_percentage,omitempty"`
	TotalSleepHours    float64  `json:"total_sleep_hours"`
	WorkoutCount       int      `json:"workout_count"`
	TotalWorkoutStrain float64  `json:"total_workout_strain"`
}

// Daily fetches all WHOOP resources for the provided date (local timezone).
func (s *Service) Daily(ctx context.Context, date time.Time) (*DailyReport, error) {
	local := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	start := local
	end := local.Add(24 * time.Hour)
	startUTC := start.UTC()
	endUTC := end.UTC()

	report := &DailyReport{Date: start.Format("2006-01-02"), Start: start, End: end}

	cycle, err := s.singleCycle(ctx, startUTC, endUTC)
	if err != nil {
		return nil, err
	}
	report.Cycle = cycle

	rec, err := s.singleRecovery(ctx, cycle, startUTC, endUTC)
	if err != nil {
		return nil, err
	}
	report.Recovery = rec

	sleeps, sleepHours, sleepPerf, err := s.sleepSessions(ctx, startUTC, endUTC)
	if err != nil {
		return nil, err
	}
	report.Sleep = sleeps

	workouts, totalStrain, err := s.workoutSessions(ctx, startUTC, endUTC)
	if err != nil {
		return nil, err
	}
	report.Workouts = workouts

	report.Summary = Summary{
		CycleStrain:        extractCycleStrain(cycle),
		RecoveryScore:      extractRecoveryScore(rec),
		SleepPerformance:   sleepPerf,
		TotalSleepHours:    sleepHours,
		WorkoutCount:       len(workouts),
		TotalWorkoutStrain: totalStrain,
	}

	return report, nil
}

func (s *Service) singleCycle(ctx context.Context, start, end time.Time) (*cycles.Cycle, error) {
	opts := rangeOpts(start, end, 5)
	res, err := s.cyclesSvc.List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("fetch cycles for stats: %w", err)
	}
	if res == nil {
		return nil, nil
	}
	for _, cycle := range res.Cycles {
		if withinDay(cycle.Start, start, end) {
			c := cycle
			return &c, nil
		}
	}
	return nil, nil
}

func (s *Service) singleRecovery(ctx context.Context, cycle *cycles.Cycle, start, end time.Time) (*recovery.Recovery, error) {
	if cycle != nil {
		rec, err := s.recoverySvc.GetByCycle(ctx, strconv.FormatInt(cycle.ID, 10))
		if err == nil {
			return rec, nil
		}
	}
	opts := rangeOpts(start, end, 10)
	res, err := s.recoverySvc.List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("fetch recoveries for stats: %w", err)
	}
	if res == nil {
		return nil, nil
	}
	for _, rec := range res.Recoveries {
		if rec.CreatedAt.After(start) && rec.CreatedAt.Before(end) {
			r := rec
			return &r, nil
		}
	}
	return nil, nil
}

func (s *Service) sleepSessions(ctx context.Context, start, end time.Time) ([]sleep.Session, float64, *float64, error) {
	opts := rangeOpts(start, end, 50)
	res, err := s.sleepSvc.List(ctx, opts)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("fetch sleep for stats: %w", err)
	}
	if res == nil {
		return nil, 0, nil, nil
	}
	var sessions []sleep.Session
	var totalHours float64
	var perf *float64
	for _, sess := range res.Sleeps {
		if withinDay(sess.Start, start, end) {
			sessions = append(sessions, sess)
			totalHours += sess.End.Sub(sess.Start).Hours()
			if perf == nil && sess.Score.SleepPerformancePercentage != nil {
				value := *sess.Score.SleepPerformancePercentage
				perf = &value
			}
		}
	}
	return sessions, totalHours, perf, nil
}

func (s *Service) workoutSessions(ctx context.Context, start, end time.Time) ([]workouts.Workout, float64, error) {
	opts := rangeOpts(start, end, 200)
	res, err := s.workoutSvc.List(ctx, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch workouts for stats: %w", err)
	}
	if res == nil {
		return nil, 0, nil
	}
	var filtered []workouts.Workout
	var totalStrain float64
	for _, w := range res.Workouts {
		if withinDay(w.Start, start, end) {
			filtered = append(filtered, w)
			totalStrain += w.Score.Strain
		}
	}
	return filtered, totalStrain, nil
}

func rangeOpts(start, end time.Time, limit int) *api.ListOptions {
	startCopy := start
	endCopy := end
	if limit <= 0 || limit > 25 {
		limit = 25
	}
	return &api.ListOptions{Start: &startCopy, End: &endCopy, Limit: limit}
}

func withinDay(ts, start, end time.Time) bool {
	if ts.IsZero() {
		return false
	}
	return !ts.Before(start) && ts.Before(end)
}

func extractCycleStrain(cycle *cycles.Cycle) *float64 {
	if cycle == nil || cycle.Score.Strain <= 0 {
		return nil
	}
	value := cycle.Score.Strain
	return &value
}

func extractRecoveryScore(rec *recovery.Recovery) *float64 {
	if rec == nil || rec.Score.RecoveryScore == nil {
		return nil
	}
	return rec.Score.RecoveryScore
}

// Interfaces for dependency injection/testing.
type cyclesFetcher interface {
	List(ctx context.Context, opts *api.ListOptions) (*cycles.ListResult, error)
}

type recoveryFetcher interface {
	List(ctx context.Context, opts *api.ListOptions) (*recovery.ListResult, error)
	GetByCycle(ctx context.Context, cycleID string) (*recovery.Recovery, error)
}

type sleepFetcher interface {
	List(ctx context.Context, opts *api.ListOptions) (*sleep.ListResult, error)
}

type workoutFetcher interface {
	List(ctx context.Context, opts *api.ListOptions) (*workouts.ListResult, error)
}
