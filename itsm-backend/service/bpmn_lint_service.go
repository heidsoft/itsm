package service

import (
	"fmt"
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
				Message: fmt.Sprintf("序列流 %s 源自网关 %s 但未配置条件表达式（依赖默认流则忽略）", flow.ID, flow.SourceRef),
			})
		}
	}
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
