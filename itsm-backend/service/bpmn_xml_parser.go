package service

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"
)

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
}

// BPMNStartEvent 开始事件
type BPMNStartEvent struct {
	ID         string `xml:"id,attr"`
	Name       string `xml:"name,attr"`
	EventType  string `xml:"eventType,attr"`
	MessageRef string `xml:"messageRef,attr"`
	TimerRef   string `xml:"timerRef,attr"`
	SignalRef  string `xml:"signalRef,attr"`
}

// BPMNEndEvent 结束事件
type BPMNEndEvent struct {
	ID         string `xml:"id,attr"`
	Name       string `xml:"name,attr"`
	EventType  string `xml:"eventType,attr"`
	MessageRef string `xml:"messageRef,attr"`
	SignalRef  string `xml:"signalRef,attr"`
	ErrorRef   string `xml:"errorRef,attr"`
}

// BPMNUserTask 用户任务
type BPMNUserTask struct {
	ID              string `xml:"id,attr"`
	Name            string `xml:"name,attr"`
	Assignee        string `xml:"assignee,attr"`
	CandidateUsers  string `xml:"candidateUsers,attr"`
	CandidateGroups string `xml:"candidateGroups,attr"`
	Priority        int    `xml:"priority,attr"`
	FormKey         string `xml:"formKey,attr"`
	DueDate         string `xml:"dueDate,attr"`
}

// BPMNServiceTask 服务任务
type BPMNServiceTask struct {
	ID                 string `xml:"id,attr"`
	Name               string `xml:"name,attr"`
	Implementation     string `xml:"implementation,attr"`
	Class              string `xml:"class,attr"`
	Expression         string `xml:"expression,attr"`
	DelegateExpression string `xml:"delegateExpression,attr"`
}

// BPMNScriptTask 脚本任务
type BPMNScriptTask struct {
	ID           string `xml:"id,attr"`
	Name         string `xml:"name,attr"`
	ScriptFormat string `xml:"scriptFormat,attr"`
	Script       string `xml:"script"`
}

// BPMNExclusiveGateway 排他网关
type BPMNExclusiveGateway struct {
	ID          string `xml:"id,attr"`
	Name        string `xml:"name,attr"`
	DefaultFlow string `xml:"default,attr"`
}

// BPMNParallelGateway 并行网关
type BPMNParallelGateway struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

// BPMNInclusiveGateway 包容网关
type BPMNInclusiveGateway struct {
	ID          string `xml:"id,attr"`
	Name        string `xml:"name,attr"`
	DefaultFlow string `xml:"default,attr"`
}

// BPMNSequenceFlow 顺序流
type BPMNSequenceFlow struct {
	ID                  string                   `xml:"id,attr"`
	Name                string                   `xml:"name,attr"`
	SourceRef           string                   `xml:"sourceRef,attr"`
	TargetRef           string                   `xml:"targetRef,attr"`
	ConditionExpression *BPMNConditionExpression `xml:"conditionExpression"`
}

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

// BPMNManualTask 手工任务
type BPMNManualTask struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

// BPMNCallActivity 调用活动
type BPMNCallActivity struct {
	ID                string `xml:"id,attr"`
	Name              string `xml:"name,attr"`
	CalledElement     string `xml:"calledElement,attr"`
	CalledElementType string `xml:"calledElementType,attr"`
	InheritVariables  bool   `xml:"inheritVariables,attr"`
}

// BPMNDataObject 数据对象
type BPMNDataObject struct {
	ID           string `xml:"id,attr"`
	Name         string `xml:"name,attr"`
	DataState    string `xml:"dataState,attr"`
	IsCollection bool   `xml:"isCollection,attr"`
}

// BPMNDataStore 数据存储
type BPMNDataStore struct {
	ID          string `xml:"id,attr"`
	Name        string `xml:"name,attr"`
	DataState   string `xml:"dataState,attr"`
	Capacity    string `xml:"capacity,attr"`
	IsUnlimited bool   `xml:"isUnlimited,attr"`
}

// BPMNBoundaryEvent 边界事件
type BPMNBoundaryEvent struct {
	ID             string `xml:"id,attr"`
	Name           string `xml:"name,attr"`
	AttachedToRef  string `xml:"attachedToRef,attr"`
	CancelActivity bool   `xml:"cancelActivity,attr"`
}

// BPMNIntermediateEvent 中间事件
type BPMNIntermediateEvent struct {
	ID         string `xml:"id,attr"`
	Name       string `xml:"name,attr"`
	EventType  string `xml:"eventType,attr"`
	MessageRef string `xml:"messageRef,attr"`
	TimerRef   string `xml:"timerRef,attr"`
	SignalRef  string `xml:"signalRef,attr"`
}

// BPMNDefinitions BPMN定义根元素
type BPMNDefinitions struct {
	XMLName         xml.Name       `xml:"definitions"`
	ID              string         `xml:"id,attr"`
	Name            string         `xml:"name,attr"`
	TargetNamespace string         `xml:"targetNamespace,attr"`
	Processes       []*BPMNProcess `xml:"process"`
}

// BPMNParser BPMN XML解析器
type BPMNParser struct{}

// NewBPMNParser 创建BPMN解析器实例
func NewBPMNParser() *BPMNParser {
	return &BPMNParser{}
}

