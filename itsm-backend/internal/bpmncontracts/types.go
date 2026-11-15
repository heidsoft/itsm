package bpmncontracts

import "encoding/xml"

// BPMNElement BPMN元素的基础接口
type BPMNElement interface {
	GetID() string
	GetName() string
	GetType() string
}

// BPMNProcess BPMN流程定义
type BPMNProcess struct {
	ID                 string                   `xml:"id,attr"`
	Name               string                   `xml:"name,attr"`
	ProcessType        string                   `xml:"processType,attr"`
	IsExecutable       bool                     `xml:"isExecutable,attr"`
	IsClosed           bool                     `xml:"isClosed,attr"`
	StartEvents        []*BPMNStartEvent        `xml:"startEvent"`
	EndEvents          []*BPMNEndEvent          `xml:"endEvent"`
	UserTasks          []*BPMNUserTask          `xml:"userTask"`
	ServiceTasks       []*BPMNServiceTask       `xml:"serviceTask"`
	ScriptTasks        []*BPMNScriptTask        `xml:"scriptTask"`
	BusinessRuleTasks  []*BPMNBusinessRuleTask  `xml:"businessRuleTask"`
	ManualTasks        []*BPMNManualTask        `xml:"manualTask"`
	CallActivities     []*BPMNCallActivity      `xml:"callActivity"`
	ExclusiveGateways  []*BPMNExclusiveGateway  `xml:"exclusiveGateway"`
	ParallelGateways   []*BPMNParallelGateway   `xml:"parallelGateway"`
	InclusiveGateways  []*BPMNInclusiveGateway  `xml:"inclusiveGateway"`
	SequenceFlows      []*BPMNSequenceFlow      `xml:"sequenceFlow"`
	SubProcesses       []*BPMNSubProcess        `xml:"subProcess"`
	BoundaryEvents     []*BPMNBoundaryEvent     `xml:"boundaryEvent"`
	IntermediateEvents []*BPMNIntermediateEvent `xml:"intermediateCatchEvent"`
	DataObjects        []*BPMNDataObject        `xml:"dataObject"`
	DataStores         []*BPMNDataStore         `xml:"dataStore"`

	// OutgoingDecls 元素级 outgoing 声明索引（elementID -> flow ids），
	// 由 parser 后处理填充（encoding/xml 无法在结构体层面表达
	// 「outgoing 挂在哪个子元素下」，见 BPMNParser.collectOutgoingDecls）。
	// 用途：① lint 校验声明与 sequenceFlow.sourceRef 一致性；
	// ② 引擎 0 出边时的 fallback 寻路（防静默卡死）。
	OutgoingDecls map[string][]string `xml:"-"`
}

// BPMNOutgoingRef 单条 outgoing 声明（收集阶段中间形态）。
type BPMNOutgoingRef struct {
	OwnerID string
	FlowID  string
}

// BPMNStartEvent 开始事件
type BPMNStartEvent struct {
	ID                   string                    `xml:"id,attr"`
	Name                 string                    `xml:"name,attr"`
	EventType            string                    `xml:"eventType,attr"`
	MessageRef           string                    `xml:"messageRef,attr"`
	TimerRef             string                    `xml:"timerRef,attr"`
	SignalRef            string                    `xml:"signalRef,attr"`
	TimerEventDefinition *BPMNTimerEventDefinition `xml:"timerEventDefinition"`
}

// GetID 获取ID
func (e *BPMNStartEvent) GetID() string { return e.ID }

// GetName 获取名称
func (e *BPMNStartEvent) GetName() string { return e.Name }

// GetType 获取类型
func (e *BPMNStartEvent) GetType() string { return "StartEvent" }

// BPMNEndEvent 结束事件
type BPMNEndEvent struct {
	ID         string `xml:"id,attr"`
	Name       string `xml:"name,attr"`
	EventType  string `xml:"eventType,attr"`
	MessageRef string `xml:"messageRef,attr"`
	SignalRef  string `xml:"signalRef,attr"`
	ErrorRef   string `xml:"errorRef,attr"`
	// CompensateEventDefinition 补偿结束事件：到达时对已登记的补偿范围
	// 执行逆序补偿，再走正常结束路径。activityRef 非空时只补偿指定活动。
	CompensateEventDefinition *BPMNCompensateEventDefinition `xml:"compensateEventDefinition"`
}

