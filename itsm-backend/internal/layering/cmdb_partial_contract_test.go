package layering

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/service"
)

// cmdb 全域归零轨迹：专项 A 归零 5 个生产文件，专项 B 用 8 个域内窄契约
// 收掉 production_service.go 的 7 个共享层具体类型与 ontology.go 的
// ToolRegistry，并下沉 LifecycleChangeParams / ToolDefinition /
// ErrInvalidRelationshipType 与 operator context 实现。至此整个域的生产
// 文件都不 import 共享层，L.3 基线里的 cmdb 行必须删除（留 cmdb=0 等于
// 保留一个能悄悄放宽回 N 的槽位）。本文件继续按「实测计数 + 基线 +
// 单文件形态」精确锁定，不进 contractDomains（该守卫假设单一契约文件）。

const cmdbSharedImport = `"itsm-backend/service"`

const cmdbPartialZeroExpectedImports = 0

func readCMDBFile(t *testing.T, rel string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "handlers", "cmdb", rel))
	if err != nil {
		t.Fatalf("读取 handlers/cmdb/%s 失败：%v", rel, err)
	}
	if len(body) < 100 {
		t.Fatalf("扫描器零命中保护：handlers/cmdb/%s 仅 %d 字节", rel, len(body))
	}
	return string(body)
}

func readServiceFile(t *testing.T, rel string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "service", rel))
	if err != nil {
		t.Fatalf("读取 service/%s 失败：%v", rel, err)
	}
	if len(body) < 100 {
		t.Fatalf("扫描器零命中保护：service/%s 仅 %d 字节", rel, len(body))
	}
	return string(body)
}

// TestCMDBPartialZeroMeasuredImports 把「实测 2 个文件」与「基线 cmdb=2」
// 双锁：只改基线不改代码、或只改代码不改基线，都必须在这里变红。
func TestCMDBPartialZeroMeasuredImports(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join("..", "..", "handlers", "cmdb"))
	if err != nil {
		t.Fatalf("读取 handlers/cmdb 目录失败：%v", err)
	}
	prod, count := 0, 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		prod++
		body, err := os.ReadFile(filepath.Join("..", "..", "handlers", "cmdb", name))
		if err != nil {
			t.Fatalf("读取 handlers/cmdb/%s 失败：%v", name, err)
		}
		if len(body) < 100 {
			t.Fatalf("扫描器零命中保护：handlers/cmdb/%s 仅 %d 字节", name, len(body))
		}
		if strings.Contains(string(body), cmdbSharedImport) {
			count++
		}
	}
	if prod < 18 {
		t.Fatalf("扫描器零命中保护：handlers/cmdb 只剩 %d 个生产 .go 文件", prod)
	}
	if count != cmdbPartialZeroExpectedImports {
		t.Errorf("handlers/cmdb 共享层依赖实测 %d，期望 %d（专项 A + B 已全域归零）", count, cmdbPartialZeroExpectedImports)
	}

	base, err := os.ReadFile(filepath.Join("..", "..", "..", "scripts", "docs-gate", "handler-service-imports-baseline.txt"))
	if err != nil {
		t.Fatalf("读取 L.3 基线失败：%v", err)
	}
	if len(base) < 100 {
		t.Fatalf("扫描器零命中保护：L.3 基线仅 %d 字节", len(base))
	}
	lines := []string{}
	for _, line := range strings.Split(string(base), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "cmdb=") {
			lines = append(lines, line)
		}
	}
	if len(lines) != 0 {
		t.Errorf("L.3 基线必须删除 cmdb 行（整域归零后不留 cmdb=0 槽位），实际 %v", lines)
	}
}

