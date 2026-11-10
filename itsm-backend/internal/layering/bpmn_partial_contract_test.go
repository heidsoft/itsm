package layering

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/service"
)

// 批次 3 第 1 步：bpmn 是「部分归零」域，只有 workflow_template.go 不再依赖
// 共享层，其余 7 个生产文件仍 import service/。因此它不能进 contractDomains
// （那个守卫逐文件扫整个域），改由本文件按「单文件 + 基线」精确锁定。
// 后续每组归零时，把 bpmnPartialZeroExpectedImports 与基线期望值同步下调即可。

const bpmnSharedImport = `"itsm-backend/service"`

const bpmnPartialZeroExpectedImports = 7

func readBPMNFile(t *testing.T, rel string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "handlers", "bpmn", rel))
	if err != nil {
		t.Fatalf("读取 handlers/bpmn/%s 失败：%v", rel, err)
	}
	if len(body) < 100 {
		t.Fatalf("扫描器零命中保护：handlers/bpmn/%s 仅 %d 字节", rel, len(body))
	}
	return string(body)
}

// TestBPMNPartialZeroWorkflowTemplate 锁住「归零是换成了域内窄契约」而不是
// 把文件删掉或把共享层限定符换成别的写法。改回旧实现必须在这里变红。
func TestBPMNPartialZeroWorkflowTemplate(t *testing.T) {
	wf := readBPMNFile(t, "workflow_template.go")
	if strings.Contains(wf, bpmnSharedImport) {
		t.Errorf("handlers/bpmn/workflow_template.go 重新引用共享层包，L.3 归零被破坏")
	}
	code := stripLineComments(wf)
	if !regexp.MustCompile(`\bcatalog\s+WorkflowTemplateCatalog\b`).MatchString(code) {
		t.Errorf("模板目录字段未使用域内窄接口 WorkflowTemplateCatalog")
	}
	if !regexp.MustCompile(`\bNewWorkflowTemplateHandler\(catalog\s+WorkflowTemplateCatalog\)`).MatchString(code) {
		t.Errorf("构造函数未接收域内窄接口 WorkflowTemplateCatalog")
	}
	if strings.Contains(code, "*service.BPMNWorkflowTemplateCatalog") {
		t.Errorf("模板目录字段回退到共享层具体类型（旧实现）")
	}
	if strings.Contains(code, "service.Err") {
		t.Errorf("错误映射回退到共享层 sentinel 限定符（旧实现）")
	}

	contract := readBPMNFile(t, "contracts.go")
	if len(contract) < 100 {
		t.Fatalf("扫描器零命中保护：contracts.go 仅 %d 字节", len(contract))
	}
	if !strings.Contains(contract, "type WorkflowTemplateCatalog interface") {
		t.Errorf("contracts.go 未声明域内接口 WorkflowTemplateCatalog")
	}
	if strings.Contains(contract, bpmnSharedImport) {
		t.Errorf("contracts.go 出现共享层包路径字面量（连注释也会让 L.3 计数 +1）")
	}

	// 零命中不能由「目录被清空 / 文件被删光」造成。
	entries, err := os.ReadDir(filepath.Join("..", "..", "handlers", "bpmn"))
	if err != nil {
		t.Fatalf("列举 handlers/bpmn 失败：%v", err)
	}
	prodFiles := 0
	for _, e := range entries {
		fn := e.Name()
		if e.IsDir() || !strings.HasSuffix(fn, ".go") || strings.HasSuffix(fn, "_test.go") {
			continue
		}
		prodFiles++
	}
	if prodFiles < 9 {
		t.Fatalf("扫描器零命中保护：handlers/bpmn 只剩 %d 个生产 .go 文件", prodFiles)
	}
}

