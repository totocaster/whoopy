package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLegacyHPXFlagRemoved(t *testing.T) {
	_, err := runCLICommandWithError(t, []string{"version", "--hpx"}, "")
	require.ErrorContains(t, err, "unknown flag: --hpx")
}

func TestHPXCommandRemoved(t *testing.T) {
	_, err := runCLICommandWithError(t, []string{"hpx"}, "")
	require.ErrorContains(t, err, "unknown command \"hpx\"")
}
