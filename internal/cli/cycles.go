package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/toto/whoopy/internal/api"
	"github.com/toto/whoopy/internal/cycles"
)

var (
	cyclesListFn = defaultCyclesList
	cyclesViewFn = defaultCyclesView
)

func init() {
	rootCmd.AddCommand(cyclesCmd)
	cyclesCmd.AddCommand(cyclesListCmd)
	cyclesCmd.AddCommand(cyclesViewCmd)
	addListFlags(cyclesListCmd)
	cyclesListCmd.Flags().Bool("text", false, "Human-readable output")
	cyclesViewCmd.Flags().Bool("text", false, "Human-readable output")
}

var cyclesCmd = &cobra.Command{
	Use:   "cycles",
	Short: "Cycle-related commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var cyclesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cycles within an optional time range",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := parseListOptions(cmd)
		if err != nil {
			return err
		}
		textMode, err := cmd.Flags().GetBool("text")
		if err != nil {
			return err
		}
		result, err := cyclesListFn(cmd.Context(), opts)
		if err != nil {
			return err
		}
		if textMode {
			fmt.Fprintln(cmd.OutOrStdout(), formatCyclesText(result))
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

var cyclesViewCmd = &cobra.Command{
	Use:   "view <cycle-id>",
	Short: "Show a single cycle by WHOOP ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		textMode, err := cmd.Flags().GetBool("text")
		if err != nil {
			return err
		}
		cycle, err := cyclesViewFn(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		if textMode {
			fmt.Fprintln(cmd.OutOrStdout(), formatCycleDetailText(cycle))
			return nil
		}
		payload, err := json.MarshalIndent(cycle, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(payload))
		return nil
	},
}

func defaultCyclesList(ctx context.Context, opts *api.ListOptions) (*cycles.ListResult, error) {
	client, err := apiClientFactory()
	if err != nil {
		return nil, err
	}
	service := cycles.NewService(client)
	return service.List(ctx, opts)
}

func defaultCyclesView(ctx context.Context, id string) (*cycles.Cycle, error) {
	client, err := apiClientFactory()
	if err != nil {
		return nil, err
	}
	service := cycles.NewService(client)
	return service.Get(ctx, id)
}

func formatCyclesText(result *cycles.ListResult) string {
	if result == nil || len(result.Cycles) == 0 {
		if result != nil && strings.TrimSpace(result.NextToken) != "" {
			return fmt.Sprintf("No cycles found. Next cursor: %s", result.NextToken)
		}
		return "No cycles found."
	}
	var b strings.Builder
	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "Start\tDuration\tStrain\tAvg HR\tMax HR\tID")
	for _, c := range result.Cycles {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%d\n",
			formatCycleStart(c),
			formatDuration(c.Start, c.End),
			formatCycleStrain(c),
			formatIntPtr(c.Score.AverageHeartRate),
			formatIntPtr(c.Score.MaxHeartRate),
			c.ID,
		)
	}
	tw.Flush()
	if strings.TrimSpace(result.NextToken) != "" {
		fmt.Fprintf(&b, "\nNext cursor: %s", result.NextToken)
	}
	return strings.TrimSpace(b.String())
}

func formatCycleStart(c cycles.Cycle) string {
	if c.Start.IsZero() {
		return "n/a"
	}
	return c.Start.Local().Format("2006-01-02")
}

func formatCycleStrain(c cycles.Cycle) string {
	if c.Score.Strain <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f", c.Score.Strain)
}

func formatCycleDetailText(cycle *cycles.Cycle) string {
	if cycle == nil {
		return "Cycle not found."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "ID: %d\n", cycle.ID)
	if cycle.UserID != nil {
		fmt.Fprintf(&b, "User ID: %d\n", *cycle.UserID)
	}
	fmt.Fprintf(&b, "State: %s\n", safeValue(cycle.ScoreState))
	fmt.Fprintf(&b, "Start: %s\n", formatTimestamp(cycle.Start))
	fmt.Fprintf(&b, "End: %s\n", formatTimestamp(cycle.End))
	fmt.Fprintf(&b, "Duration: %s\n", formatDuration(cycle.Start, cycle.End))
	fmt.Fprintf(&b, "Strain: %s\n", formatCycleStrain(*cycle))
	fmt.Fprintf(&b, "Kilojoule: %s kJ\n", formatFloatPtr(cycle.Score.Kilojoule, 1))
	fmt.Fprintf(&b, "Avg HR: %s bpm\n", formatIntPtr(cycle.Score.AverageHeartRate))
	fmt.Fprintf(&b, "Max HR: %s bpm\n", formatIntPtr(cycle.Score.MaxHeartRate))
	fmt.Fprintf(&b, "Timezone Offset: %s\n", safeValue(cycle.TimezoneOffset))
	fmt.Fprintf(&b, "Created At: %s\n", formatTimestamp(cycle.CreatedAt))
	fmt.Fprintf(&b, "Updated At: %s", formatTimestamp(cycle.UpdatedAt))
	return strings.TrimSpace(b.String())
}