// ParseXML 解析BPMN XML文件
func (p *BPMNParser) ParseXML(xmlData []byte) (*BPMNDefinitions, error) {
	var definitions BPMNDefinitions
	err := xml.Unmarshal(xmlData, &definitions)
	if err != nil {
		return nil, fmt.Errorf("解析BPMN XML失败: %w", err)
	}

	// 验证BPMN结构
	if err := p.validateBPMN(&definitions); err != nil {
		return nil, fmt.Errorf("BPMN验证失败: %w", err)
	}

	return &definitions, nil
}

// ParseXMLFromReader 从Reader解析BPMN XML
func (p *BPMNParser) ParseXMLFromReader(reader io.Reader) (*BPMNDefinitions, error) {
	xmlData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("读取XML数据失败: %w", err)
	}
	return p.ParseXML(xmlData)
}

// validateBPMN 验证BPMN结构
func (p *BPMNParser) validateBPMN(definitions *BPMNDefinitions) error {
	if len(definitions.Processes) == 0 {
		return fmt.Errorf("BPMN定义必须包含至少一个流程")
	}

	for _, process := range definitions.Processes {
		if err := p.validateProcess(process); err != nil {
			return fmt.Errorf("流程验证失败 [%s]: %w", process.ID, err)
		}
	}

	return nil
}

// validateProcess 验证单个流程
func (p *BPMNParser) validateProcess(process *BPMNProcess) error {
	// 检查必要元素
	if process.ID == "" {
		return fmt.Errorf("流程ID不能为空")
	}

	if len(process.StartEvents) == 0 {
		return fmt.Errorf("流程必须包含至少一个开始事件")
	}

	if len(process.EndEvents) == 0 {
		return fmt.Errorf("流程必须包含至少一个结束事件")
	}

	// 验证顺序流
	if err := p.validateSequenceFlows(process); err != nil {
		return fmt.Errorf("顺序流验证失败: %w", err)
	}

	// 验证网关配置
	if err := p.validateGateways(process); err != nil {
		return fmt.Errorf("网关验证失败: %w", err)
	}

	return nil
}

// validateSequenceFlows 验证顺序流
func (p *BPMNParser) validateSequenceFlows(process *BPMNProcess) error {
	// 检查所有顺序流的源和目标是否有效
	for _, flow := range process.SequenceFlows {
		if flow.SourceRef == "" {
			return fmt.Errorf("顺序流 [%s] 缺少源引用", flow.ID)
		}
		if flow.TargetRef == "" {
			return fmt.Errorf("顺序流 [%s] 缺少目标引用", flow.ID)
		}

		// 检查源元素是否存在
		if !p.elementExists(process, flow.SourceRef) {
			return fmt.Errorf("顺序流 [%s] 的源元素 [%s] 不存在", flow.ID, flow.SourceRef)
		}

		// 检查目标元素是否存在
		if !p.elementExists(process, flow.TargetRef) {
			return fmt.Errorf("顺序流 [%s] 的目标元素 [%s] 不存在", flow.ID, flow.TargetRef)
		}
	}

	return nil
}

// validateGateways 验证网关配置
func (p *BPMNParser) validateGateways(process *BPMNProcess) error {
	// 验证排他网关的默认流
	for _, gateway := range process.ExclusiveGateways {
		if gateway.DefaultFlow != "" {
			// 检查默认流是否存在
			if !p.elementExists(process, gateway.DefaultFlow) {
				return fmt.Errorf("排他网关 [%s] 的默认流 [%s] 不存在", gateway.ID, gateway.DefaultFlow)
			}
		}
	}

	// 验证包容网关的默认流
	for _, gateway := range process.InclusiveGateways {
		if gateway.DefaultFlow != "" {
			// 检查默认流是否存在
			if !p.elementExists(process, gateway.DefaultFlow) {
				return fmt.Errorf("包容网关 [%s] 的默认流 [%s] 不存在", gateway.ID, gateway.DefaultFlow)
			}
		}
	}

	return nil
}

// elementExists 检查元素是否存在
func (p *BPMNParser) elementExists(process *BPMNProcess, elementID string) bool {
	// 检查开始事件
	for _, event := range process.StartEvents {
		if event.ID == elementID {
			return true
		}
	}

	// 检查结束事件
	for _, event := range process.EndEvents {
		if event.ID == elementID {
			return true
		}
	}

	// 检查用户任务
	for _, task := range process.UserTasks {
		if task.ID == elementID {
			return true
		}
	}

	// 检查服务任务
	for _, task := range process.ServiceTasks {
		if task.ID == elementID {
			return true
		}
	}

	// 检查脚本任务
	for _, task := range process.ScriptTasks {
		if task.ID == elementID {
			return true
		}
	}

	// 检查业务规则任务
	for _, task := range process.BusinessRuleTasks {
		if task.ID == elementID {
			return true
		}
	}

	// 检查手工任务
	for _, task := range process.ManualTasks {
		if task.ID == elementID {
			return true
		}
	}

	// 检查调用活动
	for _, activity := range process.CallActivities {
		if activity.ID == elementID {
			return true
		}
	}

	// 检查排他网关
	for _, gateway := range process.ExclusiveGateways {
		if gateway.ID == elementID {
			return true
		}
	}

	// 检查并行网关
	for _, gateway := range process.ParallelGateways {
		if gateway.ID == elementID {
			return true
		}
	}

	// 检查包容网关
	for _, gateway := range process.InclusiveGateways {
		if gateway.ID == elementID {
			return true
		}
	}

	// 检查子流程
	for _, subProcess := range process.SubProcesses {
		if subProcess.ID == elementID {
			return true
		}
	}

	// 检查边界事件
	for _, event := range process.BoundaryEvents {
		if event.ID == elementID {
			return true
		}
	}

	// 检查中间事件
	for _, event := range process.IntermediateEvents {
		if event.ID == elementID {
			return true
		}
	}

	// 检查数据对象
	for _, dataObj := range process.DataObjects {
		if dataObj.ID == elementID {
			return true
		}
	}

	// 检查数据存储
	for _, dataStore := range process.DataStores {
		if dataStore.ID == elementID {
			return true
		}
	}

	// 检查顺序流（网关默认流引用的是 sequenceFlow 的 id）
	for _, flow := range process.SequenceFlows {
		if flow.ID == elementID {
			return true
		}
	}

	return false
}

