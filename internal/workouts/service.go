package workouts

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/toto/whoopy/internal/api"
)

const workoutsPath = "/workout"

// Service fetches workout data from WHOOP's developer API.
type Service struct {
	client interface {
		GetJSON(ctx context.Context, path string, query url.Values, dest any) error
	}
}

// NewService creates a workout Service backed by the shared API client.
func NewService(client *api.Client) *Service {
	return &Service{client: client}
}

// ListResult captures a single page of workouts alongside the pagination cursor.
type ListResult struct {
	Workouts  []Workout `json:"workouts"`
	NextToken string    `json:"next_token,omitempty"`
}

// HasNext reports whether WHOOP provided a pagination cursor for the next request.
func (r *ListResult) HasNext() bool {
	if r == nil {
		return false
	}
	return strings.TrimSpace(r.NextToken) != ""
}

// List retrieves a single page of workouts based on the provided options.
func (s *Service) List(ctx context.Context, opts *api.ListOptions) (*ListResult, error) {
	if opts == nil {
		opts = &api.ListOptions{}
	}
	if err := opts.Validate(); err != nil {
		return nil, err
	}
	query := opts.Apply(nil)
	var resp struct {
		Records   []workoutRecord `json:"records"`
		NextToken string          `json:"next_token"`
	}
	if err := s.client.GetJSON(ctx, workoutsPath, query, &resp); err != nil {
		return nil, fmt.Errorf("fetch workouts: %w", err)
	}
	workouts := make([]Workout, len(resp.Records))
	for i, record := range resp.Records {
		workout, err := convertRecord(record)
		if err != nil {
			return nil, err
		}
		workouts[i] = workout
	}
	return &ListResult{Workouts: workouts, NextToken: strings.TrimSpace(resp.NextToken)}, nil
}

// Get fetches a single workout by ID.
func (s *Service) Get(ctx context.Context, id string) (*Workout, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("workout id is required")
	}
	path := fmt.Sprintf("%s/%s", workoutsPath, strings.TrimSpace(id))
	var rec workoutRecord
	if err := s.client.GetJSON(ctx, path, nil, &rec); err != nil {
		return nil, fmt.Errorf("fetch workout %s: %w", id, err)
	}
	workout, err := convertRecord(rec)
	if err != nil {
		return nil, err
	}
	return &workout, nil
}

