package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/processdeployment"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
	"itsm-backend/ent/user"
	"itsm-backend/ent/workflowtask"

	"go.uber.org/zap"
)

// ProcessEngine BPMN流程引擎核心接口
type ProcessEngine interface {
	// 流程定义管理
	ProcessDefinitionService() ProcessDefinitionService
	// 流程实例管理
	ProcessInstanceService() ProcessInstanceService
	// 任务管理
	TaskService() TaskService
	// 流程执行
	StartProcess(ctx context.Context, processDefinitionKey string, businessKey string, variables map[string]interface{}) (*ent.ProcessInstance, error)
	CompleteTask(ctx context.Context, taskID string, variables map[string]interface{}) error
	SuspendProcess(ctx context.Context, processInstanceID string, reason string) error
	ResumeProcess(ctx context.Context, processInstanceID string) error
	TerminateProcess(ctx context.Context, processInstanceID string, reason string) error
}

// ProcessDefinitionService 流程定义服务接口
type ProcessDefinitionService interface {
	CreateProcessDefinition(ctx context.Context, req *CreateProcessDefinitionRequest) (*ent.ProcessDefinition, error)
	GetProcessDefinition(ctx context.Context, key string, version string) (*ent.ProcessDefinition, error)
	GetProcessDefinitionByID(ctx context.Context, id int) (*ent.ProcessDefinition, error)
	GetLatestProcessDefinition(ctx context.Context, key string) (*ent.ProcessDefinition, error)
	UpdateProcessDefinition(ctx context.Context, key string, version string, req *UpdateProcessDefinitionRequest) (*ent.ProcessDefinition, error)
	DeleteProcessDefinition(ctx context.Context, key string, version string) error
	ListProcessDefinitions(ctx context.Context, req *ListProcessDefinitionsRequest) ([]*ent.ProcessDefinition, int, error)
	SetProcessDefinitionActive(ctx context.Context, key string, version string, active bool) error
}

// ProcessInstanceService 流程实例服务接口
type ProcessInstanceService interface {
	GetProcessInstance(ctx context.Context, processInstanceID string) (*ent.ProcessInstance, error)
	ListProcessInstances(ctx context.Context, req *ListProcessInstancesRequest) ([]*ent.ProcessInstance, int, error)
	GetProcessInstanceVariables(ctx context.Context, processInstanceID string) (map[string]interface{}, error)
	SetProcessInstanceVariables(ctx context.Context, processInstanceID string, variables map[string]interface{}) error
	GetProcessInstanceHistory(ctx context.Context, processInstanceID string) ([]*ent.ProcessExecutionHistory, error)
	GetInstanceStatistics(ctx context.Context, req *InstanceStatisticsRequest) (*InstanceStatistics, error)
}

// TaskService 任务管理服务接口
type TaskService interface {
	GetTask(ctx context.Context, taskID string) (*ent.ProcessTask, error)
	ListUserTasks(ctx context.Context, req *ListUserTasksRequest) ([]*ent.ProcessTask, int, error)
	AssignTask(ctx context.Context, taskID string, assignee string) error
	CompleteTask(ctx context.Context, taskID string, variables map[string]interface{}) error
	CancelTask(ctx context.Context, taskID string, reason string) error
	GetTaskVariables(ctx context.Context, taskID string) (map[string]interface{}, error)
	SetTaskVariables(ctx context.Context, taskID string, variables map[string]interface{}) error
	HandleTaskTimeout(ctx context.Context, taskID string) error
	RetryTask(ctx context.Context, taskID string, maxRetries int) error
	DelegateTask(ctx context.Context, taskID string, newAssignee string) error
	EscalateTask(ctx context.Context, taskID string, reason string) error
	BatchAssignTasks(ctx context.Context, taskIDs []string, assignee string) error
	GetTaskStatistics(ctx context.Context, req *TaskStatisticsRequest) (*TaskStatistics, error)
	// 会签相关
	CreateCounterSignTasks(ctx context.Context, parentTaskID string, req *CounterSignRequest) ([]*ent.ProcessTask, error)
	GetCounterSignStatus(ctx context.Context, parentTaskID string) (*CounterSignStatus, error)
	Vote(ctx context.Context, taskID string, req *VoteRequest) error
}

// CustomProcessEngine 是ProcessEngine接口的实现
// 充当领域服务(Domain Service)，协调流程定义、实例和任务实体的生命周期
type CustomProcessEngine struct {
	client          *ent.Client
	logger          *zap.SugaredLogger
	parser          *BPMNParser       // 使用自定义的BPMN解析器
	exprEngine      *ExpressionEngine // 表达式引擎
	expressionVars  map[string]interface{} // 表达式变量
	// 内部服务
	processDefinitionService *bpmnProcessDefinitionService
	processInstanceService   *bpmnProcessInstanceService
	taskService              *bpmnTaskService
}

// NewCustomProcessEngine 创建自定义流程引擎实例
func NewCustomProcessEngine(client *ent.Client, logger *zap.SugaredLogger) ProcessEngine {
	engine := &CustomProcessEngine{
		client:          client,
		logger:          logger,
		parser:          NewBPMNParser(),
		exprEngine:      NewExpressionEngine(),
		expressionVars:  make(map[string]interface{}),
	}
	engine.processDefinitionService = &bpmnProcessDefinitionService{client: client, logger: logger}
	engine.processInstanceService = &bpmnProcessInstanceService{client: client, logger: logger}
	engine.taskService = &bpmnTaskService{client: client, logger: logger}

	// 注册流程相关的内置函数
	engine.registerProcessFunctions()

	return engine
}