// ExtractProcessInfo 提取流程基本信息
func (p *BPMNParser) ExtractProcessInfo(definitions *BPMNDefinitions) map[string]interface{} {
	if len(definitions.Processes) == 0 {
		return nil
	}

	process := definitions.Processes[0]
	elements := map[string]interface{}{
		"startEvents":        len(process.StartEvents),
		"endEvents":          len(process.EndEvents),
		"userTasks":          len(process.UserTasks),
		"serviceTasks":       len(process.ServiceTasks),
		"scriptTasks":        len(process.ScriptTasks),
		"businessRuleTasks":  len(process.BusinessRuleTasks),
		"manualTasks":        len(process.ManualTasks),
		"callActivities":     len(process.CallActivities),
		"exclusiveGateways":  len(process.ExclusiveGateways),
		"parallelGateways":   len(process.ParallelGateways),
		"inclusiveGateways":  len(process.InclusiveGateways),
		"sequenceFlows":      len(process.SequenceFlows),
		"subProcesses":       len(process.SubProcesses),
		"boundaryEvents":     len(process.BoundaryEvents),
		"intermediateEvents": len(process.IntermediateEvents),
		"dataObjects":        len(process.DataObjects),
		"dataStores":         len(process.DataStores),
	}
	info := map[string]interface{}{
		"id":                 process.ID,
		"name":               process.Name,
		"processType":        process.ProcessType,
		"isExecutable":       process.IsExecutable,
		"isClosed":           process.IsClosed,
		// 向后兼容：保留顶层计数，同时新增 elements 聚合
		"startEvents":   len(process.StartEvents),
		"endEvents":     len(process.EndEvents),
		"userTasks":     len(process.UserTasks),
		"serviceTasks":  len(process.ServiceTasks),
		"exclusiveGateways": len(process.ExclusiveGateways),
		"sequenceFlows": len(process.SequenceFlows),
		"elements":      elements,
	}

	return info
}

// ExtractUserTasks 提取所有用户任务
func (p *BPMNParser) ExtractUserTasks(definitions *BPMNDefinitions) []map[string]interface{} {
	var tasks []map[string]interface{}

	for _, process := range definitions.Processes {
		for _, task := range process.UserTasks {
			taskInfo := map[string]interface{}{
				"id":              task.ID,
				"name":            task.Name,
				"assignee":        task.Assignee,
				"candidateUsers":  task.CandidateUsers,
				"candidateGroups": task.CandidateGroups,
				"priority":        task.Priority,
				"formKey":         task.FormKey,
				"dueDate":         task.DueDate,
				"processId":       process.ID,
			}
			tasks = append(tasks, taskInfo)
		}
	}

	return tasks
}

// ExtractServiceTasks 提取所有服务任务
func (p *BPMNParser) ExtractServiceTasks(definitions *BPMNDefinitions) []map[string]interface{} {
	var tasks []map[string]interface{}

	for _, process := range definitions.Processes {
		for _, task := range process.ServiceTasks {
			taskInfo := map[string]interface{}{
				"id":                 task.ID,
				"name":               task.Name,
				"implementation":     task.Implementation,
				"class":              task.Class,
				"expression":         task.Expression,
				"delegateExpression": task.DelegateExpression,
				"processId":          process.ID,
			}
			tasks = append(tasks, taskInfo)
		}
	}

	return tasks
}

// ExtractBusinessRuleTasks 提取所有业务规则任务
func (p *BPMNParser) ExtractBusinessRuleTasks(definitions *BPMNDefinitions) []map[string]interface{} {
	var tasks []map[string]interface{}

	for _, process := range definitions.Processes {
		for _, task := range process.BusinessRuleTasks {
			taskInfo := map[string]interface{}{
				"id":                 task.ID,
				"name":               task.Name,
				"implementation":     task.Implementation,
				"class":              task.Class,
				"expression":         task.Expression,
				"delegateExpression": task.DelegateExpression,
				"decisionTaskRef":    task.DecisionTaskRef,
				"processId":          process.ID,
			}
			tasks = append(tasks, taskInfo)
		}
	}

	return tasks
}

