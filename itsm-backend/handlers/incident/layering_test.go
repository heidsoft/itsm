package incident

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandlerDoesNotTraverseEntEdges(t *testing.T) {
	source, err := os.ReadFile("handler.go")
	require.NoError(t, err)
	require.NotContains(t, string(source), ".Edges.", "incident edge traversal belongs in the service/repository layer")
}
