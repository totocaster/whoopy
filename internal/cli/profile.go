package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/toto/whoopy/internal/profile"
)

var profileFetchFn = defaultProfileFetch

func init() {
	rootCmd.AddCommand(profileCmd)
	profileCmd.AddCommand(profileShowCmd)

	profileShowCmd.Flags().Bool("text", false, "Human-readable output")
}

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Profile-related commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var profileShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show WHOOP profile and body measurements",
	RunE: func(cmd *cobra.Command, args []string) error {
		textMode, err := cmd.Flags().GetBool("text")
		if err != nil {
			return err
		}
		summary, err := profileFetchFn(cmd.Context())
		if err != nil {
			return err
		}
		if textMode {
			fmt.Fprintln(cmd.OutOrStdout(), formatProfileText(summary))
			return nil
		}
		data, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	},
}

func defaultProfileFetch(ctx context.Context) (*profile.Summary, error) {
	client, err := apiClientFactory()
	if err != nil {
		return nil, err
	}
	service := profile.NewService(client)
	return service.Fetch(ctx)
}

func formatProfileText(sum *profile.Summary) string {
	var b strings.Builder
	b.WriteString("Profile\n")
	b.WriteString("-------\n")
	fmt.Fprintf(&b, "Name: %s\n", safeString(sum.Name))
	fmt.Fprintf(&b, "Email: %s\n", safeString(sum.Email))
	fmt.Fprintf(&b, "User ID: %s\n", safeString(sum.UserID))
	fmt.Fprintf(&b, "Locale: %s\n", safeString(sum.Locale))
	fmt.Fprintf(&b, "Timezone: %s\n", safeString(sum.Timezone))
	fmt.Fprintf(&b, "Membership: %s\n", safeString(sum.MembershipTier))
	b.WriteString("\nBody Measurements\n------------------\n")
	fmt.Fprintf(&b, "Height: %s", formatMeasurement(sum.HeightCm, sum.HeightIn, "cm", "in"))
	fmt.Fprintf(&b, "Weight: %s", formatMeasurement(sum.WeightKg, sum.WeightLb, "kg", "lb"))
	fmt.Fprintf(&b, "Max HR: %s\n", formatInt(sum.MaxHeartRate, "bpm"))
	if sum.UpdatedAt != nil {
		fmt.Fprintf(&b, "Last Updated: %s\n", sum.UpdatedAt.Format(time.RFC1123))
	} else {
		fmt.Fprintf(&b, "Last Updated: n/a\n")
	}
	return strings.TrimSpace(b.String())
}

func safeString(value string) string {
	if strings.TrimSpace(value) == "" {
		return "n/a"
	}
	return value
}

func formatMeasurement(primary, secondary *float64, primaryUnit, secondaryUnit string) string {
	var parts []string
	if primary != nil {
		parts = append(parts, fmt.Sprintf("%.1f %s", *primary, primaryUnit))
	}
	if secondary != nil {
		parts = append(parts, fmt.Sprintf("%.1f %s", *secondary, secondaryUnit))
	}
	if len(parts) == 0 {
		return "n/a\n"
	}
	return strings.Join(parts, " / ") + "\n"
}

func formatInt(value *int, suffix string) string {
	if value == nil {
		return "n/a"
	}
	return fmt.Sprintf("%d %s", *value, suffix)
}
