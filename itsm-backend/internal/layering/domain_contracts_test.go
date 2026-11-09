package layering

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/handlers/a2ui"
	"itsm-backend/handlers/account"
	"itsm-backend/handlers/analytics"
	"itsm-backend/handlers/approval"
	"itsm-backend/handlers/approval_chain"
	"itsm-backend/handlers/assignment_smart"
	"itsm-backend/handlers/auth"
	"itsm-backend/handlers/automation_rule"
	"itsm-backend/handlers/escalation_matrix"
	"itsm-backend/handlers/prediction"
	"itsm-backend/handlers/problem_investigation"
	"itsm-backend/handlers/provisioning"
	"itsm-backend/handlers/release"
	"itsm-backend/handlers/ticket_notification"
	"itsm-backend/handlers/ticket_tag"
	"itsm-backend/handlers/ticket_type"
	"itsm-backend/handlers/ticket_workflow"
	"itsm-backend/handlers/vector_store"
	"itsm-backend/service"
)

// 编译期断言：生产具体类型必须满足域内窄契约。契约方法集由各域 handler/routes
// 的真实调用面决定，任何签名漂移都会在编译期暴露，而不是拖到运行时。
var (
	_ account.UserService                               = (*service.UserService)(nil)
	_ analytics.AnalyticsService                        = (*service.AnalyticsService)(nil)
	_ approval.ApprovalService                          = (*service.ApprovalService)(nil)
	_ approval_chain.ApprovalChainService               = (*service.ApprovalChainService)(nil)
	_ auth.EmailService                                 = (*service.EmailService)(nil)
	_ automation_rule.AutomationRuleService             = (*service.TicketAutomationRuleService)(nil)
	_ escalation_matrix.EscalationMatrixService         = (*service.EscalationMatrixService)(nil)
	_ prediction.PredictionService                      = (*service.PredictionService)(nil)
	_ problem_investigation.ProblemInvestigationService = (*service.ProblemInvestigationService)(nil)
	_ release.ReleaseService                            = (*service.ReleaseService)(nil)
	_ ticket_notification.TicketNotificationService     = (*service.TicketNotificationService)(nil)
	_ ticket_type.TicketTypeService                     = (*service.TicketTypeService)(nil)
	_ ticket_workflow.TicketWorkflowService             = (*service.TicketWorkflowService)(nil)
	_ vector_store.VectorStore                          = (*service.VectorStore)(nil)
	_ a2ui.TicketFormService                            = (*service.A2UITicketService)(nil)
	_ assignment_smart.SmartAssignmentService           = (*service.TicketAssignmentSmartService)(nil)
	_ assignment_smart.AssignmentRuleService            = (*service.TicketAssignmentRuleService)(nil)
	_ provisioning.ProvisioningTaskService              = (*service.ProvisioningService)(nil)
	_ ticket_tag.TagService                             = (*service.TicketTagService)(nil)
)

type domainSpec struct {
	name     string // handlers 下的域目录名
	file     string // 持有契约字段的生产文件
	iface    string // 域内窄接口名
	concrete string // 共享层具体类型名；多接口域用逗号分隔，逐个检查
}

var contractDomains = []domainSpec{
	{"account", "handler.go", "UserService", "UserService"},
	{"analytics", "handler.go", "AnalyticsService", "AnalyticsService"},
	{"approval", "handler.go", "ApprovalService", "ApprovalService"},
	{"escalation_matrix", "handler.go", "EscalationMatrixService", "EscalationMatrixService"},
	{"prediction", "handler.go", "PredictionService", "PredictionService"},
	{"release", "handler.go", "ReleaseService", "ReleaseService"},
	{"approval_chain", "handler.go", "ApprovalChainService", "ApprovalChainService"},
	{"auth", "service.go", "EmailService", "EmailService"},
	{"automation_rule", "handler.go", "AutomationRuleService", "TicketAutomationRuleService"},
	{"problem_investigation", "handler.go", "ProblemInvestigationService", "ProblemInvestigationService"},
	{"ticket_notification", "handler.go", "TicketNotificationService", "TicketNotificationService"},
	{"ticket_type", "handler.go", "TicketTypeService", "TicketTypeService"},
	{"ticket_workflow", "handler.go", "TicketWorkflowService", "TicketWorkflowService"},
	{"vector_store", "handler.go", "VectorStore", "VectorStore"},
	{"a2ui", "handler.go", "TicketFormService", "A2UITicketService"},
	{"assignment_smart", "handler.go", "SmartAssignmentService", "TicketAssignmentSmartService,TicketAssignmentRuleService"},
	{"provisioning", "handler.go", "ProvisioningTaskService", "ProvisioningService"},
	{"ticket_tag", "handler.go", "TagService", "TicketTagService"},
}