// Workout represents the CLI-friendly workout model with parsed timestamps.
type Workout struct {
	ID             string    `json:"id"`
	V1ID           *int64    `json:"v1_id,omitempty"`
	SportID        *int      `json:"sport_id,omitempty"`
	SportName      string    `json:"sport_name"`
	ScoreState     string    `json:"score_state"`
	Start          time.Time `json:"start"`
	End            time.Time `json:"end"`
	TimezoneOffset string    `json:"timezone_offset"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Score          Score     `json:"score"`
}

// Score contains strain, heart-rate, and effort metrics for a workout.
type Score struct {
	Strain              float64       `json:"strain"`
	AverageHeartRate    *int          `json:"average_heart_rate,omitempty"`
	MaxHeartRate        *int          `json:"max_heart_rate,omitempty"`
	Kilojoule           *float64      `json:"kilojoule,omitempty"`
	PercentRecorded     *float64      `json:"percent_recorded,omitempty"`
	DistanceMeter       *float64      `json:"distance_meter,omitempty"`
	AltitudeGainMeter   *float64      `json:"altitude_gain_meter,omitempty"`
	AltitudeChangeMeter *float64      `json:"altitude_change_meter,omitempty"`
	ZoneDurations       ZoneDurations `json:"zone_durations"`
}

// ZoneDurations (milliseconds per WHOOP zone).
type ZoneDurations struct {
	ZoneZeroMilli  *int64 `json:"zone_zero_milli,omitempty"`
	ZoneOneMilli   *int64 `json:"zone_one_milli,omitempty"`
	ZoneTwoMilli   *int64 `json:"zone_two_milli,omitempty"`
	ZoneThreeMilli *int64 `json:"zone_three_milli,omitempty"`
	ZoneFourMilli  *int64 `json:"zone_four_milli,omitempty"`
	ZoneFiveMilli  *int64 `json:"zone_five_milli,omitempty"`
}

type workoutRecord struct {
	ID             string        `json:"id"`
	V1ID           *int64        `json:"v1_id"`
	SportID        *int          `json:"sport_id"`
	SportName      string        `json:"sport_name"`
	ScoreState     string        `json:"score_state"`
	Start          string        `json:"start"`
	End            string        `json:"end"`
	TimezoneOffset string        `json:"timezone_offset"`
	CreatedAt      string        `json:"created_at"`
	UpdatedAt      string        `json:"updated_at"`
	Score          *workoutScore `json:"score"`
}

type workoutScore struct {
	Strain              float64        `json:"strain"`
	AverageHeartRate    *int           `json:"average_heart_rate"`
	MaxHeartRate        *int           `json:"max_heart_rate"`
	Kilojoule           *float64       `json:"kilojoule"`
	PercentRecorded     *float64       `json:"percent_recorded"`
	DistanceMeter       *float64       `json:"distance_meter"`
	AltitudeGainMeter   *float64       `json:"altitude_gain_meter"`
	AltitudeChangeMeter *float64       `json:"altitude_change_meter"`
	ZoneDurations       *zoneDurations `json:"zone_durations"`
}

type zoneDurations struct {
	ZoneZeroMilli  *int64 `json:"zone_zero_milli"`
	ZoneOneMilli   *int64 `json:"zone_one_milli"`
	ZoneTwoMilli   *int64 `json:"zone_two_milli"`
	ZoneThreeMilli *int64 `json:"zone_three_milli"`
	ZoneFourMilli  *int64 `json:"zone_four_milli"`
	ZoneFiveMilli  *int64 `json:"zone_five_milli"`
}

func convertRecord(rec workoutRecord) (Workout, error) {
	start, err := parseTimestamp("start", rec.Start)
	if err != nil {
		return Workout{}, err
	}
	end, err := parseTimestamp("end", rec.End)
	if err != nil {
		return Workout{}, err
	}
	created, err := parseTimestampAllowBlank(rec.CreatedAt)
	if err != nil {
		return Workout{}, fmt.Errorf("parse created_at: %w", err)
	}
	updated, err := parseTimestampAllowBlank(rec.UpdatedAt)
	if err != nil {
		return Workout{}, fmt.Errorf("parse updated_at: %w", err)
	}

	score := Score{}
	if rec.Score != nil {
		score = Score{
			Strain:              rec.Score.Strain,
			AverageHeartRate:    rec.Score.AverageHeartRate,
			MaxHeartRate:        rec.Score.MaxHeartRate,
			Kilojoule:           rec.Score.Kilojoule,
			PercentRecorded:     rec.Score.PercentRecorded,
			DistanceMeter:       rec.Score.DistanceMeter,
			AltitudeGainMeter:   rec.Score.AltitudeGainMeter,
			AltitudeChangeMeter: rec.Score.AltitudeChangeMeter,
			ZoneDurations:       convertZoneDurations(rec.Score.ZoneDurations),
		}
	}

	return Workout{
		ID:             rec.ID,
		V1ID:           rec.V1ID,
		SportID:        rec.SportID,
		SportName:      rec.SportName,
		ScoreState:     rec.ScoreState,
		Start:          start,
		End:            end,
		TimezoneOffset: rec.TimezoneOffset,
		CreatedAt:      created,
		UpdatedAt:      updated,
		Score:          score,
	}, nil
}

func parseTimestamp(field, value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, fmt.Errorf("workout %s is missing", field)
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("workout %s must be RFC3339", field)
}

func parseTimestampAllowBlank(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("value %q must be RFC3339", value)
}

func convertZoneDurations(z *zoneDurations) ZoneDurations {
	if z == nil {
		return ZoneDurations{}
	}
	return ZoneDurations{
		ZoneZeroMilli:  z.ZoneZeroMilli,
		ZoneOneMilli:   z.ZoneOneMilli,
		ZoneTwoMilli:   z.ZoneTwoMilli,
		ZoneThreeMilli: z.ZoneThreeMilli,
		ZoneFourMilli:  z.ZoneFourMilli,
		ZoneFiveMilli:  z.ZoneFiveMilli,
	}
}
