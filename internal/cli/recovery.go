package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/toto/whoopy/internal/api"
	"github.com/toto/whoopy/internal/recovery"
)

var (
	recoveryListFn = defaultRecoveryList
	recoveryViewFn = defaultRecoveryView
)

func init() {
	rootCmd.AddCommand(recoveryCmd)
	recoveryCmd.AddCommand(recoveryListCmd)
	recoveryCmd.AddCommand(recoveryTodayCmd)
	recoveryCmd.AddCommand(recoveryViewCmd)
	addListFlags(recoveryListCmd)
	recoveryListCmd.Flags().Bool("text", false, "Human-readable output")
	recoveryTodayCmd.Flags().Bool("text", false, "Human-readable output")
	recoveryViewCmd.Flags().Bool("text", false, "Human-readable output")
}

var recoveryCmd = &cobra.Command{
	Use:   "recovery",
	Short: "Recovery-related commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var recoveryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recovery scores within an optional time range",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := parseListOptions(cmd)
		if err != nil {
			return err
		}
		textMode, err := cmd.Flags().GetBool("text")
		if err != nil {
			return err
		}
		result, err := recoveryListFn(cmd.Context(), opts)
		if err != nil {
			return err
		}
		if textMode {
			fmt.Fprintln(cmd.OutOrStdout(), formatRecoveryText(result))
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

var recoveryTodayCmd = &cobra.Command{
	Use:   "today",
	Short: "Show today's recovery scores",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := todayRangeOptions(25)
		textMode, err := cmd.Flags().GetBool("text")
		if err != nil {
			return err
		}
		result, err := recoveryListFn(cmd.Context(), opts)
		if err != nil {
			return err
		}
		if textMode {
			fmt.Fprintln(cmd.OutOrStdout(), formatRecoveryText(result))
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

var recoveryViewCmd = &cobra.Command{
	Use:   "view <cycle-id>",
	Short: "Show recovery for a specific cycle",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		textMode, err := cmd.Flags().GetBool("text")
		if err != nil {
			return err
		}
		rec, err := recoveryViewFn(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		if textMode {
			fmt.Fprintln(cmd.OutOrStdout(), formatRecoveryDetailText(rec))
			return nil
		}
		payload, err := json.MarshalIndent(rec, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(payload))
		return nil
	},
}

func defaultRecoveryList(ctx context.Context, opts *api.ListOptions) (*recovery.ListResult, error) {
	client, err := apiClientFactory()
	if err != nil {
		return nil, err
	}
	service := recovery.NewService(client)
	return service.List(ctx, opts)
}

func defaultRecoveryView(ctx context.Context, cycleID string) (*recovery.Recovery, error) {
	client, err := apiClientFactory()
	if err != nil {
		return nil, err
	}
	service := recovery.NewService(client)
	return service.GetByCycle(ctx, cycleID)
}

func formatRecoveryText(result *recovery.ListResult) string {
	if result == nil || len(result.Recoveries) == 0 {
		if result != nil && strings.TrimSpace(result.NextToken) != "" {
			return fmt.Sprintf("No recoveries found. Next cursor: %s", result.NextToken)
		}
		return "No recoveries found."
	}
	var b strings.Builder
	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "Cycle\tScore\tRHR\tHRV (ms)\tResp\tCal?\tSleep ID")
	for _, rec := range result.Recoveries {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			rec.CycleID,
			formatFloatPtr(rec.Score.RecoveryScore, 0),
			formatFloatPtr(rec.Score.RestingHeartRate, 0),
			formatFloatPtr(rec.Score.HRVRMSSDMilli, 1),
			formatFloatPtr(rec.Score.RespiratoryRate, 1),
			formatBool(rec.Score.UserCalibrating),
			safeValue(rec.SleepID),
		)
	}
	tw.Flush()
	if strings.TrimSpace(result.NextToken) != "" {
		fmt.Fprintf(&b, "\nNext cursor: %s", result.NextToken)
	}
	return strings.TrimSpace(b.String())
}

func formatRecoveryDetailText(rec *recovery.Recovery) string {
	if rec == nil {
		return "Recovery not found."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Cycle ID: %d\n", rec.CycleID)
	if rec.UserID != nil {
		fmt.Fprintf(&b, "User ID: %d\n", *rec.UserID)
	}
	fmt.Fprintf(&b, "Sleep ID: %s\n", safeValue(rec.SleepID))
	fmt.Fprintf(&b, "State: %s\n", safeValue(rec.ScoreState))
	fmt.Fprintf(&b, "Recovery Score: %s\n", formatFloatPtr(rec.Score.RecoveryScore, 0))
	fmt.Fprintf(&b, "Resting HR: %s bpm\n", formatFloatPtr(rec.Score.RestingHeartRate, 0))
	fmt.Fprintf(&b, "HRV (rmssd): %s ms\n", formatFloatPtr(rec.Score.HRVRMSSDMilli, 1))
	fmt.Fprintf(&b, "Respiratory Rate: %s br/min\n", formatFloatPtr(rec.Score.RespiratoryRate, 1))
	fmt.Fprintf(&b, "SpO₂: %s %%\n", formatPercent(rec.Score.Spo2Percentage))
	fmt.Fprintf(&b, "Skin Temp: %s °C\n", formatFloatPtr(rec.Score.SkinTempCelsius, 1))
	fmt.Fprintf(&b, "Cycle Strain: %s\n", formatFloatPtr(rec.Score.CycleStrain, 1))
	fmt.Fprintf(&b, "User Calibrating: %s\n", formatBool(rec.Score.UserCalibrating))
	fmt.Fprintf(&b, "Created At: %s\n", formatTimestamp(rec.CreatedAt))
	fmt.Fprintf(&b, "Updated At: %s", formatTimestamp(rec.UpdatedAt))
	return strings.TrimSpace(b.String())
}
