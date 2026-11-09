package layering

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"itsm-backend/handlers/account"
	"itsm-backend/handlers/approval"
	"itsm-backend/handlers/approval_chain"
	"itsm-backend/handlers/auth"
	"itsm-backend/handlers/automation_rule"
	"itsm-backend/handlers/problem_investigation"
	"itsm-backend/handlers/ticket_notification"
	"itsm-backend/handlers/ticket_type"
	"itsm-backend/handlers/ticket_workflow"
	"itsm-backend/handlers/vector_store"
	"itsm-backend/service"
)

// 编译期断言：生产具体类型必须满足域内窄契约。契约方法集由各域 handler/routes
// 的真实调用面决定，任何签名漂移都会在编译期暴露，而不是拖到运行时。
var (
	_ account.UserService                               = (*service.UserService)(nil)
	_ approval.ApprovalService                          = (*service.ApprovalService)(nil)
	_ approval_chain.ApprovalChainService               = (*service.ApprovalChainService)(nil)
	_ auth.EmailService                                 = (*service.EmailService)(nil)
	_ automation_rule.AutomationRuleService             = (*service.TicketAutomationRuleService)(nil)
	_ problem_investigation.ProblemInvestigationService = (*service.ProblemInvestigationService)(nil)
	_ ticket_notification.TicketNotificationService     = (*service.TicketNotificationService)(nil)
	_ ticket_type.TicketTypeService                     = (*service.TicketTypeService)(nil)
	_ ticket_workflow.TicketWorkflowService             = (*service.TicketWorkflowService)(nil)
	_ vector_store.VectorStore                          = (*service.VectorStore)(nil)
)

type domainSpec struct {
	name     string // handlers 下的域目录名
	file     string // 持有契约字段的生产文件
	iface    string // 域内窄接口名
	concrete string // 共享层具体类型名
}

var contractDomains = []domainSpec{
	{"account", "handler.go", "UserService", "UserService"},
	{"approval", "handler.go", "ApprovalService", "ApprovalService"},
	{"approval_chain", "handler.go", "ApprovalChainService", "ApprovalChainService"},
	{"auth", "service.go", "EmailService", "EmailService"},
	{"automation_rule", "handler.go", "AutomationRuleService", "TicketAutomationRuleService"},
	{"problem_investigation", "handler.go", "ProblemInvestigationService", "ProblemInvestigationService"},
	{"ticket_notification", "handler.go", "TicketNotificationService", "TicketNotificationService"},
	{"ticket_type", "handler.go", "TicketTypeService", "TicketTypeService"},
	{"ticket_workflow", "handler.go", "TicketWorkflowService", "TicketWorkflowService"},
	{"vector_store", "handler.go", "VectorStore", "VectorStore"},
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
				if strings.Contains(stripLineComments(string(body)), "*service."+d.concrete) {
					t.Errorf("handlers/%s/%s 字段或构造参数回退到共享层具体类型 %s", d.name, fn, d.concrete)
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
			if !strings.Contains(code, d.iface) {
				t.Errorf("handlers/%s/%s 未使用域内接口 %s", d.name, d.file, d.iface)
			}
			if strings.Contains(code, "*service."+d.concrete) {
				t.Errorf("handlers/%s/%s 回退到共享层具体类型（旧实现）", d.name, d.file)
			}
		})
	}
}
