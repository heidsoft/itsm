package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTicketListDTOsUseItemsKey(t *testing.T) {
	tests := map[string]interface{}{
		"ticket views":    ListTicketViewsResponse{Items: []*TicketViewResponse{}, Total: 0},
		"ticket comments": ListTicketCommentsResponse{Items: []*TicketCommentResponse{}, Total: 0},
	}

	for name, response := range tests {
		t.Run(name, func(t *testing.T) {
			body, err := json.Marshal(response)
			require.NoError(t, err)

			var fields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(body, &fields))
			require.Contains(t, fields, "items")
			require.Contains(t, fields, "total")
			require.Len(t, fields, 2)
		})
	}
}
