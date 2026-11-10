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
	"itsm-backend/handlers/bpmn"
	"itsm-backend/handlers/escalation_matrix"
	"itsm-backend/handlers/incident"
	"itsm-backend/handlers/prediction"
	"itsm-backend/handlers/problem_investigation"
	"itsm-backend/handlers/provisioning"
	"itsm-backend/handlers/release"
	"itsm-backend/handlers/ticket"
	"itsm-backend/handlers/ticket_notification"
	"itsm-backend/handlers/ticket_tag"
	"itsm-backend/handlers/ticket_type"
	"itsm-backend/handlers/ticket_workflow"
	"itsm-backend/handlers/timer"
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
	_ incident.IncidentProductionService                = (*service.IncidentService)(nil)
	_ incident.IncidentMonitoringProvider               = (*service.IncidentMonitoringService)(nil)
	_ incident.IncidentAlertingProvider                 = (*service.IncidentAlertingService)(nil)
	_ incident.IncidentRootCauseService                 = (*service.RootCauseAnalysisService)(nil)
	_ incident.IncidentSLAMonitor                       = (*service.SLAMonitorService)(nil)
	_ ticket.TicketProductionService                    = (*service.TicketService)(nil)
	_ ticket.TicketSLAMonitor                           = (*service.SLAMonitorService)(nil)
	// 共享层是接口而非指针类型：接口到接口赋值同样在编译期检查方法集，
	// 这能锁住 List/Stats 的 dto 值类型身份不被换成另一份同名结构。
	_ timer.TimerReadStore = (service.TimerStore)(nil)
	// 批次 3 第 1 步：bpmn 域只有 workflow_template.go 归零，不能进 contractDomains
	// （那个守卫逐文件扫整个域，其余 7 个文件仍依赖共享层）；这里单独锁窄契约。
	_ bpmn.WorkflowTemplateCatalog = (*service.BPMNWorkflowTemplateCatalog)(nil)
	// 批次 3 第 2 步：组 3 五个窄契约同样单独锁（bpmn 仍是部分归零域）。
	// 值类型身份靠下面 21 条别名守卫与 dto JSON 契约共同保证。
	_ bpmn.DashboardMetricsProvider  = (*service.BPMNMetricsService)(nil)
	_ bpmn.BPMNAuditLogReader        = (*service.BPMNAuditService)(nil)
	_ bpmn.BPMNTenantStatsProvider   = (*service.BPMNTenantService)(nil)
	_ bpmn.BPMNSLAProvider           = (*service.BPMNSLAService)(nil)
	_ bpmn.ProcessMonitoringProvider = (*service.BPMNMonitoringService)(nil)
	// 批次 3 第 3 步：组 2 三个窄契约（bpmn 仍是部分归零域）。
	_ bpmn.ProcessTriggerRunner  = (*service.ProcessTriggerService)(nil)
	_ bpmn.ProcessBindingCatalog = (*service.ProcessBindingService)(nil)
	_ bpmn.DomainConfigResolver  = (*service.ConfigInheritanceService)(nil)
	// 批次 3 第 4 步：组 1 引擎 / 版本 / AI 生成器三个窄契约。
	// ProcessEngine 本身是接口，用接口到接口赋值锁住「三个访问器的返回类型
	// 与域内契约完全一致」——若共享层偷偷把访问器改回包内命名接口，这里先红。
	_ bpmn.WorkflowEngine     = (service.ProcessEngine)(nil)
	_ bpmn.VersionCatalog     = (*service.BPMNVersionService)(nil)
	_ bpmn.AIGeneratorCatalog = (*service.BPMNAIGeneratorService)(nil)
)

type domainSpec struct {
	name     string // handlers 下的域目录名
	file     string // 持有契约字段的生产文件
	iface    string // 域内窄接口名
	concrete string // 共享层具体类型名；多接口域用逗号分隔，逐个检查
	bare     string // 共享层是接口（非指针）时，额外检查裸限定符，如 service.TimerStore
}

