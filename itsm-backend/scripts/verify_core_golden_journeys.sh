#!/usr/bin/env bash
set -euo pipefail

# P0/P1 Core Golden Gate: authoritative backend lifecycle, BPMN bridge,
# tenant boundary, and audited workflow recovery checks.
GOTOOLCHAIN=auto go test ./handlers/incident \
  -run '^TestGoldenJourney_IncidentResolvedAndClosed$' -count=1
GOTOOLCHAIN=auto go test ./handlers/problem \
  -run '^TestGoldenJourney_ProblemRCAResolvedAndClosed$' -count=1
GOTOOLCHAIN=auto go test ./service \
  -run '^(TestGoldenJourney_NormalChangeCompletedStateMachine|TestTaskRecovery_)' -count=1
GOTOOLCHAIN=auto go test ./handlers/change \
  -run '^TestTransitionStatus_(BridgesBPMNTask|BridgeFailClosed)$' -count=1
GOTOOLCHAIN=auto go test ./handlers/service_request \
  -run '^(TestGoldenJourney_ServiceRequestApprovedProvisionedAndDelivered|TestServiceRequestApproval_BridgesBPMNTask|TestServiceRequestApproval_BridgeFailClosedForUnauthorizedActor)$' -count=1
