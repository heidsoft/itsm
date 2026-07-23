package dto

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"itsm-backend/ent"
)

func TestToIncidentResponseMapsIncidentSpecificFields(t *testing.T) {
	incident := &ent.Incident{
		ID: 1, Impact: "high", Urgency: "critical", Source: "monitoring",
		EscalationLevel: 2, IsMajorIncident: true,
		Metadata:   map[string]interface{}{"monitor": "prometheus"},
		DetectedAt: time.Now(),
	}

	response := ToIncidentResponse(incident)
	require.Equal(t, "high", response.Impact)
	require.Equal(t, "critical", response.Urgency)
	require.Equal(t, "monitoring", response.Source)
	require.Equal(t, 2, response.EscalationLevel)
	require.True(t, response.IsMajorIncident)
	require.Equal(t, "prometheus", response.Metadata["monitor"])
	require.Nil(t, response.ResolvedAt)
	require.Nil(t, response.ClosedAt)
	require.Nil(t, response.EscalatedAt)
}
