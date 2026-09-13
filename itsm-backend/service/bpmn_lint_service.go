package service

import (
	"fmt"
	"regexp"
	"strings"

	"itsm-backend/dto"
)

// BPMNLintService 对 BPMN XML 做结构 + 语义校验。
//
// 定位：给前端设计器「校验流程」按钮与 BPMN AI 生成器提供统一的后端 Lint 真源。
// 复用 BPMNParser 的结构化解析（部署服务同源），在此之上叠加语义层检查：
// 起止事件、任务配置、网关出入边、连通性/不可达节点等。
//
// 规则分级：error = 部署后引擎无法正确执行；warning = 可执行但有质量风险。
type BPMNLintService struct {
	parser *BPMNParser
}

// NewBPMNLintService 创建 Lint 服务（与部署服务共用同一 parser 实现）
func NewBPMNLintService() *BPMNLintService {
	return &BPMNLintService{parser: NewBPMNParser()}
}

// LintBPMNXML 校验一段 BPMN XML，返回结构化结果。
// XML 无法解析时返回 error（调用方应视为 400 类参数错误）；
// 语义问题一律进入 Issues（Error 级 issue 不应部署）。
func (s *BPMNLintService) LintBPMNXML(xmlData []byte) (*dto.BPMNLintResult, error) {
	result := &dto.BPMNLintResult{Issues: []*dto.BPMNLintIssue{}}

	definitions, err := s.parser.ParseXML(xmlData)
	if err != nil {
		return nil, fmt.Errorf("BPMN XML 解析失败: %w", err)
	}
	// 复用部署服务同款命名空间检查
	if err := s.parser.ValidateBPMNXML(xmlData); err != nil {
		return nil, err
	}

	if len(definitions.Processes) == 0 {
		result.Issues = append(result.Issues, &dto.BPMNLintIssue{
			Severity: "error",
			Category: "structure",
			Message:  "未找到流程定义（<process> 元素）",
		})
		return result, nil
	}

	for _, process := range definitions.Processes {
		s.lintProcess(process, result)
	}

	for _, issue := range result.Issues {
		if issue.Severity == "error" {
			result.HasErrors = true
			break
		}
	}
	result.ErrorCount = countSeverity(result.Issues, "error")
	result.WarningCount = countSeverity(result.Issues, "warning")
	return result, nil
}

