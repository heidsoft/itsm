package layering

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"itsm-backend/internal/bpmntemplates"
	"itsm-backend/internal/services"
)

// G0-b 内置 BPMN 模板归位专项回归锁（2026-11-12）。
//
// 本批把 20 个 .bpmn 与 digest 的两个派生口径从 service/ 整体迁入
// internal/bpmntemplates；service 侧只保留 bpmnTemplates 过渡变量与两个薄委托。
//
// 假绿风险全部落在「编译全绿但语义已经分叉」：
//   - 模板漏搬 / 误删 / 内容被改：embed FS 与磁盘仍自洽，编译与 L.1 全绿；
//   - service 侧重新长出第二份 embed 或第二份 digest 实现：两份模板清单会漂移；
//   - 硬编码路径的守卫（role 契约、seeder 清单）没跟着改路径，读到的会是空集合；
//   - 旧文件名或基线行复活，L.1 / C.6.4 棘轮留下免检空间。
//
// 因此锁同时覆盖：模板完整性清单 + 内容摘要锚点 + 独立重算 + 委托形态 +
// 全目录零复活扫描 + 读面路径同步 + 两份基线的上界棘轮。

// bteMigratedTemplates 是迁移时 HEAD 侧 service/bpmn/ 下的全部模板名。
// 作为完整性下界使用：后续允许新增模板，但任何一个被漏搬或误删都必须红。
var bteMigratedTemplates = []string{
	"change_emergency_flow.bpmn", "change_normal_flow.bpmn", "change_normal_flow_cn.bpmn",
	"cloud_private_ops_flow.bpmn", "cloud_public_ops_flow.bpmn", "cloud_security_scan_flow.bpmn",
	"compensation_recovery_flow.bpmn", "dev_approval_ops_flow.bpmn",
	"incident_emergency_flow.bpmn", "incident_emergency_flow_cn.bpmn",
	"incident_emergency_flow_v1.1.bpmn", "problem_management_flow.bpmn",
	"problem_management_flow_cn.bpmn", "release_approval_flow.bpmn",
	"release_approval_flow_cn.bpmn", "scheduled_inspection_flow.bpmn",
	"service_request_flow.bpmn", "service_request_flow_cn.bpmn",
	"ticket_assignment_flow.bpmn", "ticket_general_flow.bpmn",
}

// bteMigratedDigest 是迁移前 service/bpmn/*.bpmn 按 EmbeddedBPMNDigest 口径
// 算出的摘要。它是版本账本锚点而不是「本批指标值」：模板内容一旦变化，
// 必须走模板版本升级流程并同步更新本常量，避免初始化组件校验和静默漂移。
const bteMigratedDigest = "69a25a6808d1ef39a9f29d7a60a5e4d0937c4306c300625051099ce0d15f7ce6"

func bteReadBackend(t *testing.T, parts ...string) string {
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

func bteReadRepo(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(append([]string{"..", "..", ".."}, parts...)...)
	body, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("读取 %s 失败：%v", p, err)
	}
	if len(body) < 100 {
		t.Fatalf("扫描器零命中保护：%s 仅 %d 字节", p, len(body))
	}
	return string(body)
}

// bteDiskTemplates 返回磁盘上模板目录里的 .bpmn 文件名（已排序）。
func bteDiskTemplates(t *testing.T) []string {
	t.Helper()
	dir := filepath.Join("..", "..", "internal", "bpmntemplates", "bpmn")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取模板目录 %s 失败：%v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".bpmn") {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		t.Fatal("扫描器零命中保护：模板目录里一个 .bpmn 都没有")
	}
	sort.Strings(names)
	return names
}

// bteDiskDigest 用与生产实现完全相同的口径，从磁盘文件独立重算摘要。
func bteDiskDigest(t *testing.T, names []string) string {
	t.Helper()
	h := sha256.New()
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join("..", "..", "internal", "bpmntemplates", "bpmn", name))
		if err != nil {
			t.Fatalf("读取模板 %s 失败：%v", name, err)
		}
		fmt.Fprintf(h, "%s\x00%d\x00", name, len(data))
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func bteFunctionBody(src, name string) (string, bool) {
	idx := strings.Index(src, "func "+name+"(")
	if idx < 0 {
		return "", false
	}
	rel := strings.IndexByte(src[idx:], '{')
	if rel < 0 {
		return "", false
	}
	start := idx + rel
	depth := 0
	for i := start; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[start+1 : i], true
			}
		}
	}
	return "", false
}