// BPMNCompensateEventDefinition 补偿事件定义。
// ActivityRef 语义（BPMN 2.0）：空 = 补偿当前范围内所有已登记活动；
// 非空 = 只补偿指定的活动（必须已登记，否则 fail-closed）。
type BPMNCompensateEventDefinition struct {
	ActivityRef string `xml:"activityRef,attr"`
}

// GetID 获取ID
func (e *BPMNEndEvent) GetID() string { return e.ID }

// GetName 获取名称
func (e *BPMNEndEvent) GetName() string { return e.Name }

// GetType 获取类型
func (e *BPMNEndEvent) GetType() string { return "EndEvent" }

// BPMNTimerEventDefinition 定时器事件定义
type BPMNTimerEventDefinition struct {
	TimeDuration string `xml:"timeDuration"`
	TimeDate     string `xml:"timeDate"`
	TimeCycle    string `xml:"timeCycle"`
}

// BPMNUserTask 用户任务
type BPMNUserTask struct {
	ID                string `xml:"id,attr"`
	Name              string `xml:"name,attr"`
	Assignee          string `xml:"assignee,attr"`
	CandidateUsers    string `xml:"candidateUsers,attr"`
	CandidateGroups   string `xml:"candidateGroups,attr"`
	Priority          int    `xml:"priority,attr"`
	FormKey           string `xml:"formKey,attr"`
	DueDate           string `xml:"dueDate,attr"`
	TaskPurpose       string `xml:"taskPurpose,attr"`
	ApprovalMode      string `xml:"approvalMode,attr"`
	ApprovalThreshold int    `xml:"approvalThreshold,attr"`
	// ApprovalLevel 声明该审批节点对应审批链的第几级，审批人从流程变量
	// approval_plan 按层级解析；0 表示不参与变量驱动（沿用静态分配）。
	ApprovalLevel           int    `xml:"approvalLevel,attr"`
	RejectStrategy          string `xml:"rejectStrategy,attr"`
	TimeoutAction           string `xml:"timeoutAction,attr"`
	AllowDelegate           bool   `xml:"allowDelegate,attr"`
	AllowAddApprover        bool   `xml:"allowAddApprover,attr"`
	CommentRequiredOnReject bool   `xml:"commentRequiredOnReject,attr"`
}

// GetID 获取ID
func (e *BPMNUserTask) GetID() string { return e.ID }

// GetName 获取名称
func (e *BPMNUserTask) GetName() string { return e.Name }

// GetType 获取类型
func (e *BPMNUserTask) GetType() string { return "UserTask" }

// BPMNServiceTask 服务任务
type BPMNServiceTask struct {
	ID                 string `xml:"id,attr"`
	Name               string `xml:"name,attr"`
	Type               string `xml:"type,attr"`
	OperationRef       string `xml:"operationRef,attr"`
	Implementation     string `xml:"implementation,attr"`
	Class              string `xml:"class,attr"`
	Expression         string `xml:"expression,attr"`
	DelegateExpression string `xml:"delegateExpression,attr"`
	CCType             string `xml:"ccType,attr"`
	CCUserIDs          string `xml:"ccUserIds,attr"`
	CCGroupIDs         string `xml:"ccGroupIds,attr"`
	CCRoleIDs          string `xml:"ccRoleIds,attr"`
	CCVariable         string `xml:"ccVariable,attr"`
	CCNotify           string `xml:"ccNotify,attr"`
	NotifyChannels     string `xml:"notifyChannels,attr"`

	// ServiceTaskType 显式处理器类型，来自扩展元素
	// <metaData name="service_task_type">xxx</metaData>。
	// 非空时优先于 implementation/class 等标准属性参与 handler 匹配
	// （见 serviceTaskReference），使「##WebService」等 BPMN 标准标注
	// 不再劫持内部 handler 寻址。
	ServiceTaskType string `xml:"-"`
	// Action 该节点要执行的具体动作，来自扩展元素
	// <metaData name="action">xxx</metaData>。
	//
	// 六个 ServiceTaskHandler（incident/ticket/change/service_request/
	// notification/webhook）的 Execute 都只认 variables["action"]，
	// 因此解析后必须由 mergeServiceTaskVariables 注入 variables。
	// 2026-10-24 生产栈实测：本字段缺失导致所有 serviceTask 取到空串、
	// 36/36 全部重试 8 次后进 dead_letter（0 成功）。
	Action string `xml:"-"`
	// IsForCompensation 补偿专用活动标记（isForCompensation="true"）：
	// 主流程到达时跳过执行，只作为补偿处理器被补偿触发器调用。
	IsForCompensation bool `xml:"isForCompensation,attr"`
}

