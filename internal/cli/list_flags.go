package cli

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/toto/whoopy/internal/api"
)

const (
	listFlagStart        = "start"
	listFlagEnd          = "end"
	listFlagLimit        = "limit"
	listFlagCursor       = "cursor"
	listFlagSince        = "since"
	listFlagUntil        = "until"
	listFlagLast         = "last"
	listFlagUpdatedSince = "updated-since"
)

var (
	relativeWindowPattern = regexp.MustCompile(`^(\d+)(h|d|w|mo)$`)
	nowFunc               = time.Now
)

func addListFlags(cmd *cobra.Command) {
	cmd.Flags().String(listFlagStart, "", "Start timestamp (RFC3339 or YYYY-MM-DD, UTC if date only)")
	cmd.Flags().String(listFlagEnd, "", "End timestamp (RFC3339 or YYYY-MM-DD, UTC if date only)")
	cmd.Flags().Int(listFlagLimit, 0, "Maximum records to return (0 leaves WHOOP default)")
	cmd.Flags().String(listFlagCursor, "", "Opaque cursor token to resume pagination")
	cmd.Flags().String(listFlagSince, "", "Alias for --start; useful for bounded date ranges")
	cmd.Flags().String(listFlagUntil, "", "Alias for --end; useful for bounded date ranges")
	cmd.Flags().String(listFlagLast, "", "Relative date window such as 3d, 10d, or 1mo")
	cmd.Flags().String(listFlagUpdatedSince, "", "Requested update-time lower bound (not supported by WHOOP; use --since)")
}

func parseListOptions(cmd *cobra.Command) (*api.ListOptions, error) {
	startVal, err := getOptionalStringFlag(cmd, listFlagStart)
	if err != nil {
		return nil, err
	}
	endVal, err := getOptionalStringFlag(cmd, listFlagEnd)
	if err != nil {
		return nil, err
	}
	limit, err := getOptionalIntFlag(cmd, listFlagLimit)
	if err != nil {
		return nil, err
	}
	cursor, err := getOptionalStringFlag(cmd, listFlagCursor)
	if err != nil {
		return nil, err
	}
	sinceVal, err := getOptionalStringFlag(cmd, listFlagSince)
	if err != nil {
		return nil, err
	}
	untilVal, err := getOptionalStringFlag(cmd, listFlagUntil)
	if err != nil {
		return nil, err
	}
	lastVal, err := getOptionalStringFlag(cmd, listFlagLast)
	if err != nil {
		return nil, err
	}
	updatedSinceVal, err := getOptionalStringFlag(cmd, listFlagUpdatedSince)
	if err != nil {
		return nil, err
	}

	opts := &api.ListOptions{Limit: limit}
	if strings.TrimSpace(cursor) != "" {
		opts.NextToken = cursor
	}

	if strings.TrimSpace(updatedSinceVal) != "" {
		return nil, fmt.Errorf("--%s is not supported by WHOOP; use --%s or --%s", listFlagUpdatedSince, listFlagSince, listFlagStart)
	}
	if strings.TrimSpace(startVal) != "" && strings.TrimSpace(sinceVal) != "" {
		return nil, fmt.Errorf("--%s cannot be combined with --%s", listFlagStart, listFlagSince)
	}
	if strings.TrimSpace(endVal) != "" && strings.TrimSpace(untilVal) != "" {
		return nil, fmt.Errorf("--%s cannot be combined with --%s", listFlagEnd, listFlagUntil)
	}

	startInput := firstNonEmpty(startVal, sinceVal)
	endInput := firstNonEmpty(endVal, untilVal)
	if strings.TrimSpace(lastVal) != "" {
		if strings.TrimSpace(startInput) != "" {
			return nil, fmt.Errorf("--%s cannot be combined with --%s or --%s", listFlagLast, listFlagStart, listFlagSince)
		}
		dur, err := parseRelativeWindow(lastVal)
		if err != nil {
			return nil, fmt.Errorf("invalid --%s: %w", listFlagLast, err)
		}
		end := nowFunc().UTC()
		if strings.TrimSpace(endInput) != "" {
			end, err = parseTimeFlag(endInput)
			if err != nil {
				return nil, fmt.Errorf("invalid --%s: %w", listFlagUntil, err)
			}
		}
		start := end.Add(-dur)
		opts.Start = &start
		opts.End = &end
	} else {
		if strings.TrimSpace(startInput) != "" {
			ts, err := parseTimeFlag(startInput)
			if err != nil {
				flagName := listFlagStart
				if strings.TrimSpace(sinceVal) != "" {
					flagName = listFlagSince
				}
				return nil, fmt.Errorf("invalid --%s: %w", flagName, err)
			}
			opts.Start = &ts
		}
		if strings.TrimSpace(endInput) != "" {
			ts, err := parseTimeFlag(endInput)
			if err != nil {
				flagName := listFlagEnd
				if strings.TrimSpace(untilVal) != "" {
					flagName = listFlagUntil
				}
				return nil, fmt.Errorf("invalid --%s: %w", flagName, err)
			}
			opts.End = &ts
		}
	}

	if err := opts.Validate(); err != nil {
		return nil, err
	}
	return opts, nil
}

func getOptionalStringFlag(cmd *cobra.Command, name string) (string, error) {
	if cmd == nil || cmd.Flags().Lookup(name) == nil {
		return "", nil
	}
	return cmd.Flags().GetString(name)
}

func getOptionalIntFlag(cmd *cobra.Command, name string) (int, error) {
	if cmd == nil || cmd.Flags().Lookup(name) == nil {
		return 0, nil
	}
	return cmd.Flags().GetInt(name)
}

var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02",
}

func parseTimeFlag(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, trimmed); err == nil {
			if layout == "2006-01-02" {
				return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
			}
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("%q must be RFC3339 or YYYY-MM-DD", value)
}

func parseRelativeWindow(value string) (time.Duration, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	matches := relativeWindowPattern.FindStringSubmatch(trimmed)
	if len(matches) != 3 {
		return 0, fmt.Errorf("%q must use h, d, w, or mo units", value)
	}
	count, err := strconv.Atoi(matches[1])
	if err != nil || count <= 0 {
		return 0, fmt.Errorf("%q must be a positive duration", value)
	}
	switch matches[2] {
	case "h":
		return time.Duration(count) * time.Hour, nil
	case "d":
		return time.Duration(count) * 24 * time.Hour, nil
	case "w":
		return time.Duration(count) * 7 * 24 * time.Hour, nil
	case "mo":
		return time.Duration(count) * 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("%q must use h, d, w, or mo units", value)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