// ExtractManualTasks 提取所有手工任务
func (p *BPMNParser) ExtractManualTasks(definitions *BPMNDefinitions) []map[string]interface{} {
	var tasks []map[string]interface{}

	for _, process := range definitions.Processes {
		for _, task := range process.ManualTasks {
			taskInfo := map[string]interface{}{
				"id":        task.ID,
				"name":      task.Name,
				"processId": process.ID,
			}
			tasks = append(tasks, taskInfo)
		}
	}

	return tasks
}

// ExtractCallActivities 提取所有调用活动
func (p *BPMNParser) ExtractCallActivities(definitions *BPMNDefinitions) []map[string]interface{} {
	var activities []map[string]interface{}

	for _, process := range definitions.Processes {
		for _, activity := range process.CallActivities {
			activityInfo := map[string]interface{}{
				"id":                activity.ID,
				"name":              activity.Name,
				"calledElement":     activity.CalledElement,
				"calledElementType": activity.CalledElementType,
				"inheritVariables":  activity.InheritVariables,
				"processId":         process.ID,
			}
			activities = append(activities, activityInfo)
		}
	}

	return activities
}

// ExtractDataObjects 提取所有数据对象
func (p *BPMNParser) ExtractDataObjects(definitions *BPMNDefinitions) []map[string]interface{} {
	var dataObjects []map[string]interface{}

	for _, process := range definitions.Processes {
		for _, dataObj := range process.DataObjects {
			dataObjInfo := map[string]interface{}{
				"id":           dataObj.ID,
				"name":         dataObj.Name,
				"dataState":    dataObj.DataState,
				"isCollection": dataObj.IsCollection,
				"processId":    process.ID,
			}
			dataObjects = append(dataObjects, dataObjInfo)
		}
	}

	return dataObjects
}

// ExtractDataStores 提取所有数据存储
func (p *BPMNParser) ExtractDataStores(definitions *BPMNDefinitions) []map[string]interface{} {
	var dataStores []map[string]interface{}

	for _, process := range definitions.Processes {
		for _, dataStore := range process.DataStores {
			dataStoreInfo := map[string]interface{}{
				"id":          dataStore.ID,
				"name":        dataStore.Name,
				"dataState":   dataStore.DataState,
				"capacity":    dataStore.Capacity,
				"isUnlimited": dataStore.IsUnlimited,
				"processId":   process.ID,
			}
			dataStores = append(dataStores, dataStoreInfo)
		}
	}

	return dataStores
}

// ExtractGateways 提取所有网关
func (p *BPMNParser) ExtractGateways(definitions *BPMNDefinitions) map[string][]map[string]interface{} {
	gateways := map[string][]map[string]interface{}{
		"exclusive": make([]map[string]interface{}, 0),
		"parallel":  make([]map[string]interface{}, 0),
		"inclusive": make([]map[string]interface{}, 0),
	}

	for _, process := range definitions.Processes {
		// 排他网关
		for _, gateway := range process.ExclusiveGateways {
			gatewayInfo := map[string]interface{}{
				"id":          gateway.ID,
				"name":        gateway.Name,
				"defaultFlow": gateway.DefaultFlow,
				"processId":   process.ID,
			}
			gateways["exclusive"] = append(gateways["exclusive"], gatewayInfo)
		}

		// 并行网关
		for _, gateway := range process.ParallelGateways {
			gatewayInfo := map[string]interface{}{
				"id":        gateway.ID,
				"name":      gateway.Name,
				"processId": process.ID,
			}
			gateways["parallel"] = append(gateways["parallel"], gatewayInfo)
		}

		// 包容网关
		for _, gateway := range process.InclusiveGateways {
			gatewayInfo := map[string]interface{}{
				"id":          gateway.ID,
				"name":        gateway.Name,
				"defaultFlow": gateway.DefaultFlow,
				"processId":   process.ID,
			}
			gateways["inclusive"] = append(gateways["inclusive"], gatewayInfo)
		}
	}

	return gateways
}

// ExtractSequenceFlows 提取所有顺序流
func (p *BPMNParser) ExtractSequenceFlows(definitions *BPMNDefinitions) []map[string]interface{} {
	var flows []map[string]interface{}

	for _, process := range definitions.Processes {
		for _, flow := range process.SequenceFlows {
			flowInfo := map[string]interface{}{
				"id":        flow.ID,
				"name":      flow.Name,
				"sourceRef": flow.SourceRef,
				"targetRef": flow.TargetRef,
				"processId": process.ID,
			}

			// 添加条件表达式信息
			if flow.ConditionExpression != nil {
				flowInfo["conditionType"] = flow.ConditionExpression.Type
				flowInfo["conditionExpression"] = flow.ConditionExpression.Expression
			}

			flows = append(flows, flowInfo)
		}
	}

	return flows
}

// ValidateBPMNXML 验证BPMN XML语法
func (p *BPMNParser) ValidateBPMNXML(xmlData []byte) error {
	// 尝试解析XML
	_, err := p.ParseXML(xmlData)
	if err != nil {
		return err
	}

	// 检查是否包含BPMN命名空间
	xmlStr := string(xmlData)
	if !strings.Contains(xmlStr, "bpmn:") && !strings.Contains(xmlStr, "xmlns:bpmn") {
		return fmt.Errorf("XML文件缺少BPMN命名空间声明")
	}

	return nil
}

