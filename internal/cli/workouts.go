package cli

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
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

const (
	workoutFlagSport     = "sport"
	workoutFlagMinStrain = "min-strain"
	workoutFlagMaxStrain = "max-strain"
)

func init() {
	rootCmd.AddCommand(workoutsCmd)
	workoutsCmd.AddCommand(workoutsListCmd)
	workoutsCmd.AddCommand(workoutsTodayCmd)
	workoutsCmd.AddCommand(workoutsViewCmd)
	workoutsCmd.AddCommand(workoutsExportCmd)
	addListFlags(workoutsListCmd)
	workoutsListCmd.Flags().Bool("text", false, "Human-readable output")
	workoutsListCmd.Flags().String(workoutFlagSport, "", "Filter workouts by sport name or ID (client-side filter)")
	workoutsListCmd.Flags().Float64(workoutFlagMinStrain, 0, "Minimum strain (inclusive)")
	workoutsListCmd.Flags().Float64(workoutFlagMaxStrain, 0, "Maximum strain (inclusive)")
	workoutsViewCmd.Flags().Bool("text", false, "Human-readable output")
	workoutsTodayCmd.Flags().Bool("text", false, "Human-readable output")
	workoutsTodayCmd.Flags().String(workoutFlagSport, "", "Filter workouts by sport name or ID (client-side filter)")
	workoutsTodayCmd.Flags().Float64(workoutFlagMinStrain, 0, "Minimum strain (inclusive)")
	workoutsTodayCmd.Flags().Float64(workoutFlagMaxStrain, 0, "Maximum strain (inclusive)")
	workoutsExportCmd.Flags().String("format", "jsonl", "Export format: jsonl or csv")
	workoutsExportCmd.Flags().String("output", "-", "Output path ('-' for stdout)")
	workoutsExportCmd.Flags().String(workoutFlagSport, "", "Filter workouts by sport name or ID (client-side filter)")
	workoutsExportCmd.Flags().Float64(workoutFlagMinStrain, 0, "Minimum strain (inclusive)")
	workoutsExportCmd.Flags().Float64(workoutFlagMaxStrain, 0, "Maximum strain (inclusive)")
	addListFlags(workoutsExportCmd)
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
		filters, err := parseWorkoutFilters(cmd)
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
		if result != nil {
			result.Workouts = filterWorkouts(result.Workouts, filters)
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

var workoutsTodayCmd = &cobra.Command{
	Use:   "today",
	Short: "List today's workouts quickly",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := todayRangeOptions(25)
		filters, err := parseWorkoutFilters(cmd)
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
		if result != nil {
			result.Workouts = filterWorkouts(result.Workouts, filters)
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

var workoutsExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export workouts over a date range",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := parseListOptions(cmd)
		if err != nil {
			return err
		}
		filters, err := parseWorkoutFilters(cmd)
		if err != nil {
			return err
		}
		format, err := cmd.Flags().GetString("format")
		if err != nil {
			return err
		}
		outputPath, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}
		return exportWorkouts(cmd.Context(), opts, filters, format, outputPath, cmd.OutOrStdout())
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
	return formatDuration(w.Start, w.End)
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

type workoutFilters struct {
	sportQuery string
	minStrain  *float64
	maxStrain  *float64
}

func parseWorkoutFilters(cmd *cobra.Command) (workoutFilters, error) {
	var filters workoutFilters
	sport, err := cmd.Flags().GetString(workoutFlagSport)
	if err != nil {
		return filters, err
	}
	filters.sportQuery = strings.TrimSpace(sport)

	minStrain, err := cmd.Flags().GetFloat64(workoutFlagMinStrain)
	if err != nil {
		return filters, err
	}
	if cmd.Flags().Changed(workoutFlagMinStrain) {
		filters.minStrain = float64Ptr(minStrain)
	}

	maxStrain, err := cmd.Flags().GetFloat64(workoutFlagMaxStrain)
	if err != nil {
		return filters, err
	}
	if cmd.Flags().Changed(workoutFlagMaxStrain) {
		filters.maxStrain = float64Ptr(maxStrain)
	}

	if filters.minStrain != nil && filters.maxStrain != nil && *filters.minStrain > *filters.maxStrain {
		return filters, fmt.Errorf("--min-strain cannot be greater than --max-strain")
	}

	return filters, nil
}

func (f workoutFilters) active() bool {
	return f.sportQuery != "" || f.minStrain != nil || f.maxStrain != nil
}

func (f workoutFilters) matches(w workouts.Workout) bool {
	if f.sportQuery != "" && !matchesSportQuery(f.sportQuery, w) {
		return false
	}
	if f.minStrain != nil && w.Score.Strain < *f.minStrain {
		return false
	}
	if f.maxStrain != nil && w.Score.Strain > *f.maxStrain {
		return false
	}
	return true
}

func filterWorkouts(items []workouts.Workout, filters workoutFilters) []workouts.Workout {
	if !filters.active() {
		return items
	}
	filtered := make([]workouts.Workout, 0, len(items))
	for _, w := range items {
		if filters.matches(w) {
			filtered = append(filtered, w)
		}
	}
	return filtered
}

func matchesSportQuery(query string, w workouts.Workout) bool {
	if strings.TrimSpace(query) == "" {
		return true
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if id, err := strconv.Atoi(q); err == nil {
		if w.SportID != nil && *w.SportID == id {
			return true
		}
	}
	return strings.Contains(strings.ToLower(w.SportName), q)
}

func float64Ptr(v float64) *float64 {
	return &v
}

func exportWorkouts(ctx context.Context, baseOpts *api.ListOptions, filters workoutFilters, format, outputPath string, stdout io.Writer) error {
	if strings.TrimSpace(format) == "" {
		format = "jsonl"
	}
	format = strings.ToLower(strings.TrimSpace(format))

	writer := stdout
	var file *os.File
	if out := strings.TrimSpace(outputPath); out != "" && out != "-" {
		var err error
		file, err = os.Create(out)
		if err != nil {
			return fmt.Errorf("open export output: %w", err)
		}
		defer file.Close()
		writer = file
	}

	switch format {
	case "jsonl":
		return exportWorkoutsJSONL(ctx, baseOpts, filters, writer)
	case "csv":
		return exportWorkoutsCSV(ctx, baseOpts, filters, writer)
	default:
		return fmt.Errorf("unsupported --format %q (expected jsonl or csv)", format)
	}
}

func exportWorkoutsJSONL(ctx context.Context, baseOpts *api.ListOptions, filters workoutFilters, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return iterateWorkouts(ctx, baseOpts, filters, func(workout workouts.Workout) error {
		return encoder.Encode(workout)
	})
}

func exportWorkoutsCSV(ctx context.Context, baseOpts *api.ListOptions, filters workoutFilters, w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{
		"id", "sport_name", "sport_id", "start", "end", "duration",
		"strain", "avg_hr", "max_hr", "kilojoule", "distance_m", "percent_recorded",
	}); err != nil {
		return err
	}
	err := iterateWorkouts(ctx, baseOpts, filters, func(workout workouts.Workout) error {
		return cw.Write([]string{
			workout.ID,
			workout.SportName,
			intPtrToString(workout.SportID),
			workout.Start.Format(time.RFC3339Nano),
			workout.End.Format(time.RFC3339Nano),
			formatDuration(workout.Start, workout.End),
			fmt.Sprintf("%.3f", workout.Score.Strain),
			intPtrToString(workout.Score.AverageHeartRate),
			intPtrToString(workout.Score.MaxHeartRate),
			floatPtrString(workout.Score.Kilojoule),
			floatPtrString(workout.Score.DistanceMeter),
			floatPtrString(workout.Score.PercentRecorded),
		})
	})
	cw.Flush()
	if err != nil {
		return err
	}
	if err := cw.Error(); err != nil {
		return err
	}
	return nil
}

func iterateWorkouts(ctx context.Context, baseOpts *api.ListOptions, filters workoutFilters, fn func(workouts.Workout) error) error {
	var opts api.ListOptions
	if baseOpts != nil {
		opts = *baseOpts
	}
	maxRecords := 0
	if opts.Limit > 0 {
		maxRecords = opts.Limit
	}
	emitted := 0
	for {
		if maxRecords > 0 && emitted >= maxRecords {
			return nil
		}
		result, err := workoutsListFn(ctx, &opts)
		if err != nil {
			return err
		}
		if result == nil {
			return nil
		}
		for _, workout := range filterWorkouts(result.Workouts, filters) {
			if maxRecords > 0 && emitted >= maxRecords {
				return nil
			}
			if err := fn(workout); err != nil {
				return err
			}
			emitted++
		}
		if strings.TrimSpace(result.NextToken) == "" {
			return nil
		}
		opts.NextToken = result.NextToken
	}
}

func intPtrToString(value *int) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *value)
}

func floatPtrString(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}