// registerProcessFunctions 注册流程相关的内置函数
func (e *CustomProcessEngine) registerProcessFunctions() {
	// 获取任务列表
	e.exprEngine.RegisterFunction("getTasks", func(ctx context.Context, assignee string) []interface{} {
		// 从数据库查询任务
		tasks, err := e.client.WorkflowTask.Query().
			Where(workflowtask.Assignee(assignee)).
			Where(workflowtask.CompletedAtIsNil()).
			All(ctx)
		if err != nil {
			e.logger.Warnw("Failed to query tasks", "error", err)
			return []interface{}{}
		}
		result := make([]interface{}, len(tasks))
		for i, task := range tasks {
			result[i] = map[string]interface{}{
				"id":          task.TaskID,
				"name":        task.Name,
				"instance_id": task.InstanceID,
			}
		}
		return result
	})

	// 获取用户信息
	e.exprEngine.RegisterFunction("getUser", func(userID interface{}) interface{} {
		return map[string]interface{}{
			"id":   userID,
			"name": "User " + fmt.Sprintf("%v", userID),
		}
	})

	// 获取当前时间
	e.exprEngine.RegisterFunction("currentTime", func() int64 {
		return time.Now().Unix()
	})

	// 日期计算
	e.exprEngine.RegisterFunction("addDays", func(timestamp int64, days int) int64 {
		return timestamp + int64(days*86400)
	})

	// 数组长度
	e.exprEngine.RegisterFunction("size", func(arr []interface{}) int {
		return len(arr)
	})

	// 随机数
	e.exprEngine.RegisterFunction("random", func(min, max float64) float64 {
		return min + (max-min)*float64(time.Now().UnixNano()%10000000)/10000000
	})
}

// ProcessDefinitionService 返回流程定义服务
func (e *CustomProcessEngine) ProcessDefinitionService() ProcessDefinitionService {
	return &bpmnProcessDefinitionService{client: e.client}
}

// ProcessInstanceService 返回流程实例服务
func (e *CustomProcessEngine) ProcessInstanceService() ProcessInstanceService {
	return &bpmnProcessInstanceService{client: e.client}
}

// TaskService 返回任务服务
func (e *CustomProcessEngine) TaskService() TaskService {
	return &bpmnTaskService{client: e.client, logger: e.logger}
}

// StartProcess 启动流程实例
func (e *CustomProcessEngine) StartProcess(ctx context.Context, processDefinitionKey string, businessKey string, variables map[string]interface{}) (*ent.ProcessInstance, error) {
	// 1. 获取流程定义
	definition, err := e.client.ProcessDefinition.Query().
		Where(processdefinition.Key(processDefinitionKey)).
		Where(processdefinition.IsActive(true)).
		Where(processdefinition.IsLatest(true)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	// 2. 解析BPMN
	bpmnDefinitions, err := e.parser.ParseXML(definition.BpmnXML)
	if err != nil {
		return nil, fmt.Errorf("解析BPMN失败: %w", err)
	}

	if len(bpmnDefinitions.Processes) == 0 {
		return nil, fmt.Errorf("BPMN中未找到流程定义")
	}
	process := bpmnDefinitions.Processes[0]

	// 3. 找到开始事件
	if len(process.StartEvents) == 0 {
		return nil, fmt.Errorf("流程缺少开始事件")
	}
	startEvent := process.StartEvents[0]

	// 4. 创建流程实例
	instance, err := e.client.ProcessInstance.Create().
		SetProcessInstanceID(fmt.Sprintf("PI-%s-%d", processDefinitionKey, time.Now().UnixNano())).
		SetBusinessKey(businessKey).
		SetProcessDefinitionKey(processDefinitionKey).
		SetProcessDefinitionID(definition.ID).
		SetStatus("running").
		SetVariables(variables).
		SetStartTime(time.Now()).
		SetTenantID(definition.TenantID).
		SetCurrentActivityID(startEvent.ID).
		SetCurrentActivityName(startEvent.Name).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建流程实例失败: %w", err)
	}

	// 5. 执行流程推进（从StartEvent开始）
	if err := e.executeStep(ctx, instance, process, startEvent.ID, variables); err != nil {
		return nil, err
	}

	return instance, nil
}

// CompleteTask 完成任务
func (e *CustomProcessEngine) CompleteTask(ctx context.Context, taskID string, variables map[string]interface{}) error {
	// 1. 获取任务
	task, err := e.client.ProcessTask.Query().
		Where(processtask.TaskID(taskID)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 2. 获取流程实例 - 使用任务中存储的ProcessInstanceID (ent自动生成的ID)
	instance, err := e.client.ProcessInstance.Get(ctx, task.ProcessInstanceID)
	if err != nil {
		return fmt.Errorf("获取流程实例失败: %w", err)
	}

	// 3. 获取流程定义并解析
	definition, err := e.client.ProcessDefinition.Query().
		Where(processdefinition.Key(task.ProcessDefinitionKey)).
		Where(processdefinition.IsLatest(true)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("获取流程定义失败: %w", err)
	}

	bpmnDefinitions, err := e.parser.ParseXML(definition.BpmnXML)
	if err != nil {
		return fmt.Errorf("解析BPMN失败: %w", err)
	}
	process := bpmnDefinitions.Processes[0]

	// 4. 更新当前任务状态
	_, err = e.client.ProcessTask.UpdateOne(task).
		SetStatus("completed").
		SetCompletedTime(time.Now()).
		SetTaskVariables(variables).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 5. 合并变量
	currentVars := instance.Variables
	if currentVars == nil {
		currentVars = make(map[string]interface{})
	}
	for k, v := range variables {
		currentVars[k] = v
	}

	instance, err = e.client.ProcessInstance.UpdateOne(instance).
		SetVariables(currentVars).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("更新实例变量失败: %w", err)
	}

	// 6. 执行流程推进（从当前UserTask继续）
	return e.executeStep(ctx, instance, process, task.TaskDefinitionKey, currentVars)
}

// executeStep 执行流程步骤
func (e *CustomProcessEngine) executeStep(ctx context.Context, instance *ent.ProcessInstance, process *BPMNProcess, currentElementID string, variables map[string]interface{}) error {
	outgoingFlows := e.findOutgoingFlows(process, currentElementID)

	if len(outgoingFlows) == 0 {
		if e.isEndEvent(process, currentElementID) {
			return e.completeProcess(ctx, instance)
		}
		return nil
	}

	var targetRef string
	for _, flow := range outgoingFlows {
		if e.evaluateCondition(flow, variables) {
			targetRef = flow.TargetRef
			break
		}
	}

	if targetRef == "" {
		return fmt.Errorf("没有符合条件的路径")
	}

	return e.handleElement(ctx, instance, process, targetRef)
}