// TestBPMNPartialZeroMatchesL3Baseline 同时锁住代码实测计数与棘轮基线：
// 只改代码不收紧基线（或只改基线不改代码）都要变红。
func TestBPMNPartialZeroMatchesL3Baseline(t *testing.T) {
	dir := filepath.Join("..", "..", "handlers", "bpmn")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("列举 handlers/bpmn 失败：%v", err)
	}
	count := 0
	for _, e := range entries {
		fn := e.Name()
		if e.IsDir() || !strings.HasSuffix(fn, ".go") || strings.HasSuffix(fn, "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, fn))
		if err != nil {
			t.Fatalf("读取 handlers/bpmn/%s 失败：%v", fn, err)
		}
		if len(body) < 40 {
			t.Fatalf("扫描器零命中保护：handlers/bpmn/%s 仅 %d 字节", fn, len(body))
		}
		// 与 Gate C.8 / L.3 一致：全文 grep，注释里的包路径同样算命中。
		if strings.Contains(string(body), bpmnSharedImport) {
			count++
		}
	}
	if count != bpmnPartialZeroExpectedImports {
		t.Errorf("handlers/bpmn 共享层依赖实测 %d，期望 %d（批次 3 第 1 步只归零 workflow_template.go）", count, bpmnPartialZeroExpectedImports)
	}

	base, err := os.ReadFile(filepath.Join("..", "..", "..", "scripts", "docs-gate", "handler-service-imports-baseline.txt"))
	if err != nil {
		t.Fatalf("读取 L.3 基线失败：%v", err)
	}
	if len(base) < 100 {
		t.Fatalf("扫描器零命中保护：L.3 基线仅 %d 字节", len(base))
	}
	values := []string{}
	for _, line := range strings.Split(string(base), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "bpmn=") {
			values = append(values, strings.TrimPrefix(line, "bpmn="))
		}
	}
	want := "7"
	if len(values) != 1 || values[0] != want {
		t.Errorf("L.3 基线 bpmn 期望恰好一条 bpmn=%s，实际 %v（真下降必须同 commit 精确收紧）", want, values)
	}
}

// TestBPMNSharedValuesComeFromDTO 锁住纯值的单一来源：常量与 sentinel 定义在
// dto，service 只做别名，handler 只读 dto。任何一层另起一份都会让 errors.Is
// 或默认阈值漂移，而 L.3 计数门禁看不出来。
func TestBPMNSharedValuesComeFromDTO(t *testing.T) {
	if service.DefaultStaleThresholdHours != dto.DefaultStaleThresholdHours {
		t.Errorf("默认陈旧阈值在共享层与 dto 之间漂移：%d vs %d", service.DefaultStaleThresholdHours, dto.DefaultStaleThresholdHours)
	}

	inv := readBPMNFile(t, "inventory.go")
	invCode := stripLineComments(inv)
	if !strings.Contains(invCode, "dto.DefaultStaleThresholdHours") {
		t.Errorf("inventory.go 未从 dto 读取默认陈旧阈值")
	}
	if strings.Contains(invCode, "service.DefaultStaleThresholdHours") {
		t.Errorf("inventory.go 回退到共享层常量限定符（旧实现）")
	}

	bottom, err := os.ReadFile(filepath.Join("..", "..", "dto", "bpmn_workflow_template_sentinels.go"))
	if err != nil {
		t.Fatalf("读取 dto/bpmn_workflow_template_sentinels.go 失败：%v", err)
	}
	if len(bottom) < 100 {
		t.Fatalf("扫描器零命中保护：dto 纯值文件仅 %d 字节", len(bottom))
	}
	flat := squashSpace(string(bottom))
	for _, decl := range []string{
		"ErrWorkflowTemplateNotFound = errors.New(",
		"ErrWorkflowTemplateConflict = errors.New(",
		"ErrWorkflowTemplateInvalid = errors.New(",
		"const DefaultStaleThresholdHours = 24",
	} {
		if !strings.Contains(flat, squashSpace(decl)) {
			t.Errorf("dto 纯值文件缺少定义 %q（sentinel / 阈值必须只有一份）", decl)
		}
	}
}