// 只剔除行注释，避免把说明文字里的示例当成代码；L.3 门禁本身是全文 grep，
// 因此这里同时保留「全文零共享层路径字面量」的更严断言。
func stripLineComments(src string) string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		if i := strings.Index(line, "//"); i >= 0 {
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// squashSpace 让源码守卫不受 gofmt 对齐空格影响：var 块里对齐后的
// 「ErrX      = dto.ErrX」与声明文本「ErrX = dto.ErrX」应视为同一条。
func squashSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// TestDomainContractsKeepSharedLayerOut 是集中式归属回归锁：
// 逐域比对（不取并集）、扫描器零命中直接失败、字段类型必须能区分新旧实现。
func TestDomainContractsKeepSharedLayerOut(t *testing.T) {
	const sharedImport = `"itsm-backend/service"`

	for _, d := range contractDomains {
		t.Run(d.name, func(t *testing.T) {
			dir := filepath.Join("..", "..", "handlers", d.name)
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatalf("读取 handlers/%s 失败：%v", d.name, err)
			}

			prodFiles := 0
			for _, e := range entries {
				fn := e.Name()
				if e.IsDir() || !strings.HasSuffix(fn, ".go") || strings.HasSuffix(fn, "_test.go") {
					continue
				}
				prodFiles++
				body, err := os.ReadFile(filepath.Join(dir, fn))
				if err != nil {
					t.Fatalf("读取 %s/%s 失败：%v", d.name, fn, err)
				}
				if len(body) < 40 {
					t.Fatalf("扫描器零命中保护：%s/%s 内容异常短（%d 字节）", d.name, fn, len(body))
				}
				// 与 Gate C.8 / L.3 口径一致：全文，注释里的字面量同样算命中。
				if strings.Contains(string(body), sharedImport) {
					t.Errorf("handlers/%s/%s 重新引用了共享层包，L.3 归零被破坏", d.name, fn)
				}
				bodyCode := stripLineComments(string(body))
				for _, concrete := range strings.Split(d.concrete, ",") {
					if strings.Contains(bodyCode, "*service."+concrete) {
						t.Errorf("handlers/%s/%s 字段或构造参数回退到共享层具体类型 %s", d.name, fn, concrete)
					}
				}
			}
			// 零命中不能由「目录空了 / 文件被删光」造成。
			if prodFiles < 2 {
				t.Fatalf("扫描器零命中保护：handlers/%s 只有 %d 个生产 .go 文件（至少应为 contracts.go + handler.go）", d.name, prodFiles)
			}

			contract, err := os.ReadFile(filepath.Join(dir, "contracts.go"))
			if err != nil {
				t.Fatalf("handlers/%s/contracts.go 缺失：%v", d.name, err)
			}
			if len(contract) < 100 {
				t.Fatalf("扫描器零命中保护：contracts.go 内容异常短（%d 字节）", len(contract))
			}
			if !strings.Contains(string(contract), "type "+d.iface+" interface") {
				t.Errorf("contracts.go 未声明域内接口 %s", d.iface)
			}
			if strings.Contains(string(contract), sharedImport) {
				t.Errorf("contracts.go 不得引用共享层包，否则该域 L.3 归零失效")
			}

			holder, err := os.ReadFile(filepath.Join(dir, d.file))
			if err != nil {
				t.Fatalf("读取 handlers/%s/%s 失败：%v", d.name, d.file, err)
			}
			code := stripLineComments(string(holder))
			// 词边界而不是子串：具体类型 TicketTagService 含 TagService 子串，
			// 用 Contains 会让「字段回退到具体类型」仍然假绿。
			ifaceWord := regexp.MustCompile(`\b` + regexp.QuoteMeta(d.iface) + `\b`)
			if !ifaceWord.MatchString(code) {
				t.Errorf("handlers/%s/%s 未使用域内接口 %s", d.name, d.file, d.iface)
			}
			for _, concrete := range strings.Split(d.concrete, ",") {
				if strings.Contains(code, "*service."+concrete) {
					t.Errorf("handlers/%s/%s 回退到共享层具体类型 %s（旧实现）", d.name, d.file, concrete)
				}
			}
		})
	}
}