func (e *CustomProcessEngine) handleElement(ctx context.Context, instance *ent.ProcessInstance, process *BPMNProcess, elementID string) error {
	// Find the element name for logging
	elementName := elementID
	if task := e.findUserTask(process, elementID); task != nil {
		elementName = task.Name
	} else if endEvent := e.findEndEvent(process, elementID); endEvent != nil {
		elementName = endEvent.Name
	}

	_, err := e.client.ProcessInstance.UpdateOne(instance).
		SetCurrentActivityID(elementID).
		SetCurrentActivityName(elementName).
		Save(ctx)
	if err != nil {
		return err
	}

	// Debug: log element info
	e.logger.Debugw("handleElement called", "elementID", elementID, "elementName", elementName, "userTasksCount", len(process.UserTasks))

	if task := e.findUserTask(process, elementID); task != nil {
		e.logger.Infow("Found user task, creating task", "taskID", task.ID, "taskName", task.Name)
		return e.createUserTask(ctx, instance, task)
	} else if endEvent := e.findEndEvent(process, elementID); endEvent != nil {
		return e.completeProcess(ctx, instance)
	} else if gateway := e.findExclusiveGateway(process, elementID); gateway != nil {
		return e.executeStep(ctx, instance, process, elementID, instance.Variables)
	} else if serviceTask := e.findServiceTask(process, elementID); serviceTask != nil {
		fmt.Printf("自动执行服务任务: %s\n", serviceTask.Name)
		return e.executeStep(ctx, instance, process, elementID, instance.Variables)
	}

	return e.executeStep(ctx, instance, process, elementID, instance.Variables)
}

func (e *CustomProcessEngine) createUserTask(ctx context.Context, instance *ent.ProcessInstance, task *BPMNUserTask) error {
	// 自动分配逻辑：优先级 BPMN定义 > 流程变量(request/assignee) > 默认分配
	assignee := task.Assignee

	// 辅助函数：从变量中提取用户ID
	getUserID := func(key string) string {
		if v, ok := instance.Variables[key]; ok {
			switch val := v.(type) {
			case float64:
				// JSON numbers are float64
				if val > 0 {
					return strconv.FormatFloat(val, 'f', 0, 64)
				}
			case int:
				if val > 0 {
					return strconv.Itoa(val)
				}
			case string:
				if val != "" && val != "0" {
					return val
				}
			}
		}
		return ""
	}

	// 如果BPMN没有定义分配人，从流程变量中获取
	if assignee == "" {
		// 优先使用 requester_id（工单申请人）
		assignee = getUserID("requester_id")
		// 其次使用 triggered_by（触发者）
		if assignee == "" {
			assignee = getUserID("triggered_by")
		}
		// 再其次使用 assignee_id
		if assignee == "" {
			assignee = getUserID("assignee_id")
		}
		// 如果还是没有，根据任务名称自动分配
		if assignee == "" {
			assignee = e.getDefaultAssigntee(ctx, instance, task)
		}
	}

	// Use instance.ID (auto-generated integer) for the relationship
	_, err := e.client.ProcessTask.Create().
		SetTaskID(fmt.Sprintf("TASK-%s-%d", task.ID, time.Now().UnixNano())).
		SetProcessInstanceID(instance.ID).
		SetProcessDefinitionKey(instance.ProcessDefinitionKey).
		SetTaskDefinitionKey(task.ID).
		SetTaskName(task.Name).
		SetTaskType("user_task").
		SetStatus("created").
		SetAssignee(assignee).
		SetCandidateUsers(task.CandidateUsers).
		SetCandidateGroups(task.CandidateGroups).
		SetTenantID(instance.TenantID).
		SetCreatedTime(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("创建用户任务失败: %w", err)
	}
	e.logger.Infow("User task created with auto-assignment", "taskID", task.ID, "taskName", task.Name, "assignee", assignee)
	return nil
}

// getDefaultAssigntee 根据任务类型和业务逻辑获取默认分配人
func (e *CustomProcessEngine) getDefaultAssigntee(ctx context.Context, instance *ent.ProcessInstance, task *BPMNUserTask) string {
	taskName := task.Name

	// 1. 审批类任务 - 尝试分配给管理员或安全审批人
	if strings.Contains(taskName, "审批") || strings.Contains(taskName, "审核") || strings.Contains(taskName, "批准") {
		// 查找有审批权限的用户 (角色为 admin 或 security)
		users, err := e.client.User.Query().
			Where(user.RoleIn("admin", "security")).
			Where(user.TenantID(instance.TenantID)).
			Where(user.Active(true)).
			Limit(1).
			All(ctx)
		if err == nil && len(users) > 0 {
			return strconv.Itoa(users[0].ID)
		}
	}

	// 2. 处理类任务 - 分配给工程师
	if strings.Contains(taskName, "处理") || strings.Contains(taskName, "执行") || strings.Contains(taskName, "处理") {
		users, err := e.client.User.Query().
			Where(user.RoleIn("engineer", "admin")).
			Where(user.TenantID(instance.TenantID)).
			Where(user.Active(true)).
			Limit(1).
			All(ctx)
		if err == nil && len(users) > 0 {
			return strconv.Itoa(users[0].ID)
		}
	}

	// 3. 默认分配 - 返回第一个活跃用户
	users, err := e.client.User.Query().
		Where(user.TenantID(instance.TenantID)).
		Where(user.Active(true)).
		Limit(1).
		All(ctx)
	if err == nil && len(users) > 0 {
		return strconv.Itoa(users[0].ID)
	}

	return ""
}

func (e *CustomProcessEngine) completeProcess(ctx context.Context, instance *ent.ProcessInstance) error {
	_, err := e.client.ProcessInstance.UpdateOne(instance).
		SetStatus("completed").
		SetEndTime(time.Now()).
		Save(ctx)
	return err
}

func (e *CustomProcessEngine) findOutgoingFlows(process *BPMNProcess, sourceRef string) []*BPMNSequenceFlow {
	var flows []*BPMNSequenceFlow
	for _, flow := range process.SequenceFlows {
		if flow.SourceRef == sourceRef {
			flows = append(flows, flow)
		}
	}
	return flows
}

