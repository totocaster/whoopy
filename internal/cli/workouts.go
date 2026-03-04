package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/toto/whoopy/internal/api"
	"github.com/toto/whoopy/internal/workouts"
)

var (
	workoutsListFn = defaultWorkoutsList
	workoutsViewFn = defaultWorkoutsView
)

func init() {
	rootCmd.AddCommand(workoutsCmd)
	workoutsCmd.AddCommand(workoutsListCmd)
	workoutsCmd.AddCommand(workoutsViewCmd)
	addListFlags(workoutsListCmd)
	workoutsListCmd.Flags().Bool("text", false, "Human-readable output")
	workoutsViewCmd.Flags().Bool("text", false, "Human-readable output")
}

var workoutsCmd = &cobra.Command{
	Use:   "workouts",
	Short: "Workout-related commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var workoutsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workouts within an optional time range",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := parseListOptions(cmd)
		if err != nil {
			return err
		}
		textMode, err := cmd.Flags().GetBool("text")
		if err != nil {
			return err
		}
		result, err := workoutsListFn(cmd.Context(), opts)
		if err != nil {
			return err
		}
		if textMode {
			fmt.Fprintln(cmd.OutOrStdout(), formatWorkoutsText(result))
			return nil
		}
		payload, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(payload))
		return nil
	},
}

var workoutsViewCmd = &cobra.Command{
	Use:   "view <workout-id>",
	Short: "Show a single workout by WHOOP ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		textMode, err := cmd.Flags().GetBool("text")
		if err != nil {
			return err
		}
		workout, err := workoutsViewFn(cmd.Context(), id)
		if err != nil {
			return err
		}
		if textMode {
			fmt.Fprintln(cmd.OutOrStdout(), formatWorkoutDetailText(workout))
			return nil
		}
		payload, err := json.MarshalIndent(workout, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(payload))
		return nil
	},
}

func defaultWorkoutsList(ctx context.Context, opts *api.ListOptions) (*workouts.ListResult, error) {
	client, err := apiClientFactory()
	if err != nil {
		return nil, err
	}
	service := workouts.NewService(client)
	return service.List(ctx, opts)
}

func defaultWorkoutsView(ctx context.Context, id string) (*workouts.Workout, error) {
	client, err := apiClientFactory()
	if err != nil {
		return nil, err
	}
	service := workouts.NewService(client)
	return service.Get(ctx, id)
}

func formatWorkoutsText(result *workouts.ListResult) string {
	if result == nil || len(result.Workouts) == 0 {
		if result != nil && strings.TrimSpace(result.NextToken) != "" {
			return fmt.Sprintf("No workouts found. Next cursor: %s", result.NextToken)
		}
		return "No workouts found."
	}
	var b strings.Builder
	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "Start\tDuration\tSport\tStrain\tAvg HR\tID")
	for _, w := range result.Workouts {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			formatWorkoutStart(w),
			formatWorkoutDuration(w),
			w.SportName,
			formatWorkoutStrain(w),
			formatWorkoutHeartRate(w),
			w.ID,
		)
	}
	tw.Flush()
	if strings.TrimSpace(result.NextToken) != "" {
		fmt.Fprintf(&b, "\nNext cursor: %s", result.NextToken)
	}
	return strings.TrimSpace(b.String())
}

func formatWorkoutStart(w workouts.Workout) string {
	if w.Start.IsZero() {
		return "n/a"
	}
	return w.Start.Local().Format("2006-01-02 15:04")
}

func formatWorkoutDuration(w workouts.Workout) string {
	dur := w.End.Sub(w.Start)
	if dur <= 0 {
		return "n/a"
	}
	dur = dur.Round(time.Second)
	hours := int(dur.Hours())
	minutes := int(dur.Minutes()) % 60
	seconds := int(dur.Seconds()) % 60
	switch {
	case hours > 0:
		return fmt.Sprintf("%dh%02dm", hours, minutes)
	case minutes > 0:
		if seconds > 0 {
			return fmt.Sprintf("%dm%02ds", minutes, seconds)
		}
		return fmt.Sprintf("%dm", minutes)
	default:
		return fmt.Sprintf("%ds", seconds)
	}
}

func formatWorkoutStrain(w workouts.Workout) string {
	if w.Score.Strain <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f", w.Score.Strain)
}

func formatWorkoutHeartRate(w workouts.Workout) string {
	if w.Score.AverageHeartRate == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *w.Score.AverageHeartRate)
}

func formatWorkoutDetailText(w *workouts.Workout) string {
	if w == nil {
		return "Workout not found."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "ID: %s\n", w.ID)
	fmt.Fprintf(&b, "Sport: %s\n", safeValue(w.SportName))
	fmt.Fprintf(&b, "State: %s\n", safeValue(w.ScoreState))
	fmt.Fprintf(&b, "Start: %s\n", formatWorkoutStart(*w))
	fmt.Fprintf(&b, "Duration: %s\n", formatWorkoutDuration(*w))
	fmt.Fprintf(&b, "Strain: %s\n", formatWorkoutStrain(*w))
	fmt.Fprintf(&b, "Avg HR: %s bpm\n", formatWorkoutHeartRate(*w))
	fmt.Fprintf(&b, "Max HR: %s bpm\n", formatIntPtr(w.Score.MaxHeartRate))
	fmt.Fprintf(&b, "Kilojoule: %s kJ\n", formatFloatPtr(w.Score.Kilojoule, 1))
	fmt.Fprintf(&b, "Distance: %s m\n", formatFloatPtr(w.Score.DistanceMeter, 1))
	fmt.Fprintf(&b, "Percent Recorded: %s%%\n", formatPercent(w.Score.PercentRecorded))
	fmt.Fprintf(&b, "Created At: %s\n", formatTimestamp(w.CreatedAt))
	fmt.Fprintf(&b, "Updated At: %s\n", formatTimestamp(w.UpdatedAt))
	fmt.Fprintf(&b, "Timezone Offset: %s\n", safeValue(w.TimezoneOffset))
	fmt.Fprintf(&b, "Zone Durations (ms): Z0=%s Z1=%s Z2=%s Z3=%s Z4=%s Z5=%s\n",
		formatInt64Ptr(w.Score.ZoneDurations.ZoneZeroMilli),
		formatInt64Ptr(w.Score.ZoneDurations.ZoneOneMilli),
		formatInt64Ptr(w.Score.ZoneDurations.ZoneTwoMilli),
		formatInt64Ptr(w.Score.ZoneDurations.ZoneThreeMilli),
		formatInt64Ptr(w.Score.ZoneDurations.ZoneFourMilli),
		formatInt64Ptr(w.Score.ZoneDurations.ZoneFiveMilli),
	)
	return strings.TrimSpace(b.String())
}

func safeValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "n/a"
	}
	return value
}

func formatIntPtr(v *int) string {
	if v == nil {
		return "n/a"
	}
	return fmt.Sprintf("%d", *v)
}

func formatFloatPtr(v *float64, precision int) string {
	if v == nil {
		return "n/a"
	}
	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, *v)
}

func formatPercent(v *float64) string {
	if v == nil {
		return "n/a"
	}
	return fmt.Sprintf("%.1f", *v*100)
}

func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return "n/a"
	}
	return t.Local().Format(time.RFC1123)
}

func formatInt64Ptr(v *int64) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *v)
}
