package scenario

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 部门默认流程清单是本用例唯一来源，因此它自身的形状必须在包内锁住：
// 业务类型词表封闭、同一部门内标识唯一、优先级与分类不得为空。
// 绑定侧（service.ProcessBindingService.InitDepartmentDefaultBindings）直接消费这里，
// 任何缺字段的条目都会变成写入 process_bindings 的脏数据。

var allowedBusinessTypes = map[string]bool{
	"incident":        true,
	"problem":         true,
	"change":          true,
	"release":         true,
	"service_request": true,
	"ticket":          true,
}

func TestGetAllTemplates_ShapeIsWellFormed(t *testing.T) {
	catalog := GetAllTemplates()
	require.Equal(t, []string{"finance", "hr", "operations", "rd"}, sortedDepartmentTypes(catalog))

	for departmentType, templates := range catalog {
		require.NotEmptyf(t, templates, "部门类型 %s 没有默认流程条目", departmentType)

		seen := make(map[string]bool, len(templates))
		for _, tmpl := range templates {
			identity := tmpl.BusinessType + "/" + tmpl.BusinessSubType + "/" + string(tmpl.Scenario)
			assert.Falsef(t, seen[identity], "%s: 重复的默认绑定标识 %s", departmentType, identity)
			seen[identity] = true

			assert.Truef(t, allowedBusinessTypes[tmpl.BusinessType], "%s: 未知业务类型 %q", departmentType, tmpl.BusinessType)
			assert.NotEmptyf(t, tmpl.ProcessKey, "%s: %s 缺少流程 key", departmentType, identity)
			assert.NotEmptyf(t, string(tmpl.Scenario), "%s: %s 缺少场景", departmentType, identity)
			assert.Equalf(t, departmentType, tmpl.Category, "%s: Category 必须与部门类型一致", identity)
			assert.Greaterf(t, tmpl.Priority, 0, "%s: 优先级必须为正", identity)
		}
	}
}

func TestGetAllTemplates_ScenarioVocabularyIsClosed(t *testing.T) {
	known := map[ScenarioType]bool{
		ScenarioAlertHandling:       true,
		ScenarioChangeRelease:       true,
		ScenarioEmergencyChange:     true,
		ScenarioCodeReleaseProd:     true,
		ScenarioCodeReleaseTest:     true,
		ScenarioRequirementChange:   true,
		ScenarioExpenseApproval:     true,
		ScenarioBudgetApproval:      true,
		ScenarioProcurement:         true,
		ScenarioLeaveApproval:       true,
		ScenarioRecruitmentApproval: true,
		ScenarioGeneralTicket:       true,
		ScenarioServiceRequest:      true,
		ScenarioStandardChange:      true,
		ScenarioTechReview:          true,
		ScenarioOnboardingApproval:  true,
	}

	for _, templates := range GetAllTemplates() {
		for _, tmpl := range templates {
			assert.Truef(t, known[tmpl.Scenario], "清单使用了未登记的场景 %q；新增场景必须同时补词表与绑定语义", tmpl.Scenario)
		}
	}
}

func sortedDepartmentTypes(catalog map[string][]DepartmentProcessTemplate) []string {
	names := make([]string, 0, len(catalog))
	for name := range catalog {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