// evaluateCondition 评估流转条件 (Domain Logic)
// 使用表达式引擎评估条件
func (e *CustomProcessEngine) evaluateCondition(flow *BPMNSequenceFlow, variables map[string]interface{}) bool {
	if flow.ConditionExpression == nil || flow.ConditionExpression.Expression == "" {
		return true // 无条件则默认通过
	}

	// 合并变量
	evalVars := make(map[string]interface{})
	for k, v := range e.expressionVars {
		evalVars[k] = v
	}
	for k, v := range variables {
		evalVars[k] = v
	}

	// 使用表达式引擎评估条件
	result, err := e.exprEngine.EvaluateCondition(flow.ConditionExpression.Expression, evalVars)
	if err != nil {
		e.logger.Warnw("Failed to evaluate condition, defaulting to true",
			"expression", flow.ConditionExpression.Expression,
			"error", err,
		)
		return true // 评估失败时默认通过
	}

	return result
}

func (e *CustomProcessEngine) isEndEvent(process *BPMNProcess, id string) bool {
	for _, event := range process.EndEvents {
		if event.ID == id {
			return true
		}
	}
	return false
}

func (e *CustomProcessEngine) findUserTask(process *BPMNProcess, id string) *BPMNUserTask {
	for _, task := range process.UserTasks {
		if task.ID == id {
			return task
		}
	}
	return nil
}

func (e *CustomProcessEngine) findEndEvent(process *BPMNProcess, id string) *BPMNEndEvent {
	for _, event := range process.EndEvents {
		if event.ID == id {
			return event
		}
	}
	return nil
}

func (e *CustomProcessEngine) findExclusiveGateway(process *BPMNProcess, id string) *BPMNExclusiveGateway {
	for _, gateway := range process.ExclusiveGateways {
		if gateway.ID == id {
			return gateway
		}
	}
	return nil
}

func (e *CustomProcessEngine) findServiceTask(process *BPMNProcess, id string) *BPMNServiceTask {
	for _, task := range process.ServiceTasks {
		if task.ID == id {
			return task
		}
	}
	return nil
}

func (e *CustomProcessEngine) SuspendProcess(ctx context.Context, processInstanceID string, reason string) error {
	return nil
}
func (e *CustomProcessEngine) ResumeProcess(ctx context.Context, processInstanceID string) error {
	return nil
}
func (e *CustomProcessEngine) TerminateProcess(ctx context.Context, processInstanceID string, reason string) error {
	return nil
}

// Request/Response structs
type CreateProcessDefinitionRequest struct {
	Key              string                 `json:"key" binding:"required"`
	Name             string                 `json:"name" binding:"required"`
	Description      string                 `json:"description"`
	Category         string                 `json:"category"`
	BPMNXML          string                 `json:"bpmn_xml" binding:"required"`
	ProcessVariables map[string]interface{} `json:"process_variables"`
	TenantID         int                    `json:"tenant_id" binding:"required"`
}

type UpdateProcessDefinitionRequest struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Category         string                 `json:"category"`
	BPMNXML          string                 `json:"bpmn_xml"`
	ProcessVariables map[string]interface{} `json:"process_variables"`
	IsActive         *bool                  `json:"is_active"`
}