// TestSharedValueTypesRemainDTOAliases 锁住「共享层值类型下沉到 dto，service 侧类型别名」：
// 若有人把定义搬回 service 包另起一份 struct，域内窄契约的签名会立刻不兼容，
// 而 L.3 计数门禁看不出来（域内仍不引用共享层）。
func TestSharedValueTypesRemainDTOAliases(t *testing.T) {
	pairs := []struct {
		name   string
		shared reflect.Type
		bottom reflect.Type
	}{
		{"DashboardViewer", reflect.TypeOf(service.DashboardViewer{}), reflect.TypeOf(dto.DashboardViewer{})},
		{"EscalationMatrix", reflect.TypeOf(service.EscalationMatrix{}), reflect.TypeOf(dto.EscalationMatrix{})},
		{"EscalationLevel", reflect.TypeOf(service.EscalationLevel{}), reflect.TypeOf(dto.EscalationLevel{})},
		{"CreateTagRequest", reflect.TypeOf(service.CreateTagRequest{}), reflect.TypeOf(dto.CreateTagRequest{})},
		{"UpdateTagRequest", reflect.TypeOf(service.UpdateTagRequest{}), reflect.TypeOf(dto.UpdateTagRequest{})},
		{"ListTagsRequest", reflect.TypeOf(service.ListTagsRequest{}), reflect.TypeOf(dto.ListTagsRequest{})},
		{"A2UIDataModelUpdate", reflect.TypeOf(service.A2UIDataModelUpdate{}), reflect.TypeOf(dto.A2UIDataModelUpdate{})},
		{"DataModelUpdate", reflect.TypeOf(service.DataModelUpdate{}), reflect.TypeOf(dto.DataModelUpdate{})},
		{"Content", reflect.TypeOf(service.Content{}), reflect.TypeOf(dto.Content{})},
		{"A2UIDeleteSurface", reflect.TypeOf(service.A2UIDeleteSurface{}), reflect.TypeOf(dto.A2UIDeleteSurface{})},
		{"DeleteSurface", reflect.TypeOf(service.DeleteSurface{}), reflect.TypeOf(dto.DeleteSurface{})},
	}
	for _, p := range pairs {
		if p.shared != p.bottom {
			t.Errorf("service.%s 必须是 dto.%s 的类型别名，否则域内契约签名不再匹配", p.name, p.name)
		}
	}

	// sentinel 必须是同一个 error 实例，否则 handler 侧 errors.Is 会静默失效。
	sentinels := []struct {
		name           string
		shared, bottom error
	}{
		{"ErrInvalidReleaseTransition", service.ErrInvalidReleaseTransition, dto.ErrInvalidReleaseTransition},
		{"ErrAssignmentRuleNotFound", service.ErrAssignmentRuleNotFound, dto.ErrAssignmentRuleNotFound},
		{"ErrTestTicketNotFound", service.ErrTestTicketNotFound, dto.ErrTestTicketNotFound},
		{"ErrProvisioningTaskSucceeded", service.ErrProvisioningTaskSucceeded, dto.ErrProvisioningTaskSucceeded},
		{"ErrProvisioningTaskInProgress", service.ErrProvisioningTaskInProgress, dto.ErrProvisioningTaskInProgress},
		{"ErrProvisioningTaskAttemptsExhausted", service.ErrProvisioningTaskAttemptsExhausted, dto.ErrProvisioningTaskAttemptsExhausted},
		{"ErrProvisioningTaskNotRetryable", service.ErrProvisioningTaskNotRetryable, dto.ErrProvisioningTaskNotRetryable},
		{"ErrProvisioningTaskNotResolvable", service.ErrProvisioningTaskNotResolvable, dto.ErrProvisioningTaskNotResolvable},
		{"ErrProvisioningNotExecutable", service.ErrProvisioningNotExecutable, dto.ErrProvisioningNotExecutable},
		{"ErrProvisioningPermanentlyFailed", service.ErrProvisioningPermanentlyFailed, dto.ErrProvisioningPermanentlyFailed},
		{"ErrTicketTagNameExists", service.ErrTicketTagNameExists, dto.ErrTicketTagNameExists},
		{"ErrTicketTagInUse", service.ErrTicketTagInUse, dto.ErrTicketTagInUse},
		{"ErrTicketTagNameBlank", service.ErrTicketTagNameBlank, dto.ErrTicketTagNameBlank},
		{"ErrTicketTagNotFound", service.ErrTicketTagNotFound, dto.ErrTicketTagNotFound},
	}
	for _, s := range sentinels {
		if !errors.Is(fmt.Errorf("wrapped: %w", s.shared), s.bottom) {
			t.Errorf("service.%s 必须与 dto 的 sentinel 指向同一实例", s.name)
		}
	}

	// 导出的 String 方法必须随类型一起下沉（service/escalation_matrix_test.go 仍调用 m.String）。
	matrix := dto.EscalationMatrix{"critical": {{Level: 1, AfterMinutes: 15, NotifyRoles: []string{"manager"}}}}
	if got := matrix.String("critical"); !strings.Contains(got, "L1(15min") {
		t.Errorf("EscalationMatrix.String 行为漂移：%q", got)
	}

	// 源码守卫：service 侧必须保留别名声明，不能重新定义值类型。
	aliases := []struct{ file, decl string }{
		{"dashboard_service.go", "type DashboardViewer = dto.DashboardViewer"},
		{"escalation_matrix.go", "type EscalationMatrix = dto.EscalationMatrix"},
		{"release_service.go", "var ErrInvalidReleaseTransition = dto.ErrInvalidReleaseTransition"},
		{"ticket_assignment_rule_service.go", "ErrAssignmentRuleNotFound = dto.ErrAssignmentRuleNotFound"},
		{"ticket_assignment_rule_service.go", "ErrTestTicketNotFound = dto.ErrTestTicketNotFound"},
		{"provisioning_service.go", "ErrProvisioningTaskSucceeded = dto.ErrProvisioningTaskSucceeded"},
		{"provisioning_service.go", "ErrProvisioningPermanentlyFailed = dto.ErrProvisioningPermanentlyFailed"},
		{"ticket_tag_service.go", "ErrTicketTagNameExists = dto.ErrTicketTagNameExists"},
		{"ticket_tag_service.go", "ErrTicketTagNotFound = dto.ErrTicketTagNotFound"},
		{"ticket_tag_service.go", "type CreateTagRequest = dto.CreateTagRequest"},
		{"ticket_tag_service.go", "type UpdateTagRequest = dto.UpdateTagRequest"},
		{"ticket_tag_service.go", "type ListTagsRequest = dto.ListTagsRequest"},
		{"a2ui_ticket_service.go", "type A2UIDataModelUpdate = dto.A2UIDataModelUpdate"},
		{"a2ui_ticket_service.go", "type DataModelUpdate = dto.DataModelUpdate"},
		{"a2ui_ticket_service.go", "type Content = dto.Content"},
		{"a2ui_ticket_service.go", "type A2UIDeleteSurface = dto.A2UIDeleteSurface"},
		{"a2ui_ticket_service.go", "type DeleteSurface = dto.DeleteSurface"},
	}
	for _, a := range aliases {
		body, err := os.ReadFile(filepath.Join("..", "..", "service", a.file))
		if err != nil {
			t.Fatalf("读取 service/%s 失败：%v", a.file, err)
		}
		if len(body) < 100 {
			t.Fatalf("扫描器零命中保护：service/%s 内容异常短（%d 字节）", a.file, len(body))
		}
		if !strings.Contains(squashSpace(stripLineComments(string(body))), squashSpace(a.decl)) {
			t.Errorf("service/%s 丢失别名声明 %q", a.file, a.decl)
		}
	}
}

