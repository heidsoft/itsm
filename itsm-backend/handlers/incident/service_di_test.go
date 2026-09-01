package incident

import (
	"testing"

	production "itsm-backend/service"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewServiceWiresProductionIncidentDependencies(t *testing.T) {
	repo := newMockRepository()
	incidentSvc := production.NewIncidentService(nil, zap.NewNop().Sugar(), nil)
	monitoringSvc := production.NewIncidentMonitoringService(nil, zap.NewNop().Sugar())
	alertingSvc := production.NewIncidentAlertingService(nil, zap.NewNop().Sugar())
	rootCauseSvc := production.NewRootCauseAnalysisService(nil)

	svc := NewService(repo, incidentSvc, monitoringSvc, alertingSvc, rootCauseSvc, zap.NewNop().Sugar())

	require.Same(t, incidentSvc, svc.productionService)
	require.Same(t, monitoringSvc, svc.monitoringService)
	require.Same(t, alertingSvc, svc.alertingSvc)
	require.Same(t, rootCauseSvc, svc.rootCauseSvc)
}
