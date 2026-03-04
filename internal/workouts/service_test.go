package workouts

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/toto/whoopy/internal/api"
)

func TestServiceListAppliesOptionsAndParses(t *testing.T) {
	response := struct {
		Records   []workoutRecord `json:"records"`
		NextToken string          `json:"next_token"`
	}{
		Records: []workoutRecord{
			{
				ID:             "w1",
				SportName:      "Running",
				SportID:        intPtr(1),
				ScoreState:     "SCORED",
				Start:          "2026-03-04T00:00:00Z",
				End:            "2026-03-04T00:45:00Z",
				TimezoneOffset: "-05:00",
				CreatedAt:      "2026-03-04T01:00:00Z",
				UpdatedAt:      "2026-03-04T01:10:00Z",
				Score: &workoutScore{
					Strain:            12.3,
					AverageHeartRate:  intPtr(140),
					MaxHeartRate:      intPtr(165),
					Kilojoule:         floatPtr(500),
					PercentRecorded:   floatPtr(0.98),
					DistanceMeter:     floatPtr(8000),
					AltitudeGainMeter: floatPtr(120),
					ZoneDurations: &zoneDurations{
						ZoneZeroMilli: int64Ptr(1000),
					},
				},
			},
		},
		NextToken: "cursor123",
	}

	fake := &fakeClient{response: response}
	svc := &Service{client: fake}
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	opts := &api.ListOptions{Start: &start, Limit: 25}

	result, err := svc.List(context.Background(), opts)
	require.NoError(t, err)
	require.Equal(t, workoutsPath, fake.lastPath)
	require.Equal(t, "cursor123", result.NextToken)
	require.True(t, result.HasNext())
	require.Equal(t, start.Format(time.RFC3339Nano), fake.lastQuery.Get("start"))
	require.Equal(t, "25", fake.lastQuery.Get("limit"))
	require.Len(t, result.Workouts, 1)
	workout := result.Workouts[0]
	require.Equal(t, "w1", workout.ID)
	require.Equal(t, "Running", workout.SportName)
	require.Equal(t, "SCORED", workout.ScoreState)
	require.Equal(t, time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC), workout.Start)
	require.Equal(t, 140, *workout.Score.AverageHeartRate)
	require.Equal(t, float64(500), *workout.Score.Kilojoule)
	require.Equal(t, int64(1000), *workout.Score.ZoneDurations.ZoneZeroMilli)
}

func TestServiceListFailsOnInvalidTimestamp(t *testing.T) {
	response := struct {
		Records   []workoutRecord `json:"records"`
		NextToken string          `json:"next_token"`
	}{
		Records: []workoutRecord{{ID: "w1", SportName: "Run", ScoreState: "SCORED", Start: "bad", End: "2026-03-04T00:00:00Z"}},
	}
	fake := &fakeClient{response: response}
	svc := &Service{client: fake}
	_, err := svc.List(context.Background(), &api.ListOptions{})
	require.Error(t, err)
}

func TestServiceListValidatesOptions(t *testing.T) {
	svc := &Service{client: &fakeClient{}}
	opts := &api.ListOptions{Limit: -1}
	_, err := svc.List(context.Background(), opts)
	require.Error(t, err)
}

func TestServiceGetFetchesWorkout(t *testing.T) {
	rec := workoutRecord{
		ID:         "w123",
		SportName:  "Rowing",
		ScoreState: "SCORED",
		Start:      "2026-03-03T01:00:00Z",
		End:        "2026-03-03T01:30:00Z",
	}
	fake := &fakeClient{response: rec}
	svc := &Service{client: fake}
	got, err := svc.Get(context.Background(), "w123")
	require.NoError(t, err)
	require.Equal(t, workoutsPath+"/w123", fake.lastPath)
	require.Equal(t, "w123", got.ID)
	require.Equal(t, "Rowing", got.SportName)
}

func TestServiceGetRequiresID(t *testing.T) {
	svc := &Service{client: &fakeClient{}}
	_, err := svc.Get(context.Background(), "")
	require.Error(t, err)
}

type fakeClient struct {
	response  any
	err       error
	lastPath  string
	lastQuery url.Values
}

func (f *fakeClient) GetJSON(ctx context.Context, path string, query url.Values, dest any) error {
	f.lastPath = path
	f.lastQuery = query
	if f.err != nil {
		return f.err
	}
	data, err := json.Marshal(f.response)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func intPtr(v int) *int           { return &v }
func int64Ptr(v int64) *int64     { return &v }
func floatPtr(v float64) *float64 { return &v }