// GetID 获取ID
func (e *BPMNServiceTask) GetID() string { return e.ID }

// GetName 获取名称
func (e *BPMNServiceTask) GetName() string { return e.Name }

// GetType 获取类型
func (e *BPMNServiceTask) GetType() string { return "ServiceTask" }

// BPMNScriptTask 脚本任务
type BPMNScriptTask struct {
	ID           string `xml:"id,attr"`
	Name         string `xml:"name,attr"`
	ScriptFormat string `xml:"scriptFormat,attr"`
	Script       string `xml:"script"`
}

// GetID 获取ID
func (e *BPMNScriptTask) GetID() string { return e.ID }

// GetName 获取名称
func (e *BPMNScriptTask) GetName() string { return e.Name }

// GetType 获取类型
func (e *BPMNScriptTask) GetType() string { return "ScriptTask" }

// BPMNExclusiveGateway 排他网关
type BPMNExclusiveGateway struct {
	ID          string `xml:"id,attr"`
	Name        string `xml:"name,attr"`
	DefaultFlow string `xml:"default,attr"`
}

// GetID 获取ID
func (e *BPMNExclusiveGateway) GetID() string { return e.ID }

// GetName 获取名称
func (e *BPMNExclusiveGateway) GetName() string { return e.Name }

// GetType 获取类型
func (e *BPMNExclusiveGateway) GetType() string { return "ExclusiveGateway" }

// BPMNParallelGateway 并行网关
type BPMNParallelGateway struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

// GetID 获取ID
func (e *BPMNParallelGateway) GetID() string { return e.ID }

// GetName 获取名称
func (e *BPMNParallelGateway) GetName() string { return e.Name }

// GetType 获取类型
func (e *BPMNParallelGateway) GetType() string { return "ParallelGateway" }

// BPMNInclusiveGateway 包容网关
type BPMNInclusiveGateway struct {
	ID          string `xml:"id,attr"`
	Name        string `xml:"name,attr"`
	DefaultFlow string `xml:"default,attr"`
}

// GetID 获取ID
func (e *BPMNInclusiveGateway) GetID() string { return e.ID }

// GetName 获取名称
func (e *BPMNInclusiveGateway) GetName() string { return e.Name }

// GetType 获取类型
func (e *BPMNInclusiveGateway) GetType() string { return "InclusiveGateway" }

// BPMNSequenceFlow 顺序流
type BPMNSequenceFlow struct {
	ID                  string                   `xml:"id,attr"`
	Name                string                   `xml:"name,attr"`
	SourceRef           string                   `xml:"sourceRef,attr"`
	TargetRef           string                   `xml:"targetRef,attr"`
	ConditionExpression *BPMNConditionExpression `xml:"conditionExpression"`
}

// GetID 获取ID
func (e *BPMNSequenceFlow) GetID() string { return e.ID }

// GetName 获取名称
func (e *BPMNSequenceFlow) GetName() string { return e.Name }

// GetType 获取类型
func (e *BPMNSequenceFlow) GetType() string { return "SequenceFlow" }

// BPMNConditionExpression 条件表达式
type BPMNConditionExpression struct {
	Type       string `xml:"type,attr"`
	Expression string `xml:",chardata"`
}

// BPMNSubProcess 子流程
type BPMNSubProcess struct {
	ID               string `xml:"id,attr"`
	Name             string `xml:"name,attr"`
	TriggeredByEvent bool   `xml:"triggeredByEvent,attr"`
}

// GetID 获取ID
func (e *BPMNSubProcess) GetID() string { return e.ID }

// GetName 获取名称
func (e *BPMNSubProcess) GetName() string { return e.Name }

// GetType 获取类型
func (e *BPMNSubProcess) GetType() string { return "SubProcess" }

// BPMNBusinessRuleTask 业务规则任务
type BPMNBusinessRuleTask struct {
	ID                 string `xml:"id,attr"`
	Name               string `xml:"name,attr"`
	Implementation     string `xml:"implementation,attr"`
	Class              string `xml:"class,attr"`
	Expression         string `xml:"expression,attr"`
	DelegateExpression string `xml:"delegateExpression,attr"`
	DecisionTaskRef    string `xml:"decisionTaskRef,attr"`
}

