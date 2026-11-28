package group

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"itsm-backend/ent"
	"itsm-backend/ent/approvalchain"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/internal/bpmncontracts"
)

// GroupReference 描述「某个审批组名被何处引用」。
//
// 组名是审批人解析键，而不是展示名：
//   - BPMN userTask 的 candidateGroups 在流程实例化时落库到 process_task.candidate_groups，
//     引擎用 GroupResolver.ExpandGroupsToUsers 按 group.name 精确查成员；
//   - 审批链 chain[].role 的 "group:<name>" 形式由 resolveChainApprovers 走同一个
//     ExpandGroupsToUsers 分支。
//
// 两条路径都按精确组名匹配（ent 的 group.NameIn，非模糊匹配），所以一旦改名或删除，
// 候选审批人会静默变空——流程仍能发布、任务仍能创建，只是永远没人能审批。
// 因此组变更前必须先确认它没有被引用。
type GroupReference struct {
	// Kind 取值：process_definition（BPMN userTask）或 approval_chain（审批链 step）。
	Kind string `json:"kind"`
	// RefID 引用来源的实体 ID（流程定义 ID / 审批链 ID）。
	RefID int `json:"refId"`
	// RefKey 引用来源的可读标识（流程 key / 审批链 name）。
	RefKey string `json:"refKey"`
	// NodeID 引用位置（BPMN userTask id / 审批链 level）。
	NodeID string `json:"nodeId"`
}

// findGroupReferences 扫描指定租户内所有对 groupName 的审批引用。
//
// 只覆盖两条运行时真实消费 groupName 的路径，且与运行时共用同一套 token 分割口径：
//  1. 生效中的流程定义 BpmnXML 里 userTask 的 candidateGroups（CSV，按 token 精确匹配，
//     对应 GroupResolver.ExpandGroupsToUsers 的查询输入）；
//  2. 审批链 chain[].role 中的 "group:<name>"（对应 resolveChainApprovers 的 group 分支）。
//
// 不扫描 <metaData name="assignee_type">：运行时并未把它当作组名消费
// （内置模板里它的值是 manager/technician 等角色 code），扫它会凭空造出引用面。
//
// 失败关闭：任一流程定义 XML 无法解析时返回 error，调用方必须拒绝变更，
// 避免用坏 XML 绕过引用检查。
//
// 落点说明：本扫描器留在 handlers/group 域内是 layering 约束的结果——
// L.3 棘轮要求 handler 生产文件对共享实现包（itsm-backend/service 与
// itsm-backend/internal/services）的依赖恒为 0，而 XML 类型真源
// internal/bpmncontracts 只依赖 dto/ent，是域内可直接复用的纯类型包。
// 这里用标准库 encoding/xml 直接反序列化到 bpmncontracts.BPMNDefinitions，
// 与运行时解析器使用的是同一份 XML tag 定义，不另立一套结构。
func findGroupReferences(ctx context.Context, client *ent.Client, tenantID int, groupName string) ([]GroupReference, error) {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" || client == nil || tenantID <= 0 {
		return nil, nil
	}

	var refs []GroupReference

	defs, err := client.ProcessDefinition.Query().
		Where(
			processdefinition.TenantID(tenantID),
			processdefinition.IsActive(true),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询流程定义失败: %w", err)
	}
	for _, def := range defs {
		var parsed bpmncontracts.BPMNDefinitions
		if perr := xml.Unmarshal(def.BpmnXML, &parsed); perr != nil {
			return nil, fmt.Errorf("流程定义 %s (id=%d) XML 无法解析，无法确认组引用: %w", def.Key, def.ID, perr)
		}
		for _, proc := range parsed.Processes {
			for _, task := range proc.UserTasks {
				if csvContainsToken(task.CandidateGroups, groupName) {
					refs = append(refs, GroupReference{
						Kind:   "process_definition",
						RefID:  def.ID,
						RefKey: def.Key,
						NodeID: task.ID,
					})
				}
			}
		}
	}

	chains, err := client.ApprovalChain.Query().
		Where(approvalchain.TenantID(tenantID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询审批链失败: %w", err)
	}
	for _, ch := range chains {
		for _, st := range ch.Chain {
			if st.Role == "group:"+groupName {
				refs = append(refs, GroupReference{
					Kind:   "approval_chain",
					RefID:  ch.ID,
					RefKey: ch.Name,
					NodeID: fmt.Sprintf("level-%d", st.Level),
				})
			}
		}
	}

	return refs, nil
}

// csvContainsToken 按 CSV 语义精确匹配一个 token。
// 分割口径与运行时一致：逗号分隔 + 逐项 TrimSpace + 丢弃空项
// （与 service/bpmn 的 splitCSV、internal/services 的 csvTokenEQ 同语义）。
func csvContainsToken(csv, token string) bool {
	if token == "" {
		return false
	}
	for _, part := range strings.Split(csv, ",") {
		if strings.TrimSpace(part) == token {
			return true
		}
	}
	return false
}
