package service

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 引擎完成任务时写入的流程变量是 camelCase（approvalAction / approvalResult /
// approvalComment，见 service/bpmn_process_executor.go 与 handlers/bpmn/workflow.go）。
// 内置模板里的连线条件若写蛇形名，条件恒为 false，网关会走错分支且不报错，
// 所以这类笔误必须由语料级守卫锁死，而不是靠人工 review。
var (
	conditionExpressionRe = regexp.MustCompile(`(?s)<[^>]*conditionExpression[^>]*>(.*?)</[^>]*conditionExpression>`)
	cdataRe               = regexp.MustCompile(`(?s)^<!\[CDATA\[(.*?)\]\]>$`)
	snakeApprovalVarRe    = regexp.MustCompile(`approval[_-](result|action|comment)`)
)

func TestEmbeddedBPMNTemplatesNeverReadSnakeCaseApprovalVariable(t *testing.T) {
	entries, err := fs.ReadDir(bpmnTemplates, "bpmn")
	require.NoError(t, err)

	scanned := 0
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".bpmn" {
			continue
		}
		data, err := bpmnTemplates.ReadFile("bpmn/" + e.Name())
		require.NoError(t, err)
		scanned++

		for _, cond := range extractConditionExpressions(string(data)) {
			if loc := snakeApprovalVarRe.FindString(cond); loc != "" {
				t.Errorf("模板 %s 的条件用了蛇形审批变量 %q，引擎只写驼峰 approvalResult/approvalAction：\n%s",
					e.Name(), loc, cond)
			}
		}
	}
	require.Positive(t, scanned, "未扫描到任何内置模板，守卫已失效")
}

// TestSnakeCaseApprovalGuardIsNotVacuous 用一段已知错误条件验证守卫确实会拦截，
// 防止正则为空或 CDATA 剥离失效导致测试永远通过。
func TestSnakeCaseApprovalGuardIsNotVacuous(t *testing.T) {
	bad := `<bpmn:conditionExpression><![CDATA[${variables['approval_result'] == 'approved'}]]></bpmn:conditionExpression>`
	conditions := extractConditionExpressions(bad)
	require.Len(t, conditions, 1)
	require.True(t, snakeApprovalVarRe.MatchString(conditions[0]))

	good := `<bpmn:conditionExpression><![CDATA[${variables['approvalResult'] == 'approved'}]]></bpmn:conditionExpression>`
	goodConditions := extractConditionExpressions(good)
	require.Len(t, goodConditions, 1)
	require.False(t, snakeApprovalVarRe.MatchString(goodConditions[0]))
}

func extractConditionExpressions(xml string) []string {
	matches := conditionExpressionRe.FindAllStringSubmatch(xml, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		body := strings.TrimSpace(m[1])
		if sub := cdataRe.FindStringSubmatch(body); sub != nil {
			body = strings.TrimSpace(sub[1])
		}
		out = append(out, body)
	}
	return out
}