// TestBTEEmbeddedTemplatesAreSingleSource 锁「唯一 embed FS + 内容等价」。
func TestBTEEmbeddedTemplatesAreSingleSource(t *testing.T) {
	diskNames := bteDiskTemplates(t)

	embedded := bpmntemplates.FS()
	entries, err := fs.ReadDir(embedded, "bpmn")
	if err != nil {
		t.Fatalf("读取 embed FS 的 bpmn 目录失败：%v", err)
	}
	embeddedNames := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			embeddedNames = append(embeddedNames, e.Name())
		}
	}
	sort.Strings(embeddedNames)
	if !reflect.DeepEqual(embeddedNames, diskNames) {
		t.Fatalf("embed FS 的模板集合与磁盘不一致：embed=%v disk=%v", embeddedNames, diskNames)
	}

	for _, name := range diskNames {
		fromFS, err := embedded.ReadFile("bpmn/" + name)
		if err != nil {
			t.Fatalf("embed FS 读不到 %s：%v", name, err)
		}
		fromDisk, err := os.ReadFile(filepath.Join("..", "..", "internal", "bpmntemplates", "bpmn", name))
		if err != nil {
			t.Fatalf("磁盘读不到 %s：%v", name, err)
		}
		if !reflect.DeepEqual(fromFS, fromDisk) {
			t.Errorf("模板 %s 的 embed 内容与磁盘不一致", name)
		}
	}

	set := make(map[string]bool, len(diskNames))
	for _, n := range diskNames {
		set[n] = true
	}
	for _, want := range bteMigratedTemplates {
		if !set[want] {
			t.Errorf("迁移前的内置模板 %s 在新目录中缺失——漏搬会让对应流程无载体", want)
		}
	}

	wantKeys := make(map[string]bool, len(diskNames))
	for _, n := range diskNames {
		wantKeys[strings.TrimSuffix(n, ".bpmn")] = true
	}
	keys, err := service.BuiltinProcessTemplateKeys()
	if err != nil {
		t.Fatalf("service.BuiltinProcessTemplateKeys 失败：%v", err)
	}
	if !reflect.DeepEqual(keys, wantKeys) {
		t.Errorf("service 侧可部署 key 集合与模板文件不一致：got=%v want=%v", keys, wantKeys)
	}
	pkgKeys, err := bpmntemplates.BuiltinProcessTemplateKeys()
	if err != nil {
		t.Fatalf("bpmntemplates.BuiltinProcessTemplateKeys 失败：%v", err)
	}
	if !reflect.DeepEqual(keys, pkgKeys) {
		t.Errorf("service 侧薄委托与模板包的 key 集合不一致：service=%v pkg=%v", keys, pkgKeys)
	}

	got, err := service.EmbeddedBPMNDigest()
	if err != nil {
		t.Fatalf("service.EmbeddedBPMNDigest 失败：%v", err)
	}
	pkgDigest, err := bpmntemplates.EmbeddedBPMNDigest()
	if err != nil {
		t.Fatalf("bpmntemplates.EmbeddedBPMNDigest 失败：%v", err)
	}
	independent := bteDiskDigest(t, diskNames)
	if got != independent {
		t.Errorf("service 侧摘要与磁盘独立重算不一致：got=%s want=%s", got, independent)
	}
	if got != pkgDigest {
		t.Errorf("service 侧摘要与模板包摘要不一致：service=%s pkg=%s", got, pkgDigest)
	}
	if got != bteMigratedDigest {
		t.Errorf("内置模板摘要偏离迁移基线 %s（got=%s）：模板内容变更必须走版本升级并同步更新锚点", bteMigratedDigest, got)
	}
	again, err := service.EmbeddedBPMNDigest()
	if err != nil || again != got {
		t.Errorf("摘要不具备确定性：first=%s second=%s err=%v", got, again, err)
	}
}

