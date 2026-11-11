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
	"itsm-backend/handlers/skill"
	"itsm-backend/service"
)

// L.3 归零收官专项回归锁（2026-11-10）。
//
// 本批把最后两个域（change / service_request）归零，L.3 达到 0 文件 / 0 域：
// 全部 handler 生产文件零共享层 import。本文件锁住三件事：
//   1. 全仓 handler 生产文件零共享层 import（且基线不得复活任何 domain= 行）；
//   2. 审批链/变更状态机的值类型与纯函数组唯一实现在 dto（service 侧别名）；
//   3. 两域的窄契约形态与组合根接线（app.go 必须注入桥接与收口函数）。
//
// 最大的假绿风险有两个：
//   - 迁移表下沉后 service 侧若保留第二份字面量，状态机与数据库 CHECK 词表
//     会各说各话（mock 全绿、生产撞 23514）；
//   - 桥接构造上移组合根后，若 app.go 漏注入，approvalBridge 为 nil，
//     流程推进路径被静默跳过——编译与 L.3 计数都看不出问题。

// TestHandlersHaveZeroSharedLayerImport：全仓 handler 生产文件零共享层 import。
func TestHandlersHaveZeroSharedLayerImport(t *testing.T) {
	root := filepath.Join("..", "..", "handlers")
	importing := []string{}
	prodTotal := 0
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("读取 handlers 失败：%v", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("读取 %s 失败：%v", dir, err)
		}
		for _, f := range files {
			fn := f.Name()
			if f.IsDir() || !strings.HasSuffix(fn, ".go") || strings.HasSuffix(fn, "_test.go") {
				continue
			}
			prodTotal++
			body, err := os.ReadFile(filepath.Join(dir, fn))
			if err != nil {
				t.Fatalf("读取 %s 失败：%v", fn, err)
			}
			if len(body) < 40 {
				t.Fatalf("扫描器零命中保护：handlers/%s/%s 仅 %d 字节", e.Name(), fn, len(body))
			}
			if strings.Contains(string(body), `"itsm-backend/service"`) {
				importing = append(importing, e.Name()+"/"+fn)
			}
		}
	}
	if prodTotal < 50 {
		t.Fatalf("扫描器零命中保护：handlers 生产文件仅 %d 个", prodTotal)
	}
	if len(importing) != 0 {
		t.Errorf("L.3 归零被破坏：仍 import 共享层的生产文件 %v", importing)
	}

	// 基线不得残留任何 domain= 行（含 domain=0 假账）。
	base := readL3RepoFile(t, "..", "scripts", "docs-gate", "handler-service-imports-baseline.txt")
	if m := regexp.MustCompile(`(?m)^[a-z_]+=\d+$`).FindString(base); m != "" {
		t.Errorf("L.3 基线不得残留 domain= 行（归零后应全部删除），仍见 %q", m)
	}
}

func readL3RepoFile(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(append([]string{"..", ".."}, parts...)...)
	body, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("读取 %s 失败：%v", p, err)
	}
	if len(body) < 100 {
		t.Fatalf("扫描器零命中保护：%s 仅 %d 字节", p, len(body))
	}
	return string(body)
}

// TestApprovalChainValueTypesAreDTOAliases：审批链值类型身份与形状锁。
func TestApprovalChainValueTypesAreDTOAliases(t *testing.T) {
	pairs := []struct {
		name   string
		shared reflect.Type
		bottom reflect.Type
	}{
		{"ApprovalEvalContext", reflect.TypeOf(service.ApprovalEvalContext{}), reflect.TypeOf(dto.ApprovalEvalContext{})},
		{"ApprovalLevelEval", reflect.TypeOf(service.ApprovalLevelEval{}), reflect.TypeOf(dto.ApprovalLevelEval{})},
		{"ApprovalChainEvaluation", reflect.TypeOf(service.ApprovalChainEvaluation{}), reflect.TypeOf(dto.ApprovalChainEvaluation{})},
		{"ShadowResult", reflect.TypeOf(service.ShadowResult{}), reflect.TypeOf(dto.ShadowResult{})},
	}
	for _, p := range pairs {
		if p.shared != p.bottom {
			t.Errorf("service.%s 与 dto.%s 不再是同一类型（别名被换成独立副本）", p.name, p.name)
		}
	}
	shapes := []struct {
		name string
		typ  reflect.Type
		want string
	}{
		{"ApprovalEvalContext", reflect.TypeOf(dto.ApprovalEvalContext{}), "TenantID,EntityType,RequesterID,Priority,Amount"},
		{"ApprovalLevelEval", reflect.TypeOf(dto.ApprovalLevelEval{}), "Level,Required,ApprovalType,Threshold,ApproverIDs,ApproverNames,FallbackTriggered,FallbackAction,Status,Skipped,SkipReason"},
		{"ApprovalChainEvaluation", reflect.TypeOf(dto.ApprovalChainEvaluation{}), "ChainID,Levels,Passed,PendingLevel,Blocked"},
		{"ShadowResult", reflect.TypeOf(dto.ShadowResult{}), "ChainID,Passed,PendingLevel,Blocked,LevelsJSON,Err"},
	}
	for _, s := range shapes {
		if got := fieldNames(s.typ); got != s.want {
			t.Errorf("dto.%s 字段形状漂移：%s", s.name, got)
		}
	}
	// service 侧必须是别名声明（逐处比对，不取并集）。
	for _, a := range []struct{ file, decl string }{
		{"approval_chain_evaluation.go", "type ApprovalEvalContext = dto.ApprovalEvalContext"},
		{"approval_chain_evaluation.go", "type ApprovalLevelEval = dto.ApprovalLevelEval"},
		{"approval_chain_evaluation.go", "type ApprovalChainEvaluation = dto.ApprovalChainEvaluation"},
		{"approval_chain_shadow.go", "type ShadowResult = dto.ShadowResult"},
	} {
		src := readL3RepoFile(t, "service", a.file)
		if !strings.Contains(squashSpace(src), squashSpace(a.decl)) {
			t.Errorf("service/%s 缺少别名声明 %q", a.file, a.decl)
		}
	}
}