func (s *BPMNLintService) lintProcess(process *BPMNProcess, result *dto.BPMNLintResult) {
	prefix := fmt.Sprintf("流程 %s", process.ID)

	// --- 规则组 1：起止事件 ---
	if len(process.StartEvents) == 0 {
		result.Issues = append(result.Issues, &dto.BPMNLintIssue{
			Severity: "error", Category: "events",
			Message: prefix + " 缺少开始事件（StartEvent）",
		})
	} else if len(process.StartEvents) > 1 {
		result.Issues = append(result.Issues, &dto.BPMNLintIssue{
			Severity: "warning", Category: "events",
			Message: fmt.Sprintf("%s 存在 %d 个开始事件，引擎按首个执行，其余将被忽略", prefix, len(process.StartEvents)),
		})
	}
	if len(process.EndEvents) == 0 {
		result.Issues = append(result.Issues, &dto.BPMNLintIssue{
			Severity: "error", Category: "events",
			Message: prefix + " 缺少结束事件（EndEvent）",
		})
	}

	// --- 规则组 2：任务配置 ---
	for _, task := range process.UserTasks {
		if strings.EqualFold(strings.TrimSpace(task.TaskPurpose), "rework") {
			continue
		}
		if task.Assignee == "" && task.CandidateUsers == "" && task.CandidateGroups == "" {
			result.Issues = append(result.Issues, &dto.BPMNLintIssue{
				Severity: "warning", Category: "tasks",
				ElementID: task.ID, ElementName: task.Name,
				Message: fmt.Sprintf("用户任务 %s 未配置执行人（assignee/candidateUsers/candidateGroups 均为空）", display(task.Name, task.ID)),
			})
		}
	}

	// --- 规则组 3：连通性（入边/出边统计 + 不可达检测） ---
	adjacency, nodeSet, flowTargets := buildGraph(process)

	// --- 规则组 3.1：审批拒绝策略可执行性 ---
	s.lintApprovalRejectionConfiguration(process, adjacency, result)

	// --- 规则组 3.0：重复序列流 ID（2026-09-07 补）---
	// encoding/xml 解析同名 sequenceFlow 时后者静默覆盖前者，
	// 图会缺边——problem_management_flow_cn / service_request_flow_cn 曾各有一例。
	flowSeen := make(map[string]bool, len(process.SequenceFlows))
	for _, flow := range process.SequenceFlows {
		if flow.ID == "" || flowSeen[flow.ID] {
			continue
		}
		flowSeen[flow.ID] = true
	}
	dupCount := len(process.SequenceFlows) - len(flowSeen)
	if dupCount > 0 {
		result.Issues = append(result.Issues, &dto.BPMNLintIssue{
			Severity: "error", Category: "flows",
			Message: fmt.Sprintf("%s 存在 %d 条重复 id 的序列流（XML 解析时后者覆盖前者，图将缺边）", prefix, dupCount),
		})
	}

	for id, node := range nodeSet {
		inDegree := flowTargets[id]
		outDegree := len(adjacency[id])
		_, isStart := findStart(process, id)
		_, isEnd := findEnd(process, id)

		if !isStart && inDegree == 0 {
			result.Issues = append(result.Issues, &dto.BPMNLintIssue{
				Severity: "warning", Category: "connectivity",
				ElementID: id, ElementName: node,
				Message: fmt.Sprintf("节点 %s 不可达（无入边且非开始事件，引擎将静默跳过）", display(node, id)),
			})
		}
		if !isEnd && outDegree == 0 {
			result.Issues = append(result.Issues, &dto.BPMNLintIssue{
				Severity: "warning", Category: "connectivity",
				ElementID: id, ElementName: node,
				Message: fmt.Sprintf("节点 %s 是死路（无出边且非结束事件，流程将在此静默停滞）", display(node, id)),
			})
		}
	}

	// --- 规则组 4：网关语义 ---
	for _, gw := range process.ExclusiveGateways {
		s.lintGateway(prefix, gw.ID, gw.Name, gw.DefaultFlow, adjacency, result, "排他网关")
	}
	for _, gw := range process.ParallelGateways {
		s.lintGateway(prefix, gw.ID, gw.Name, "", adjacency, result, "并行网关")
	}
	for _, gw := range process.InclusiveGateways {
		s.lintGateway(prefix, gw.ID, gw.Name, gw.DefaultFlow, adjacency, result, "包容网关")
	}

	// --- 规则组 5：序列流条件 ---
	for _, flow := range process.SequenceFlows {
		sourceIsGateway := isGateway(process, flow.SourceRef)
		if sourceIsGateway && flow.ConditionExpression == nil {
			// 排他/包容网关的出边缺条件是风险（有 default 的除外——无法从 XML 结构直接判定的保守提示）
			result.Issues = append(result.Issues, &dto.BPMNLintIssue{
				Severity: "warning", Category: "flows",
				ElementID: flow.ID,
				Message:   fmt.Sprintf("序列流 %s 源自网关 %s 但未配置条件表达式（依赖默认流则忽略）", flow.ID, flow.SourceRef),
			})
		}
	}

	// --- 规则组 6：outgoing 声明 ↔ sequenceFlow 一致性（2026-09-07 新增）---
	// 引擎寻路只按 sequenceFlow.sourceRef。元素声明的 <outgoing> 若指向
	// 一条 sourceRef 不是自己的 flow（悬空），该节点在运行时会 0 出边——
	// 曾致 10/16 内置模板卡死（executeStep 静默 return nil）。
	for elementID, flowIDs := range process.OutgoingDecls {
		_, isEnd := findEnd(process, elementID)
		if isEnd {
			continue // 结束事件的 outgoing 声明本身即非法，但由规则组 3 兜底
		}
		for _, flowID := range flowIDs {
			flow := findFlowByID(process, flowID)
			if flow == nil {
				result.Issues = append(result.Issues, &dto.BPMNLintIssue{
					Severity: "error", Category: "connectivity",
					ElementID: elementID,
					Message:   fmt.Sprintf("节点 %s 声明 outgoing=%s 但该序列流不存在（运行时将无路可走）", elementID, flowID),
				})
				continue
			}
			if flow.SourceRef != elementID {
				result.Issues = append(result.Issues, &dto.BPMNLintIssue{
					Severity: "error", Category: "connectivity",
					ElementID: elementID,
					Message: fmt.Sprintf("节点 %s 声明 outgoing=%s 但该流 sourceRef=%s（连线悬空，运行时该节点 0 出边）",
						elementID, flowID, flow.SourceRef),
				})
			}
		}
	}

	// --- 规则组 7：serviceTask 处理器可达性（2026-09-07 新增）---
	// implementation="##WebService" 等 BPMN 标准标注不是本引擎的 handler ID；
	// 缺少 metaData service_task_type 的 serviceTask 将按 name/ID 寻址，
	// 大概率找不到 handler 而 dead_letter。
	for _, st := range process.ServiceTasks {
		if st.ServiceTaskType == "" && strings.HasPrefix(st.Implementation, "##") {
			result.Issues = append(result.Issues, &dto.BPMNLintIssue{
				Severity: "warning", Category: "tasks",
				ElementID: st.ID, ElementName: st.Name,
				Message: fmt.Sprintf("服务任务 %s 使用 %q 标准标注但未配置 metaData service_task_type，将按名称寻址 handler（找不到即 dead_letter）",
					display(st.Name, st.ID), st.Implementation),
			})
		}
	}
}