// GetID 获取ID
func (e *BPMNBusinessRuleTask) GetID() string { return e.ID }

// GetName 获取名称
func (e *BPMNBusinessRuleTask) GetName() string { return e.Name }

// GetType 获取类型
func (e *BPMNBusinessRuleTask) GetType() string { return "BusinessRuleTask" }

// BPMNManualTask 手工任务
type BPMNManualTask struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

// GetID 获取ID
func (e *BPMNManualTask) GetID() string { return e.ID }

// GetName 获取名称
func (e *BPMNManualTask) GetName() string { return e.Name }

// GetType 获取类型
func (e *BPMNManualTask) GetType() string { return "ManualTask" }

// BPMNCallActivity 调用活动
type BPMNCallActivity struct {
	ID                string `xml:"id,attr"`
	Name              string `xml:"name,attr"`
	CalledElement     string `xml:"calledElement,attr"`
	CalledElementType string `xml:"calledElementType,attr"`
	InheritVariables  bool   `xml:"inheritVariables,attr"`
}

// GetID 获取ID
func (e *BPMNCallActivity) GetID() string { return e.ID }

// GetName 获取名称
func (e *BPMNCallActivity) GetName() string { return e.Name }

// GetType 获取类型
func (e *BPMNCallActivity) GetType() string { return "CallActivity" }

// BPMNDataObject 数据对象
type BPMNDataObject struct {
	ID           string `xml:"id,attr"`
	Name         string `xml:"name,attr"`
	DataState    string `xml:"dataState,attr"`
	IsCollection bool   `xml:"isCollection,attr"`
}

// GetID 获取ID
func (e *BPMNDataObject) GetID() string { return e.ID }

// GetName 获取名称
func (e *BPMNDataObject) GetName() string { return e.Name }

// GetType 获取类型
func (e *BPMNDataObject) GetType() string { return "DataObject" }

// BPMNDataStore 数据存储
type BPMNDataStore struct {
	ID          string `xml:"id,attr"`
	Name        string `xml:"name,attr"`
	DataState   string `xml:"dataState,attr"`
	Capacity    string `xml:"capacity,attr"`
	IsUnlimited bool   `xml:"isUnlimited,attr"`
}

// GetID 获取ID
func (e *BPMNDataStore) GetID() string { return e.ID }

// GetName 获取名称
func (e *BPMNDataStore) GetName() string { return e.Name }

// GetType 获取类型
func (e *BPMNDataStore) GetType() string { return "DataStore" }

// BPMNBoundaryEvent 边界事件
type BPMNBoundaryEvent struct {
	ID                   string                    `xml:"id,attr"`
	Name                 string                    `xml:"name,attr"`
	AttachedToRef        string                    `xml:"attachedToRef,attr"`
	CancelActivity       bool                      `xml:"cancelActivity,attr"`
	TimerEventDefinition *BPMNTimerEventDefinition `xml:"timerEventDefinition"`
	// CompensateEventDefinition 补偿边界事件：挂在活动上的补偿触发器。
	CompensateEventDefinition *BPMNCompensateEventDefinition `xml:"compensateEventDefinition"`
}

// GetID 获取ID
func (e *BPMNBoundaryEvent) GetID() string { return e.ID }

// GetName 获取名称
func (e *BPMNBoundaryEvent) GetName() string { return e.Name }

// GetType 获取类型
func (e *BPMNBoundaryEvent) GetType() string { return "BoundaryEvent" }

// BPMNIntermediateEvent 中间事件
type BPMNIntermediateEvent struct {
	ID                   string                    `xml:"id,attr"`
	Name                 string                    `xml:"name,attr"`
	EventType            string                    `xml:"eventType,attr"`
	MessageRef           string                    `xml:"messageRef,attr"`
	TimerRef             string                    `xml:"timerRef,attr"`
	SignalRef            string                    `xml:"signalRef,attr"`
	TimerEventDefinition *BPMNTimerEventDefinition `xml:"timerEventDefinition"`
}

// GetID 获取ID
func (e *BPMNIntermediateEvent) GetID() string { return e.ID }

// GetName 获取名称
func (e *BPMNIntermediateEvent) GetName() string { return e.Name }

// GetType 获取类型
func (e *BPMNIntermediateEvent) GetType() string { return "IntermediateEvent" }