// GenerateBPMNSummary 生成BPMN摘要报告
func (p *BPMNParser) GenerateBPMNSummary(definitions *BPMNDefinitions) map[string]interface{} {
	summary := map[string]interface{}{
		"totalProcesses": len(definitions.Processes),
		"processes":      make([]map[string]interface{}, 0),
	}

	for _, process := range definitions.Processes {
		processSummary := map[string]interface{}{
			"id":           process.ID,
			"name":         process.Name,
			"isExecutable": process.IsExecutable,
			"elements": map[string]interface{}{
				"startEvents":        len(process.StartEvents),
				"endEvents":          len(process.EndEvents),
				"userTasks":          len(process.UserTasks),
				"serviceTasks":       len(process.ServiceTasks),
				"scriptTasks":        len(process.ScriptTasks),
				"businessRuleTasks":  len(process.BusinessRuleTasks),
				"manualTasks":        len(process.ManualTasks),
				"callActivities":     len(process.CallActivities),
				"exclusiveGateways":  len(process.ExclusiveGateways),
				"parallelGateways":   len(process.ParallelGateways),
				"inclusiveGateways":  len(process.InclusiveGateways),
				"sequenceFlows":      len(process.SequenceFlows),
				"subProcesses":       len(process.SubProcesses),
				"boundaryEvents":     len(process.BoundaryEvents),
				"intermediateEvents": len(process.IntermediateEvents),
				"dataObjects":        len(process.DataObjects),
				"dataStores":         len(process.DataStores),
			},
		}
		summary["processes"] = append(summary["processes"].([]map[string]interface{}), processSummary)
	}

	return summary
}

// ConvertToProcessDefinition 将BPMN定义转换为流程定义结构
func (p *BPMNParser) ConvertToProcessDefinition(definitions *BPMNDefinitions, tenantID int) map[string]interface{} {
	if len(definitions.Processes) == 0 {
		return nil
	}

	process := definitions.Processes[0]

	// 转换为流程定义格式
	processDef := map[string]interface{}{
		"key":           process.ID,
		"name":          process.Name,
		"version":       "1.0.0", // 默认版本
		"tenant_id":     tenantID,
		"is_active":     true,
		"is_latest":     true,
		"bpmn_xml":      "", // 需要原始XML
		"deployment_id": "",
		"created_at":    time.Now(),
		"updated_at":    time.Now(),
		"metadata": map[string]interface{}{
			"processType":   process.ProcessType,
			"isExecutable":  process.IsExecutable,
			"isClosed":      process.IsClosed,
			"totalElements": p.countTotalElements(process),
			"elementTypes":  p.getElementTypeCounts(process),
		},
	}

	return processDef
}

// countTotalElements 计算流程中的总元素数量
func (p *BPMNParser) countTotalElements(process *BPMNProcess) int {
	return len(process.StartEvents) + len(process.EndEvents) +
		len(process.UserTasks) + len(process.ServiceTasks) + len(process.ScriptTasks) +
		len(process.BusinessRuleTasks) + len(process.ManualTasks) + len(process.CallActivities) +
		len(process.ExclusiveGateways) + len(process.ParallelGateways) + len(process.InclusiveGateways) +
		len(process.SequenceFlows) + len(process.SubProcesses) +
		len(process.BoundaryEvents) + len(process.IntermediateEvents) +
		len(process.DataObjects) + len(process.DataStores)
}

// getElementTypeCounts 获取各类型元素的数量统计
func (p *BPMNParser) getElementTypeCounts(process *BPMNProcess) map[string]int {
	return map[string]int{
		"startEvents":        len(process.StartEvents),
		"endEvents":          len(process.EndEvents),
		"userTasks":          len(process.UserTasks),
		"serviceTasks":       len(process.ServiceTasks),
		"scriptTasks":        len(process.ScriptTasks),
		"businessRuleTasks":  len(process.BusinessRuleTasks),
		"manualTasks":        len(process.ManualTasks),
		"callActivities":     len(process.CallActivities),
		"exclusiveGateways":  len(process.ExclusiveGateways),
		"parallelGateways":   len(process.ParallelGateways),
		"inclusiveGateways":  len(process.InclusiveGateways),
		"sequenceFlows":      len(process.SequenceFlows),
		"subProcesses":       len(process.SubProcesses),
		"boundaryEvents":     len(process.BoundaryEvents),
		"intermediateEvents": len(process.IntermediateEvents),
		"dataObjects":        len(process.DataObjects),
		"dataStores":         len(process.DataStores),
	}
}