func (s *BPMNLintService) lintApprovalRejectionConfiguration(process *BPMNProcess, adjacency map[string][]string, result *dto.BPMNLintResult) {
	reworkTasks := make(map[string]*BPMNUserTask)
	approvalTaskIDs := make(map[string]bool)
	for _, task := range process.UserTasks {
		switch strings.ToLower(strings.TrimSpace(task.TaskPurpose)) {
		case "rework":
			reworkTasks[task.ID] = task
			if hasStaticTaskAssignment(task) {
				result.Issues = append(result.Issues, &dto.BPMNLintIssue{
					Severity: "error", Category: "approval",
					ElementID: task.ID, ElementName: task.Name,
					Message: fmt.Sprintf("返工任务 %s 不能配置固定执行人或候选人，必须由发起人回退规则分配", display(task.Name, task.ID)),
				})
			}
		case "approval":
			approvalTaskIDs[task.ID] = true
		}
	}

	for _, task := range process.UserTasks {
		strategy := strings.ToLower(strings.TrimSpace(task.RejectStrategy))
		if strategy == "" || strategy == "terminate" {
			continue
		}
		if !approvalTaskIDs[task.ID] {
			result.Issues = append(result.Issues, &dto.BPMNLintIssue{
				Severity: "error", Category: "approval",
				ElementID: task.ID, ElementName: task.Name,
				Message: fmt.Sprintf("用户任务 %s 配置了拒绝策略 %q，但 taskPurpose 必须为 approval", display(task.Name, task.ID), task.RejectStrategy),
			})
			continue
		}

		switch strategy {
		case "gateway":
			if !hasApprovalRejectionFlow(process, task.ID) {
				result.Issues = append(result.Issues, &dto.BPMNLintIssue{
					Severity: "error", Category: "approval",
					ElementID: task.ID, ElementName: task.Name,
					Message: fmt.Sprintf("审批任务 %s 的拒绝分支必须有直接出边，并显式判断 approvalAction == 'reject' 或 approvalResult == 'rejected'", display(task.Name, task.ID)),
				})
			}
		case "to_requester":
			reworkTask := findRejectedReworkTask(process, task.ID, reworkTasks)
			if reworkTask == nil {
				result.Issues = append(result.Issues, &dto.BPMNLintIssue{
					Severity: "error", Category: "approval",
					ElementID: task.ID, ElementName: task.Name,
					Message: fmt.Sprintf("审批任务 %s 的退回发起人分支必须直接连接到 taskPurpose=rework 的返工任务，并使用拒绝条件", display(task.Name, task.ID)),
				})
				continue
			}
			if !reworkCanReachApproval(reworkTask.ID, adjacency, approvalTaskIDs) {
				result.Issues = append(result.Issues, &dto.BPMNLintIssue{
					Severity: "error", Category: "approval",
					ElementID: reworkTask.ID, ElementName: reworkTask.Name,
					Message: fmt.Sprintf("返工任务 %s 必须存在回到审批任务的路径", display(reworkTask.Name, reworkTask.ID)),
				})
			}
		default:
			result.Issues = append(result.Issues, &dto.BPMNLintIssue{
				Severity: "error", Category: "approval",
				ElementID: task.ID, ElementName: task.Name,
				Message: fmt.Sprintf("审批任务 %s 使用了不支持的拒绝策略 %q", display(task.Name, task.ID), task.RejectStrategy),
			})
		}
	}
}

