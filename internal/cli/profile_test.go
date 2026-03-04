package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/toto/whoopy/internal/profile"
)

func TestProfileShowJSON(t *testing.T) {
	orig := profileFetchFn
	defer func() { profileFetchFn = orig }()

	summary := &profile.Summary{
		UserID:         "user-1",
		Name:           "Ada Lovelace",
		Email:          "ada@example.com",
		Locale:         "en_US",
		Timezone:       "America/New_York",
		MembershipTier: "pro",
	}
	profileFetchFn = func(_ context.Context) (*profile.Summary, error) { return summary, nil }

	output := runCLICommand(t, []string{"profile", "show"}, "")
	require.Contains(t, output, `"user_id": "user-1"`)
	require.Contains(t, output, `"name": "Ada Lovelace"`)
}

func TestProfileShowText(t *testing.T) {
	orig := profileFetchFn
	defer func() { profileFetchFn = orig }()

	height := 170.0
	heightIn := 66.9
	weightKg := 65.0
	weightLb := 143.3
	maxHR := 185
	ts := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	profileFetchFn = func(_ context.Context) (*profile.Summary, error) {
		return &profile.Summary{
			Name:           "Ada Lovelace",
			Email:          "ada@example.com",
			UserID:         "user-1",
			Locale:         "en_US",
			Timezone:       "America/New_York",
			MembershipTier: "pro",
			HeightCm:       &height,
			HeightIn:       &heightIn,
			WeightKg:       &weightKg,
			WeightLb:       &weightLb,
			MaxHeartRate:   &maxHR,
			UpdatedAt:      &ts,
		}, nil
	}

	output := runCLICommand(t, []string{"profile", "show", "--text"}, "")
	require.Contains(t, output, "Ada Lovelace")
	require.Contains(t, output, "Height: 170.0 cm / 66.9 in")
	require.Contains(t, output, "Weight: 65.0 kg / 143.3 lb")
	require.Contains(t, output, "Max HR: 185 bpm")
}