// ExportToXML 将BPMN定义导出为XML格式
func (p *BPMNParser) ExportToXML(definitions *BPMNDefinitions) ([]byte, error) {
	// 设置XML命名空间
	if definitions.TargetNamespace == "" {
		definitions.TargetNamespace = "http://bpmn.io/schema/bpmn"
	}

	// 添加XML声明和格式化
	xmlData, err := xml.MarshalIndent(definitions, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("XML序列化失败: %w", err)
	}

	// 添加XML声明
	xmlDeclaration := `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	result := append([]byte(xmlDeclaration), xmlData...)

	return result, nil
}

// ExportToXMLString 将BPMN定义导出为XML字符串
func (p *BPMNParser) ExportToXMLString(definitions *BPMNDefinitions) (string, error) {
	xmlData, err := p.ExportToXML(definitions)
	if err != nil {
		return "", err
	}
	return string(xmlData), nil
}

// ExportProcessToXML 导出单个流程为XML
func (p *BPMNParser) ExportProcessToXML(process *BPMNProcess, targetNamespace string) ([]byte, error) {
	if targetNamespace == "" {
		targetNamespace = "http://bpmn.io/schema/bpmn"
	}

	// 创建临时的definitions结构
	tempDefinitions := &BPMNDefinitions{
		ID:              "Definitions_1",
		Name:            "Process Export",
		TargetNamespace: targetNamespace,
		Processes:       []*BPMNProcess{process},
	}

	return p.ExportToXML(tempDefinitions)
}

// ValidateBPMNCompleteness 验证BPMN流程的完整性
func (p *BPMNParser) ValidateBPMNCompleteness(definitions *BPMNDefinitions) map[string]interface{} {
	validation := map[string]interface{}{
		"isValid":     true,
		"errors":      []string{},
		"warnings":    []string{},
		"suggestions": []string{},
	}

	for _, process := range definitions.Processes {
		// 检查流程基本属性
		if process.ID == "" {
			validation["isValid"] = false
			validation["errors"] = append(validation["errors"].([]string), "流程缺少ID")
		}

		if process.Name == "" {
			validation["warnings"] = append(validation["warnings"].([]string), "流程缺少名称")
		}

		// 检查必要元素
		if len(process.StartEvents) == 0 {
			validation["isValid"] = false
			validation["errors"] = append(validation["errors"].([]string), "流程必须包含至少一个开始事件")
		}

		if len(process.EndEvents) == 0 {
			validation["isValid"] = false
			validation["errors"] = append(validation["errors"].([]string), "流程必须包含至少一个结束事件")
		}

		// 检查任务分配
		for _, task := range process.UserTasks {
			if task.Assignee == "" && task.CandidateUsers == "" && task.CandidateGroups == "" {
				validation["warnings"] = append(validation["warnings"].([]string),
					fmt.Sprintf("用户任务 [%s] 缺少分配者或候选用户/组", task.ID))
			}
		}

		// 检查网关配置
		for _, gateway := range process.ExclusiveGateways {
			if gateway.DefaultFlow != "" {
				// 检查默认流是否存在
				if !p.elementExists(process, gateway.DefaultFlow) {
					validation["isValid"] = false
					validation["errors"] = append(validation["errors"].([]string),
						fmt.Sprintf("排他网关 [%s] 的默认流 [%s] 不存在", gateway.ID, gateway.DefaultFlow))
				}
			}
		}

		// 检查顺序流连接
		if len(process.SequenceFlows) > 0 {
			connectedElements := make(map[string]bool)
			for _, flow := range process.SequenceFlows {
				connectedElements[flow.SourceRef] = true
				connectedElements[flow.TargetRef] = true
			}

			// 检查是否有孤立的元素
			allElements := p.getAllElementIDs(process)
			for _, elementID := range allElements {
				if !connectedElements[elementID] {
					validation["warnings"] = append(validation["warnings"].([]string),
						fmt.Sprintf("元素 [%s] 未连接到流程中", elementID))
				}
			}
		}

		// 检查循环引用
		if p.hasCircularReference(process) {
			validation["warnings"] = append(validation["warnings"].([]string),
				"流程中存在循环引用，可能导致无限循环")
		}
	}

	return validation
}

// getAllElementIDs 获取流程中所有元素的ID
func (p *BPMNParser) getAllElementIDs(process *BPMNProcess) []string {
	var ids []string

	// 添加所有类型元素的ID
	for _, event := range process.StartEvents {
		ids = append(ids, event.ID)
	}
	for _, event := range process.EndEvents {
		ids = append(ids, event.ID)
	}
	for _, task := range process.UserTasks {
		ids = append(ids, task.ID)
	}
	for _, task := range process.ServiceTasks {
		ids = append(ids, task.ID)
	}
	for _, task := range process.ScriptTasks {
		ids = append(ids, task.ID)
	}
	for _, task := range process.BusinessRuleTasks {
		ids = append(ids, task.ID)
	}
	for _, task := range process.ManualTasks {
		ids = append(ids, task.ID)
	}
	for _, activity := range process.CallActivities {
		ids = append(ids, activity.ID)
	}
	for _, gateway := range process.ExclusiveGateways {
		ids = append(ids, gateway.ID)
	}
	for _, gateway := range process.ParallelGateways {
		ids = append(ids, gateway.ID)
	}
	for _, gateway := range process.InclusiveGateways {
		ids = append(ids, gateway.ID)
	}
	for _, subProcess := range process.SubProcesses {
		ids = append(ids, subProcess.ID)
	}
	for _, event := range process.BoundaryEvents {
		ids = append(ids, event.ID)
	}
	for _, event := range process.IntermediateEvents {
		ids = append(ids, event.ID)
	}
	for _, dataObj := range process.DataObjects {
		ids = append(ids, dataObj.ID)
	}
	for _, dataStore := range process.DataStores {
		ids = append(ids, dataStore.ID)
	}

	return ids
}

// hasCircularReference 检查流程是否存在循环引用
func (p *BPMNParser) hasCircularReference(process *BPMNProcess) bool {
	// 使用深度优先搜索检测循环
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for _, flow := range process.SequenceFlows {
		if p.isCyclicUtil(process, flow.SourceRef, visited, recStack) {
			return true
		}
	}

	return false
}

// isCyclicUtil 深度优先搜索的辅助函数，用于检测循环
func (p *BPMNParser) isCyclicUtil(process *BPMNProcess, elementID string, visited, recStack map[string]bool) bool {
	if recStack[elementID] {
		return true
	}

	if visited[elementID] {
		return false
	}

	visited[elementID] = true
	recStack[elementID] = true

	// 查找所有以当前元素为源的顺序流
	for _, flow := range process.SequenceFlows {
		if flow.SourceRef == elementID {
			if p.isCyclicUtil(process, flow.TargetRef, visited, recStack) {
				return true
			}
		}
	}

	recStack[elementID] = false
	return false
}

// GetBPMNStatistics 获取BPMN统计信息
func (p *BPMNParser) GetBPMNStatistics(definitions *BPMNDefinitions) map[string]interface{} {
	stats := map[string]interface{}{
		"totalProcesses": len(definitions.Processes),
		"totalElements":  0,
		"elementTypes":   make(map[string]int),
		"complexity":     "low",
	}

	for _, process := range definitions.Processes {
		elementCount := p.countTotalElements(process)
		stats["totalElements"] = stats["totalElements"].(int) + elementCount

		// 统计各类型元素
		elementTypes := p.getElementTypeCounts(process)
		for elementType, count := range elementTypes {
			if count > 0 {
				currentCount := stats["elementTypes"].(map[string]int)[elementType]
				stats["elementTypes"].(map[string]int)[elementType] = currentCount + count
			}
		}

		// 评估复杂度
		if elementCount > 50 {
			stats["complexity"] = "high"
		} else if elementCount > 20 {
			stats["complexity"] = "medium"
		}
	}

	return stats
}

// FindPathBetweenElements 查找两个元素之间的路径
func (p *BPMNParser) FindPathBetweenElements(process *BPMNProcess, sourceID, targetID string) ([]string, error) {
	if !p.elementExists(process, sourceID) {
		return nil, fmt.Errorf("源元素 [%s] 不存在", sourceID)
	}
	if !p.elementExists(process, targetID) {
		return nil, fmt.Errorf("目标元素 [%s] 不存在", targetID)
	}

	// 使用广度优先搜索查找最短路径
	queue := [][]string{{sourceID}}
	visited := make(map[string]bool)
	visited[sourceID] = true

	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		currentID := path[len(path)-1]

		if currentID == targetID {
			return path, nil
		}

		// 查找所有以当前元素为源的顺序流
		for _, flow := range process.SequenceFlows {
			if flow.SourceRef == currentID && !visited[flow.TargetRef] {
				visited[flow.TargetRef] = true
				newPath := make([]string, len(path))
				copy(newPath, path)
				newPath = append(newPath, flow.TargetRef)
				queue = append(queue, newPath)
			}
		}
	}

	return nil, fmt.Errorf("无法找到从 [%s] 到 [%s] 的路径", sourceID, targetID)
}

// GetElementDependencies 获取元素的依赖关系
func (p *BPMNParser) GetElementDependencies(process *BPMNProcess, elementID string) map[string][]string {
	dependencies := make(map[string][]string)
	dependencies["incoming"] = []string{}
	dependencies["outgoing"] = []string{}

	// 查找入向依赖（以当前元素为目标的顺序流）
	for _, flow := range process.SequenceFlows {
		if flow.TargetRef == elementID {
			dependencies["incoming"] = append(dependencies["incoming"], flow.SourceRef)
		}
	}

	// 查找出向依赖（以当前元素为源的顺序流）
	for _, flow := range process.SequenceFlows {
		if flow.SourceRef == elementID {
			dependencies["outgoing"] = append(dependencies["outgoing"], flow.TargetRef)
		}
	}

	return dependencies
}

// GetProcessExecutionPaths 获取流程的所有执行路径
func (p *BPMNParser) GetProcessExecutionPaths(process *BPMNProcess) [][]string {
	var allPaths [][]string

	// 从每个开始事件开始查找路径
	for _, startEvent := range process.StartEvents {
		paths := p.findPathsFromElement(process, startEvent.ID, make(map[string]bool))
		allPaths = append(allPaths, paths...)
	}

	return allPaths
}

// findPathsFromElement 从指定元素开始查找所有可能的路径
func (p *BPMNParser) findPathsFromElement(process *BPMNProcess, elementID string, visited map[string]bool) [][]string {
	var paths [][]string

	// 如果元素已被访问，返回空路径（避免循环）
	if visited[elementID] {
		return paths
	}

	// 标记当前元素为已访问
	visited[elementID] = true
	defer func() { visited[elementID] = false }()

	// 检查是否为结束事件
	isEndEvent := false
	for _, endEvent := range process.EndEvents {
		if endEvent.ID == elementID {
			isEndEvent = true
			break
		}
	}

	if isEndEvent {
		// 找到结束事件，返回当前路径
		return [][]string{{elementID}}
	}

	// 查找所有以当前元素为源的顺序流
	for _, flow := range process.SequenceFlows {
		if flow.SourceRef == elementID {
			// 递归查找后续路径
			subPaths := p.findPathsFromElement(process, flow.TargetRef, visited)
			for _, subPath := range subPaths {
				// 将当前元素添加到路径开头
				newPath := make([]string, len(subPath)+1)
				newPath[0] = elementID
				copy(newPath[1:], subPath)
				paths = append(paths, newPath)
			}
		}
	}

	return paths
}

// ValidateBPMNBusinessRules 验证BPMN业务规则
func (p *BPMNParser) ValidateBPMNBusinessRules(definitions *BPMNDefinitions) map[string]interface{} {
	validation := map[string]interface{}{
		"isValid":     true,
		"errors":      []string{},
		"warnings":    []string{},
		"suggestions": []string{},
	}

	for _, process := range definitions.Processes {
		// 检查用户任务的业务规则
		for _, task := range process.UserTasks {
			// 检查任务优先级
			if task.Priority < 0 || task.Priority > 10 {
				validation["warnings"] = append(validation["warnings"].([]string),
					fmt.Sprintf("用户任务 [%s] 的优先级应在0-10之间", task.ID))
			}

			// 检查截止日期格式
			if task.DueDate != "" {
				if _, err := time.Parse("2006-01-02", task.DueDate); err != nil {
					validation["warnings"] = append(validation["warnings"].([]string),
						fmt.Sprintf("用户任务 [%s] 的截止日期格式应为YYYY-MM-DD", task.ID))
				}
			}
		}

		// 检查服务任务的实现
		for _, task := range process.ServiceTasks {
			if task.Implementation == "" && task.Class == "" && task.Expression == "" && task.DelegateExpression == "" {
				validation["warnings"] = append(validation["warnings"].([]string),
					fmt.Sprintf("服务任务 [%s] 缺少实现方式", task.ID))
			}
		}

		// 检查脚本任务的脚本格式
		for _, task := range process.ScriptTasks {
			if task.ScriptFormat == "" {
				validation["warnings"] = append(validation["warnings"].([]string),
					fmt.Sprintf("脚本任务 [%s] 缺少脚本格式", task.ID))
			}
			if task.Script == "" {
				validation["warnings"] = append(validation["warnings"].([]string),
					fmt.Sprintf("脚本任务 [%s] 缺少脚本内容", task.ID))
			}
		}

		// 检查调用活动的引用
		for _, activity := range process.CallActivities {
			if activity.CalledElement == "" {
				validation["errors"] = append(validation["errors"].([]string),
					fmt.Sprintf("调用活动 [%s] 缺少被调用元素", activity.ID))
				validation["isValid"] = false
			}
		}
	}

	return validation
}

// GetBPMNElementMetrics 获取BPMN元素的度量指标
func (p *BPMNParser) GetBPMNElementMetrics(definitions *BPMNDefinitions) map[string]interface{} {
	metrics := map[string]interface{}{
		"processMetrics": make(map[string]interface{}),
		"elementMetrics": make(map[string]interface{}),
		"flowMetrics":    make(map[string]interface{}),
	}

	for _, process := range definitions.Processes {
		// 流程度量
		processMetrics := map[string]interface{}{
			"id":                 process.ID,
			"name":               process.Name,
			"totalElements":      p.countTotalElements(process),
			"startEvents":        len(process.StartEvents),
			"endEvents":          len(process.EndEvents),
			"totalTasks":         len(process.UserTasks) + len(process.ServiceTasks) + len(process.ScriptTasks) + len(process.BusinessRuleTasks) + len(process.ManualTasks),
			"totalGateways":      len(process.ExclusiveGateways) + len(process.ParallelGateways) + len(process.InclusiveGateways),
			"totalFlows":         len(process.SequenceFlows),
			"subProcesses":       len(process.SubProcesses),
			"boundaryEvents":     len(process.BoundaryEvents),
			"intermediateEvents": len(process.IntermediateEvents),
			"dataElements":       len(process.DataObjects) + len(process.DataStores),
		}

		// 计算流程复杂度指标
		elementCount := p.countTotalElements(process)
		flowCount := len(process.SequenceFlows)

		if elementCount > 0 {
			processMetrics["flowToElementRatio"] = float64(flowCount) / float64(elementCount)
		} else {
			processMetrics["flowToElementRatio"] = 0.0
		}

		// 计算网关密度
		gatewayCount := len(process.ExclusiveGateways) + len(process.ParallelGateways) + len(process.InclusiveGateways)
		if elementCount > 0 {
			processMetrics["gatewayDensity"] = float64(gatewayCount) / float64(elementCount)
		} else {
			processMetrics["gatewayDensity"] = 0.0
		}

		metrics["processMetrics"].(map[string]interface{})[process.ID] = processMetrics

		// 元素度量
		elementMetrics := p.getElementTypeCounts(process)
		for elementType, count := range elementMetrics {
			if count > 0 {
				if metrics["elementMetrics"].(map[string]interface{})[elementType] == nil {
					metrics["elementMetrics"].(map[string]interface{})[elementType] = 0
				}
				currentCount := metrics["elementMetrics"].(map[string]interface{})[elementType].(int)
				metrics["elementMetrics"].(map[string]interface{})[elementType] = currentCount + count
			}
		}

		// 流程度量
		if metrics["flowMetrics"].(map[string]interface{})["totalFlows"] == nil {
			metrics["flowMetrics"].(map[string]interface{})["totalFlows"] = 0
		}
		currentFlows := metrics["flowMetrics"].(map[string]interface{})["totalFlows"].(int)
		metrics["flowMetrics"].(map[string]interface{})["totalFlows"] = currentFlows + len(process.SequenceFlows)
	}

	return metrics
}
