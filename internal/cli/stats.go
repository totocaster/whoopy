package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/toto/whoopy/internal/stats"
)

var statsDailyFn = defaultStatsDaily

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.AddCommand(statsDailyCmd)
	statsDailyCmd.Flags().String("date", "", "Target date (YYYY-MM-DD, default today)")
	statsDailyCmd.Flags().Bool("text", false, "Human-readable output")
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Aggregated WHOOP statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var statsDailyCmd = &cobra.Command{
	Use:   "daily",
	Short: "Show workouts, recovery, sleep, and cycles for a single day",
	RunE: func(cmd *cobra.Command, args []string) error {
		dateValue, err := cmd.Flags().GetString("date")
		if err != nil {
			return err
		}
		targetDate, err := parseStatsDate(dateValue)
		if err != nil {
			return err
		}
		textMode, err := cmd.Flags().GetBool("text")
		if err != nil {
			return err
		}
		report, err := statsDailyFn(cmd.Context(), targetDate)
		if err != nil {
			return err
		}
		if textMode {
			fmt.Fprintln(cmd.OutOrStdout(), formatStatsText(report))
			return nil
		}
		payload, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(payload))
		return nil
	},
}

func parseStatsDate(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), nil
	}
	date, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --date: expected YYYY-MM-DD")
	}
	local := date.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location()), nil
}

func defaultStatsDaily(ctx context.Context, date time.Time) (*stats.DailyReport, error) {
	client, err := apiClientFactory()
	if err != nil {
		return nil, err
	}
	service := stats.NewService(client)
	return service.Daily(ctx, date)
}

func formatStatsText(report *stats.DailyReport) string {
	if report == nil {
		return "No stats available."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "WHOOP Stats – %s\n", report.Date)
	fmt.Fprintf(&b, "Summary: \n")
	fmt.Fprintf(&b, "  Cycle Strain: %s\n", formatFloatPtr(report.Summary.CycleStrain, 1))
	fmt.Fprintf(&b, "  Recovery Score: %s\n", formatFloatPtr(report.Summary.RecoveryScore, 0))
	fmt.Fprintf(&b, "  Sleep Performance: %s %%\n", formatPercent(report.Summary.SleepPerformance))
	fmt.Fprintf(&b, "  Total Sleep: %.1f h\n", report.Summary.TotalSleepHours)
	fmt.Fprintf(&b, "  Workouts: %d (total strain %.1f)\n\n", report.Summary.WorkoutCount, report.Summary.TotalWorkoutStrain)

	fmt.Fprintf(&b, "Cycle:\n")
	if report.Cycle != nil {
		fmt.Fprintf(&b, "  Strain: %.1f\n", report.Cycle.Score.Strain)
		fmt.Fprintf(&b, "  Start: %s\n", formatTimestamp(report.Cycle.Start))
		fmt.Fprintf(&b, "  End: %s\n\n", formatTimestamp(report.Cycle.End))
	} else {
		fmt.Fprintf(&b, "  n/a\n\n")
	}

	fmt.Fprintf(&b, "Recovery:\n")
	if report.Recovery != nil {
		fmt.Fprintf(&b, "  Score: %s\n", formatFloatPtr(report.Recovery.Score.RecoveryScore, 0))
		fmt.Fprintf(&b, "  Resting HR: %s bpm\n", formatFloatPtr(report.Recovery.Score.RestingHeartRate, 0))
		fmt.Fprintf(&b, "  Respiratory Rate: %s br/min\n\n", formatFloatPtr(report.Recovery.Score.RespiratoryRate, 1))
	} else {
		fmt.Fprintf(&b, "  n/a\n\n")
	}

	fmt.Fprintf(&b, "Sleep Sessions (%d):\n", len(report.Sleep))
	if len(report.Sleep) == 0 {
		fmt.Fprintf(&b, "  n/a\n\n")
	} else {
		for _, sess := range report.Sleep {
			fmt.Fprintf(&b, "  %s – %s (%s) perf=%s%%\n",
				formatTimestamp(sess.Start),
				formatTimestamp(sess.End),
				formatDuration(sess.Start, sess.End),
				formatFloatPtr(sess.Score.SleepPerformancePercentage, 0))
		}
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprintf(&b, "Workouts (%d):\n", len(report.Workouts))
	if len(report.Workouts) == 0 {
		fmt.Fprintf(&b, "  n/a")
	} else {
		for _, w := range report.Workouts {
			fmt.Fprintf(&b, "  %s – %s (%s) %s strain=%.1f\n",
				formatTimestamp(w.Start),
				formatTimestamp(w.End),
				formatDuration(w.Start, w.End),
				safeValue(w.SportName),
				w.Score.Strain)
		}
	}
	return strings.TrimSpace(b.String())
}
