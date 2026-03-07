package cli

import (
	"context"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/toto/whoopy/internal/api"
	"github.com/toto/whoopy/internal/cycles"
	"github.com/toto/whoopy/internal/recovery"
	"github.com/toto/whoopy/internal/sleep"
	"github.com/toto/whoopy/internal/workouts"
)

func init() {
	rootCmd.AddCommand(hpxCmd)
	hpxCmd.AddCommand(hpxExportCmd)
	addListWindowFlags(hpxExportCmd)
}

var hpxCmd = &cobra.Command{
	Use:   "hpx",
	Short: "HyperContext export commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var hpxExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export profile, sleep, cycles, recovery, and workouts as HyperContext NDJSON",
	Example: strings.Join([]string{
		"  whoopy hpx export --last 7d | hpx import",
		"  whoopy hpx export --since 2026-03-01 --until 2026-03-07",
	}, "\n"),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := parseListWindowOptions(cmd)
		if err != nil {
			return err
		}
		return exportHPX(cmd.Context(), cmd.OutOrStdout(), opts)
	},
}

func exportHPX(ctx context.Context, w io.Writer, opts *api.ListOptions) error {
	writer := newHPXWriter(w)

	summary, err := profileFetchFn(ctx)
	if err != nil {
		return err
	}
	if err := writer.writeProfile(summary); err != nil {
		return err
	}

	if err := iterateSleepSessions(ctx, opts, func(sess sleep.Session) error {
		return writer.writeSleepSession(sess)
	}); err != nil {
		return err
	}

	cycleByID := make(map[int64]cycles.Cycle)
	if err := iterateCycles(ctx, opts, func(cycle cycles.Cycle) error {
		cycleByID[cycle.ID] = cycle
		return writer.writeCycle(cycle)
	}); err != nil {
		return err
	}

	if err := iterateRecoveries(ctx, opts, func(rec recovery.Recovery) error {
		return writer.writeRecovery(rec, hpxCycleContext(ctx, cycleByID, rec.CycleID))
	}); err != nil {
		return err
	}

	return iterateWorkouts(ctx, opts, workoutFilters{}, func(workout workouts.Workout) error {
		return writer.writeWorkout(workout)
	})
}

func hpxCycleContext(ctx context.Context, cycleByID map[int64]cycles.Cycle, cycleID int64) *cycles.Cycle {
	if cycle, ok := cycleByID[cycleID]; ok {
		cycleCopy := cycle
		return &cycleCopy
	}
	return lookupCycle(ctx, cyclesViewFn, cycleID)
}

func iterateSleepSessions(ctx context.Context, baseOpts *api.ListOptions, fn func(sleep.Session) error) error {
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
		result, err := sleepListFn(ctx, &opts)
		if err != nil {
			return err
		}
		if result == nil {
			return nil
		}
		for _, sess := range result.Sleeps {
			if maxRecords > 0 && emitted >= maxRecords {
				return nil
			}
			if err := fn(sess); err != nil {
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

func iterateRecoveries(ctx context.Context, baseOpts *api.ListOptions, fn func(recovery.Recovery) error) error {
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
		result, err := recoveryListFn(ctx, &opts)
		if err != nil {
			return err
		}
		if result == nil {
			return nil
		}
		for _, rec := range result.Recoveries {
			if maxRecords > 0 && emitted >= maxRecords {
				return nil
			}
			if err := fn(rec); err != nil {
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

func iterateCycles(ctx context.Context, baseOpts *api.ListOptions, fn func(cycles.Cycle) error) error {
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
		result, err := cyclesListFn(ctx, &opts)
		if err != nil {
			return err
		}
		if result == nil {
			return nil
		}
		for _, cycle := range result.Cycles {
			if maxRecords > 0 && emitted >= maxRecords {
				return nil
			}
			if err := fn(cycle); err != nil {
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