var contractDomains = []domainSpec{
	{"account", "handler.go", "UserService", "UserService", ""},
	{"analytics", "handler.go", "AnalyticsService", "AnalyticsService", ""},
	{"approval", "handler.go", "ApprovalService", "ApprovalService", ""},
	{"escalation_matrix", "handler.go", "EscalationMatrixService", "EscalationMatrixService", ""},
	{"prediction", "handler.go", "PredictionService", "PredictionService", ""},
	{"release", "handler.go", "ReleaseService", "ReleaseService", ""},
	{"approval_chain", "handler.go", "ApprovalChainService", "ApprovalChainService", ""},
	{"auth", "service.go", "EmailService", "EmailService", ""},
	{"automation_rule", "handler.go", "AutomationRuleService", "TicketAutomationRuleService", ""},
	{"problem_investigation", "handler.go", "ProblemInvestigationService", "ProblemInvestigationService", ""},
	{"ticket_notification", "handler.go", "TicketNotificationService", "TicketNotificationService", ""},
	{"ticket_type", "handler.go", "TicketTypeService", "TicketTypeService", ""},
	{"ticket_workflow", "handler.go", "TicketWorkflowService", "TicketWorkflowService", ""},
	{"vector_store", "handler.go", "VectorStore", "VectorStore", ""},
	{"a2ui", "handler.go", "TicketFormService", "A2UITicketService", ""},
	{"assignment_smart", "handler.go", "SmartAssignmentService", "TicketAssignmentSmartService,TicketAssignmentRuleService", ""},
	{"provisioning", "handler.go", "ProvisioningTaskService", "ProvisioningService", ""},
	{"ticket_tag", "handler.go", "TagService", "TicketTagService", ""},
	{"incident", "service.go", "IncidentProductionService", "IncidentService,IncidentMonitoringService,IncidentAlertingService,RootCauseAnalysisService,SLAMonitorService", ""},
	{"ticket", "service.go", "TicketProductionService", "TicketService,SLAMonitorService", ""},
	{"timer", "service.go", "TimerReadStore", "TimerStore", "TimerStore"},
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
				// 共享层是接口类型时没有 "*service.X" 形态，需单独锁裸限定符，
				// 否则把 store 改回 service.TimerStore 仍能靠全文 import 检查之外的路径漏过。
				if d.bare != "" && strings.Contains(bodyCode, "service."+d.bare) {
					t.Errorf("handlers/%s/%s 字段回退到共享层接口 service.%s", d.name, fn, d.bare)
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
		{"ResolvedConfig", reflect.TypeOf(service.ResolvedConfig{}), reflect.TypeOf(dto.ResolvedConfig{})},
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
		{"ErrIncidentNotFound", service.ErrIncidentNotFound, dto.ErrIncidentNotFound},
		{"ErrIncidentTerminal", service.ErrIncidentTerminal, dto.ErrIncidentTerminal},
		{"ErrIncidentEscalationLevelInvalid", service.ErrIncidentEscalationLevelInvalid, dto.ErrIncidentEscalationLevelInvalid},
		{"ErrIncidentEscalationLevelNotGreater", service.ErrIncidentEscalationLevelNotGreater, dto.ErrIncidentEscalationLevelNotGreater},
		{"ErrIncidentEscalationReasonRequired", service.ErrIncidentEscalationReasonRequired, dto.ErrIncidentEscalationReasonRequired},
		{"ErrEmailContractNotActive", service.ErrEmailContractNotActive, dto.ErrEmailContractNotActive},
		{"ErrIncidentInvalidTransition", service.ErrIncidentInvalidTransition, dto.ErrIncidentInvalidTransition},
		{"ErrIncidentResolutionRequired", service.ErrIncidentResolutionRequired, dto.ErrIncidentResolutionRequired},
		{"ErrIncidentCloseNotesRequired", service.ErrIncidentCloseNotesRequired, dto.ErrIncidentCloseNotesRequired},
		{"ErrIncidentVersionConflict", service.ErrIncidentVersionConflict, dto.ErrIncidentVersionConflict},
		{"ErrWorkflowTemplateNotFound", service.ErrWorkflowTemplateNotFound, dto.ErrWorkflowTemplateNotFound},
		{"ErrWorkflowTemplateConflict", service.ErrWorkflowTemplateConflict, dto.ErrWorkflowTemplateConflict},
		{"ErrWorkflowTemplateInvalid", service.ErrWorkflowTemplateInvalid, dto.ErrWorkflowTemplateInvalid},
		{"ErrConfigNotFound", service.ErrConfigNotFound, dto.ErrConfigNotFound},
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
		{"incident_service.go", "ErrIncidentNotFound = dto.ErrIncidentNotFound"},
		{"incident_service.go", "ErrIncidentVersionConflict = dto.ErrIncidentVersionConflict"},
		{"sla_monitor_service.go", "type SLAMetrics = dto.SLAMonitorMetrics"},
		{"ticket_service.go", "type TicketSLAInfo = dto.TicketSLAInfo"},
		{"timer_store.go", "type TimerListFilter = dto.TimerListFilter"},
		{"timer_store.go", "type TimerStats = dto.TimerStats"},
		{"bpmn_workflow_template_catalog.go", "ErrWorkflowTemplateNotFound = dto.ErrWorkflowTemplateNotFound"},
		{"bpmn_workflow_template_catalog.go", "ErrWorkflowTemplateConflict = dto.ErrWorkflowTemplateConflict"},
		{"bpmn_workflow_template_catalog.go", "ErrWorkflowTemplateInvalid = dto.ErrWorkflowTemplateInvalid"},
		{"bpmn_pending_inventory.go", "const DefaultStaleThresholdHours = dto.DefaultStaleThresholdHours"},
		// 批次 3 第 2 步：21 个 BPMN 监控值类型逐处锁别名，不取并集。
		{"bpmn_audit_service.go", "type QueryAuditLogsRequest = dto.QueryAuditLogsRequest"},
		{"bpmn_metrics_service.go", "type DashboardMetrics = dto.DashboardMetrics"},
		{"bpmn_metrics_service.go", "type ProcessHealth = dto.ProcessHealth"},
		{"bpmn_metrics_service.go", "type ProcessStat = dto.ProcessStat"},
		{"bpmn_metrics_service.go", "type TaskStat = dto.TaskStat"},
		{"bpmn_metrics_service.go", "type TrendPoint = dto.TrendPoint"},
		{"bpmn_metrics_service.go", "type BPMNProcessMetrics = dto.BPMNProcessMetrics"},
		{"bpmn_metrics_service.go", "type BottleneckInfo = dto.BottleneckInfo"},
		{"bpmn_monitoring_service.go", "type ProcessMetricsRequest = dto.ProcessMetricsRequest"},
		{"bpmn_monitoring_service.go", "type ListProcessInstanceStatusQuery = dto.ListProcessInstanceStatusQuery"},
		{"bpmn_monitoring_service.go", "type ProcessMetrics = dto.ProcessMetrics"},
		{"bpmn_monitoring_service.go", "type PerformanceMetrics = dto.PerformanceMetrics"},
		{"bpmn_monitoring_service.go", "type TaskMetrics = dto.TaskMetrics"},
		{"bpmn_monitoring_service.go", "type BottleneckAnalysis = dto.BottleneckAnalysis"},
		{"bpmn_monitoring_service.go", "type BottleneckTask = dto.BottleneckTask"},
		{"bpmn_monitoring_service.go", "type SlowestPath = dto.SlowestPath"},
		{"bpmn_monitoring_service.go", "type ResourceConstraint = dto.ResourceConstraint"},
		{"bpmn_monitoring_service.go", "type ProcessInstanceStatus = dto.ProcessInstanceStatus"},
		{"bpmn_monitoring_service.go", "type ProcessTimelineEntry = dto.ProcessTimelineEntry"},
		{"bpmn_sla_service.go", "type SLAViolation = dto.SLAViolation"},
		{"bpmn_tenant_service.go", "type TenantBPMNStats = dto.TenantBPMNStats"},
		// 批次 3 第 3 步：层级域配置的 sentinel 与解析结果逐处锁别名。
		{"config_inheritance_service.go", "var ErrConfigNotFound = dto.ErrConfigNotFound"},
		{"config_inheritance_service.go", "type ResolvedConfig = dto.ResolvedConfig"},
		// 批次 3 第 4 步：17 个引擎组值类型 + 3 个子服务接口别名，逐处比对不取并集。
		{"bpmn_process_definition_service.go", "type CreateProcessDefinitionRequest = dto.CreateProcessDefinitionRequest"},
		{"bpmn_process_definition_service.go", "type UpdateProcessDefinitionRequest = dto.UpdateProcessDefinitionRequest"},
		{"bpmn_process_definition_service.go", "type ListProcessDefinitionsRequest = dto.ListProcessDefinitionsRequest"},
		{"bpmn_process_instance_service.go", "type ListProcessInstancesRequest = dto.ListProcessInstancesRequest"},
		{"bpmn_process_instance_service.go", "type InstanceStatisticsRequest = dto.InstanceStatisticsRequest"},
		{"bpmn_process_instance_service.go", "type InstanceStatistics = dto.InstanceStatistics"},
		{"bpmn_task_service.go", "type ListUserTasksRequest = dto.ListUserTasksRequest"},
		{"bpmn_task_service.go", "type TaskStatisticsRequest = dto.TaskStatisticsRequest"},
		{"bpmn_task_service.go", "type TaskStatistics = dto.TaskStatistics"},
		{"bpmn_task_service.go", "type CounterSignStatus = dto.CounterSignStatus"},
		{"bpmn_task_service.go", "type CounterSignRequest = dto.CounterSignRequest"},
		{"bpmn_task_service.go", "type VoteRequest = dto.VoteRequest"},
		{"bpmn_version_service.go", "type ProcessVersion = dto.ProcessVersion"},
		{"bpmn_version_service.go", "type CreateVersionRequest = dto.CreateVersionRequest"},
		{"bpmn_version_service.go", "type UpdateVersionRequest = dto.UpdateVersionRequest"},
		{"bpmn_version_service.go", "type VersionComparison = dto.VersionComparison"},
		{"bpmn_version_service.go", "type ChangeDetail = dto.ChangeDetail"},
		{"bpmn_process_engine.go", "type ProcessDefinitionService = dto.ProcessDefinitionCatalog"},
		{"bpmn_process_engine.go", "type ProcessInstanceService = dto.ProcessInstanceCatalog"},
		{"bpmn_process_engine.go", "type TaskService = dto.TaskCatalog"},
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

// TestRenamedDTOAliasesShareTypeIdentity 锁住「下沉后改名」的别名身份：
// dto.SLAMetrics 已被仪表盘达成率视图占用，监控指标只能叫 SLAMonitorMetrics，
// 但 service.SLAMetrics 必须仍指向同一个类型，否则 CalculateSLAMetrics 的返回
// 签名会与 ticket 域窄契约不兼容（L.3 计数门禁看不出这种退化）。
func TestRenamedDTOAliasesShareTypeIdentity(t *testing.T) {
	cases := []struct {
		name           string
		shared, bottom reflect.Type
	}{
		{"SLAMetrics/SLAMonitorMetrics", reflect.TypeOf(service.SLAMetrics{}), reflect.TypeOf(dto.SLAMonitorMetrics{})},
		{"TicketSLAInfo", reflect.TypeOf(service.TicketSLAInfo{}), reflect.TypeOf(dto.TicketSLAInfo{})},
		{"TimerListFilter", reflect.TypeOf(service.TimerListFilter{}), reflect.TypeOf(dto.TimerListFilter{})},
		{"TimerStats", reflect.TypeOf(service.TimerStats{}), reflect.TypeOf(dto.TimerStats{})},
	}
	for _, c := range cases {
		if c.shared != c.bottom {
			t.Errorf("%s：共享层类型与 dto 下沉类型必须身份一致（%v vs %v）", c.name, c.shared, c.bottom)
		}
	}
	// 仪表盘视图与监控指标是两个不同结构，禁止顺手合并成一个类型。
	if reflect.TypeOf(dto.SLAMetrics{}) == reflect.TypeOf(dto.SLAMonitorMetrics{}) {
		t.Error("dto.SLAMetrics 与 dto.SLAMonitorMetrics 是两个不同视图，禁止合并")
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
		{"incident", "service.go", []string{
			"dto.ErrIncidentVersionConflict", "dto.ErrIncidentInvalidTransition",
			"dto.ErrIncidentResolutionRequired", "dto.ErrIncidentCloseNotesRequired",
			"dto.ErrIncidentNotFound",
		}},
		{"bpmn", "workflow_template.go", []string{
			"dto.ErrWorkflowTemplateNotFound", "dto.ErrWorkflowTemplateConflict",
			"dto.ErrWorkflowTemplateInvalid",
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
