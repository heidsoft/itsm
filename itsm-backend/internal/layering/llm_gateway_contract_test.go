package layering

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/internal/services"
)

// LLMGateway 契约化专项回归锁（2026-11-10 批次 A）。
//
// 这一批的难点不是「换个窄接口」：流式问答接口的方法签名里携带网关具体
// 指针，Go 接口要求签名逐字一致，所以必须把网关能力收敛成 dto 端口，
// 再让共享层实现侧的方法参数也泛化到同一个端口。三个容易假绿的点：
// ① 值类型另起一份同名 struct 时，域内不引用共享层、编译也可能过，
//    只有类型身份/字段形状断言能拦；② 端口接口悄悄少一个方法时，
//    编译与 L.3 计数都不变，但契约语义已丢；③ 把 RAG 方法参数改回具体
//    指针会让域内 ragStreamer 断言失效，必须锁住参数类型本身。

const llmSharedImport = `"itsm-backend/internal/services"`

var _ dto.LLMGatewayPort = (*service.LLMGateway)(nil)

func readLLMRepoFile(t *testing.T, parts ...string) string {
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

// TestEmailIntakeDomainHasZeroSharedImports：整域生产文件零共享层 import，
// 且基线必须删行（留 email_intake=0 等于保留可悄悄放宽的槽位）。
func TestEmailIntakeDomainHasZeroSharedImports(t *testing.T) {
	dir := filepath.Join("..", "..", "handlers", "email_intake")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取 handlers/email_intake 失败：%v", err)
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
			t.Fatalf("读取 handlers/email_intake/%s 失败：%v", fn, err)
		}
		if len(body) < 40 {
			t.Fatalf("扫描器零命中保护：handlers/email_intake/%s 仅 %d 字节", fn, len(body))
		}
		if strings.Contains(string(body), llmSharedImport) {
			t.Errorf("handlers/email_intake/%s 重新引用共享层包，整域归零被破坏", fn)
		}
	}
	// contracts.go / dto.go / extractor.go / handler.go / orchestrator.go / outbound_handler.go / service.go
	if prod < 7 {
		t.Fatalf("扫描器零命中保护：handlers/email_intake 只剩 %d 个生产 .go 文件（期望至少 7）", prod)
	}
	base := readLLMRepoFile(t, "..", "scripts", "docs-gate", "handler-service-imports-baseline.txt")
	if regexp.MustCompile(`(?m)^email_intake=`).MatchString(base) {
		t.Errorf("L.3 基线仍保留 email_intake= 行（整域归零必须删除，不能留 =0 槽位）")
	}
}

