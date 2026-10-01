package seeder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// verifyWorkflowTemplates（initialization_adapter.go:571）要求每条 active
// process_bindings 都能按 (key, tenant, is_active) 找到 process_definition，而
// process_definition 只能由 service/bpmn/*.bpmn 经 BPMNTemplateService
// LoadAndDeployTemplates 部署。清单里出现没有模板载体的 key，全新安装的
// workflow-core 组件就会校验失败并回滚，整套 backend/worker/frontend 起不来
// （2026-10-01 清卷全新部署实证：迁移写入的 incident_general_flow、
// change_emergency_flow、release_test_flow、expense_approval_flow 均无载体）。
//
// 部门级默认绑定（service/bpmn_process_binding_service.go
// getDepartmentDefaultBindings）同样引用了未部署的 key，属于运行期缺陷，另行处理。
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
