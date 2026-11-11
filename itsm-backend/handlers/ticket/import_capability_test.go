package ticket

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"itsm-backend/common"
)

// 产品完整性计划 批次 1「消灭假成功」：占位能力必须二选一——接真实现，
// 或显式 unready 且不注册路由。工单导入属于后者，回归锁同时锁住三件事：
// ① 返回的是 5003 分类错误而不是会被兜底成 500 的裸 fmt.Errorf；
// ② handler 用 RespondError 透传服务端文案，不用 FailWithErr 覆盖；
// ③ 路由未注册；将来一旦注册，测试立刻变红，强制先实现真实导入。

func TestImportTickets_IsExplicitlyUnavailable(t *testing.T) {
	svc := &Service{}
	err := svc.ImportTickets(context.Background(), 1, []byte("{}"), "json")
	if err == nil {
		t.Fatal("工单导入未实现时必须返回错误，不得静默成功")
	}
	var bizErr *common.BusinessError
	if !errors.As(err, &bizErr) {
		t.Fatalf("必须是分类错误 *common.BusinessError，实际 %T: %v", err, err)
	}
	if bizErr.Code != common.ServiceUnavailableCode {
		t.Errorf("业务码应为 5003 能力不可用，实际 %d", bizErr.Code)
	}
	if !strings.Contains(bizErr.Message, "未实现") {
		t.Errorf("文案必须明确「未实现」，实际 %q", bizErr.Message)
	}
	if strings.Contains(err.Error(), "import not implemented in handlers layer") {
		t.Error("旧占位文案仍存在：会被 RespondError 兜底成 500 而非 5003")
	}
}

// functionBody 截取某个顶层函数到下一个顶层 func 声明之间的源码，
// 避免对全文件做字符串包含时命中别的端点（假绿通道）。
func functionBody(src, signature string) (string, bool) {
	start := strings.Index(src, signature)
	if start < 0 {
		return "", false
	}
	rest := src[start:]
	if next := strings.Index(rest[len(signature):], "\nfunc "); next >= 0 {
		return rest[:len(signature)+next], true
	}
	return rest, true
}

func TestImportTickets_SourceGuard_UnreadyWithoutRoute(t *testing.T) {
	svcSrc, err := os.ReadFile("service.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(svcSrc), "func (s *Service) ImportTickets") {
		t.Fatal("扫描器没找到 ImportTickets 定义，测试已失去意义")
	}
	if strings.Contains(string(svcSrc), "not implemented in handlers layer") {
		t.Error("service.go 仍含普通 fmt.Errorf 占位实现")
	}

	handlerSrc, err := os.ReadFile("handler.go")
	if err != nil {
		t.Fatal(err)
	}
	body, ok := functionBody(string(handlerSrc), "func (h *Handler) ImportTickets(")
	if !ok {
		t.Fatal("扫描器没找到 handler ImportTickets 定义，测试已失去意义")
	}
	if !strings.Contains(body, "common.RespondError(c, err") {
		t.Error("handler 必须用 RespondError 透传 5003 文案，FailWithErr 会覆盖成「操作失败」")
	}
	if strings.Contains(body, "common.FailWithErr") {
		t.Error("handler 仍在用 FailWithErr 覆盖分类错误文案")
	}

	routerSrc, err := os.ReadFile("../../router/ticket_routes.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(routerSrc), "func SetupTicketRoutes") {
		t.Fatal("扫描器没找到路由注册函数，零命中不能算绿")
	}
	if strings.Contains(string(routerSrc), `"/import"`) {
		t.Error("工单导入路由已注册但实现仍是 unready：必须先实现真实导入并同步 authz 预检生成物")
	}
}
