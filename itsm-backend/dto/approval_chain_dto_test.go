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

func TestApprovalChainListResponseUsesStandardPaginationKeys(t *testing.T) {
	data, err := json.Marshal(ApprovalChainListResponse{
		Items:      []ApprovalChainResponse{},
		Total:      25,
		Page:       2,
		PageSize:   10,
		TotalPages: 3,
	})
	require.NoError(t, err)

	var response map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &response))
	require.Contains(t, response, "page")
	require.Contains(t, response, "pageSize")
	require.Contains(t, response, "totalPages")
	require.NotContains(t, response, "size")
	require.NotContains(t, response, "totalCount")

	require.JSONEq(t, `2`, string(response["page"]))
	require.JSONEq(t, `10`, string(response["pageSize"]))
	require.JSONEq(t, `3`, string(response["totalPages"]))
}

func TestApprovalChainStepDTOCarriesAdvancedFieldsAsCamelCase(t *testing.T) {
	data, err := json.Marshal(ApprovalChainStepDTO{
		Level:               1,
		ApproverID:          7,
		Name:                "会签步骤",
		ApprovalType:        "parallel",
		Threshold:           2,
		FallbackAction:      "escalate",
		FallbackApproverID:  42,
		FallbackRole:        "role:ops_manager",
		ConditionPriorities: []string{"high", "urgent"},
		ConditionAmountMin:  1000,
		ConditionAmountMax:  5000,
	})
	require.NoError(t, err)

	var step map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &step))
	for _, key := range []string{
		"approvalType",
		"threshold",
		"fallbackAction",
		"fallbackApproverId",
		"fallbackRole",
		"conditionPriorities",
		"conditionAmountMin",
		"conditionAmountMax",
	} {
		require.Contains(t, step, key)
	}
	require.NotContains(t, step, "timeoutHours")
	require.NotContains(t, step, "timeout_hours")
	require.NotContains(t, step, "conditions")
	require.NotContains(t, step, "stepName")
	require.NotContains(t, step, "stepOrder")
}
