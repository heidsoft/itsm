package layering

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/handlers/escalation_matrix"
)

// escalation 域整组归位专项回归锁（2026-11-11）。
//
// 本批把 service 的三个生产实现（升级矩阵服务、system_configs store、SLA 升级
// 扫描器）连同四份专属测试 git mv 进 handlers/escalation_matrix；域内窄接口改名
// 为 MatrixService 以避开与具体实现 EscalationMatrixService 的同包重名，扫描器
// 的通知依赖改为 NotificationSender 单方法端口，组合根仍注入共享层具体实现。
// 最大假绿风险是「旧文件名复活但 C.8 未重跑」或「字段悄悄改回共享层具体类型，
// 而 handler 侧 L.3 计数仍全绿」——编译能过、指标能过，但域归属被破坏。
func readEscalationRepoFile(t *testing.T, parts ...string) string {
	t.Helper()
	path := filepath.Join(append([]string{"..", ".."}, parts...)...)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 %s 失败：%v", filepath.Join(parts...), err)
	}
	if len(body) < 40 {
		t.Fatalf("扫描器零命中保护：%s 内容异常短（%d 字节）", filepath.Join(parts...), len(body))
	}
	return string(body)
}

func TestEscalationDomainImplementationIsDomainOwned(t *testing.T) {
	for _, old := range []string{"escalation_matrix.go", "escalation_matrix_store.go", "escalation_service.go"} {
		path := filepath.Join("..", "..", "internal", "services", old)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("internal/services/%s 复活：实现必须只存在于 handlers/escalation_matrix", old)
		}
	}

	for _, fn := range []string{
		"contracts.go", "handler.go", "escalation_matrix.go",
		"escalation_matrix_store.go", "escalation_service.go",
	} {
		src := readEscalationRepoFile(t, "handlers", "escalation_matrix", fn)
		if strings.Contains(src, `"itsm-backend/internal/services"`) {
			t.Errorf("handlers/escalation_matrix/%s 重新引用共享层包，域归零被破坏", fn)
		}
	}

	contracts := squashSpace(readEscalationRepoFile(t, "handlers", "escalation_matrix", "contracts.go"))
	for _, decl := range []string{
		"type MatrixService interface",
		"type NotificationSender interface",
		"SendNotification(ctx context.Context, ticketID int, req *dto.SendTicketNotificationRequest, tenantID int) error",
	} {
		if !strings.Contains(contracts, squashSpace(decl)) {
			t.Errorf("contracts.go 缺少域内契约声明 %q", decl)
		}
	}

	serviceRaw := readEscalationRepoFile(t, "handlers", "escalation_matrix", "escalation_service.go")
	if !strings.Contains(serviceRaw, "\tnotificationSvc NotificationSender\n") {
		t.Error("escalation_service.go 的 notificationSvc 字段必须精确声明为 NotificationSender（禁止改回共享层具体类型或独立匿名接口）")
	}
	service := squashSpace(serviceRaw)
	for _, decl := range []string{
		"func (e *EscalationService) SetNotificationService(notificationSvc NotificationSender)",
		"func NewEscalationService(client *ent.Client, logger *zap.SugaredLogger) *EscalationService",
		"var ErrEscalationLevelClaimed = errors.New(\"escalation level already advanced by concurrent scan\")",
	} {
		if !strings.Contains(service, squashSpace(decl)) {
			t.Errorf("escalation_service.go 缺少归域后的声明 %q", decl)
		}
	}

	matrix := squashSpace(readEscalationRepoFile(t, "handlers", "escalation_matrix", "escalation_matrix.go"))
	for _, decl := range []string{
		"type EscalationLevel = dto.EscalationLevel",
		"type EscalationMatrix = dto.EscalationMatrix",
		"func NewEscalationMatrixServiceWithStore(logger *zap.SugaredLogger, store EscalationMatrixStore) *EscalationMatrixService",
	} {
		if !strings.Contains(matrix, squashSpace(decl)) {
			t.Errorf("escalation_matrix.go 缺少归域后的声明 %q", decl)
		}
	}

	store := squashSpace(readEscalationRepoFile(t, "handlers", "escalation_matrix", "escalation_matrix_store.go"))
	if !strings.Contains(store, squashSpace("func NewSystemConfigEscalationMatrixStore(client *ent.Client, logger *zap.SugaredLogger) *SystemConfigEscalationMatrixStore")) {
		t.Error("escalation_matrix_store.go 缺少 system_configs store 构造器")
	}

	baseline := readEscalationRepoFile(t, "..", "scripts", "docs-gate", "service-prod-baseline.txt")
	baselineLines := strings.Split(baseline, "\n")
	for _, old := range []string{"escalation_matrix.go", "escalation_matrix_store.go", "escalation_service.go"} {
		for _, line := range baselineLines {
			if line == old {
				t.Errorf("L.1 基线仍登记已迁出的 service/%s，必须删除该行", old)
			}
		}
	}

	drift := readEscalationRepoFile(t, "..", "scripts", "docs-gate", "product-surface-baseline.txt")
	// 棘轮语义：本批把上界从 350 压到 343；后续批次继续下降合规，
	// 但任何回退到 350（含）以上的值都必须红。硬编码「精确 343」会在
	// 下一批下调时误报（2026-11-11 sla 预测归位批实测踩到）。
	serviceFiles := -1
	for _, line := range strings.Split(drift, "\n") {
		if strings.HasPrefix(line, "service_go_files=") {
			fields := strings.Fields(strings.TrimPrefix(line, "service_go_files="))
			if len(fields) == 0 {
				t.Fatal("C.6.4 service_go_files 行无数值")
			}
			v, err := strconv.Atoi(fields[0])
			if err != nil {
				t.Fatalf("C.6.4 service_go_files 数值非法：%v", err)
			}
			serviceFiles = v
			break
		}
	}
	if serviceFiles < 0 || serviceFiles > 343 {
		t.Errorf("C.6.4 service_go_files 上界必须 ≤343（本批 350 → 343，后续只许继续下降），当前 %d", serviceFiles)
	}
}

func TestEscalationValueTypesRemainDTOAliases(t *testing.T) {
	if reflect.TypeOf(escalation_matrix.EscalationMatrix{}) != reflect.TypeOf(dto.EscalationMatrix{}) {
		t.Error("escalation_matrix.EscalationMatrix 必须是 dto.EscalationMatrix 的类型别名")
	}
	if reflect.TypeOf(escalation_matrix.EscalationLevel{}) != reflect.TypeOf(dto.EscalationLevel{}) {
		t.Error("escalation_matrix.EscalationLevel 必须是 dto.EscalationLevel 的类型别名")
	}
}