func hasStaticTaskAssignment(task *BPMNUserTask) bool {
	return strings.TrimSpace(task.Assignee) != "" ||
		strings.TrimSpace(task.CandidateUsers) != "" ||
		strings.TrimSpace(task.CandidateGroups) != ""
}

func hasApprovalRejectionFlow(process *BPMNProcess, taskID string) bool {
	for _, flow := range process.SequenceFlows {
		if flow.SourceRef == taskID && isApprovalRejectionCondition(flow.ConditionExpression) {
			return true
		}
	}
	return false
}

func findRejectedReworkTask(process *BPMNProcess, taskID string, reworkTasks map[string]*BPMNUserTask) *BPMNUserTask {
	for _, flow := range process.SequenceFlows {
		if flow.SourceRef != taskID || !isApprovalRejectionCondition(flow.ConditionExpression) {
			continue
		}
		if reworkTask := reworkTasks[flow.TargetRef]; reworkTask != nil {
			return reworkTask
		}
	}
	return nil
}

var (
	approvalActionRejectCondition   = regexp.MustCompile(`(?i)(?:variables\s*\[\s*['"]approvalaction['"]\s*\]|approvalaction)\s*==\s*['"]reject['"]`)
	approvalResultRejectedCondition = regexp.MustCompile(`(?i)(?:variables\s*\[\s*['"]approvalresult['"]\s*\]|approvalresult)\s*==\s*['"]rejected['"]`)
)

func isApprovalRejectionCondition(condition *BPMNConditionExpression) bool {
	if condition == nil {
		return false
	}
	return approvalActionRejectCondition.MatchString(condition.Expression) ||
		approvalResultRejectedCondition.MatchString(condition.Expression)
}

