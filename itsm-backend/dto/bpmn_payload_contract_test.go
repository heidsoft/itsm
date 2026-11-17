package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"itsm-backend/ent"
)

// 流程定义列表曾经把每条定义的完整 BPMN XML 一起返回：单份十几 KB，一页 20 条就是几百 KB，
// 而前端预览/设计器都按 key 走详情接口取 XML，列表里的载荷没有任何消费者。
// 这里锁住「列表不带 XML、详情带 XML」的契约，防止回退。
func TestBPMNProcessDefinitionList_DropsXMLButDetailKeepsIt(t *testing.T) {
	xml := []byte(`<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"/>`)
	definitions := []*ent.ProcessDefinition{
		{ID: 1, Key: "flow_a", Name: "A", Version: "2", IsLatest: true, IsActive: true, BpmnXML: xml},
		{ID: 2, Key: "flow_b", Name: "B", Version: "1", BpmnXML: xml},
	}

	list := ToBPMNProcessDefinitionListResponse(definitions)
	require.Len(t, list, 2)
	for i, item := range list {
		assert.Empty(t, item.BpmnXML, "列表第 %d 条不得携带 bpmnXml", i)
		assert.Equal(t, definitions[i].Key, item.Key)
		assert.Equal(t, definitions[i].Version, item.Version)
	}

	detail := ToBPMNProcessDefinitionResponse(definitions[0])
	assert.NotEmpty(t, detail.BpmnXML, "详情接口仍然是 XML 的唯一来源")
}

// 实例响应不得外泄引擎内部控制键（`_` 前缀：完成位图、幂等键、补偿顺序等）。
func TestBPMNProcessInstanceResponse_FiltersEngineControlVariables(t *testing.T) {
	instance := &ent.ProcessInstance{
		ID:                   7,
		ProcessInstanceID:    "PI-flow_a-1",
		ProcessDefinitionKey: "flow_a",
		Status:               "running",
		Variables: map[string]interface{}{
			"_done_":          []string{"startEvent"},
			"_commandId":      "cmd-1",
			"_idempotencyKey": "idem-1",
			"ticketId":        "42",
			"assignee_id":     "7",
		},
	}

	resp := ToBPMNProcessInstanceResponse(instance)
	assert.NotContains(t, resp.Variables, "_done_")
	assert.NotContains(t, resp.Variables, "_commandId")
	assert.NotContains(t, resp.Variables, "_idempotencyKey")
	assert.Equal(t, "42", resp.Variables["ticketId"], "业务变量必须原样保留")

	list := ToBPMNProcessInstanceListResponse([]*ent.ProcessInstance{instance})
	require.Len(t, list, 1)
	assert.NotContains(t, list[0].Variables, "_commandId")
}
