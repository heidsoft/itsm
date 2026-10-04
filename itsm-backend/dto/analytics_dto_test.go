package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyticsTicketStatsResponse_JSONKeysAreCamelCase(t *testing.T) {
	resp := &AnalyticsTicketStatsResponse{
		Total:          10,
		StatusGroups:   []TicketStatusGroup{{Status: "open", Count: 5}},
		PriorityGroups: []TicketPriorityGroup{{Priority: "high", Count: 3}},
		Trend30d:       []TicketTrendPoint{{Date: "2026-10-01", Count: 2}},
		GeneratedAt:    "2026-10-04T00:00:00Z",
	}
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))

	expectedKeys := []string{"total", "statusGroups", "priorityGroups", "trend30d", "generatedAt"}
	for _, k := range expectedKeys {
		_, ok := raw[k]
		assert.True(t, ok, "expected camelCase key %q in JSON output", k)
	}

	for key := range raw {
		assert.NotContains(t, key, "_", "JSON key %q must not contain snake_case", key)
	}
}
