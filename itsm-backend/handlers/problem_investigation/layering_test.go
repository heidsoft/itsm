package probleminvestigation

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandlerDoesNotAccessKnowledgePersistence(t *testing.T) {
	handlerSource, err := os.ReadFile("handler.go")
	require.NoError(t, err)
	routesSource, err := os.ReadFile("routes.go")
	require.NoError(t, err)

	require.NotContains(t, string(handlerSource), "*ent.Client")
	require.NotContains(t, string(handlerSource), "entClient")
	require.NotContains(t, string(routesSource), "KnowledgeArticle.Query")
	require.NotContains(t, string(routesSource), "KnowledgeArticle.Create")
}
