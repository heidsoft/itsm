package layering

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/service"
)

// 批次 3 第 5 步 B：operations / problem 两域靠「组合根注入」整域归零。
//
// 与前面各组的差别：这两域的共享层依赖不是「字段是具体类型」，而是
// ① 包级函数 RecoverInterruptedToolInvocationTx、② 域内直接构造
// NewBPMNApprovalBridge。前者改成构造期注入的函数字段，后者把构造上移
// app.go。两者的共同风险是「域内改完了但组合根没接线」——编译能过、
// L.3 计数也是 0，运行时却会 nil panic 或静默不推进流程，因此本文件的
// 断言必须同时覆盖域内形态与装配点源码。

const sharedLayerImport = `"itsm-backend/service"`

func readRepoFile(t *testing.T, parts ...string) string {
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

// TestCompositionRootDomainsHaveZeroSharedImports 是 L.3 归零的代码侧锁：
// 只改基线不改代码（或反过来）都要变红。
func TestCompositionRootDomainsHaveZeroSharedImports(t *testing.T) {
	// 生产文件下限按实测逐域锁定：零命中不能由「文件被删掉」造成。
	minimums := map[string]int{"operations": 2, "problem": 7}
	for _, domain := range []string{"operations", "problem"} {
		dir := filepath.Join("..", "..", "handlers", domain)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("列举 handlers/%s 失败：%v", domain, err)
		}
		prod := 0
		for _, e := range entries {
			fn := e.Name()
			if e.IsDir() || !strings.HasSuffix(fn, ".go") || strings.HasSuffix(fn, "_test.go") {
				continue
			}
			prod++
			body, err := os.ReadFile(filepath.Join(dir, fn))
			if err != nil {
				t.Fatalf("读取 handlers/%s/%s 失败：%v", domain, fn, err)
			}
			if len(body) < 40 {
				t.Fatalf("扫描器零命中保护：handlers/%s/%s 仅 %d 字节", domain, fn, len(body))
			}
			if strings.Contains(string(body), sharedLayerImport) {
				t.Errorf("handlers/%s/%s 重新引用共享层包，整域归零被破坏", domain, fn)
			}
		}
		if prod < minimums[domain] {
			t.Fatalf("扫描器零命中保护：handlers/%s 只剩 %d 个生产 .go 文件（期望至少 %d）",
				domain, prod, minimums[domain])
		}
	}

	base := readRepoFile(t, "..", "scripts", "docs-gate", "handler-service-imports-baseline.txt")
	for _, domain := range []string{"operations", "problem"} {
		if regexp.MustCompile(`(?m)^` + domain + `=`).MatchString(base) {
			t.Errorf("L.3 基线仍保留 %s= 行（整域归零必须删除，不能留 %s=0 槽位）", domain, domain)
		}
	}
}

// TestOperationsRecoveryIsInjected 锁住「注入而非隐式降级」：恢复函数必须由
// 组合根与每一个测试调用点显式传入。任何一处漏传都会让死信重放静默跳过
// 中断的 AI 工具调用解锁，而编译与 L.3 计数都看不出来。
func TestOperationsRecoveryIsInjected(t *testing.T) {
	svc := readRepoFile(t, "handlers", "operations", "service.go")
	if strings.Contains(svc, sharedLayerImport) {
		t.Errorf("operations/service.go 重新引用共享层包")
	}
	code := stripLineComments(svc)
	for _, want := range []string{
		"type ToolInvocationRecovery func(",
		`recoverToolInvocation ToolInvocationRecovery`,
		`NewService(client *ent.Client, recoverToolInvocation ToolInvocationRecovery)`,
		"toolsexec.",
	} {
		if want == "toolsexec." {
			if strings.Contains(code, want) {
				t.Errorf("operations/service.go 残留共享层包限定符（旧实现）")
			}
			continue
		}
		if !strings.Contains(code, want) {
			t.Errorf("operations/service.go 缺少注入形态 %q", want)
		}
	}
	if got := strings.Count(code, "s.recoverToolInvocation("); got != 2 {
		t.Errorf("恢复函数调用点 %d 处，期望 2（单条 replay + 批量 replay）", got)
	}
	if strings.Contains(code, "RecoverInterruptedToolInvocationTx") {
		t.Errorf("operations/service.go 仍直接调用共享层包级函数（旧实现）")
	}

	router := readRepoFile(t, "router", "router.go")
	if !strings.Contains(stripLineComments(router),
		"operations.NewService(config.Client, service.RecoverInterruptedToolInvocationTx)") {
		t.Errorf("router.go 未把真实恢复函数注入 operations.NewService")
	}

	// 测试调用点同样必须显式传：否则「恢复生效」的 E2E 断言会被悄悄绕过。
	for _, rel := range [][]string{
		{"handlers", "operations", "service_test.go"},
		{"handlers", "operations", "handler_test.go"},
		{"tests", "scenarios", "scenario9_outbox_retry_deadletter_test.go"},
	} {
		body := stripLineComments(readRepoFile(t, rel...))
		total := strings.Count(body, "NewService(client")
		injected := strings.Count(body, "NewService(client, toolsexec.RecoverInterruptedToolInvocationTx")
		if total == 0 || total != injected {
			t.Errorf("%s 的 NewService 调用点 %d 处，显式注入 %d 处（必须全量传入真实恢复函数）",
				strings.Join(rel, "/"), total, injected)
		}
	}
}