// TestCMDBZeroedFilesUseDTOContracts 逐文件锁住「归零是换成 dto 契约」，
// 而不是删掉文件、把 import 换路径、或把共享层符号抄进域内。
func TestCMDBZeroedFilesUseDTOContracts(t *testing.T) {
	cases := []struct {
		file    string
		needles []string
	}{
		{"discovery_ci_projection.go", []string{"dto.ErrCIAttributeValidation"}},
		{"discovery_manual_completion.go", []string{
			"[]dto.CIAttributeRequirement, error)", "dto.CIAttributeRequirement\n",
			"map[int][]dto.CIAttributeRequirement", "dto.ErrCIAttributeValidation",
		}},
		{"discovery_scheduler.go", []string{
			"dto.ResolveLocation(\"\")", "dto.ParseTimerExpression(expr)", "dto.NextFireAt(expr, exprType, loc, anchor)",
		}},
		{"handler.go", []string{"dto.ResolveLocation(\"\")"}},
		{"provisioning_ci_projection.go", []string{
			"dto.ProvisioningCIProjector = (*ProvisioningCIProjector)(nil)",
			"req dto.ProvisioningCIProjectionRequest) dto.ProvisioningCIProjectionResult",
			"dto.ProvisioningCIProjectionUnavailable", "dto.ProvisioningCIProjectionCreated",
			"dto.ProvisioningCIProjectionFailed", "dto.ErrCIAttributeValidation",
		}},
		{"production_service.go", []string{
			"ciTypeService                CITypeCatalog",
			"toolRegistry                 ToolRegistryCatalog",
			"dto.WithOperator(ctx.Request.Context()",
			"params := dto.LifecycleChangeParams{",
			"dto.ErrInvalidRelationshipType",
		}},
		{"ontology.go", []string{
			"SetToolRegistry(tr ToolRegistryCatalog)",
		}},
	}
	for _, c := range cases {
		t.Run(c.file, func(t *testing.T) {
			body := readCMDBFile(t, c.file)
			if strings.Contains(body, cmdbSharedImport) {
				t.Errorf("%s 重新引用共享层包，L.3 归零被破坏", c.file)
			}
			code := stripLineComments(body)
			if regexp.MustCompile(`(?m)(^|[^.\w])service\.`).MatchString(code) {
				t.Errorf("%s 仍以包限定符引用共享层（旧实现）", c.file)
			}
			for _, needle := range c.needles {
				if !strings.Contains(squashSpace(code), squashSpace(needle)) {
					t.Errorf("%s 丢失 dto 契约 %q", c.file, needle)
				}
			}
		})
	}

	// 组合根零改动：本批不得在 app.go / router.go 为 timer 或投影新增装配。
	for _, rel := range []string{"internal/bootstrap/app.go", "router/router.go"} {
		body, err := os.ReadFile(filepath.Join("..", "..", rel))
		if err != nil {
			t.Fatalf("读取 %s 失败：%v", rel, err)
		}
		if strings.Contains(string(body), "dto.ParseTimerExpression") || strings.Contains(string(body), "dto.ResolveLocation") {
			t.Errorf("%s 不应为 timer 纯函数新增装配（组合根零改动前提）", rel)
		}
	}
}

// TestCMDBProjectionDTOValuesAndShape 锁住常量字面值、JSON tag 与错误身份。
// 把 "created" 改成 "create" 或把 displayName 改成 DisplayName 都能编译、
// L.3 计数也不变，但发现/交付回写的状态机与人工补全读面会静默错乱。
func TestCMDBProjectionDTOValuesAndShape(t *testing.T) {
	want := map[string]string{
		"Created": "created", "Updated": "updated", "Deferred": "deferred",
		"Skipped": "skipped", "Failed": "failed", "Unavailable": "unavailable",
	}
	got := map[string]string{
		"Created": dto.ProvisioningCIProjectionCreated, "Updated": dto.ProvisioningCIProjectionUpdated,
		"Deferred": dto.ProvisioningCIProjectionDeferred, "Skipped": dto.ProvisioningCIProjectionSkipped,
		"Failed": dto.ProvisioningCIProjectionFailed, "Unavailable": dto.ProvisioningCIProjectionUnavailable,
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("dto.ProvisioningCIProjection%s = %q，期望 %q（service 侧常量必须同时保持同值）", k, got[k], v)
		}
	}
	if service.ProvisioningCIProjectionCreated != dto.ProvisioningCIProjectionCreated ||
		service.ProvisioningCIProjectionDeferred != dto.ProvisioningCIProjectionDeferred {
		t.Error("service 侧投影状态常量与 dto 不同值（别名退化为独立定义）")
	}

	payload, err := json.Marshal(dto.CIAttributeRequirement{
		Name: "os", DisplayName: "操作系统", DataType: "string", EnumValues: []string{"linux"},
		DefaultValue: "linux", Placeholder: "请输入", HelpText: "帮助",
	})
	if err != nil {
		t.Fatalf("序列化 CIAttributeRequirement 失败：%v", err)
	}
	shape := string(payload)
	for _, key := range []string{
		`"name":"os"`, `"displayName":"操作系统"`, `"dataType":"string"`,
		`"enumValues":["linux"]`, `"defaultValue":"linux"`, `"placeholder":"请输入"`, `"helpText":"帮助"`,
	} {
		if !strings.Contains(shape, key) {
			t.Errorf("CIAttributeRequirement JSON 形状丢失 %s：%s", key, shape)
		}
	}
	for _, key := range []string{`"Name"`, `"DisplayName"`, `"slATotalSamples"`} {
		if strings.Contains(shape, key) {
			t.Errorf("CIAttributeRequirement JSON 形状意外出现 %s：%s", key, shape)
		}
	}

	// errors.Is 身份：共享层重新 errors.New 一个同文案的 error 会让
	// 「必填项不齐 → deferred」退化成 500，编译与 L.3 计数都看不出来。
	if !errors.Is(service.ErrCIAttributeValidation, dto.ErrCIAttributeValidation) {
		t.Error("service.ErrCIAttributeValidation 必须就是 dto.ErrCIAttributeValidation（errors.Is 身份）")
	}
	if service.ErrCIAttributeValidation.Error() != "ci attribute validation" {
		t.Errorf("sentinel 文案漂移：%q", service.ErrCIAttributeValidation.Error())
	}

	// 类型身份：别名退化成新 struct 会撕裂接口签名。
	pairs := []struct {
		name           string
		shared, bottom reflect.Type
	}{
		{"CIAttributeRequirement", reflect.TypeOf(service.CIAttributeRequirement{}), reflect.TypeOf(dto.CIAttributeRequirement{})},
		{"ProvisioningCIProjectionRequest", reflect.TypeOf(service.ProvisioningCIProjectionRequest{}), reflect.TypeOf(dto.ProvisioningCIProjectionRequest{})},
		{"ProvisioningCIProjectionResult", reflect.TypeOf(service.ProvisioningCIProjectionResult{}), reflect.TypeOf(dto.ProvisioningCIProjectionResult{})},
		{"ExpressionType", reflect.TypeOf(service.ExprTypeDuration), reflect.TypeOf(dto.ExprTypeDuration)},
	}
	for _, p := range pairs {
		if p.shared != p.bottom {
			t.Errorf("%s：共享层与 dto 类型身份不一致（%v vs %v）", p.name, p.shared, p.bottom)
		}
	}
}

