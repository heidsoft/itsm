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

// DepartmentProcessTemplate defines a department-specific process template.
//
// 这里是「部门类型 -> 业务类型/子类型/场景/流程 key」的唯一清单。之前
// service/bpmn_process_binding_service.go 与 service/department_process_service.go
// 各自维护了一份同义列表并已经漂移（三份清单引用的 key 不一致），所以部门初始化
// 只允许读取本包；新场景在这里加条目，不要再新增第二份 switch。
//
// ProcessKey 是否真的可用由 service 包按 go:embed bpmn/*.bpmn 判定：没有模板载体的
// key 不会被绑定（详见 service.BuiltinProcessTemplateKeys 与绑定守卫测试）。
type DepartmentProcessTemplate struct {
	DepartmentCode    string                 `json:"departmentCode"`
	BusinessType      string                 `json:"businessType"`
	BusinessSubType   string                 `json:"businessSubType,omitempty"`
	Scenario          ScenarioType           `json:"scenario"`
	ProcessKey        string                 `json:"processKey"`
	Description       string                 `json:"description"`
	Priority          int                    `json:"priority"`
	Category          string                 `json:"category"`
	Conditions        map[string]interface{} `json:"conditions,omitempty"`
	RequiresOwnImport bool                   `json:"-"`
}

// GetOperationsTemplates returns default process templates for Operations department
func GetOperationsTemplates() []DepartmentProcessTemplate {
	return []DepartmentProcessTemplate{
		{
			DepartmentCode:  "OPS",
			BusinessType:    "incident",
			BusinessSubType: "alert_p0",
			Scenario:        ScenarioAlertHandling,
			ProcessKey:      "incident_emergency_flow",
			Description:     "P0 alert handling process",
			Priority:        100,
			Category:        "operations",
			Conditions:      map[string]interface{}{"severity": "p0"},
		},
		{
			DepartmentCode:  "OPS",
			BusinessType:    "incident",
			BusinessSubType: "alert_p1",
			Scenario:        ScenarioAlertHandling,
			ProcessKey:      "incident_emergency_flow",
			Description:     "P1 alert handling process",
			Priority:        90,
			Category:        "operations",
			Conditions:      map[string]interface{}{"severity": "p1"},
		},
		{
			DepartmentCode:  "OPS",
			BusinessType:    "change",
			BusinessSubType: "normal",
			Scenario:        ScenarioChangeRelease,
			ProcessKey:      "change_normal_flow",
			Description:     "Standard change release process",
			Priority:        70,
			Category:        "operations",
		},
		{
			DepartmentCode:  "OPS",
			BusinessType:    "change",
			BusinessSubType: "emergency",
			Scenario:        ScenarioEmergencyChange,
			ProcessKey:      "change_emergency_flow",
			Description:     "Emergency change process",
			Priority:        90,
			Category:        "operations",
		},
	}
}

// GetRDTemplates returns default process templates for R&D department
func GetRDTemplates() []DepartmentProcessTemplate {
	return []DepartmentProcessTemplate{
		{
			DepartmentCode:  "RD",
			BusinessType:    "release",
			BusinessSubType: "production",
			Scenario:        ScenarioCodeReleaseProd,
			ProcessKey:      "release_approval_flow",
			Description:     "Production release approval",
			Priority:        90,
			Category:        "rd",
			Conditions:      map[string]interface{}{"environment": "production"},
		},
		{
			DepartmentCode:  "RD",
			BusinessType:    "release",
			BusinessSubType: "testing",
			Scenario:        ScenarioCodeReleaseTest,
			ProcessKey:      "release_test_flow",
			Description:     "Test environment release",
			Priority:        70,
			Category:        "rd",
			Conditions:      map[string]interface{}{"environment": "testing"},
		},
		{
			DepartmentCode:  "RD",
			BusinessType:    "change",
			BusinessSubType: "requirement",
			Scenario:        ScenarioRequirementChange,
			ProcessKey:      "change_requirement_flow",
			Description:     "Requirement change process",
			Priority:        80,
			Category:        "rd",
		},
	}
}

// GetFinanceTemplates returns default process templates for Finance department
func GetFinanceTemplates() []DepartmentProcessTemplate {
	return []DepartmentProcessTemplate{
		{
			DepartmentCode:  "FIN",
			BusinessType:    "service_request",
			BusinessSubType: "expense",
			Scenario:        ScenarioExpenseApproval,
			ProcessKey:      "expense_approval_flow",
			Description:     "Expense approval process",
			Priority:        80,
			Category:        "finance",
			Conditions:      map[string]interface{}{"type": "expense"},
		},
		{
			DepartmentCode:  "FIN",
			BusinessType:    "service_request",
			BusinessSubType: "budget",
			Scenario:        ScenarioBudgetApproval,
			ProcessKey:      "budget_approval_flow",
			Description:     "Budget approval process",
			Priority:        90,
			Category:        "finance",
			Conditions:      map[string]interface{}{"type": "budget"},
		},
		{
			DepartmentCode:  "FIN",
			BusinessType:    "service_request",
			BusinessSubType: "procurement",
			Scenario:        ScenarioProcurement,
			ProcessKey:      "procurement_flow",
			Description:     "Procurement approval process",
			Priority:        85,
			Category:        "finance",
		},
	}
}

// GetHRTemplates returns default process templates for HR department
func GetHRTemplates() []DepartmentProcessTemplate {
	return []DepartmentProcessTemplate{
		{
			DepartmentCode:  "HR",
			BusinessType:    "service_request",
			BusinessSubType: "leave",
			Scenario:        ScenarioLeaveApproval,
			ProcessKey:      "leave_approval_flow",
			Description:     "Leave approval process",
			Priority:        70,
			Category:        "hr",
		},
		{
			DepartmentCode:  "HR",
			BusinessType:    "service_request",
			BusinessSubType: "recruitment",
			Scenario:        ScenarioRecruitmentApproval,
			ProcessKey:      "recruitment_approval_flow",
			Description:     "Recruitment approval process",
			Priority:        80,
			Category:        "hr",
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
