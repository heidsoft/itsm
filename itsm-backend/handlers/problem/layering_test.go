package problem

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandlerDoesNotImportEnt(t *testing.T) {
	source, err := os.ReadFile("handler.go")
	require.NoError(t, err)
	require.NotContains(t, string(source), "itsm-backend/ent")
	require.NotContains(t, string(source), "TenantIDEQ")
}