// TestCMDBTimerPureFunctionsAreSingleSource 同时锁「service 是薄委托」与
// 「行为与既有口径一致」。只锁委托不够：若 dto 实现被改坏，service 仍会
// 忠实委托一个错误结果；只锁行为也不够：service 里再抄一份实现也能过。
func TestCMDBTimerPureFunctionsAreSingleSource(t *testing.T) {
	cron := readServiceFile(t, "timer_cron.go")
	for _, needle := range []string{
		"const DefaultTimerTimezone = dto.DefaultTimerTimezone",
		"return dto.ParseTimerExpression(expr)",
		"return dto.ResolveLocation(tz)",
		"return dto.NextFireAt(expr, exprType, loc, after)",
		"dto.SplitCycleExpression(strings.TrimSpace(expr))",
	} {
		if !strings.Contains(squashSpace(stripLineComments(cron)), squashSpace(needle)) {
			t.Errorf("service/timer_cron.go 丢失薄委托 %q（禁止复制第二份实现）", needle)
		}
	}
	if strings.Contains(stripLineComments(cron), "cron.NewParser") {
		t.Error("service/timer_cron.go 重新持有 cron 解析器（单一实现被破坏）")
	}
	extractor := readServiceFile(t, "timer_event_extractor.go")
	for _, name := range []string{
		"func parseISO8601Duration(", "func splitCycleExpression(",
		"func hasISO8601Component(", "func findChar(", "func extractNumber(",
	} {
		if strings.Contains(extractor, name) {
			t.Errorf("service/timer_event_extractor.go 仍持有私有 helper %s（应整体下沉 dto）", name)
		}
	}
	store := readServiceFile(t, "timer_store.go")
	if !strings.Contains(squashSpace(stripLineComments(store)), squashSpace("type ExpressionType = dto.ExpressionType")) {
		t.Error("service/timer_store.go 的 ExpressionType 别名丢失")
	}

	// 分类口径：ISO duration / RFC3339 / cycle / cron / 非法输入。
	classify := []struct {
		expr string
		want dto.ExpressionType
		bad  bool
	}{
		{"PT30M", dto.ExprTypeDuration, false},
		{"2026-12-31T23:59:59Z", dto.ExprTypeDate, false},
		{"R5/PT10M", dto.ExprTypeCycle, false},
		{"0 9 * * 1-5", dto.ExprTypeCron, false},
		{"", "", true},
		{"not-a-schedule", "", true},
	}
	for _, c := range classify {
		gotService, errS := service.ParseTimerExpression(c.expr)
		gotDTO, errD := dto.ParseTimerExpression(c.expr)
		if (errS != nil) != c.bad || (errD != nil) != c.bad {
			t.Errorf("ParseTimerExpression(%q) 错误口径漂移：service=%v dto=%v", c.expr, errS, errD)
			continue
		}
		if gotService != c.want || gotDTO != c.want {
			t.Errorf("ParseTimerExpression(%q) = (%q, %q)，期望 %q", c.expr, gotService, gotDTO, c.want)
		}
	}

	loc := dto.ResolveLocation("Asia/Shanghai")
	if loc == nil || loc.String() != "Asia/Shanghai" {
		t.Fatalf("ResolveLocation(Asia/Shanghai) = %v", loc)
	}
	if service.ResolveLocation("").String() != dto.ResolveLocation("").String() {
		t.Error("service 与 dto 的默认时区口径不一致")
	}
	if service.ResolveLocation("Not/AZone").String() != "Asia/Shanghai" {
		t.Error("非法时区未回退到 Asia/Shanghai")
	}

	anchor := time.Date(2026, 11, 10, 1, 2, 3, 0, time.UTC)
	wantFire := anchor.Add(30 * time.Minute).UTC()
	fireS, errS := service.NextFireAt("PT30M", service.ExprTypeDuration, loc, anchor)
	fireD, errD := dto.NextFireAt("PT30M", dto.ExprTypeDuration, loc, anchor)
	if errS != nil || errD != nil || !fireS.Equal(wantFire) || !fireD.Equal(wantFire) {
		t.Errorf("NextFireAt(PT30M) = (%v, %v) / (%v, %v)，期望 %v", fireS, errS, fireD, errD, wantFire)
	}
	if _, err := dto.NextFireAt("PTxxX", dto.ExprTypeDuration, loc, anchor); err == nil {
		t.Error("无法识别的 duration 必须显式报错，不得解析成 0 时长立即触发")
	}
	if got := dto.SplitCycleExpression("R5/PT10M"); len(got) != 2 || got[0] != "R5" || got[1] != "PT10M" {
		t.Errorf("SplitCycleExpression(R5/PT10M) = %v", got)
	}
	if _, err := dto.ParseISO8601Duration("PT"); err == nil {
		t.Error("空 duration 片段必须报错（静默 0 时长治理）")
	}
}

