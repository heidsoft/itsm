package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateTicketRequestDoesNotBindForceFromJSON(t *testing.T) {
	var req UpdateTicketRequest
	require.NoError(t, json.Unmarshal([]byte(`{"title":"updated","force":true,"version":1}`), &req))
	require.False(t, req.Force)
	require.Equal(t, 1, req.Version)
}