// TestLLMValueTypesShareOneDefinition：别名身份、sentinel 身份、逐处源码形态。
func TestLLMValueTypesShareOneDefinition(t *testing.T) {
	pairs := []struct {
		name   string
		shared reflect.Type
		bottom reflect.Type
	}{
		{"LLMTool", reflect.TypeOf(service.LLMTool{}), reflect.TypeOf(dto.LLMTool{})},
		{"LLMToolCall", reflect.TypeOf(service.LLMToolCall{}), reflect.TypeOf(dto.LLMToolCall{})},
		{"LLMMessage", reflect.TypeOf(service.LLMMessage{}), reflect.TypeOf(dto.LLMMessage{})},
		{"RateLimitError", reflect.TypeOf(service.RateLimitError{}), reflect.TypeOf(dto.RateLimitError{})},
		{"EmailIncidentCommand", reflect.TypeOf(service.EmailIncidentCommand{}), reflect.TypeOf(dto.EmailIncidentCommand{})},
	}
	for _, p := range pairs {
		if p.shared != p.bottom {
			t.Errorf("service.%s 与 dto.%s 不再是同一类型（别名被换成独立副本）", p.name, p.name)
		}
	}

	sentinels := []struct {
		name, file, decl string
		shared, bottom   error
	}{
		{"ErrRateLimited", "llm_gateway.go", "var ErrRateLimited = dto.ErrRateLimited", service.ErrRateLimited, dto.ErrRateLimited},
		{"ErrQuotaExceeded", "llm_gateway.go", "var ErrQuotaExceeded = dto.ErrQuotaExceeded", service.ErrQuotaExceeded, dto.ErrQuotaExceeded},
	}
	gatewaySrc := readLLMRepoFile(t, "internal", "services", "llm_gateway.go")
	for _, s := range sentinels {
		if !strings.Contains(squashSpace(gatewaySrc), squashSpace(s.decl)) {
			t.Errorf("internal/services/%s 缺少 %s 的同实例别名声明", s.file, s.name)
		}
		if !errors.Is(fmt.Errorf("wrapped: %w", s.bottom), s.shared) {
			t.Errorf("service.%s 必须与 dto sentinel 是同一实例（errors.Is 身份漂移）", s.name)
		}
	}

	// 逐处比对，不取并集：任何一条别名被改回独立定义都要变红。
	aliasDecls := []struct{ file, decl string }{
		{"llm_gateway.go", "type LLMTool = dto.LLMTool"},
		{"llm_gateway.go", "type LLMToolCall = dto.LLMToolCall"},
		{"llm_gateway.go", "type LLMMessage = dto.LLMMessage"},
		{"llm_gateway.go", "type RateLimitError = dto.RateLimitError"},
		{"llm_gateway.go", "type LLMGatewayPort = dto.LLMGatewayPort"},
		{"incident_service.go", "type EmailIncidentCommand = dto.EmailIncidentCommand"},
	}
	byFile := map[string]string{
		"llm_gateway.go":      gatewaySrc,
		"incident_service.go": readLLMRepoFile(t, "internal", "services", "incident_service.go"),
	}
	for _, a := range aliasDecls {
		src := squashSpace(byFile[a.file])
		if !strings.Contains(src, squashSpace(a.decl)) {
			t.Errorf("internal/services/%s 缺少别名声明 %q", a.file, a.decl)
		}
	}

	// 字段形状锁：值类型下沉后字段集合不能悄悄增删（别名身份相等时该断言
	// 天然成立，反向验证里把 dto 侧字段改掉时它会独立变红）。
	if got := fieldNames(reflect.TypeOf(dto.LLMMessage{})); got != "Role,Content,Tools,ToolCalls,ToolCallID" {
		t.Errorf("dto.LLMMessage 字段形状漂移：%s", got)
	}
	if got := fieldNames(reflect.TypeOf(dto.LLMTool{})); got != "Name,Description,Parameters" {
		t.Errorf("dto.LLMTool 字段形状漂移：%s", got)
	}
	if got := fieldNames(reflect.TypeOf(dto.LLMToolCall{})); got != "ID,Name,Arguments" {
		t.Errorf("dto.LLMToolCall 字段形状漂移：%s", got)
	}
	// JSON tag 与序列化形状：LLM 工具声明走 function calling 协议，tag 漂移
	// 不会让 Go 编译失败，但会让模型收到的工具清单字段错名。
	tagOf := func(v reflect.Type, field string) reflect.StructTag { f, _ := v.FieldByName(field); return f.Tag }
	if got := string(tagOf(reflect.TypeOf(dto.LLMTool{}), "Parameters")); got != `json:"parameters,omitempty"` {
		t.Errorf("dto.LLMTool.Parameters tag 漂移：%s", got)
	}
	if got := string(tagOf(reflect.TypeOf(dto.LLMToolCall{}), "Arguments")); got != `json:"arguments"` {
		t.Errorf("dto.LLMToolCall.Arguments tag 漂移：%s", got)
	}
	encoded, err := json.Marshal(dto.LLMTool{Name: "list_cis", Description: "list"})
	if err != nil || string(encoded) != `{"name":"list_cis","description":"list"}` {
		t.Errorf("dto.LLMTool 序列化形状漂移：%s err=%v", encoded, err)
	}
	const cmdFields = "ConversationID,SupportContractID,ReporterUserID,AssignmentGroupID,AssigneeID,Title,Description,Impact,Urgency,Category,Metadata,OverrideContract,OverrideReason"
	if got := fieldNames(reflect.TypeOf(dto.EmailIncidentCommand{})); got != cmdFields {
		t.Errorf("dto.EmailIncidentCommand 字段形状漂移：%s", got)
	}
}

func fieldNames(t reflect.Type) string {
	out := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		out = append(out, t.Field(i).Name)
	}
	return strings.Join(out, ",")
}