// TestProblemContractsAndBridgeWiring 锁住 problem 域的两个窄契约、值类型
// 单一来源、以及 app.go / 测试夹具的真实接线。
func TestProblemContractsAndBridgeWiring(t *testing.T) {
	svc := readRepoFile(t, "handlers", "problem", "service.go")
	if strings.Contains(svc, sharedLayerImport) {
		t.Errorf("problem/service.go 重新引用共享层包")
	}
	code := stripLineComments(svc)
	for _, want := range []string{
		`slaSvc SLAService`,
		`approvalBridge WorkflowBridge`,
		`bridge ...WorkflowBridge`,
		`svc.approvalBridge = bridge[0]`,
		`SetSLAService(slaSvc SLAService)`,
	} {
		if !strings.Contains(code, want) {
			t.Errorf("problem/service.go 缺少窄契约/注入写法 %q", want)
		}
	}
	if strings.Contains(code, "service.NewBPMNApprovalBridge") ||
		strings.Contains(code, "*service.TicketSLAService") {
		t.Errorf("problem/service.go 回退到共享层具体类型或域内构造（旧实现）")
	}

	contract := readRepoFile(t, "handlers", "problem", "contracts.go")
	if strings.Contains(contract, sharedLayerImport) {
		t.Errorf("problem/contracts.go 出现共享层包路径字面量（连注释也会让 L.3 计数 +1）")
	}
	for _, want := range []string{
		"type SLAService interface",
		"type WorkflowBridge interface",
		"(*dto.SLADeadlineResult, error)",
		"SLAStore() *sla.Store",
		"AdvanceBusinessWorkflowWithClient(ctx context.Context, txc *ent.Client",
	} {
		if !strings.Contains(contract, want) {
			t.Errorf("problem/contracts.go 缺少契约声明 %q", want)
		}
	}

	app := readRepoFile(t, "internal", "bootstrap", "app.go")
	if !strings.Contains(stripLineComments(app),
		"problem.NewService(problemRepo, client, ticketService, sugar, service.NewBPMNApprovalBridge(client, sugar))") {
		t.Errorf("app.go 未把真实桥接注入 problem.NewService（生命周期动作会静默不推进 BPMN 待办）")
	}

	fixture := readRepoFile(t, "handlers", "problem", "service_test.go")
	if !strings.Contains(stripLineComments(fixture), "service.NewBPMNApprovalBridge(client, logger)") {
		t.Errorf("problem 测试夹具未显式注入桥接，workflow_bridge_test.go 的 E2E 会失去被测对象")
	}

	// 值类型只有一份定义：共享层别名 + reflect.Type 身份 + JSON 形状。
	slaSvc := readRepoFile(t, "service", "ticket_sla_service.go")
	if !strings.Contains(squashSpace(stripLineComments(slaSvc)),
		squashSpace("type SLADeadlineResult = dto.SLADeadlineResult")) {
		t.Errorf("service/ticket_sla_service.go 丢失 SLADeadlineResult 别名")
	}
	if reflect.TypeOf(service.SLADeadlineResult{}) != reflect.TypeOf(dto.SLADeadlineResult{}) {
		t.Errorf("SLADeadlineResult 在共享层与 dto 之间类型身份分裂")
	}
	raw, err := json.Marshal(dto.SLADeadlineResult{SLADefinitionID: 7, BusinessHoursOnly: true})
	if err != nil {
		t.Fatalf("序列化 SLADeadlineResult 失败：%v", err)
	}
	for _, key := range []string{`"SLADefinitionID"`, `"ResponseDeadline"`, `"ResolutionDeadline"`, `"BusinessHoursOnly"`} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("SLADeadlineResult 字段漂移，缺少 %s：%s", key, string(raw))
		}
	}
}
