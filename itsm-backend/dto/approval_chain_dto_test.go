package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApprovalChainListResponseUsesStandardItemsKey(t *testing.T) {
	data, err := json.Marshal(ApprovalChainListResponse{
		Items: []ApprovalChainResponse{{ID: 1, Name: "Default"}},
		Total: 1,
	})
	require.NoError(t, err)

	var response map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &response))
	require.Contains(t, response, "items")
	require.Contains(t, response, "total")
	require.NotContains(t, response, "data")
}
