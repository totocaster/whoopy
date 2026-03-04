package sleep

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/toto/whoopy/internal/api"
)

const sleepPath = "/activity/sleep"

// Service fetches sleep data from WHOOP's API.
type Service struct {
	client interface {
		GetJSON(ctx context.Context, path string, query url.Values, dest any) error
	}
}

// NewService constructs a sleep Service backed by the shared API client.
func NewService(client *api.Client) *Service {
	return &Service{client: client}
}

// ListResult captures a page of sleep sessions.
type ListResult struct {
	Sleeps    []Session `json:"sleeps"`
	NextToken string    `json:"next_token,omitempty"`
}

// Session represents a WHOOP sleep activity.
type Session struct {
	ID             string    `json:"id"`
	CycleID        int64     `json:"cycle_id"`
	UserID         *int64    `json:"user_id,omitempty"`
	ScoreState     string    `json:"score_state"`
	Start          time.Time `json:"start"`
	End            time.Time `json:"end"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	TimezoneOffset string    `json:"timezone_offset"`
	Nap            bool      `json:"nap"`
	Score          Score     `json:"score"`
}

// Score contains the performance metrics for a sleep session.
type Score struct {
	SleepPerformancePercentage *float64     `json:"sleep_performance_percentage,omitempty"`
	SleepConsistencyPercentage *float64     `json:"sleep_consistency_percentage,omitempty"`
	RespiratoryRate            *float64     `json:"respiratory_rate,omitempty"`
	SleepEfficiencyPercentage  *float64     `json:"sleep_efficiency_percentage,omitempty"`
	StageSummary               StageSummary `json:"stage_summary"`
}

// StageSummary contains duration-in-milliseconds for each sleep stage.
type StageSummary struct {
	TotalInBedTimeMilli         *int64 `json:"total_in_bed_time_ms,omitempty"`
	TotalAwakeTimeMilli         *int64 `json:"total_awake_time_ms,omitempty"`
	TotalLightSleepTimeMilli    *int64 `json:"total_light_sleep_time_ms,omitempty"`
	TotalSlowWaveSleepTimeMilli *int64 `json:"total_slow_wave_sleep_time_ms,omitempty"`
	TotalRemSleepTimeMilli      *int64 `json:"total_rem_sleep_time_ms,omitempty"`
}

// List retrieves sleep sessions using the shared pagination options.
func (s *Service) List(ctx context.Context, opts *api.ListOptions) (*ListResult, error) {
	if opts == nil {
		opts = &api.ListOptions{}
	}
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	query := opts.Apply(nil)
	var resp struct {
		Records   []sessionRecord `json:"records"`
		NextToken string          `json:"next_token"`
	}
	if err := s.client.GetJSON(ctx, sleepPath, query, &resp); err != nil {
		return nil, fmt.Errorf("fetch sleep sessions: %w", err)
	}
	sessions := make([]Session, len(resp.Records))
	for i, record := range resp.Records {
		sess, err := convertRecord(record)
		if err != nil {
			return nil, err
		}
		sessions[i] = sess
	}
	return &ListResult{Sleeps: sessions, NextToken: strings.TrimSpace(resp.NextToken)}, nil
}

// Get returns a single sleep session by ID.
func (s *Service) Get(ctx context.Context, id string) (*Session, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("sleep id is required")
	}
	path := fmt.Sprintf("%s/%s", sleepPath, strings.TrimSpace(id))
	var record sessionRecord
	if err := s.client.GetJSON(ctx, path, nil, &record); err != nil {
		return nil, fmt.Errorf("fetch sleep %s: %w", id, err)
	}
	sess, err := convertRecord(record)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

type sessionRecord struct {
	ID             string      `json:"id"`
	CycleID        int64       `json:"cycle_id"`
	UserID         *int64      `json:"user_id"`
	ScoreState     string      `json:"score_state"`
	Start          string      `json:"start"`
	End            string      `json:"end"`
	CreatedAt      string      `json:"created_at"`
	UpdatedAt      string      `json:"updated_at"`
	TimezoneOffset string      `json:"timezone_offset"`
	Nap            bool        `json:"nap"`
	Score          *sleepScore `json:"score"`
}

type sleepScore struct {
	SleepPerformancePercentage *float64      `json:"sleep_performance_percentage"`
	SleepConsistencyPercentage *float64      `json:"sleep_consistency_percentage"`
	RespiratoryRate            *float64      `json:"respiratory_rate"`
	SleepEfficiencyPercentage  *float64      `json:"sleep_efficiency_percentage"`
	StageSummary               *stageSummary `json:"stage_summary"`
}

type stageSummary struct {
	TotalInBedTimeMilli         *int64 `json:"total_in_bed_time_ms"`
	TotalAwakeTimeMilli         *int64 `json:"total_awake_time_ms"`
	TotalLightSleepTimeMilli    *int64 `json:"total_light_sleep_time_ms"`
	TotalSlowWaveSleepTimeMilli *int64 `json:"total_slow_wave_sleep_time_ms"`
	TotalRemSleepTimeMilli      *int64 `json:"total_rem_sleep_time_ms"`
}

func convertRecord(rec sessionRecord) (Session, error) {
	start, err := parseTime(rec.Start)
	if err != nil {
		return Session{}, fmt.Errorf("parse start: %w", err)
	}
	end, err := parseTime(rec.End)
	if err != nil {
		return Session{}, fmt.Errorf("parse end: %w", err)
	}
	created, err := parseTime(rec.CreatedAt)
	if err != nil {
		return Session{}, fmt.Errorf("parse created_at: %w", err)
	}
	updated, err := parseTime(rec.UpdatedAt)
	if err != nil {
		return Session{}, fmt.Errorf("parse updated_at: %w", err)
	}

	score := Score{}
	if rec.Score != nil {
		score = Score{
			SleepPerformancePercentage: rec.Score.SleepPerformancePercentage,
			SleepConsistencyPercentage: rec.Score.SleepConsistencyPercentage,
			RespiratoryRate:            rec.Score.RespiratoryRate,
			SleepEfficiencyPercentage:  rec.Score.SleepEfficiencyPercentage,
			StageSummary:               convertStageSummary(rec.Score.StageSummary),
		}
	}

	return Session{
		ID:             rec.ID,
		CycleID:        rec.CycleID,
		UserID:         rec.UserID,
		ScoreState:     rec.ScoreState,
		Start:          start,
		End:            end,
		CreatedAt:      created,
		UpdatedAt:      updated,
		TimezoneOffset: rec.TimezoneOffset,
		Nap:            rec.Nap,
		Score:          score,
	}, nil
}

func convertStageSummary(summary *stageSummary) StageSummary {
	if summary == nil {
		return StageSummary{}
	}
	return StageSummary{
		TotalInBedTimeMilli:         summary.TotalInBedTimeMilli,
		TotalAwakeTimeMilli:         summary.TotalAwakeTimeMilli,
		TotalLightSleepTimeMilli:    summary.TotalLightSleepTimeMilli,
		TotalSlowWaveSleepTimeMilli: summary.TotalSlowWaveSleepTimeMilli,
		TotalRemSleepTimeMilli:      summary.TotalRemSleepTimeMilli,
	}
}

func parseTime(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("timestamps must be RFC3339: %q", value)
}