// TestCMDBFullZeroContractShape 锁住专项 B 的「具体类型 → 域内窄契约」形态。
// 编译期接口断言在 domain_contracts_test.go，这里额外锁「契约文件本身」
// 与「组合根接线」：把 contracts.go 塞进共享层路径、或漏掉 SetToolRegistry
// 装配，都能编译通过且 L.3 计数不变，但会在运行时静默退化。
func TestCMDBFullZeroContractShape(t *testing.T) {
	contract := readCMDBFile(t, "contracts.go")
	if strings.Contains(contract, cmdbSharedImport) {
		t.Error("handlers/cmdb/contracts.go 含共享层包路径字面量（连注释都算，L.3 全文 grep）")
	}
	for _, name := range []string{
		"CITypeCatalog", "CIAttributeDefinitionCatalog", "ConfigurationItemCatalog",
		"CIRelationshipCatalog", "CIHistoryCatalog", "CITagCatalog",
		"CMDBImportExportCatalog", "ToolRegistryCatalog",
	} {
		if !regexp.MustCompile(`(?m)^type ` + name + ` interface \{`).MatchString(contract) {
			t.Errorf("contracts.go 丢失窄契约 %s", name)
		}
	}
	for _, needle := range []string{
		"dto.LifecycleChangeParams", "[]dto.ToolDefinition", "*dto.ListResponse[dto.ExportCIResult]",
	} {
		if !strings.Contains(contract, needle) {
			t.Errorf("contracts.go 签名闭包丢失 %q（下沉类型必须显式带 dto 限定符）", needle)
		}
	}

	prod := readCMDBFile(t, "production_service.go")
	if strings.Contains(prod, "*service.") {
		t.Error("production_service.go 仍持有共享层具体类型指针（旧实现）")
	}
	for _, needle := range []string{
		"ciTypeService                CITypeCatalog",
		"ciService                    ConfigurationItemCatalog",
		"importExportService          CMDBImportExportCatalog",
		"toolRegistry                 ToolRegistryCatalog",
		"ciTypeService CITypeCatalog,",
	} {
		if !strings.Contains(squashSpace(stripLineComments(prod)), squashSpace(needle)) {
			t.Errorf("production_service.go 字段或构造参数未改用窄契约：%q", needle)
		}
	}

	// 组合根零改动：装配语句必须仍在，且不应出现「域内自建 operator 上下文」。
	app, err := os.ReadFile(filepath.Join("..", "..", "internal", "bootstrap", "app.go"))
	if err != nil {
		t.Fatalf("读取 internal/bootstrap/app.go 失败：%v", err)
	}
	appCode := string(app)
	for _, needle := range []string{
		"cmdbProductionService := cmdb.NewProductionService(",
		"cmdbProductionService.SetToolRegistry(toolRegistry)",
	} {
		if !strings.Contains(appCode, needle) {
			t.Errorf("app.go 装配接线丢失 %q（编译与 L.3 都看不出漏接线）", needle)
		}
	}
	if strings.Contains(appCode, "dto.WithOperator") {
		t.Error("app.go 不应为 operator 上下文新增装配（组合根零改动前提）")
	}
}

