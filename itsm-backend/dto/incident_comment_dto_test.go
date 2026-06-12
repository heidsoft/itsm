package dto

import (
	"testing"

	"itsm-backend/ent"

	"github.com/stretchr/testify/require"
)

func TestToIncidentCommentResponseParsesJSONNumberSlices(t *testing.T) {
	event := &ent.IncidentEvent{
		ID:          10,
		IncidentID:  20,
		UserID:      30,
		Description: "需要关注数据库告警",
		Data: map[string]interface{}{
			"isInternal":  true,
			"mentions":    []interface{}{float64(1), float64(2)},
			"attachments": []interface{}{float64(8), float64(13)},
		},
	}

	resp := ToIncidentCommentResponse(event, nil)

	require.True(t, resp.IsInternal)
	require.Equal(t, []int{1, 2}, resp.Mentions)
	require.Equal(t, []int{8, 13}, resp.Attachments)
}