// BPMNDefinitions BPMN定义根元素
type BPMNDefinitions struct {
	XMLName         xml.Name       `xml:"definitions"`
	ID              string         `xml:"id,attr"`
	Name            string         `xml:"name,attr"`
	TargetNamespace string         `xml:"targetNamespace,attr"`
	Processes       []*BPMNProcess `xml:"process"`
}

// FindElementName 按元素 ID 解析节点的展示名（BPMN name 属性）；
// 找不到该元素、或元素本身没有 name 时返回空串，由调用方决定回退口径。
//
// 为什么需要它：引擎写 ProcessInstance.CurrentActivityName 时曾经只认 userTask
// 与 endEvent，其余节点（serviceTask / 网关 / 开始事件 / 中间事件 / 子流程 / callActivity）
// 一律回退成原始元素 ID。prod 实测 incident_emergency_flow 的当前节点是
// <bpmn:serviceTask id="Activity_AutoAssign" name="自动分配">，XML 里有中文名，
// 但用户看到的是 "Activity_AutoAssign" —— 前端实例详情、工单详情、监控看板全都显示
// 这个英文 ID，等于让非技术用户去读 XML 标识符。
//
// 用 BPMNElement 接口做统一遍历，而不是逐个类型穷举：新增节点类型时只要实现了
// GetID/GetName 就自动被覆盖，不会再出现"新类型又漏了"的同类回归。
func (p *BPMNProcess) FindElementName(id string) string {
	if p == nil || id == "" {
		return ""
	}
	if name := findNameInElements(p.flowElements(), id); name != "" {
		return name
	}
	return ""
}

// flowElements 汇总本 Process 内所有可实现命名的节点，按 BPMNElement 接口暴露。
//
// 只收集「会成为 CurrentActivity」（即引擎 handleElement 可能落到的节点）：
// DataObject / DataStore / SequenceFlow 不参与流转，不进这个集合。
func (p *BPMNProcess) flowElements() []BPMNElement {
	if p == nil {
		return nil
	}
	// 预分配容量，避免每次调用多次扩容（典型流程 10~40 个节点）
	capacity := len(p.StartEvents) + len(p.EndEvents) + len(p.UserTasks) +
		len(p.ServiceTasks) + len(p.ScriptTasks) + len(p.BusinessRuleTasks) +
		len(p.ManualTasks) + len(p.CallActivities) + len(p.ExclusiveGateways) +
		len(p.ParallelGateways) + len(p.InclusiveGateways) + len(p.SubProcesses) +
		len(p.BoundaryEvents) + len(p.IntermediateEvents)
	elements := make([]BPMNElement, 0, capacity)

	for _, e := range p.StartEvents {
		if e != nil {
			elements = append(elements, e)
		}
	}
	for _, e := range p.EndEvents {
		if e != nil {
			elements = append(elements, e)
		}
	}
	for _, e := range p.UserTasks {
		if e != nil {
			elements = append(elements, e)
		}
	}
	for _, e := range p.ServiceTasks {
		if e != nil {
			elements = append(elements, e)
		}
	}
	for _, e := range p.ScriptTasks {
		if e != nil {
			elements = append(elements, e)
		}
	}
	for _, e := range p.BusinessRuleTasks {
		if e != nil {
			elements = append(elements, e)
		}
	}
	for _, e := range p.ManualTasks {
		if e != nil {
			elements = append(elements, e)
		}
	}
	for _, e := range p.CallActivities {
		if e != nil {
			elements = append(elements, e)
		}
	}
	for _, e := range p.ExclusiveGateways {
		if e != nil {
			elements = append(elements, e)
		}
	}
	for _, e := range p.ParallelGateways {
		if e != nil {
			elements = append(elements, e)
		}
	}
	for _, e := range p.InclusiveGateways {
		if e != nil {
			elements = append(elements, e)
		}
	}
	for _, e := range p.SubProcesses {
		if e != nil {
			elements = append(elements, e)
		}
	}
	for _, e := range p.BoundaryEvents {
		if e != nil {
			elements = append(elements, e)
		}
	}
	for _, e := range p.IntermediateEvents {
		if e != nil {
			elements = append(elements, e)
		}
	}

	return elements
}

// findNameInElements 在元素集合里按 ID 找第一个匹配项的 name。
func findNameInElements(elements []BPMNElement, id string) string {
	for _, el := range elements {
		if el.GetID() == id {
			return el.GetName()
		}
	}
	return ""
}