type ListProcessDefinitionsRequest struct {
	Key      string `json:"key"`
	Category string `json:"category"`
	IsActive *bool  `json:"is_active"`
	TenantID int    `json:"tenant_id"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

type ListProcessInstancesRequest struct {
	ProcessDefinitionKey string `json:"process_definition_key"`
	Status               string `json:"status"`
	BusinessKey          string `json:"business_key"`
	TenantID             int    `json:"tenant_id"`
	Page                 int    `json:"page"`
	PageSize             int    `json:"page_size"`
}

type ListUserTasksRequest struct {
	Assignee             string `json:"assignee"`
	CandidateUsers       string `json:"candidate_users"`
	CandidateGroups      string `json:"candidate_groups"`
	Status               string `json:"status"`
	ProcessDefinitionKey string `json:"process_definition_key"`
	TenantID             int    `json:"tenant_id"`
	Page                 int    `json:"page"`
	PageSize             int    `json:"page_size"`
}

type TaskStatisticsRequest struct {
	ProcessDefinitionKey string     `json:"process_definition_key"`
	Assignee             string     `json:"assignee"`
	Status               string     `json:"status"`
	TenantID             int        `json:"tenant_id"`
	StartDate            *time.Time `json:"start_date"`
	EndDate              *time.Time `json:"end_date"`
}

type TaskStatistics struct {
	TotalTasks        int                    `json:"total_tasks"`
	CompletedTasks    int                    `json:"completed_tasks"`
	PendingTasks      int                    `json:"pending_tasks"`
	OverdueTasks      int                    `json:"overdue_tasks"`
	AverageCompletion float64                `json:"average_completion"`
	StatusBreakdown   map[string]int         `json:"status_breakdown"`
	AssigneeBreakdown map[string]int         `json:"assignee_breakdown"`
	TimeDistribution  map[string]interface{} `json:"time_distribution"`
}

// InstanceStatisticsRequest 实例统计请求
type InstanceStatisticsRequest struct {
	ProcessDefinitionKey string     `json:"process_definition_key"`
	Status               string     `json:"status"`
	TenantID             int        `json:"tenant_id"`
	StartDate            *time.Time `json:"start_date"`
	EndDate              *time.Time `json:"end_date"`
}

// InstanceStatistics 实例统计
type InstanceStatistics struct {
	Total      int            `json:"total"`
	Running    int            `json:"running"`
	Completed  int            `json:"completed"`
	Suspended  int            `json:"suspended"`
	Terminated int            `json:"terminated"`
}

// CounterSignStatus 会签状态
type CounterSignStatus struct {
	ParentTaskID string `json:"parent_task_id"`
	Total        int    `json:"total"`
	Completed    int    `json:"completed"`
	Approved     int    `json:"approved"`
	Rejected     int    `json:"rejected"`
	Pending      int    `json:"pending"`
	Status       string `json:"status"` // pending, approved, rejected
}

// CounterSignRequest 会签请求
type CounterSignRequest struct {
	ApprovalType string   `json:"approval_type"` // serial, parallel
	Approvers   []string `json:"approvers"`
	Threshold   int      `json:"threshold"`
}

// VoteRequest 投票请求
type VoteRequest struct {
	Approved bool   `json:"approved"`
	Comment  string `json:"comment"`
}

// Service implementations
type bpmnProcessDefinitionService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func (s *bpmnProcessDefinitionService) CreateProcessDefinition(ctx context.Context, req *CreateProcessDefinitionRequest) (*ent.ProcessDefinition, error) {
	// 首先检查或创建 ProcessDeployment
	var deployment *ent.ProcessDeployment
	existingDeployments, err := s.client.ProcessDeployment.Query().
		Where(processdeployment.TenantID(req.TenantID)).
		Order(ent.Desc("created_at")).
		Limit(1).
		All(ctx)

	if err == nil && len(existingDeployments) > 0 {
		deployment = existingDeployments[0]
	} else {
		// 创建新的部署记录
		deployment, err = s.client.ProcessDeployment.Create().
			SetDeploymentID(fmt.Sprintf("deploy-%d", time.Now().UnixNano())).
			SetDeploymentName(req.Name + "-deployment").
			SetDeploymentSource("api").
			SetTenantID(req.TenantID).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("创建部署记录失败: %w", err)
		}
	}

	// 获取当前最高版本号
	nextVersion := s.getNextVersion(ctx, req.Key, req.TenantID)

	// 将旧版本标记为非最新
	existing, err := s.client.ProcessDefinition.Query().
		Where(processdefinition.Key(req.Key)).
		Where(processdefinition.IsLatest(true)).
		Where(processdefinition.TenantID(req.TenantID)).
		First(ctx)

	if err == nil && existing != nil {
		_, err = s.client.ProcessDefinition.UpdateOne(existing).
			SetIsLatest(false).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("更新旧版本失败: %w", err)
		}
	}

	definition, err := s.client.ProcessDefinition.Create().
		SetKey(req.Key).
		SetName(req.Name).
		SetDescription(req.Description).
		SetCategory(req.Category).
		SetBpmnXML([]byte(req.BPMNXML)).
		SetProcessVariables(req.ProcessVariables).
		SetVersion(nextVersion).
		SetIsActive(true).
		SetIsLatest(true).
		SetTenantID(req.TenantID).
		SetDeploymentID(deployment.ID).
		SetDeploymentName(deployment.DeploymentName).
		Save(ctx)

	if err != nil {
		return nil, fmt.Errorf("创建流程定义失败: %w", err)
	}

	return definition, nil
}

// getNextVersion 获取下一个版本号
func (s *bpmnProcessDefinitionService) getNextVersion(ctx context.Context, key string, tenantID int) string {
	// 查询当前最高版本
	existing, err := s.client.ProcessDefinition.Query().
		Where(processdefinition.Key(key)).
		Where(processdefinition.TenantID(tenantID)).
		Order(ent.Desc("version")).
		First(ctx)

	if err != nil {
		// 没有版本，返回初始版本
		return "1.0.0"
	}

	// 解析当前版本号并递增
	currentVersion := existing.Version
	// 支持 v1.0.0, 1.0.0, 1 等格式
	versionNum := 1
	if currentVersion != "" {
		// 尝试提取数字部分
		var major, minor, patch int
		_, err := fmt.Sscanf(currentVersion, "%d.%d.%d", &major, &minor, &patch)
		if err != nil {
			// 如果解析失败，尝试只解析主版本号
			fmt.Sscanf(currentVersion, "%d", &major)
			versionNum = major
		} else {
			versionNum = major*100 + minor*10 + patch
		}
	}

	// 递增版本号 (使用语义化版本号)
	newMajor := versionNum / 100
	newMinor := (versionNum / 10) % 10
	newPatch := versionNum % 10

	// 如果patch已达9，重置并递增minor
	if newPatch >= 9 {
		newPatch = 0
		newMinor++
		if newMinor >= 9 {
			newMinor = 0
			newMajor++
		}
	} else {
		newPatch++
	}

	return fmt.Sprintf("%d.%d.%d", newMajor, newMinor, newPatch)
}

func (s *bpmnProcessDefinitionService) GetProcessDefinition(ctx context.Context, key string, version string) (*ent.ProcessDefinition, error) {
	definition, err := s.client.ProcessDefinition.Query().
		Where(processdefinition.Key(key)).
		Where(processdefinition.Version(version)).
		First(ctx)

	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	return definition, nil
}

// GetProcessDefinitionByID 根据ID获取流程定义
func (s *bpmnProcessDefinitionService) GetProcessDefinitionByID(ctx context.Context, id int) (*ent.ProcessDefinition, error) {
	definition, err := s.client.ProcessDefinition.Get(ctx, id)

	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	return definition, nil
}

func (s *bpmnProcessDefinitionService) GetLatestProcessDefinition(ctx context.Context, key string) (*ent.ProcessDefinition, error) {
	definition, err := s.client.ProcessDefinition.Query().
		Where(processdefinition.Key(key)).
		Where(processdefinition.IsLatest(true)).
		First(ctx)

	if err != nil {
		return nil, fmt.Errorf("获取最新流程定义失败: %w", err)
	}

	return definition, nil
}

func (s *bpmnProcessDefinitionService) UpdateProcessDefinition(ctx context.Context, key string, version string, req *UpdateProcessDefinitionRequest) (*ent.ProcessDefinition, error) {
	definition, err := s.GetProcessDefinition(ctx, key, version)
	if err != nil {
		return nil, err
	}

	update := s.client.ProcessDefinition.UpdateOne(definition)

	if req.Name != "" {
		update.SetName(req.Name)
	}
	if req.Description != "" {
		update.SetDescription(req.Description)
	}
	if req.Category != "" {
		update.SetCategory(req.Category)
	}
	if req.BPMNXML != "" {
		update.SetBpmnXML([]byte(req.BPMNXML))
	}
	if req.ProcessVariables != nil {
		update.SetProcessVariables(req.ProcessVariables)
	}
	if req.IsActive != nil {
		update.SetIsActive(*req.IsActive)
	}

	updated, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新流程定义失败: %w", err)
	}

	return updated, nil
}

func (s *bpmnProcessDefinitionService) DeleteProcessDefinition(ctx context.Context, key string, version string) error {
	definition, err := s.GetProcessDefinition(ctx, key, version)
	if err != nil {
		return err
	}

	return s.client.ProcessDefinition.DeleteOne(definition).Exec(ctx)
}

func (s *bpmnProcessDefinitionService) ListProcessDefinitions(ctx context.Context, req *ListProcessDefinitionsRequest) ([]*ent.ProcessDefinition, int, error) {
	query := s.client.ProcessDefinition.Query()

	if req.Key != "" {
		query = query.Where(processdefinition.Key(req.Key))
	}
	if req.Category != "" {
		query = query.Where(processdefinition.Category(req.Category))
	}
	if req.IsActive != nil {
		query = query.Where(processdefinition.IsActive(*req.IsActive))
	}
	if req.TenantID > 0 {
		query = query.Where(processdefinition.TenantID(req.TenantID))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取流程定义总数失败: %w", err)
	}

	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	definitions, err := query.Order(ent.Desc(processdefinition.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取流程定义列表失败: %w", err)
	}

	return definitions, total, nil
}

func (s *bpmnProcessDefinitionService) SetProcessDefinitionActive(ctx context.Context, key string, version string, active bool) error {
	definition, err := s.GetProcessDefinition(ctx, key, version)
	if err != nil {
		return err
	}

	_, err = s.client.ProcessDefinition.UpdateOne(definition).
		SetIsActive(active).
		Save(ctx)

	return err
}

type bpmnProcessInstanceService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func (s *bpmnProcessInstanceService) GetProcessInstance(ctx context.Context, processInstanceID string) (*ent.ProcessInstance, error) {
	instance, err := s.client.ProcessInstance.Query().
		Where(processinstance.ProcessInstanceID(processInstanceID)).
		First(ctx)

	if err != nil {
		return nil, fmt.Errorf("获取流程实例失败: %w", err)
	}

	return instance, nil
}

func (s *bpmnProcessInstanceService) ListProcessInstances(ctx context.Context, req *ListProcessInstancesRequest) ([]*ent.ProcessInstance, int, error) {
	query := s.client.ProcessInstance.Query()

	if req.ProcessDefinitionKey != "" {
		query = query.Where(processinstance.ProcessDefinitionKey(req.ProcessDefinitionKey))
	}
	if req.Status != "" {
		query = query.Where(processinstance.Status(req.Status))
	}
	if req.BusinessKey != "" {
		query = query.Where(processinstance.BusinessKey(req.BusinessKey))
	}
	if req.TenantID > 0 {
		query = query.Where(processinstance.TenantID(req.TenantID))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取流程实例总数失败: %w", err)
	}

	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	instances, err := query.Order(ent.Desc(processinstance.FieldStartTime)).All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取流程实例列表失败: %w", err)
	}

	return instances, total, nil
}

func (s *bpmnProcessInstanceService) GetProcessInstanceVariables(ctx context.Context, processInstanceID string) (map[string]interface{}, error) {
	instance, err := s.GetProcessInstance(ctx, processInstanceID)
	if err != nil {
		return nil, err
	}

	return instance.Variables, nil
}

func (s *bpmnProcessInstanceService) SetProcessInstanceVariables(ctx context.Context, processInstanceID string, variables map[string]interface{}) error {
	instance, err := s.GetProcessInstance(ctx, processInstanceID)
	if err != nil {
		return err
	}

	_, err = s.client.ProcessInstance.UpdateOne(instance).
		SetVariables(variables).
		Save(ctx)

	return err
}

func (s *bpmnProcessInstanceService) GetProcessInstanceHistory(ctx context.Context, processInstanceID string) ([]*ent.ProcessExecutionHistory, error) {
	return []*ent.ProcessExecutionHistory{}, nil
}

// GetInstanceStatistics 获取实例统计
func (s *bpmnProcessInstanceService) GetInstanceStatistics(ctx context.Context, req *InstanceStatisticsRequest) (*InstanceStatistics, error) {
	query := s.client.ProcessInstance.Query()

	if req.ProcessDefinitionKey != "" {
		query = query.Where(processinstance.ProcessDefinitionKey(req.ProcessDefinitionKey))
	}
	if req.TenantID > 0 {
		query = query.Where(processinstance.TenantID(req.TenantID))
	}
	if req.StartDate != nil {
		query = query.Where(processinstance.StartTimeGTE(*req.StartDate))
	}
	if req.EndDate != nil {
		query = query.Where(processinstance.StartTimeLTE(*req.EndDate))
	}

	instances, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取实例统计失败: %w", err)
	}

	stats := &InstanceStatistics{
		Total:      len(instances),
		Running:    0,
		Completed:  0,
		Suspended:  0,
		Terminated: 0,
	}

	for _, inst := range instances {
		switch inst.Status {
		case "running":
			stats.Running++
		case "completed":
			stats.Completed++
		case "suspended":
			stats.Suspended++
		case "terminated":
			stats.Terminated++
		}
	}

	// 如果有状态筛选，返回筛选后的统计
	if req.Status != "" {
		stats.Total = 0
		switch req.Status {
		case "running":
			stats.Total = stats.Running
		case "completed":
			stats.Total = stats.Completed
		case "suspended":
			stats.Total = stats.Suspended
		case "terminated":
			stats.Total = stats.Terminated
		}
	}

	return stats, nil
}

type bpmnTaskService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

func (s *bpmnTaskService) GetTask(ctx context.Context, taskID string) (*ent.ProcessTask, error) {
	task, err := s.client.ProcessTask.Query().
		Where(processtask.TaskID(taskID)).
		First(ctx)

	if err != nil {
		return nil, fmt.Errorf("获取任务失败: %w", err)
	}

	return task, nil
}

func (s *bpmnTaskService) ListUserTasks(ctx context.Context, req *ListUserTasksRequest) ([]*ent.ProcessTask, int, error) {
	s.logger.Debugw("ListUserTasks called", "assignee", req.Assignee, "tenantID", req.TenantID)
	query := s.client.ProcessTask.Query()

	// Debug: count all tasks
	totalAll, _ := s.client.ProcessTask.Query().Count(ctx)
	s.logger.Debugw("Total tasks in DB", "count", totalAll)

	if req.Assignee != "" {
		query = query.Where(processtask.Assignee(req.Assignee))
	}
	if req.CandidateUsers != "" {
		query = query.Where(processtask.CandidateUsersContains(req.CandidateUsers))
	}
	if req.CandidateGroups != "" {
		query = query.Where(processtask.CandidateGroupsContains(req.CandidateGroups))
	}
	if req.Status != "" {
		query = query.Where(processtask.Status(req.Status))
	}
	if req.ProcessDefinitionKey != "" {
		query = query.Where(processtask.ProcessDefinitionKey(req.ProcessDefinitionKey))
	}
	if req.TenantID > 0 {
		query = query.Where(processtask.TenantID(req.TenantID))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取任务总数失败: %w", err)
	}

	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	tasks, err := query.Order(ent.Desc(processtask.FieldCreatedTime)).All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("获取任务列表失败: %w", err)
	}

	return tasks, total, nil
}

func (s *bpmnTaskService) AssignTask(ctx context.Context, taskID string, assignee string) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	_, err = s.client.ProcessTask.UpdateOne(task).
		SetAssignee(assignee).
		SetStatus("assigned").
		SetAssignedTime(time.Now()).
		Save(ctx)

	return err
}

func (s *bpmnTaskService) CompleteTask(ctx context.Context, taskID string, variables map[string]interface{}) error {
	engine := NewCustomProcessEngine(s.client, s.logger)
	return engine.CompleteTask(ctx, taskID, variables)
}

func (s *bpmnTaskService) CancelTask(ctx context.Context, taskID string, reason string) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	_, err = s.client.ProcessTask.UpdateOne(task).
		SetStatus("cancelled").
		Save(ctx)

	return err
}

func (s *bpmnTaskService) GetTaskVariables(ctx context.Context, taskID string) (map[string]interface{}, error) {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return task.TaskVariables, nil
}

func (s *bpmnTaskService) SetTaskVariables(ctx context.Context, taskID string, variables map[string]interface{}) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	_, err = s.client.ProcessTask.UpdateOne(task).
		SetTaskVariables(variables).
		Save(ctx)

	return err
}

func (s *bpmnTaskService) HandleTaskTimeout(ctx context.Context, taskID string) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	if !task.DueDate.IsZero() && time.Now().After(task.DueDate) {
		_, err = s.client.ProcessTask.UpdateOne(task).
			SetStatus("timeout").
			Save(ctx)
		return err
	}

	return fmt.Errorf("任务未超时")
}

func (s *bpmnTaskService) RetryTask(ctx context.Context, taskID string, maxRetries int) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	retryCount := 0
	if task.TaskVariables != nil {
		if count, exists := task.TaskVariables["retry_count"]; exists {
			if countInt, ok := count.(float64); ok {
				retryCount = int(countInt)
			}
		}
	}

	if retryCount >= maxRetries {
		return fmt.Errorf("任务重试次数已达上限: %d", maxRetries)
	}

	if task.TaskVariables == nil {
		task.TaskVariables = make(map[string]interface{})
	}
	task.TaskVariables["retry_count"] = retryCount + 1
	task.TaskVariables["last_retry_time"] = time.Now().Format(time.RFC3339)

	_, err = s.client.ProcessTask.UpdateOne(task).
		SetStatus("pending").
		SetTaskVariables(task.TaskVariables).
		Save(ctx)

	return err
}

func (s *bpmnTaskService) DelegateTask(ctx context.Context, taskID string, newAssignee string) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	if task.TaskVariables == nil {
		task.TaskVariables = make(map[string]interface{})
	}
	task.TaskVariables["delegated_from"] = task.Assignee
	task.TaskVariables["delegated_time"] = time.Now().Format(time.RFC3339)

	_, err = s.client.ProcessTask.UpdateOne(task).
		SetAssignee(newAssignee).
		SetStatus("delegated").
		SetTaskVariables(task.TaskVariables).
		Save(ctx)

	return err
}

func (s *bpmnTaskService) EscalateTask(ctx context.Context, taskID string, reason string) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	if task.TaskVariables == nil {
		task.TaskVariables = make(map[string]interface{})
	}
	task.TaskVariables["escalation_reason"] = reason
	task.TaskVariables["escalated_time"] = time.Now().Format(time.RFC3339)

	_, err = s.client.ProcessTask.UpdateOne(task).
		SetStatus("escalated").
		SetTaskVariables(task.TaskVariables).
		Save(ctx)

	return err
}

func (s *bpmnTaskService) BatchAssignTasks(ctx context.Context, taskIDs []string, assignee string) error {
	if len(taskIDs) == 0 {
		return fmt.Errorf("任务ID列表为空")
	}

	_, err := s.client.ProcessTask.Update().
		Where(processtask.TaskIDIn(taskIDs...)).
		SetAssignee(assignee).
		SetStatus("assigned").
		SetAssignedTime(time.Now()).
		Save(ctx)

	return err
}

func (s *bpmnTaskService) GetTaskStatistics(ctx context.Context, req *TaskStatisticsRequest) (*TaskStatistics, error) {
	query := s.client.ProcessTask.Query()

	if req.ProcessDefinitionKey != "" {
		query = query.Where(processtask.ProcessDefinitionKey(req.ProcessDefinitionKey))
	}
	if req.Assignee != "" {
		query = query.Where(processtask.Assignee(req.Assignee))
	}
	if req.Status != "" {
		query = query.Where(processtask.Status(req.Status))
	}
	if req.TenantID > 0 {
		query = query.Where(processtask.TenantID(req.TenantID))
	}
	if req.StartDate != nil {
		query = query.Where(processtask.CreatedTimeGTE(*req.StartDate))
	}
	if req.EndDate != nil {
		query = query.Where(processtask.CreatedTimeLTE(*req.EndDate))
	}

	tasks, err := query.All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取任务统计信息失败: %w", err)
	}

	stats := &TaskStatistics{
		TotalTasks:        len(tasks),
		StatusBreakdown:   make(map[string]int),
		AssigneeBreakdown: make(map[string]int),
		TimeDistribution:  make(map[string]interface{}),
	}

	var totalCompletionTime time.Duration
	completedCount := 0

	for _, task := range tasks {
		stats.StatusBreakdown[task.Status]++

		if task.Assignee != "" {
			stats.AssigneeBreakdown[task.Assignee]++
		}

		if task.Status == "completed" && !task.CompletedTime.IsZero() && !task.AssignedTime.IsZero() {
			completionTime := task.CompletedTime.Sub(task.AssignedTime)
			totalCompletionTime += completionTime
			completedCount++
		}

		if !task.DueDate.IsZero() && time.Now().After(task.DueDate) && task.Status != "completed" {
			stats.OverdueTasks++
		}
	}

	if completedCount > 0 {
		stats.AverageCompletion = float64(totalCompletionTime.Milliseconds()) / float64(completedCount)
	}

	stats.CompletedTasks = stats.StatusBreakdown["completed"]
	stats.PendingTasks = stats.StatusBreakdown["pending"] + stats.StatusBreakdown["assigned"]

	return stats, nil
}

// CreateCounterSignTasks 创建会签子任务
func (s *bpmnTaskService) CreateCounterSignTasks(ctx context.Context, parentTaskID string, req *CounterSignRequest) ([]*ent.ProcessTask, error) {
	// 获取父任务
	parentTask, err := s.client.ProcessTask.Query().
		Where(processtask.TaskID(parentTaskID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取父任务失败: %w", err)
	}

	// 生成根任务ID（如果是第一个会签任务）
	rootTaskID := parentTaskID
	if parentTask.RootTaskID != "" {
		rootTaskID = parentTask.RootTaskID
	}

	threshold := req.Threshold
	if threshold == 0 {
		threshold = len(req.Approvers)
	}

	var tasks []*ent.ProcessTask
	for i, approver := range req.Approvers {
		taskID := fmt.Sprintf("%s_countersign_%d", parentTaskID, i)
		task, err := s.client.ProcessTask.Create().
			SetTaskID(taskID).
			SetProcessInstanceID(parentTask.ProcessInstanceID).
			SetProcessDefinitionKey(parentTask.ProcessDefinitionKey).
			SetTaskDefinitionKey(parentTask.TaskDefinitionKey + "_counter").
			SetTaskName(parentTask.TaskName + "_会签").
			SetTaskType("user_task").
			SetAssignee(approver).
			SetStatus("assigned").
			SetPriority(parentTask.Priority).
			SetParentTaskID(parentTaskID).
			SetRootTaskID(rootTaskID).
			SetTenantID(parentTask.TenantID).
			SetCreatedTime(time.Now()).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("创建会签任务失败: %w", err)
		}
		tasks = append(tasks, task)
	}

	// 更新父任务状态为会签中
	_, err = s.client.ProcessTask.UpdateOneID(parentTask.ID).
		SetTaskVariables(map[string]interface{}{
			"approval_type": req.ApprovalType,
			"threshold":     threshold,
			"total":          len(req.Approvers),
			"completed":      0,
			"approved":       0,
			"rejected":       0,
		}).
		Save(ctx)
	if err != nil {
		s.logger.Warnf("更新父任务变量失败: %v", err)
	}

	return tasks, nil
}

// GetCounterSignStatus 获取会签状态
func (s *bpmnTaskService) GetCounterSignStatus(ctx context.Context, parentTaskID string) (*CounterSignStatus, error) {
	// 获取所有会签子任务
	subTasks, err := s.client.ProcessTask.Query().
		Where(processtask.ParentTaskID(parentTaskID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取会签子任务失败: %w", err)
	}

	status := &CounterSignStatus{
		ParentTaskID: parentTaskID,
		Total:        len(subTasks),
		Completed:    0,
		Approved:     0,
		Rejected:     0,
		Pending:      len(subTasks),
		Status:       "pending",
	}

	for _, task := range subTasks {
		switch task.Status {
		case "completed":
			status.Completed++
			status.Pending--
			// 检查审批结果
			if vars := task.TaskVariables; vars != nil {
				if approved, ok := vars["approved"].(bool); ok && approved {
					status.Approved++
				} else {
					status.Rejected++
				}
			}
		case "assigned", "created":
			// still pending
		}
	}

	// 确定会签整体状态
	if status.Rejected > 0 {
		status.Status = "rejected"
	} else if status.Completed == status.Total {
		status.Status = "approved"
	} else {
		status.Status = "pending"
	}

	return status, nil
}

// Vote 投票（完成会签任务）
func (s *bpmnTaskService) Vote(ctx context.Context, taskID string, req *VoteRequest) error {
	task, err := s.client.ProcessTask.Query().
		Where(processtask.TaskID(taskID)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}

	// 更新任务状态为完成
	_, err = s.client.ProcessTask.UpdateOneID(task.ID).
		SetStatus("completed").
		SetCompletedTime(time.Now()).
		SetTaskVariables(map[string]interface{}{
			"approved": req.Approved,
			"comment":  req.Comment,
		}).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("完成任务失败: %w", err)
	}

	// 获取会签状态
	parentTaskID := task.ParentTaskID
	if parentTaskID == "" {
		return nil // 没有父任务，不需要检查会签状态
	}

	status, err := s.GetCounterSignStatus(ctx, parentTaskID)
	if err != nil {
		return fmt.Errorf("获取会签状态失败: %w", err)
	}

	// 根据会签类型和阈值判断是否需要终止其他任务
	parentTask, err := s.client.ProcessTask.Query().
		Where(processtask.TaskID(parentTaskID)).
		First(ctx)
	if err != nil {
		return nil
	}

	vars := parentTask.TaskVariables
	if vars == nil {
		vars = make(map[string]interface{})
	}
	threshold := 1
	if t, ok := vars["threshold"].(int); ok {
		threshold = t
	}
	approvalType := "parallel"
	if at, ok := vars["approval_type"].(string); ok {
		approvalType = at
	}

	// 并行会签：一人拒绝则全部拒绝
	if approvalType == "parallel" && req.Approved == false {
		// 取消其他未完成的会签任务
		subTasks, _ := s.client.ProcessTask.Query().
			Where(processtask.ParentTaskID(parentTaskID)).
			Where(processtask.StatusNEQ("completed")).
			All(ctx)
		for _, st := range subTasks {
			s.client.ProcessTask.UpdateOneID(st.ID).
				SetStatus("cancelled").
				SetCompletedTime(time.Now()).
				Exec(ctx)
		}
	}

	// 检查是否达到阈值
	if status.Status == "approved" || status.Status == "rejected" {
		// 更新父任务
		s.client.ProcessTask.UpdateOneID(parentTask.ID).
			SetTaskVariables(map[string]interface{}{
				"approval_type": approvalType,
				"threshold":     threshold,
				"total":         status.Total,
				"completed":     status.Completed,
				"approved":      status.Approved,
				"rejected":      status.Rejected,
				"final_status":  status.Status,
			}).
			Exec(ctx)
	}

	return nil
}
