package bootstrap

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseProcessMode(t *testing.T) {
	tests := []struct {
		value string
		want  ProcessMode
	}{
		{"", ProcessModeAll},
		{"all", ProcessModeAll},
		{" API ", ProcessModeAPI},
		{"worker", ProcessModeWorker},
	}
	for _, test := range tests {
		got, err := ParseProcessMode(test.value)
		require.NoError(t, err)
		require.Equal(t, test.want, got)
	}
	_, err := ParseProcessMode("scheduler")
	require.Error(t, err)
	require.Error(t, ValidateProcessMode(ProcessModeAll, "production"))
	require.NoError(t, ValidateProcessMode(ProcessModeAll, "development"))
	require.NoError(t, ValidateProcessMode(ProcessModeAPI, "production"))
}
