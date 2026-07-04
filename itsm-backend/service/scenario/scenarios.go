package scenario

// ScenarioType defines available scenarios for department-specific processes
type ScenarioType string

const (
	// Operations Department Scenarios
	ScenarioAlertHandling   ScenarioType = "alert_handling"
	ScenarioChangeRelease   ScenarioType = "change_release"
	ScenarioEmergencyChange ScenarioType = "emergency_change"
	ScenarioStandardChange  ScenarioType = "standard_change"
	ScenarioRoutineOps      ScenarioType = "routine_ops"

	// R&D Department Scenarios
	ScenarioCodeReleaseProd   ScenarioType = "code_release_prod"
	ScenarioCodeReleaseTest   ScenarioType = "code_release_test"
	ScenarioRequirementChange ScenarioType = "requirement_change"
	ScenarioTechReview        ScenarioType = "tech_review"

	// Finance Department Scenarios
	ScenarioExpenseApproval ScenarioType = "expense_approval"
	ScenarioBudgetApproval  ScenarioType = "budget_approval"
	ScenarioProcurement     ScenarioType = "procurement"

	// HR Department Scenarios
	ScenarioLeaveApproval       ScenarioType = "leave_approval"
	ScenarioRecruitmentApproval ScenarioType = "recruitment_approval"
	ScenarioOnboardingApproval  ScenarioType = "onboarding_approval"

	// General Scenarios
	ScenarioGeneralTicket  ScenarioType = "general_ticket"
	ScenarioServiceRequest ScenarioType = "service_request"
)

// AlertSeverity defines alert severity levels
type AlertSeverity string

const (
	AlertP0 AlertSeverity = "p0" // Critical - immediate response required
	AlertP1 AlertSeverity = "p1" // High - response within 30 minutes
	AlertP2 AlertSeverity = "p2" // Medium - response within 2 hours
	AlertP3 AlertSeverity = "p3" // Low - response within 24 hours
)

// DepartmentProcessTemplate defines a department-specific process template
type DepartmentProcessTemplate struct {
	DepartmentCode string                 `json:"departmentCode"`
	Scenario       ScenarioType           `json:"scenario"`
	ProcessKey     string                 `json:"processKey"`
	Description    string                 `json:"description"`
	Priority       int                    `json:"priority"`
	Conditions     map[string]interface{} `json:"conditions,omitempty"`
}

// GetOperationsTemplates returns default process templates for Operations department
func GetOperationsTemplates() []DepartmentProcessTemplate {
	return []DepartmentProcessTemplate{
		{
			DepartmentCode: "OPS",
			Scenario:       ScenarioAlertHandling,
			ProcessKey:     "incident_emergency_flow",
			Description:    "P0/P1 alert handling process",
			Priority:       100,
			Conditions:     map[string]interface{}{"severity": "critical"},
		},
		{
			DepartmentCode: "OPS",
			Scenario:       ScenarioChangeRelease,
			ProcessKey:     "change_normal_flow",
			Description:    "Standard change release process",
			Priority:       70,
		},
		{
			DepartmentCode: "OPS",
			Scenario:       ScenarioEmergencyChange,
			ProcessKey:     "change_emergency_flow",
			Description:    "Emergency change process",
			Priority:       90,
		},
		{
			DepartmentCode: "OPS",
			Scenario:       ScenarioStandardChange,
			ProcessKey:     "change_normal_flow",
			Description:    "Standard change process",
			Priority:       60,
		},
	}
}

// GetRDTemplates returns default process templates for R&D department
func GetRDTemplates() []DepartmentProcessTemplate {
	return []DepartmentProcessTemplate{
		{
			DepartmentCode: "RD",
			Scenario:       ScenarioCodeReleaseProd,
			ProcessKey:     "release_approval_flow",
			Description:    "Production release approval",
			Priority:       90,
			Conditions:     map[string]interface{}{"environment": "production"},
		},
		{
			DepartmentCode: "RD",
			Scenario:       ScenarioCodeReleaseTest,
			ProcessKey:     "release_test_flow",
			Description:    "Test environment release",
			Priority:       70,
			Conditions:     map[string]interface{}{"environment": "testing"},
		},
		{
			DepartmentCode: "RD",
			Scenario:       ScenarioRequirementChange,
			ProcessKey:     "change_requirement_flow",
			Description:    "Requirement change process",
			Priority:       80,
		},
		{
			DepartmentCode: "RD",
			Scenario:       ScenarioTechReview,
			ProcessKey:     "tech_review_flow",
			Description:    "Technical review process",
			Priority:       60,
		},
	}
}

// GetFinanceTemplates returns default process templates for Finance department
func GetFinanceTemplates() []DepartmentProcessTemplate {
	return []DepartmentProcessTemplate{
		{
			DepartmentCode: "FIN",
			Scenario:       ScenarioExpenseApproval,
			ProcessKey:     "expense_approval_flow",
			Description:    "Expense approval process",
			Priority:       80,
			Conditions:     map[string]interface{}{"type": "expense"},
		},
		{
			DepartmentCode: "FIN",
			Scenario:       ScenarioBudgetApproval,
			ProcessKey:     "budget_approval_flow",
			Description:    "Budget approval process",
			Priority:       90,
			Conditions:     map[string]interface{}{"type": "budget"},
		},
		{
			DepartmentCode: "FIN",
			Scenario:       ScenarioProcurement,
			ProcessKey:     "procurement_flow",
			Description:    "Procurement approval process",
			Priority:       85,
		},
	}
}

// GetHRTemplates returns default process templates for HR department
func GetHRTemplates() []DepartmentProcessTemplate {
	return []DepartmentProcessTemplate{
		{
			DepartmentCode: "HR",
			Scenario:       ScenarioLeaveApproval,
			ProcessKey:     "leave_approval_flow",
			Description:    "Leave approval process",
			Priority:       70,
		},
		{
			DepartmentCode: "HR",
			Scenario:       ScenarioRecruitmentApproval,
			ProcessKey:     "recruitment_approval_flow",
			Description:    "Recruitment approval process",
			Priority:       80,
		},
		{
			DepartmentCode: "HR",
			Scenario:       ScenarioOnboardingApproval,
			ProcessKey:     "onboarding_approval_flow",
			Description:    "Onboarding approval process",
			Priority:       75,
		},
	}
}

// GetAllTemplates returns all department process templates
func GetAllTemplates() map[string][]DepartmentProcessTemplate {
	return map[string][]DepartmentProcessTemplate{
		"operations": GetOperationsTemplates(),
		"rd":         GetRDTemplates(),
		"finance":    GetFinanceTemplates(),
		"hr":         GetHRTemplates(),
	}
}
