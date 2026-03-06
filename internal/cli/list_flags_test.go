package cli

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestParseListOptionsSuccess(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	addListFlags(cmd)
	require.NoError(t, cmd.Flags().Set("start", "2026-03-01"))
	require.NoError(t, cmd.Flags().Set("end", "2026-03-02T10:00:00Z"))
	require.NoError(t, cmd.Flags().Set("limit", "50"))
	require.NoError(t, cmd.Flags().Set("cursor", "abc"))

	opts, err := parseListOptions(cmd)
	require.NoError(t, err)
	require.NotNil(t, opts.Start)
	require.Equal(t, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), opts.Start.UTC())
	require.NotNil(t, opts.End)
	require.Equal(t, time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC), opts.End.UTC())
	require.Equal(t, 50, opts.Limit)
	require.Equal(t, "abc", opts.NextToken)
}

func TestParseListOptionsErrors(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	addListFlags(cmd)
	require.NoError(t, cmd.Flags().Set("start", "not-a-date"))
	_, err := parseListOptions(cmd)
	require.Error(t, err)

	cmd = &cobra.Command{Use: "test"}
	addListFlags(cmd)
	require.NoError(t, cmd.Flags().Set("start", "2026-03-02T00:00:00Z"))
	require.NoError(t, cmd.Flags().Set("end", "2026-03-01T00:00:00Z"))
	_, err = parseListOptions(cmd)
	require.Error(t, err)

	cmd = &cobra.Command{Use: "test"}
	addListFlags(cmd)
	require.NoError(t, cmd.Flags().Set("limit", "-1"))
	_, err = parseListOptions(cmd)
	require.Error(t, err)

	cmd = &cobra.Command{Use: "test"}
	addListFlags(cmd)
	require.NoError(t, cmd.Flags().Set("updated-since", "2026-03-01T00:00:00Z"))
	_, err = parseListOptions(cmd)
	require.Error(t, err)
}

func TestParseListOptionsSinceUntilAliases(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	addListFlags(cmd)
	require.NoError(t, cmd.Flags().Set("since", "2026-03-01"))
	require.NoError(t, cmd.Flags().Set("until", "2026-03-02T10:00:00Z"))

	opts, err := parseListOptions(cmd)
	require.NoError(t, err)
	require.NotNil(t, opts.Start)
	require.Equal(t, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), opts.Start.UTC())
	require.NotNil(t, opts.End)
	require.Equal(t, time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC), opts.End.UTC())
}

func TestParseListOptionsLast(t *testing.T) {
	origNow := nowFunc
	nowFunc = func() time.Time {
		return time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)
	}
	defer func() { nowFunc = origNow }()

	cmd := &cobra.Command{Use: "test"}
	addListFlags(cmd)
	require.NoError(t, cmd.Flags().Set("last", "3d"))

	opts, err := parseListOptions(cmd)
	require.NoError(t, err)
	require.NotNil(t, opts.Start)
	require.NotNil(t, opts.End)
	require.Equal(t, time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC), opts.Start.UTC())
	require.Equal(t, time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC), opts.End.UTC())
}