// TestChangeStatusMachineIsSingleSource：变更状态机纯函数组唯一实现在 dto；
// service 侧的迁移表变量必须是 dto 变量的别名（否则状态机与 DB CHECK 词表各说各话）。
func TestChangeStatusMachineIsSingleSource(t *testing.T) {
	src := readL3RepoFile(t, "service", "change_service.go")
	for _, decl := range []string{
		"var changeBaseTransitions = dto.ChangeBaseTransitions",
		"var changeStandardTransitions = dto.ChangeStandardTransitions",
		"var changeEmergencyTransitions = dto.ChangeEmergencyTransitions",
		"var changeNormalTransitions = dto.ChangeNormalTransitions",
		"func IsValidChangeStatusTransition(currentStatus, newStatus, changeType string) bool {\n\treturn dto.IsValidChangeStatusTransition(currentStatus, newStatus, changeType)",
	} {
		if !strings.Contains(squashSpace(src), squashSpace(decl)) {
			t.Errorf("service/change_service.go 缺少薄委托/别名 %q", decl)
		}
	}
	// service 侧不得再出现第二份迁移表字面量。
	if strings.Contains(src, "changeBaseTransitions = map[string][]string{") {
		t.Errorf("service/change_service.go 仍有第二份迁移表字面量：唯一实现在 dto")
	}
	// 行为锁：状态机的判定结果必须与下沉前一致（抽样若干组合）。
	cases := []struct {
		from, to, typ string
		want          bool
	}{
		{"draft", "submitted", "normal", true},
		{"draft", "pending", "normal", true}, // pending 与 submitted 同义
		{"draft", "approved", "normal", false},
		{"draft", "approved", "standard", true},
		{"submitted", "approved", "normal", true},
		{"submitted", "rejected", "normal", true},
		{"completed", "closed", "normal", true},
		{"cancelled", "approved", "normal", false},
		{"unknown_status", "approved", "normal", false}, // 未知状态 fail-closed
	}
	for _, c := range cases {
		if got := service.IsValidChangeStatusTransition(c.from, c.to, c.typ); got != c.want {
			t.Errorf("IsValidChangeStatusTransition(%q,%q,%q)=%v，期望 %v", c.from, c.to, c.typ, got, c.want)
		}
		if got := dto.IsValidChangeStatusTransition(c.from, c.to, c.typ); got != c.want {
			t.Errorf("dto.IsValidChangeStatusTransition(%q,%q,%q)=%v，期望 %v", c.from, c.to, c.typ, got, c.want)
		}
	}
}