// TestLLMGatewayPortSignatureIsStable：端口方法集 + RAG 方法参数类型 + 域侧签名。
func TestLLMGatewayPortSignatureIsStable(t *testing.T) {
	port := reflect.TypeOf((*dto.LLMGatewayPort)(nil)).Elem()
	want := []string{"Chat", "ChatStream", "ChatStreamWithTools", "EffectiveModel", "ProviderName", "SupportsToolCalling"}
	got := make([]string, 0, port.NumMethod())
	for i := 0; i < port.NumMethod(); i++ {
		got = append(got, port.Method(i).Name)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("dto.LLMGatewayPort 方法集漂移：got %v want %v", got, want)
	}

	m, ok := reflect.TypeOf((*service.RAGService)(nil)).MethodByName("AskWithLLMStreamWithToolsMulti")
	if !ok {
		t.Fatal("service.RAGService 缺少 AskWithLLMStreamWithToolsMulti（扫描器零命中保护）")
	}
	// In(0) 是 receiver；gateway 是第 5 个入参，tools 是第 7 个。
	if m.Type.NumIn() < 8 {
		t.Fatalf("方法签名参数数异常：%d", m.Type.NumIn())
	}
	if m.Type.In(5) != port {
		t.Errorf("RAG 流式方法的 gateway 参数回退为 %s，域内 ragStreamer 将不再匹配", m.Type.In(5))
	}
	if m.Type.In(7) != reflect.TypeOf([]dto.LLMTool{}) {
		t.Errorf("RAG 流式方法的 tools 参数不是 []dto.LLMTool：%s", m.Type.In(7))
	}

	aiSvc := readLLMRepoFile(t, "handlers", "ai", "service.go")
	for _, decl := range []string{
		"llmGateway dto.LLMGatewayPort",
		"gateway dto.LLMGatewayPort",
		"tools []dto.LLMTool",
		"func (s *Service) SetLLMGateway(gateway dto.LLMGatewayPort)",
	} {
		if !strings.Contains(squashSpace(aiSvc), squashSpace(decl)) {
			t.Errorf("handlers/ai/service.go 缺少端口契约声明 %q", decl)
		}
	}
	if strings.Contains(stripLineComments(aiSvc), "*service.LLMGateway") {
		t.Errorf("handlers/ai/service.go 回退到网关具体指针")
	}

	ragSrc := readLLMRepoFile(t, "internal", "services", "rag_service.go")
	if n := len(regexp.MustCompile(`\bgateway LLMGatewayPort\b`).FindAllString(ragSrc, -1)); n != 5 {
		t.Errorf("internal/services/rag_service.go 的网关端口参数应为 5 处，实测 %d（改回具体指针会让域内签名失配）", n)
	}
}

// TestEmailIncidentCreatorWiring：窄契约签名、域内字段与组合根装配。
func TestEmailIncidentCreatorWiring(t *testing.T) {
	contract := readLLMRepoFile(t, "handlers", "email_intake", "contracts.go")
	if !strings.Contains(contract, "type EmailIncidentCreator interface") {
		t.Errorf("contracts.go 未声明 EmailIncidentCreator")
	}
	if !strings.Contains(squashSpace(contract),
		squashSpace("CreateFromEmail(ctx context.Context, tenantID int, command dto.EmailIncidentCommand) (*dto.IncidentResponse, error)")) {
		t.Errorf("窄契约 CreateFromEmail 签名漂移")
	}
	if strings.Contains(contract, llmSharedImport) {
		t.Errorf("contracts.go 不得引用共享层包")
	}

	orch := readLLMRepoFile(t, "handlers", "email_intake", "orchestrator.go")
	if n := strings.Count(orch, "EmailIncidentCreator"); n != 2 {
		t.Errorf("orchestrator.go 的窄契约字段/构造参数出现次数应为 2，实测 %d", n)
	}
	if n := strings.Count(orch, "dto.EmailIncidentCommand{"); n != 2 {
		t.Errorf("orchestrator.go 构造 dto 命令值类型应为 2 处，实测 %d", n)
	}

	app := readLLMRepoFile(t, "internal", "bootstrap", "app.go")
	if !regexp.MustCompile(`NewEmailIntakeOrchestrator\(`).MatchString(app) {
		t.Errorf("组合根缺少 NewEmailIntakeOrchestrator 装配接线")
	}
	if !regexp.MustCompile(`SetLLMGateway\(`).MatchString(app) {
		t.Errorf("组合根缺少 SetLLMGateway 装配接线")
	}
}