// TestCMDBContractDTOAndOperatorIdentity 锁住 B 批 4 个下沉符号的身份与
// operator context 的跨包往返。WithOperator 与 OperatorFromContext 若拆成
// 两份实现（各自定义私有 context key），类型身份、L.3 计数和编译都不变，
// 但 CI 变更历史会静默落成 SystemOperatorID。
func TestCMDBContractDTOAndOperatorIdentity(t *testing.T) {
	pairs := []struct {
		name           string
		shared, bottom reflect.Type
	}{
		{"LifecycleChangeParams", reflect.TypeOf(service.LifecycleChangeParams{}), reflect.TypeOf(dto.LifecycleChangeParams{})},
		{"ToolDefinition", reflect.TypeOf(service.ToolDefinition{}), reflect.TypeOf(dto.ToolDefinition{})},
	}
	for _, p := range pairs {
		if p.shared != p.bottom {
			t.Errorf("%s：共享层与 dto 类型身份不一致（%v vs %v）", p.name, p.shared, p.bottom)
		}
	}
	if !errors.Is(service.ErrInvalidRelationshipType, dto.ErrInvalidRelationshipType) {
		t.Error("service.ErrInvalidRelationshipType 必须就是 dto 的同一实例（errors.Is）")
	}
	if service.SystemOperatorID != dto.SystemOperatorID {
		t.Error("SystemOperatorID 在共享层与 dto 必须是同一个常量值")
	}

	ctx := dto.WithOperator(context.Background(), 42, "alice")
	if id, name := service.OperatorFromContext(ctx); id != 42 || name != "alice" {
		t.Errorf("dto.WithOperator → service.OperatorFromContext 往返失败：(%d, %q)", id, name)
	}
	ctx2 := service.WithOperator(context.Background(), 7, "bob")
	if id, name := dto.OperatorFromContext(ctx2); id != 7 || name != "bob" {
		t.Errorf("service.WithOperator → dto.OperatorFromContext 往返失败：(%d, %q)", id, name)
	}
	if id, _ := service.OperatorFromContext(dto.WithOperator(context.Background(), -1, "x")); id != dto.SystemOperatorID {
		t.Errorf("非法 operatorID 未收敛到 SystemOperatorID：%d", id)
	}

	payload, err := json.Marshal(dto.ToolDefinition{
		Name: "create_ticket", Description: "建单", ReadOnly: false, Resource: "ticket", Action: "write",
		ArgsSchema: map[string]interface{}{"title": "string"}, ResultSchema: map[string]interface{}{"id": "int"},
	})
	if err != nil {
		t.Fatalf("序列化 ToolDefinition 失败：%v", err)
	}
	for _, key := range []string{
		`"name":"create_ticket"`, `"description":"建单"`, `"readOnly":false`,
		`"resource":"ticket"`, `"action":"write"`, `"argsSchema":{`, `"resultSchema":{`,
	} {
		if !strings.Contains(string(payload), key) {
			t.Errorf("ToolDefinition JSON 形状丢失 %s：%s", key, payload)
		}
	}

	// service 侧必须是别名/薄委托，而不是重新定义 struct 或第二份 context key。
	guards := []struct{ file, decl string }{
		{"configuration_item_service.go", "type LifecycleChangeParams = dto.LifecycleChangeParams"},
		{"tool_registry.go", "type ToolDefinition = dto.ToolDefinition"},
		{"ci_relationship_service.go", "var ErrInvalidRelationshipType = dto.ErrInvalidRelationshipType"},
		{"operator_context.go", "const SystemOperatorID = dto.SystemOperatorID"},
		{"operator_context.go", "return dto.WithOperator(ctx, operatorID, operatorName)"},
		{"operator_context.go", "return dto.OperatorFromContext(ctx)"},
	}
	for _, g := range guards {
		body := readServiceFile(t, g.file)
		if !strings.Contains(squashSpace(stripLineComments(body)), squashSpace(g.decl)) {
			t.Errorf("service/%s 丢失别名/薄委托 %q", g.file, g.decl)
		}
	}
}
