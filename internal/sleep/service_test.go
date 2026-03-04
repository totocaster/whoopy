package sleep

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/toto/whoopy/internal/api"
)

func TestServiceListParsesSleep(t *testing.T) {
	response := struct {
		Records   []sessionRecord `json:"records"`
		NextToken string          `json:"next_token"`
	}{
		Records: []sessionRecord{
			{
				ID:         "sleep-1",
				CycleID:    200,
				ScoreState: "SCORED",
				Start:      "2026-03-04T00:00:00Z",
				End:        "2026-03-04T07:45:00Z",
				CreatedAt:  "2026-03-04T00:05:00Z",
				UpdatedAt:  "2026-03-04T08:00:00Z",
				Score: &sleepScore{
					SleepPerformancePercentage: intPtr(92),
					RespiratoryRate:            floatPtr(14.5),
					StageSummary: &stageSummary{
						TotalInBedTimeMilli:      int64Ptr(28000000),
						TotalAwakeTimeMilli:      int64Ptr(1500000),
						TotalLightSleepTimeMilli: int64Ptr(13000000),
					},
				},
			},
		},
		NextToken: "cursor",
	}

	fake := &fakeClient{response: response}
	svc := &Service{client: fake}
	result, err := svc.List(context.Background(), &api.ListOptions{})
	require.NoError(t, err)
	require.Equal(t, sleepPath, fake.lastPath)
	require.Len(t, result.Sleeps, 1)
	sess := result.Sleeps[0]
	require.Equal(t, "sleep-1", sess.ID)
	require.Equal(t, float64(14.5), *sess.Score.RespiratoryRate)
	require.NotNil(t, sess.Score.StageSummary.TotalLightSleepTimeMilli)
}

func TestServiceGet(t *testing.T) {
	record := sessionRecord{ID: "sleep-2", Start: "2026-03-03T00:00:00Z", End: "2026-03-03T06:00:00Z"}
	fake := &fakeClient{response: record}
	svc := &Service{client: fake}
	sess, err := svc.Get(context.Background(), "sleep-2")
	require.NoError(t, err)
	require.Equal(t, sleepPath+"/sleep-2", fake.lastPath)
	require.Equal(t, "sleep-2", sess.ID)
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
