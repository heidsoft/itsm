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

// actionRequiringServiceTaskTypes 是「Execute 只从 variables["action"] 取动作」的
// handler 集合，与 service/bpmn 下六个 handler 的 switch 一一对应：
// incident_handler / ticket_handler / change_handler / service_request_handler /
// notification_handler / webhook_handler。
//
// 不在此列的：generic_task（读 operation，恒成功）、approval_task、cc_task、
// approval_chain_task（走各自的任务语义，不看 action）。
// 新增依赖 action 的 handler 时必须同步加到这里，否则 lint 会放过漏配的模板。
var actionRequiringServiceTaskTypes = map[string]bool{
	"incident_task":        true,
	"ticket_task":          true,
	"change_task":          true,
	"service_request_task": true,
	"notification_task":    true,
	"webhook_task":         true,
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

	// --- 规则组 1.1：定时事件表达式---
	// TimerEventDefinition 存在但三种时间表达式全空属于非法定义：
	// extractor 无法计算 fire_at，运行时将无法注册/触发定时器。
	checkTimerDefinition := func(kind, elementID, elementName string, def *BPMNTimerEventDefinition) {
		if def == nil {
			return
		}
		if strings.TrimSpace(def.TimeDuration) == "" && strings.TrimSpace(def.TimeDate) == "" && strings.TrimSpace(def.TimeCycle) == "" {
			result.Issues = append(result.Issues, &dto.BPMNLintIssue{
				Severity: "error", Category: "events",
				ElementID: elementID, ElementName: elementName,
				Message: fmt.Sprintf("%s %s 的定时事件缺少时间表达式（timeDuration/timeDate/timeCycle 至少配置一项）", kind, display(elementName, elementID)),
			})
		}
	}
	for _, ev := range process.StartEvents {
		checkTimerDefinition("开始事件", ev.ID, ev.Name, ev.TimerEventDefinition)
	}
	for _, ev := range process.IntermediateEvents {
		checkTimerDefinition("中间事件", ev.ID, ev.Name, ev.TimerEventDefinition)
	}
	for _, ev := range process.BoundaryEvents {
		checkTimerDefinition("边界事件", ev.ID, ev.Name, ev.TimerEventDefinition)
		if ev.TimerEventDefinition != nil && strings.TrimSpace(ev.AttachedToRef) == "" {
			result.Issues = append(result.Issues, &dto.BPMNLintIssue{
				Severity: "error", Category: "events",
				ElementID: ev.ID, ElementName: ev.Name,
				Message: fmt.Sprintf("边界事件 %s 缺少 attachedToRef（未声明附加到哪个活动）", display(ev.Name, ev.ID)),
			})
		}
	}

	// --- 规则组 2：任务配置 ---
	for _, task := range process.UserTasks {
		if strings.EqualFold(strings.TrimSpace(task.TaskPurpose), "rework") {
			continue
		}
		if strings.TrimSpace(task.TaskPurpose) == "" {
			result.Issues = append(result.Issues, &dto.BPMNLintIssue{
				Severity: "warning", Category: "tasks",
				ElementID: task.ID, ElementName: task.Name,
				Message: fmt.Sprintf("用户任务 %s 未声明 taskPurpose，引擎将按 work 处理；审批节点必须显式声明 approval", display(task.Name, task.ID)),
			})
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

	// --- 规则组 3.0：重复序列流 ID---
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
		// 包容网关分叉时若所有出边条件都不命中，引擎只打 Warn 日志然后静默
		// 等待（handleInclusiveGateway !matched 分支）——实例停在网关上无任何
		// 调度来源。声明 default 分支是 BPMN 语义的标准兜底，缺省时提示。
		if strings.TrimSpace(gw.DefaultFlow) == "" && len(adjacency[gw.ID]) > 1 {
			result.Issues = append(result.Issues, &dto.BPMNLintIssue{
				Severity: "warning", Category: "gateways",
				ElementID: gw.ID, ElementName: gw.Name,
				Message: fmt.Sprintf("%s：包容网关 %s 未声明 default 分支，运行时所有出边条件都不命中时实例将静默挂起（无定时器、无信号可唤醒）",
					prefix, display(gw.Name, gw.ID)),
			})
		}
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

	// --- 规则组 6：outgoing 声明 ↔ sequenceFlow 一致性---
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

	// --- 规则组 7：serviceTask 处理器可达性---
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

		// 依赖 variables["action"] 的 handler：节点没声明 action 就必然落 default
		// 分支、重试 8 次后进 dead_letter。2026-10-24 生产栈实测 36/36 全死信
		// 就是这个原因（模板声明了 action，但解析器/合并器两端都没接上）。
		// 现在解析与注入都修好了，这条 lint 把「模板漏写 action」挡在部署期。
		if actionRequiringServiceTaskTypes[st.ServiceTaskType] && strings.TrimSpace(st.Action) == "" {
			result.Issues = append(result.Issues, &dto.BPMNLintIssue{
				Severity: "error", Category: "tasks",
				ElementID: st.ID, ElementName: st.Name,
				Message: fmt.Sprintf("服务任务 %s 的 handler=%s 只认 variables[\"action\"]，但节点没有声明 action："+
					"请加 <metaData name=\"action\">动作名</metaData>（或 camunda:inputParameter），否则运行时必然死信",
					display(st.Name, st.ID), st.ServiceTaskType),
			})
		}
	}

	// --- 规则组 8：审批节点能力可执行性---
	s.lintApprovalNodeCapabilities(process, result)

	// --- 规则组 9：引擎不支持的元素类型拦截 ---
	// 子流程/调用活动在解析器里有类型（bpmncontracts.BPMNSubProcess /
	// BPMNCallActivity），但 handleElement 没有对应分支——到达时落入兜底
	// 路径（markElementDone + executeStep），**整个子流程内容被静默跳过**，
	// 用户得到「看似跑通实则跳过」的实例。必须在部署期拒绝，而不是让
	// 运行时静默丢活动。支持之前请展平为顶层元素。
	if len(process.SubProcesses) > 0 {
		for _, sp := range process.SubProcesses {
			result.Issues = append(result.Issues, &dto.BPMNLintIssue{
				Severity: "error", Category: "structure",
				ElementID: sp.ID, ElementName: sp.Name,
				Message: fmt.Sprintf("子流程 %s 暂不被引擎执行（将被静默跳过）：请展平为顶层元素后再部署",
					display(sp.Name, sp.ID)),
			})
		}
	}
	if len(process.CallActivities) > 0 {
		for _, ca := range process.CallActivities {
			result.Issues = append(result.Issues, &dto.BPMNLintIssue{
				Severity: "error", Category: "structure",
				ElementID: ca.ID, ElementName: ca.Name,
				Message: fmt.Sprintf("调用活动 %s 暂不被引擎执行（将被静默跳过）：请展平为顶层元素后再部署",
					display(ca.Name, ca.ID)),
			})
		}
	}

	// --- 规则组 10：令牌/汇聚语义（审计 P1-06 的部署期缓解） ---
	s.lintTokenSemantics(process, adjacency, flowTargets, result)
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

// lintApprovalNodeCapabilities 校验审批节点的能力配置是否在运行时可执行。
//
// 规则：
//   - allowDelegate：仅对 taskPurpose=approval 有意义，非审批节点开启给 warning
//   - approvalThreshold：显式阈值大于候选人数时不可能满足，给 error
func (s *BPMNLintService) lintApprovalNodeCapabilities(process *BPMNProcess, result *dto.BPMNLintResult) {
	for _, task := range process.UserTasks {
		isApproval := strings.EqualFold(strings.TrimSpace(task.TaskPurpose), "approval")
		label := display(task.Name, task.ID)

		// allowDelegate 仅对审批节点有意义
		if task.AllowDelegate && !isApproval {
			result.Issues = append(result.Issues, &dto.BPMNLintIssue{
				Severity: "warning", Category: "approval",
				ElementID: task.ID, ElementName: task.Name,
				Message: fmt.Sprintf("用户任务 %s 开启了委托（allowDelegate），但 taskPurpose 不是 approval，委托能力仅在审批节点生效", label),
			})
		}

		// 显式 approvalThreshold 大于候选人数
		if isApproval && task.ApprovalThreshold > 0 {
			candidateCount := countCandidates(task)
			if candidateCount > 0 && task.ApprovalThreshold > candidateCount {
				result.Issues = append(result.Issues, &dto.BPMNLintIssue{
					Severity: "error", Category: "approval",
					ElementID: task.ID, ElementName: task.Name,
					Message: fmt.Sprintf("审批任务 %s 的通过阈值 %d 大于候选审批人数 %d，审批永远无法满足", label, task.ApprovalThreshold, candidateCount),
				})
			}
		}
	}
}

// countCandidates 统计任务的候选审批人数。
// assignee 算 1 人；candidateUsers/candidateGroups 按逗号分隔计数。
// candidateGroups 按 1 人计（实际人数需查运行时，此处保守估计）。
func countCandidates(task *BPMNUserTask) int {
	n := 0
	if strings.TrimSpace(task.Assignee) != "" {
		n++
	}
	n += len(splitNonEmptyCSV(task.CandidateUsers))
	n += len(splitNonEmptyCSV(task.CandidateGroups))
	return n
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

// forkRef 记录一个分叉网关的类型与展示名，供 R2 的错误消息引用。
type forkRef struct {
	kind string // "并行" / "包容"
	name string
}

// lintTokenSemantics 拒绝引擎令牌语义无法正确执行的图形（审计 P1-06 的部署期缓解）。
//
// 引擎的汇聚与结束都建立在「节点完成过」而非令牌/occurrence 上：
//   - 汇聚（并行/包容）对每个入边源要么读 _done_ 这个永久 bool，要么查用户任务
//     是否还存在未结束任务——没有「该分支本轮是否被激活」的概念；
//   - 任意 endEvent 到达即把整个实例置 completed。
//
// 真修复是持久化 token/occurrence；在那之前这里在部署期拒绝两类必然被错误
// 执行的图形（规则分级：error = 部署后引擎无法正确执行）：
//
//	R1 收敛型包容网关（入度 ≥2）：包容分叉只激活命中条件的分支，而包容汇聚
//	   等待全部静态入边，未激活的分支永远不会「完成」→ 实例永久停在网关上。
//	R2 并行/包容分叉存在「不经汇聚直达 endEvent」的路径：先到的分支把整个
//	   实例置 completed，另一分支的任务与后续推进被丢弃。
//
// 已知不完备（如实声明，不作完备性承诺）：
//   - barrier 只认收敛型并行/包容网关，嵌套分叉/汇聚与排他汇聚交错的形状可能漏过；
//   - 循环再次经过汇聚时上一轮的 _done_ 仍会提前满足本轮（occurrence 缺口），本规则不查；
//   - 排他分叉 → 并行汇聚这类「只激活一半入边」的死锁形状不在 R1/R2 覆盖内。
//
// 已核对：内置模板与 seeder 模板均无包容汇聚；并行分叉模板一律先汇聚再进 endEvent，
// 故两条规则不打破任何已发布模板（由 TestBuiltinTemplates_Gate 锁定）。
func (s *BPMNLintService) lintTokenSemantics(process *BPMNProcess, adjacency map[string][]string, flowTargets map[string]int, result *dto.BPMNLintResult) {
	prefix := fmt.Sprintf("流程 %s", process.ID)

	// R1：收敛型包容网关必然死锁（引擎的包容汇聚 = 等全部静态入边）。
	for _, gw := range process.InclusiveGateways {
		if flowTargets[gw.ID] < 2 {
			continue // 纯分叉/直通（入度 <2）不受影响
		}
		result.Issues = append(result.Issues, &dto.BPMNLintIssue{
			Severity: "error", Category: "gateways",
			ElementID: gw.ID, ElementName: gw.Name,
			Message: fmt.Sprintf("%s：包容网关 %s 是汇聚节点（%d 条入边），引擎的汇聚按「全部静态入边是否完成」判断，"+
				"而包容分叉只激活命中条件的分支——未激活分支永远不会完成，实例将永久停在该网关上。"+
				"请改用成对的并行分叉/汇聚，或把分支展平为排他路径",
				prefix, display(gw.Name, gw.ID), flowTargets[gw.ID]),
		})
	}

	// R2：分叉后存在不经汇聚直达 endEvent 的路径 → 提前完成整个实例。
	forks := map[string]forkRef{}
	for _, g := range process.ParallelGateways {
		if len(adjacency[g.ID]) >= 2 {
			forks[g.ID] = forkRef{kind: "并行", name: g.Name}
		}
	}
	for _, g := range process.InclusiveGateways {
		if len(adjacency[g.ID]) >= 2 {
			forks[g.ID] = forkRef{kind: "包容", name: g.Name}
		}
	}
	if len(forks) == 0 {
		return // 没有并行/包容分叉：排他分支每次只激活一条路径，多 endEvent 是安全的
	}

	// barrier = 收敛型并行/包容网关（入度 ≥2）：只有它们真的会等其他分支。
	// 排他汇聚不等待（到达即继续往下走），不能当 barrier，否则会漏报提前完成。
	barrier := map[string]bool{}
	for _, g := range process.ParallelGateways {
		if flowTargets[g.ID] >= 2 {
			barrier[g.ID] = true
		}
	}
	for _, g := range process.InclusiveGateways {
		if flowTargets[g.ID] >= 2 {
			barrier[g.ID] = true
		}
	}
	endIDs := map[string]bool{}
	for _, e := range process.EndEvents {
		endIDs[e.ID] = true
	}

	for forkID, fork := range forks {
		visited := make(map[string]bool, len(adjacency))
		queue := make([]string, 0, len(adjacency[forkID]))
		for _, next := range adjacency[forkID] {
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
		earlyEnd := ""
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			if barrier[current] {
				continue // 该分支在汇聚处被等待，其下游由汇聚自身的入边规则负责
			}
			if endIDs[current] {
				earlyEnd = current
				break
			}
			for _, next := range adjacency[current] {
				if !visited[next] {
					visited[next] = true
					queue = append(queue, next)
				}
			}
		}
		if earlyEnd == "" {
			continue
		}
		result.Issues = append(result.Issues, &dto.BPMNLintIssue{
			Severity: "error", Category: "gateways",
			ElementID: forkID, ElementName: fork.name,
			Message: fmt.Sprintf("%s：%s网关 %s 分叉后存在不经任何汇聚网关就直达结束事件 %s 的路径："+
				"引擎一到达 endEvent 就把整个实例置为 completed，另一分支的任务与后续推进会被丢弃。"+
				"请让所有分支先在并行/包容汇聚网关上会合，再进入结束事件",
				prefix, fork.kind, display(fork.name, forkID), earlyEnd),
		})
	}
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
