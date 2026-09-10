package pii

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractStrategiesRecognizesPIIAnnotation(t *testing.T) {
	got := extractStrategies(map[string]any{
		"PII": map[string]any{"Strategy": "email"},
	})
	require.Equal(t, []string{"email"}, got)
}

func TestExtractStrategiesIgnoresOtherAnnotations(t *testing.T) {
	got := extractStrategies(map[string]any{
		"Multitenant":   map[string]any{"Fields": "tenant_id"},
		"GQLField":      map[string]any{"Name": "id"},
		"OrderByFields": map[string]any{"Name": "created_at"},
	})
	require.Empty(t, got)
}

func TestExtractStrategiesHandlesMissingStrategyField(t *testing.T) {
	got := extractStrategies(map[string]any{
		"PII": map[string]any{"Foo": "bar"},
	})
	require.Empty(t, got)
}

func TestExtractStrategiesIgnoresMalformedValues(t *testing.T) {
	got := extractStrategies(map[string]any{
		"PII": "not a map",
	})
	require.Empty(t, got)
}

func TestExtractStrategiesReturnsMultiple(t *testing.T) {
	got := extractStrategies(map[string]any{
		"PII": map[string]any{"Strategy": "phone"},
		"Multitenant": map[string]any{"Foo": "bar"},
	})
	require.Equal(t, []string{"phone"}, got)
}