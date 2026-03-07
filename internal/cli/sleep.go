package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/toto/whoopy/internal/api"
	"github.com/toto/whoopy/internal/sleep"
)

var (
	sleepListFn = defaultSleepList
	sleepViewFn = defaultSleepView
)

func init() {
	rootCmd.AddCommand(sleepCmd)
	sleepCmd.AddCommand(sleepListCmd)
	sleepCmd.AddCommand(sleepTodayCmd)
	sleepCmd.AddCommand(sleepViewCmd)
	addListFlags(sleepListCmd)
	sleepListCmd.Flags().Bool("text", false, "Human-readable output")
	sleepTodayCmd.Flags().Bool("text", false, "Human-readable output")
	sleepViewCmd.Flags().Bool("text", false, "Human-readable output")
}

var sleepCmd = &cobra.Command{
	Use:   "sleep",
	Short: "Sleep-related commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var sleepListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sleep sessions within an optional time range",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := parseListOptions(cmd)
		if err != nil {
			return err
		}
		textMode, err := cmd.Flags().GetBool("text")
		if err != nil {
			return err
		}
		result, err := sleepListFn(cmd.Context(), opts)
		if err != nil {
			return err
		}
		if textMode {
			fmt.Fprintln(cmd.OutOrStdout(), formatSleepListText(result))
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

var sleepTodayCmd = &cobra.Command{
	Use:   "today",
	Short: "Show today's sleep sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := todayRangeOptions(25)
		textMode, err := cmd.Flags().GetBool("text")
		if err != nil {
			return err
		}
		result, err := sleepListFn(cmd.Context(), opts)
		if err != nil {
			return err
		}
		if textMode {
			fmt.Fprintln(cmd.OutOrStdout(), formatSleepListText(result))
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

var sleepViewCmd = &cobra.Command{
	Use:   "view <sleep-id>",
	Short: "Show a single sleep session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		textMode, err := cmd.Flags().GetBool("text")
		if err != nil {
			return err
		}
		sess, err := sleepViewFn(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		if textMode {
			fmt.Fprintln(cmd.OutOrStdout(), formatSleepDetailText(sess))
			return nil
		}
		payload, err := json.MarshalIndent(sess, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(payload))
		return nil
	},
}

func defaultSleepList(ctx context.Context, opts *api.ListOptions) (*sleep.ListResult, error) {
	client, err := apiClientFactory()
	if err != nil {
		return nil, err
	}
	service := sleep.NewService(client)
	return service.List(ctx, opts)
}

func defaultSleepView(ctx context.Context, id string) (*sleep.Session, error) {
	client, err := apiClientFactory()
	if err != nil {
		return nil, err
	}
	service := sleep.NewService(client)
	return service.Get(ctx, id)
}

func formatSleepListText(result *sleep.ListResult) string {
	if result == nil || len(result.Sleeps) == 0 {
		if result != nil && strings.TrimSpace(result.NextToken) != "" {
			return fmt.Sprintf("No sleep sessions found. Next cursor: %s", result.NextToken)
		}
		return "No sleep sessions found."
	}
	var b strings.Builder
	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "Start\tDuration\tPerf%\tResp\tNap\tID")
	for _, sess := range result.Sleeps {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			formatTimestamp(sess.Start),
			formatDuration(sess.Start, sess.End),
			formatFloatPtr(sess.Score.SleepPerformancePercentage, 1),
			formatFloatPtr(sess.Score.RespiratoryRate, 1),
			formatBool(sess.Nap),
			sess.ID,
		)
	}
	tw.Flush()
	if strings.TrimSpace(result.NextToken) != "" {
		fmt.Fprintf(&b, "\nNext cursor: %s", result.NextToken)
	}
	return strings.TrimSpace(b.String())
}

func formatSleepDetailText(sess *sleep.Session) string {
	if sess == nil {
		return "Sleep session not found."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "ID: %s\n", sess.ID)
	fmt.Fprintf(&b, "Cycle ID: %d\n", sess.CycleID)
	fmt.Fprintf(&b, "State: %s\n", safeValue(sess.ScoreState))
	fmt.Fprintf(&b, "Start: %s\n", formatTimestamp(sess.Start))
	fmt.Fprintf(&b, "End: %s\n", formatTimestamp(sess.End))
	fmt.Fprintf(&b, "Duration: %s\n", formatDuration(sess.Start, sess.End))
	fmt.Fprintf(&b, "Nap: %s\n", formatBool(sess.Nap))
	fmt.Fprintf(&b, "Performance: %s %%\n", formatFloatPtr(sess.Score.SleepPerformancePercentage, 1))
	fmt.Fprintf(&b, "Consistency: %s %%\n", formatFloatPtr(sess.Score.SleepConsistencyPercentage, 1))
	fmt.Fprintf(&b, "Respiratory Rate: %s br/min\n", formatFloatPtr(sess.Score.RespiratoryRate, 1))
	fmt.Fprintf(&b, "Efficiency: %s %%\n", formatFloatPtr(sess.Score.SleepEfficiencyPercentage, 1))
	fmt.Fprintf(&b, "Stage Summary:\n")
	fmt.Fprintf(&b, "  In Bed: %s\n", formatMillisDuration(sess.Score.StageSummary.TotalInBedTimeMilli))
	fmt.Fprintf(&b, "  Awake: %s\n", formatMillisDuration(sess.Score.StageSummary.TotalAwakeTimeMilli))
	fmt.Fprintf(&b, "  Light: %s\n", formatMillisDuration(sess.Score.StageSummary.TotalLightSleepTimeMilli))
	fmt.Fprintf(&b, "  Slow Wave: %s\n", formatMillisDuration(sess.Score.StageSummary.TotalSlowWaveSleepTimeMilli))
	fmt.Fprintf(&b, "  REM: %s\n", formatMillisDuration(sess.Score.StageSummary.TotalRemSleepTimeMilli))
	fmt.Fprintf(&b, "Timezone Offset: %s\n", safeValue(sess.TimezoneOffset))
	fmt.Fprintf(&b, "Created At: %s\n", formatTimestamp(sess.CreatedAt))
	fmt.Fprintf(&b, "Updated At: %s", formatTimestamp(sess.UpdatedAt))
	return strings.TrimSpace(b.String())
}