// TestBTEServiceKeepsThinDelegates 锁 service 侧只有过渡变量与两个薄委托，
// 不允许重新长出第二份 embed FS 或第二份 digest 实现。
func TestBTEServiceKeepsThinDelegates(t *testing.T) {
	src := bteReadBackend(t, "internal", "services", "bpmn_template_service.go")

	if !strings.Contains(src, `"itsm-backend/internal/bpmntemplates"`) {
		t.Error("internal/services/bpmn_template_service.go 不再引用 internal/bpmntemplates")
	}
	if strings.Contains(src, "\t\"embed\"") {
		t.Error("service 侧重新 import embed——模板 embed FS 只允许存在于 internal/bpmntemplates")
	}

	foundDelegate := false
	for _, line := range strings.Split(src, "\n") {
		if strings.TrimSpace(line) == "var bpmnTemplates = bpmntemplates.FS()" {
			foundDelegate = true
			break
		}
	}
	if !foundDelegate {
		t.Error("bpmnTemplates 必须是 bpmntemplates.FS() 的过渡变量（逐行精确锚定）")
	}

	for _, tc := range []struct{ fn, call string }{
		{"BuiltinProcessTemplateKeys", "return bpmntemplates.BuiltinProcessTemplateKeys()"},
		{"EmbeddedBPMNDigest", "return bpmntemplates.EmbeddedBPMNDigest()"},
	} {
		body, ok := bteFunctionBody(src, tc.fn)
		if !ok {
			t.Errorf("扫描器零命中保护：找不到 %s 的函数体", tc.fn)
			continue
		}
		if !strings.Contains(body, tc.call) {
			t.Errorf("%s 不再是薄委托：函数体必须含 %q，got=%q", tc.fn, tc.call, body)
		}
		if strings.Contains(body, "sha256") || strings.Contains(body, "fs.ReadDir") {
			t.Errorf("%s 在 service 侧重新实现了摘要/枚举口径（第二份模板清单）", tc.fn)
		}
	}

	// 全目录扫描：service/ 顶层生产文件不得再声明内置模板 embed；
	// 旧 digest 文件与旧模板目录不得复活。
	serviceDir := filepath.Join("..", "..", "internal", "services")
	entries, err := os.ReadDir(serviceDir)
	if err != nil {
		t.Fatalf("读取 service/ 失败：%v", err)
	}
	scanned := 0
	sawTemplateService := false
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		scanned++
		if e.Name() == "bpmn_template_service.go" {
			sawTemplateService = true
		}
		body, err := os.ReadFile(filepath.Join(serviceDir, e.Name()))
		if err != nil {
			t.Fatalf("读取 service/%s 失败：%v", e.Name(), err)
		}
		if strings.Contains(string(body), "//go:embed bpmn/") {
			t.Errorf("internal/services/%s 重新声明内置模板 embed——会出现两份模板清单", e.Name())
		}
	}
	if scanned < 100 || !sawTemplateService {
		t.Fatalf("扫描器零命中保护：只扫到 %d 个生产文件 / sawTemplateService=%v", scanned, sawTemplateService)
	}
	if _, err := os.Stat(filepath.Join(serviceDir, "bpmn_template_digest.go")); !os.IsNotExist(err) {
		t.Errorf("internal/services/bpmn_template_digest.go 复活（stat err=%v）", err)
	}
	leftovers, err := filepath.Glob(filepath.Join(serviceDir, "bpmn", "*.bpmn"))
	if err != nil {
		t.Fatalf("扫描旧模板目录失败：%v", err)
	}
	if len(leftovers) != 0 {
		t.Errorf("internal/services/bpmn/ 下仍有 .bpmn 模板：%v", leftovers)
	}

	// 模板包不得反向依赖共享层或 handler 层（否则 internal 归位名存实亡）。
	pkgDir := filepath.Join("..", "..", "internal", "bpmntemplates")
	pkgEntries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatalf("读取 internal/bpmntemplates 失败：%v", err)
	}
	pkgFiles := 0
	for _, e := range pkgEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		pkgFiles++
		body, err := os.ReadFile(filepath.Join(pkgDir, e.Name()))
		if err != nil {
			t.Fatalf("读取 internal/bpmntemplates/%s 失败：%v", e.Name(), err)
		}
		if strings.Contains(string(body), `"itsm-backend/internal/services"`) || strings.Contains(string(body), `"itsm-backend/handlers`) {
			t.Errorf("internal/bpmntemplates/%s 反向引用共享层/handler 层", e.Name())
		}
	}
	if pkgFiles < 2 {
		t.Fatalf("扫描器零命中保护：模板包只扫到 %d 个 .go", pkgFiles)
	}
}

