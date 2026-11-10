package seeder

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// verifyWorkflowTemplates（initialization_adapter.go:571）要求每条 active
// process_bindings 都能按 (key, tenant, is_active) 找到 process_definition，而
// process_definition 只能由 service/bpmn/*.bpmn 经 BPMNTemplateService
// LoadAndDeployTemplates 部署。清单里出现没有模板载体的 key，全新安装的
// workflow-core 组件就会校验失败并回滚，整套 backend/worker/frontend 起不来。
//
// 部门级默认绑定不自建第二份 key 清单：service/bpmn_process_binding_service.go
// 只消费 service/scenario 的唯一目录，可用性由 go:embed bpmn/*.bpmn 决定，
// 见 service/bpmn_department_bindings_guard_test.go 的双向基线。
func TestBuiltinProcessBindingsHaveDeployableTemplates(t *testing.T) {
	templateDir := filepath.Join("..", "..", "service", "bpmn")
	entries, err := os.ReadDir(templateDir)
	require.NoError(t, err, "BPMN 模板目录缺失：部署来源已变，需同步本守卫")

	deployable := make(map[string]bool, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".bpmn") {
			continue
		}
		deployable[strings.TrimSuffix(name, ".bpmn")] = true
	}
	require.NotEmpty(t, deployable)

	for _, binding := range getEmbeddedConfig().ProcessBindings {
		require.Truef(t, deployable[binding.ProcessDefinitionKey],
			"内置绑定 %s/%s 指向 %q，但 service/bpmn 下没有同名模板；全新安装会在 workflow-core verify 处回滚",
			binding.BusinessType, binding.BusinessSubType, binding.ProcessDefinitionKey)
	}
}

func TestBuiltinWorkflowCatalogLabelsAreEnglish(t *testing.T) {
	comments := regexp.MustCompile(`(?s)<!--.*?-->`)
	for _, template := range workflowTemplateDefinitions() {
		t.Run(template.key, func(t *testing.T) {
			require.NotRegexp(t, `[\p{Han}]`, template.name)
			require.NotRegexp(t, `[\p{Han}]`, template.desc)
			content, err := seedWorkflowTemplateFS.ReadFile(template.bpmnFile)
			require.NoError(t, err)
			require.NotRegexp(t, `[\p{Han}]`, comments.ReplaceAllString(string(content), ""),
				"built-in catalog diagrams must use English display labels")
		})
	}
}

// TestTicketProcessBindingsStayInTicketDomain 锁定「工单聚合只跑 ticket 域流程」。
//
// ticket 的 type=incident/problem/change/service_request 是工单分类，不是另外四个
// 领域的单据。把工单绑到 incident_emergency_flow 这类外域流程后，服务任务按
// incident_id/change_id 写库，工单聚合读到 0，workflow.start 重试 8 次进死信；
// ID 撞号时还会改到另一张域表。运行时虽已加回退守卫，但种子仍在制造错误配置。
func TestTicketProcessBindingsStayInTicketDomain(t *testing.T) {
	templateDir := filepath.Join("..", "..", "service", "bpmn")
	categories := bpmnTemplateCategories(t, templateDir)
	require.NotEmpty(t, categories)

	allowed := map[string]bool{"": true, "default": true, "ticket": true}
	for _, binding := range getEmbeddedConfig().ProcessBindings {
		if binding.BusinessType != "ticket" {
			continue
		}
		category, ok := categories[binding.ProcessDefinitionKey]
		require.Truef(t, ok, "工单绑定指向 %q，但模板不存在", binding.ProcessDefinitionKey)
		require.Truef(t, allowed[strings.ToLower(strings.TrimSpace(category))],
			"工单 %s/%s 绑定了 %q（category=%s）：工单聚合不能执行其他领域的流程",
			binding.BusinessType, binding.BusinessSubType, binding.ProcessDefinitionKey, category)
	}
}

func bpmnTemplateCategories(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	out := make(map[string]string, len(entries))
	categoryRE := regexp.MustCompile(`<bpmn:metaData\s+name="category">([^<]*)</bpmn:metaData>`)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".bpmn") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		require.NoError(t, err)
		key := strings.TrimSuffix(entry.Name(), ".bpmn")
		if m := categoryRE.FindSubmatch(raw); len(m) == 2 {
			out[key] = strings.TrimSpace(string(m[1]))
		} else {
			out[key] = ""
		}
	}
	return out
}