// TestChangeAndSRContractShapes：两域窄契约与构造形态源码级锚定。
func TestChangeAndSRContractShapes(t *testing.T) {
	changeContract := readL3RepoFile(t, "handlers", "change", "contracts.go")
	for _, decl := range []string{
		"type ApprovalChainResolver interface",
		"ResolveApprovalPlan(ctx context.Context, tenantID int, entityType string, evalCtx dto.ApprovalEvalContext, approvals map[int][]int) (*dto.ApprovalChainEvaluation, error)",
		"ShadowCompare(ctx context.Context, tenantID int, entityType string, evalCtx dto.ApprovalEvalContext, approvals map[int][]int, directPlan *dto.ApprovalChainEvaluation, directErr error) *dto.ShadowResult",
		"type ChangeApprovalBridge interface",
		"AdvanceBusinessWorkTaskWithClient(ctx context.Context, txc *ent.Client, tenantID, actorUserID int, businessType string, businessID int, variables map[string]interface{}) (int, error)",
		"AdvanceBusinessWorkflowWithClient(ctx context.Context, txc *ent.Client, tenantID, actorUserID int, businessType string, businessID int, variables map[string]interface{}, maxSteps int) (bool, error)",
		"CompleteBusinessApprovalTaskWithClient(ctx context.Context, txc *ent.Client, tenantID, actorUserID int, businessType string, businessID int, action, comment string) (bool, error)",
		"type ApprovalChainCloser func(ctx context.Context, changeID, tenantID int) error",
		"func WithApprovalBridge(bridge ChangeApprovalBridge) ServiceOption",
		"func WithApprovalChainCloser(closer ApprovalChainCloser) ServiceOption",
	} {
		if !strings.Contains(squashSpace(changeContract), squashSpace(decl)) {
			t.Errorf("handlers/change/contracts.go 契约形态漂移：%q", decl)
		}
	}
	srContract := readL3RepoFile(t, "handlers", "service_request", "contracts.go")
	for _, decl := range []string{
		"type ApprovalChainResolver interface",
		"type SRApprovalBridge interface",
		"CompleteBusinessWorkTasks(ctx context.Context, tenantID, actorUserID int, businessType string, businessID int, variables map[string]interface{}, maxSteps int) (int, error)",
		"CheckActionableApprovalTask(ctx context.Context, tenantID int, businessType string, businessID int) (bool, error)",
		"TerminateBusinessWorkflow(ctx context.Context, tenantID, actorUserID int, businessType string, businessID int, reason string) error",
		"type SRSLAService interface",
		"CalculateSLADeadlineFromRequest(ctx context.Context, tenantID int, ticketType, priority string) (*dto.SLADeadlineResult, error)",
		"SLAStore() *sla.Store",
		"type ProvisioningDeliveryListener interface",
		"OnProvisioningStarted(ctx context.Context, tenantID, serviceRequestID, actorUserID int)",
		"OnProvisioningFinished(ctx context.Context, tenantID, serviceRequestID, actorUserID int, delivered bool, detail string)",
		"func WithApprovalBridge(bridge SRApprovalBridge) ServiceOption",
	} {
		if !strings.Contains(squashSpace(srContract), squashSpace(decl)) {
			t.Errorf("handlers/service_request/contracts.go 契约形态漂移：%q", decl)
		}
	}
	// 两域生产文件不得引用共享层包。
	for _, rel := range [][]string{
		{"handlers", "change", "service.go"},
		{"handlers", "service_request", "service.go"},
	} {
		body := readL3RepoFile(t, rel...)
		if strings.Contains(body, `"itsm-backend/service"`) {
			t.Errorf("handlers/%s/%s 重新引用共享层包", rel[1], rel[2])
		}
	}
}

// TestCompositionRootInjectsBridgeAndCloser：组合根必须注入桥接与收口函数——
// 漏注入时 approvalBridge 为 nil，流程推进路径被静默跳过，编译与 L.3 计数都绿。
func TestCompositionRootInjectsBridgeAndCloser(t *testing.T) {
	app := readL3RepoFile(t, "internal", "bootstrap", "app.go")
	for _, decl := range []string{
		"change.NewService(changeRepo, client, sugar, approvalChainService, change.WithApprovalBridge(service.NewBPMNApprovalBridge(client, sugar)), change.WithApprovalChainCloser(service.CloseChangeApprovalChains))",
		"service_request.NewService(srRepo, scRepo, cmdbRepo, client, sugar, approvalChainService, service_request.WithApprovalBridge(service.NewBPMNApprovalBridge(client, sugar)))",
	} {
		if !strings.Contains(squashSpace(app), squashSpace(decl)) {
			t.Errorf("组合根缺少依赖注入：%q", decl)
		}
	}
}

// TestChangeServiceBridgeNilGuard：桥接未接线时必须失败关闭而不是 panic。
func TestChangeServiceBridgeNilGuard(t *testing.T) {
	body := readL3RepoFile(t, "handlers", "service_request", "stale_process_instances.go")
	if !strings.Contains(body, "s.approvalBridge == nil") {
		t.Errorf("handlers/service_request/stale_process_instances.go 缺少桥接未接线的 nil 防护（会 panic）")
	}
}

// TestServiceSentinelsUnchanged：确认本批没有动到既有 sentinel 语义。
func TestServiceSentinelsUnchanged(t *testing.T) {
	if !errors.Is(fmt.Errorf("wrapped: %w", dto.ErrSkillNotFound), skill.ErrSkillNotFound) {
		t.Errorf("技能 sentinel 同一实例语义被破坏")
	}
}