// TestBTEServiceTestsReadEmbeddedTemplates 锁 service 包内模板门禁的读面：
// 模板搬离 service/ 之后，任何按包内相对路径 "bpmn/..." 读磁盘的测试都会静默
// 读成空集合（目录里只剩 Go 文件），必须统一走与生产同源的 bpmnTemplates FS。
func TestBTEServiceTestsReadEmbeddedTemplates(t *testing.T) {
	dir := filepath.Join("..", "..", "internal", "services")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取 service/ 失败：%v", err)
	}
	scanned := 0
	withEmbed := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		scanned++
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("读取 service/%s 失败：%v", e.Name(), err)
		}
		s := string(body)
		if strings.Contains(s, `os.ReadDir("bpmn"`) ||
			strings.Contains(s, `os.ReadFile("bpmn/`) ||
			strings.Contains(s, `os.ReadFile(filepath.Join("bpmn"`) {
			t.Errorf("internal/services/%s 仍按包内相对路径读 bpmn/ 目录；模板已迁至 internal/bpmntemplates，请改用 bpmnTemplates.Read*", e.Name())
		}
		if strings.Contains(s, "bpmnTemplates.Read") || strings.Contains(s, "fs.ReadDir(bpmnTemplates") {
			withEmbed++
		}
	}
	if scanned < 100 {
		t.Fatalf("扫描器零命中保护：只扫到 %d 个 service 测试文件", scanned)
	}
	if withEmbed < 5 {
		t.Errorf("模板门禁测试应走 embed FS 同源读面，实测只有 %d 个文件含 bpmnTemplates.Read", withEmbed)
	}
}

// TestBTEReadFacesFollowMovedTemplates 锁两处磁盘读面与一处运行时文案，
// 防止有人只改实现、忘记同步「按目录枚举模板」的守卫。
func TestBTEReadFacesFollowMovedTemplates(t *testing.T) {
	role := bteReadBackend(t, "domain", "role", "contract_test.go")
	if !strings.Contains(role, `"../../internal/bpmntemplates/bpmn/*.bpmn"`) {
		t.Error("domain/role/contract_test.go 未指向新模板目录")
	}
	if strings.Contains(role, `"../../service/bpmn/*.bpmn"`) {
		t.Error("domain/role/contract_test.go 仍指向已迁走的 service/bpmn 目录")
	}

	seeder := bteReadBackend(t, "pkg", "seeder", "process_bindings_manifest_test.go")
	newJoin := `filepath.Join("..", "..", "internal", "bpmntemplates", "bpmn")`
	oldJoin := `filepath.Join("..", "..", "internal", "services", "bpmn")`
	if got := strings.Count(seeder, newJoin); got != 2 {
		t.Errorf("seeder 清单守卫应有 2 处新模板目录读面，实测 %d", got)
	}
	if strings.Contains(seeder, oldJoin) {
		t.Error("seeder 清单守卫仍从 service/bpmn 枚举模板（迁移后会读成空集合）")
	}

	adapter := bteReadBackend(t, "pkg", "seeder", "initialization_adapter.go")
	if !strings.Contains(adapter, "internal/bpmntemplates/bpmn/*.bpmn") {
		t.Error("workflow-core 校验失败文案仍指向旧模板路径")
	}
}

// TestBTEBaselinesAreTightened 锁 L.1 与 C.6.4 两本账：
// 旧文件名零复活、文件名行数只许继续下降、service_go_files 只许继续下降。
// 上界语义保证后续批次继续归位不会误报，回退到本批之前的值必红。
func TestBTEBaselinesAreTightened(t *testing.T) {
	l1 := bteReadRepo(t, "scripts", "docs-gate", "service-prod-baseline.txt")
	l1Rows := 0
	for _, line := range strings.Split(l1, "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if line == "" {
			continue
		}
		l1Rows++
		if line == "bpmn_template_digest.go" {
			t.Error("L.1 基线里 bpmn_template_digest.go 复活")
		}
	}
	if l1Rows < 100 {
		t.Fatalf("扫描器零命中保护：L.1 基线只解析出 %d 行", l1Rows)
	}
	if l1Rows > 125 {
		t.Errorf("L.1 基线上界被回退：%d > 125（G0-b 迁移后应精确收紧到 125，且只许继续下降）", l1Rows)
	}

	surface := bteReadRepo(t, "scripts", "docs-gate", "product-surface-baseline.txt")
	parsed := ""
	for _, line := range strings.Split(surface, "\n") {
		line = strings.SplitN(line, "#", 2)[0]
		kv := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(kv) == 2 && strings.TrimSpace(kv[0]) == "service_go_files" {
			parsed = strings.TrimSpace(kv[1])
			break
		}
	}
	if parsed == "" {
		t.Fatal("扫描器零命中保护：C.6.4 基线里找不到 service_go_files 键")
	}
	got, err := strconv.Atoi(parsed)
	if err != nil {
		t.Fatalf("service_go_files 基线值 %q 不是整数：%v", parsed, err)
	}
	if got > 332 {
		t.Errorf("C.6.4 service_go_files 上界被回退：%d > 332", got)
	}
}
