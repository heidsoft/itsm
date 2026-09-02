package bootstrap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProductionBootstrapDoesNotImportLegacyController(t *testing.T) {
	for _, path := range []string{"app.go", "../../router/router.go"} {
		source, err := os.ReadFile(path)
		require.NoError(t, err)
		require.NotContains(t, string(source), `"itsm-backend/controller`)
	}
}