// TestDomainErrorMappingsUseDTOSentinels 锁住 handler 的错误映射必须读 dto 的
// sentinel：改回共享层限定符会重新引入共享层 import，改成域内自建 error 会让
// errors.Is 静默失效。两种退化都要在这里变红，而不是等到线上返回码漂移。
func TestDomainErrorMappingsUseDTOSentinels(t *testing.T) {
	cases := []struct {
		domain    string
		file      string
		sentinels []string
	}{
		{"assignment_smart", "handler.go", []string{"dto.ErrAssignmentRuleNotFound", "dto.ErrTestTicketNotFound"}},
		{"provisioning", "handler.go", []string{
			"dto.ErrProvisioningTaskSucceeded", "dto.ErrProvisioningTaskInProgress",
			"dto.ErrProvisioningTaskAttemptsExhausted", "dto.ErrProvisioningTaskNotRetryable",
			"dto.ErrProvisioningTaskNotResolvable",
		}},
		{"ticket_tag", "handler.go", []string{
			"dto.ErrTicketTagNameExists", "dto.ErrTicketTagInUse",
			"dto.ErrTicketTagNameBlank", "dto.ErrTicketTagNotFound",
		}},
	}
	for _, c := range cases {
		t.Run(c.domain, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join("..", "..", "handlers", c.domain, c.file))
			if err != nil {
				t.Fatalf("读取 handlers/%s/%s 失败：%v", c.domain, c.file, err)
			}
			if len(body) < 100 {
				t.Fatalf("扫描器零命中保护：handlers/%s/%s 内容异常短（%d 字节）", c.domain, c.file, len(body))
			}
			code := stripLineComments(string(body))
			for _, s := range c.sentinels {
				if !strings.Contains(code, s) {
					t.Errorf("handlers/%s/%s 未使用 %s 做错误映射", c.domain, c.file, s)
				}
			}
			if strings.Contains(code, "service.Err") {
				t.Errorf("handlers/%s/%s 错误映射回退到共享层限定符", c.domain, c.file)
			}
		})
	}
}
