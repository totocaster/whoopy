package cli

import (
	"context"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/toto/whoopy/internal/debuglog"
)

var debugFlag bool

func init() {
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable structured debug logging")
}

var rootCmd = &cobra.Command{
	Use:           "whoopy",
	Short:         "Unofficial WHOOP data CLI written in Go",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		enabled := debugFlag || parseDebugEnv(os.Getenv("WHOOPY_DEBUG"))
		if err := debuglog.Configure(enabled); err != nil {
			return err
		}
		if enabled {
			path, err := debuglog.Path()
			if err != nil {
				return err
			}
			debuglog.Info("debug logging enabled", "command", cmd.CommandPath(), "path", path)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// Execute runs the root command with the provided context.
func Execute(ctx context.Context) error {
	rootCmd.SetContext(ctx)
	return rootCmd.ExecuteContext(ctx)
}

func parseDebugEnv(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