func reworkCanReachApproval(reworkTaskID string, adjacency map[string][]string, approvalTaskIDs map[string]bool) bool {
	visited := map[string]bool{reworkTaskID: true}
	queue := []string{reworkTaskID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, next := range adjacency[current] {
			if approvalTaskIDs[next] {
				return true
			}
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}
	return false
}

// findFlowByID 按 id 查找序列流。
func findFlowByID(process *BPMNProcess, flowID string) *BPMNSequenceFlow {
	for _, flow := range process.SequenceFlows {
		if flow.ID == flowID {
			return flow
		}
	}
	return nil
}

// lintGateway 网关出入边检查：分叉需 >=1 出边，汇聚需 >=1 入边，单向网关混用给警告。
func (s *BPMNLintService) lintGateway(prefix, id, name, defaultFlow string, adjacency map[string][]string, result *dto.BPMNLintResult, kind string) {
	outs := len(adjacency[id])
	// 入边数由调用方统计不便，这里用 flowTargets 反查——简化：出边为 0 即死路已由连通性规则覆盖，
	// 此处只做网关特有语义：出边 == 1 的分叉网关给 warning（无分流意义）。
	if outs == 1 {
		result.Issues = append(result.Issues, &dto.BPMNLintIssue{
			Severity: "warning", Category: "gateways",
			ElementID: id, ElementName: name,
			Message: fmt.Sprintf("%s：%s网关 %s 仅 1 条出边，没有分流作用（请确认是否应为汇聚网关或直接删除）", prefix, kind, display(name, id)),
		})
	}
	_ = defaultFlow
}

// buildGraph 构建邻接表与节点集合。
// 返回：邻接表（node -> 出边目标列表）、节点集合（id -> 显示名）、每个节点的入边数。
func buildGraph(process *BPMNProcess) (map[string][]string, map[string]string, map[string]int) {
	adjacency := map[string][]string{}
	nodeSet := map[string]string{}
	flowTargets := map[string]int{}

	register := func(id, name string) {
		if id != "" {
			if _, ok := nodeSet[id]; !ok {
				nodeSet[id] = name
			}
		}
	}

	for _, e := range process.StartEvents {
		register(e.ID, e.Name)
	}
	for _, e := range process.EndEvents {
		register(e.ID, e.Name)
	}
	for _, t := range process.UserTasks {
		register(t.ID, t.Name)
	}
	for _, t := range process.ServiceTasks {
		register(t.ID, t.Name)
	}
	for _, t := range process.ScriptTasks {
		register(t.ID, t.Name)
	}
	for _, t := range process.BusinessRuleTasks {
		register(t.ID, t.Name)
	}
	for _, t := range process.ManualTasks {
		register(t.ID, t.Name)
	}
	for _, c := range process.CallActivities {
		register(c.ID, c.Name)
	}
	for _, g := range process.ExclusiveGateways {
		register(g.ID, g.Name)
	}
	for _, g := range process.ParallelGateways {
		register(g.ID, g.Name)
	}
	for _, g := range process.InclusiveGateways {
		register(g.ID, g.Name)
	}
	for _, sp := range process.SubProcesses {
		register(sp.ID, sp.Name)
	}

	for _, flow := range process.SequenceFlows {
		if flow.SourceRef == "" || flow.TargetRef == "" {
			continue
		}
		adjacency[flow.SourceRef] = append(adjacency[flow.SourceRef], flow.TargetRef)
		flowTargets[flow.TargetRef]++
		// 边端点也注册进节点集（防 LLM 生成引用了未声明节点）
		register(flow.SourceRef, flow.SourceRef)
		register(flow.TargetRef, flow.TargetRef)
	}
	return adjacency, nodeSet, flowTargets
}

func findStart(process *BPMNProcess, id string) (*BPMNStartEvent, bool) {
	for _, e := range process.StartEvents {
		if e.ID == id {
			return e, true
		}
	}
	return nil, false
}

func findEnd(process *BPMNProcess, id string) (*BPMNEndEvent, bool) {
	for _, e := range process.EndEvents {
		if e.ID == id {
			return e, true
		}
	}
	return nil, false
}

func isGateway(process *BPMNProcess, id string) bool {
	for _, g := range process.ExclusiveGateways {
		if g.ID == id {
			return true
		}
	}
	for _, g := range process.ParallelGateways {
		if g.ID == id {
			return true
		}
	}
	for _, g := range process.InclusiveGateways {
		if g.ID == id {
			return true
		}
	}
	return false
}

func display(name, id string) string {
	if strings.TrimSpace(name) != "" {
		return fmt.Sprintf("「%s」", name)
	}
	return id
}

func countSeverity(issues []*dto.BPMNLintIssue, severity string) int {
	n := 0
	for _, i := range issues {
		if i.Severity == severity {
			n++
		}
	}
	return n
}
