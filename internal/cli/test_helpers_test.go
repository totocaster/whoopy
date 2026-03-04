package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/toto/whoopy/internal/paths"
)

func runCLICommand(t *testing.T, args []string, input string) string {
	t.Helper()
	var buf strings.Builder
	rootCmd.SetArgs(args)
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	reader := strings.NewReader(input)
	rootCmd.SetIn(reader)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := Execute(ctx)
	rootCmd.SetIn(os.Stdin)
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	rootCmd.SetArgs(nil)
	require.NoError(t, err)
	return buf.String()
}

func setTestConfigDir(t *testing.T) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "whoopy")
	paths.SetConfigDirOverride(dir)
	t.Cleanup(func() { paths.SetConfigDirOverride("") })
}

func getConfigDir(t *testing.T) string {
	t.Helper()
	dir, err := paths.ConfigDir()
	require.NoError(t, err)
	return dir
}
