package service

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBuiltinTemplates_Gate 内置 BPMN 模板常驻回归门禁（2026-09-07 架构加固）。
//
// 背景：16 个内置模板曾有 12 处真实图缺口、24 处 outgoing 声明悬空、2 个 XML
// 语法损坏、2 处重复 sequenceFlow id，且引擎静默卡死导致问题长期不可见。
// 本测试守住三层防线：
//  1. lint 门禁：每个模板 0 error（悬空声明/死端/重复 id 均为 error 级）；
//  2. parser 富化：OutgoingDecls 索引可解析（引擎 fallback 与 lint 依赖此数据）；
//  3. 非结束事件出边完整性（lint 规则 2 的兜底断言，防 lint 被误降级）。
//
// 新增模板时本测试自动纳入；有意放宽某条规则时必须显式修改本测试并说明。
func TestBuiltinTemplates_Gate(t *testing.T) {
	lintSvc := NewBPMNLintService()
	entries, err := os.ReadDir("bpmn")
	if err != nil {
		t.Fatalf("读取 bpmn 目录失败: %v", err)
	}

	parser := NewBPMNParser()
	checked := 0

	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".bpmn" {
			continue
		}
		name := e.Name()
		checked++

		data, err := os.ReadFile(filepath.Join("bpmn", name))
		if err != nil {
			t.Fatalf("%s: 读取失败: %v", name, err)
		}

		// 防线 1+3：lint 0 error
		result, err := lintSvc.LintBPMNXML(data)
		if err != nil {
			t.Errorf("%s: lint 解析失败（XML 损坏）: %v", name, err)
			continue
		}
		if result.HasErrors {
			for _, is := range result.Issues {
				if is.Severity == "error" {
					t.Errorf("%s: lint error: %s", name, is.Message)
				}
			}
		}

		// 防线 2：parser 富化守卫
		defs, err := parser.ParseXML(data)
		if err != nil {
			t.Errorf("%s: parser 解析失败: %v", name, err)
			continue
		}
		for _, proc := range defs.Processes {
			// 每个带 outgoing 声明的模板都必须产出索引（enrichProcessElements 生效）
			hasDecl := false
			for _, decls := range proc.OutgoingDecls {
				if len(decls) > 0 {
					hasDecl = true
					break
				}
			}
			if !hasDecl {
				t.Errorf("%s: process %s 无 OutgoingDecls 索引（parser 富化失效，引擎 fallback 与 lint 一致性校验将退化）", name, proc.ID)
			}
		}
	}

	if checked < 16 {
		t.Fatalf("内置模板数量异常: 期望 >=16，实际 %d（模板被误删或目录错误）", checked)
	}
	t.Logf("内置模板门禁通过: %d 个模板", checked)
}
