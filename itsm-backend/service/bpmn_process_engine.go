package service

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/processtask"
	"itsm-backend/service/bpmn"

	"go.uber.org/zap"
)

// ProcessEngine 流程引擎接口
type ProcessEngine interface {
	// 流程定义管理
	ProcessDefinitionService() ProcessDefinitionService
	// 流程实例管理
	ProcessInstanceService() ProcessInstanceService
	// 任务管理
	TaskService() TaskService
	// 注入 ApprovalService（解决审批处理器循环依赖）
	SetApprovalService(svc bpmn.ApprovalServiceInterface)
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
	PublishProcessDefinition(ctx context.Context, key string, version string, req *UpdateProcessDefinitionRequest) (*ent.ProcessDefinition, error)
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
	GetTaskByID(ctx context.Context, id int) (*ent.ProcessTask, error)
	CompleteTaskByID(ctx context.Context, id int, variables map[string]interface{}) error
	ClaimTask(ctx context.Context, taskID string, userID string) error
	ClaimTaskByID(ctx context.Context, id int, userID int) error
	ListUserTasks(ctx context.Context, req *ListUserTasksRequest) ([]*ent.ProcessTask, int, error)
	ListUserTaskViews(ctx context.Context, req *ListUserTasksRequest) ([]*dto.BPMNTaskResponse, int, error)
	AssignTask(ctx context.Context, taskID string, assignee string) error
	ReassignTask(ctx context.Context, taskID string, newAssigneeID int, reason string) error
	TerminateTask(ctx context.Context, taskID string, reason string) error
	CompleteTask(ctx context.Context, taskID string, variables map[string]interface{}) error
	CancelTask(ctx context.Context, taskID string, reason string) error
	GetTaskVariables(ctx context.Context, taskID string) (map[string]interface{}, error)
	SetTaskVariables(ctx context.Context, taskID string, variables map[string]interface{}) error
	HandleTaskTimeout(ctx context.Context, taskID string) error
	RetryTask(ctx context.Context, taskID string, maxRetries int) error
	DelegateTask(ctx context.Context, taskID string, newAssignee string) error
	DelegateTaskByID(ctx context.Context, id int, newAssignee string) error
	AddApproverTask(ctx context.Context, taskID string, newApprover string) error
	AddApproverTaskByID(ctx context.Context, id int, newApprover string) error
	EscalateTask(ctx context.Context, taskID string, reason string) error
	BatchAssignTasks(ctx context.Context, taskIDs []string, assignee string, tenantID int) error
	GetTaskStatistics(ctx context.Context, req *TaskStatisticsRequest) (*TaskStatistics, error)
	ListApprovalDecisions(ctx context.Context, processInstanceKey string) ([]*ent.ProcessApprovalDecision, error)
	// 会签相关
	CreateCounterSignTasks(ctx context.Context, parentTaskID string, req *CounterSignRequest) ([]*ent.ProcessTask, error)
	GetCounterSignStatus(ctx context.Context, parentTaskID string) (*CounterSignStatus, error)
	Vote(ctx context.Context, taskID string, req *VoteRequest) error
}

// CustomProcessEngine 是ProcessEngine接口的实现
// 充当领域服务(Domain Service)，协调流程定义、实例和任务实体的生命周期
type CustomProcessEngine struct {
	client           *ent.Client
	logger           *zap.SugaredLogger
	parser           *BPMNParser            // 使用自定义的BPMN解析器
	exprEngine       *ExpressionEngine      // 表达式引擎
	expressionVars   map[string]interface{} // 表达式变量
	callbackRegistry *bpmn.CallbackRegistry // 服务任务回调注册中心
	groupResolver    *bpmn.GroupResolver    // 审批组解析器：candidateGroups → 候选用户
	// 内部服务
	processDefinitionService *bpmnProcessDefinitionService
	processInstanceService   *bpmnProcessInstanceService
	taskService              *bpmnTaskService
	// 审计服务
	auditService *BPMNAuditService
	// 定时器事件支持（Phase 4: 运行时注册）
	timerStore     TimerStore
	timerScheduler *TimerScheduler
}

// NewCustomProcessEngine 创建自定义流程引擎实例
func NewCustomProcessEngine(client *ent.Client, logger *zap.SugaredLogger) ProcessEngine {
	engine := &CustomProcessEngine{
		client:           client,
		logger:           logger,
		parser:           NewBPMNParser(),
		exprEngine:       NewExpressionEngine(),
		expressionVars:   make(map[string]interface{}),
		callbackRegistry: bpmn.NewCallbackRegistry(client, logger),
		groupResolver:    bpmn.NewGroupResolver(client),
	}
	engine.processDefinitionService = &bpmnProcessDefinitionService{client: client, logger: logger}
	engine.processInstanceService = &bpmnProcessInstanceService{client: client, logger: logger}
	engine.taskService = &bpmnTaskService{client: client, logger: logger, groupResolver: engine.groupResolver}
	engine.auditService = NewBPMNAuditService(client, logger)

	// 注册流程相关的内置函数
	engine.registerProcessFunctions()

	return engine
}

// SetTimerServices injects timer store and scheduler for runtime timer event registration.
// Called after engine construction in app.go wiring phase.
// SetTimerServices 注入 TimerStore 与调度器；同时把 store 传播给流程定义服务，
// 使发布/停用流程定义时能同步 start timer 时间表（Phase 5）。
func (e *CustomProcessEngine) SetTimerServices(store TimerStore, scheduler *TimerScheduler) {
	e.timerStore = store
	e.timerScheduler = scheduler
	if e.processDefinitionService != nil {
		e.processDefinitionService.SetTimerStore(store)
	}
}

// registerProcessFunctions 注册流程相关的内置函数
func (e *CustomProcessEngine) registerProcessFunctions() {
	// 获取任务列表
	e.exprEngine.RegisterFunction("getTasks", func(ctx context.Context, assignee string) []interface{} {
		// 从数据库查询任务
		tasks, err := e.client.ProcessTask.Query().
			Where(processtask.Assignee(assignee)).
			Where(processtask.CompletedTimeIsNil()).
			All(ctx)
		if err != nil {
			e.logger.Warnw("Failed to query tasks", "error", err)
			return []interface{}{}
		}
		result := make([]interface{}, len(tasks))
		for i, task := range tasks {
			result[i] = map[string]interface{}{
				"id":          task.TaskID,
				"name":        task.TaskName,
				"instance_id": task.ProcessInstanceID,
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
	return &bpmnTaskService{client: e.client, logger: e.logger, groupResolver: e.groupResolver}
}

// SetApprovalService 注入 ApprovalService，解决循环依赖
func (e *CustomProcessEngine) SetApprovalService(svc bpmn.ApprovalServiceInterface) {
	e.callbackRegistry.SetApprovalService(svc)
}

// requireBPMNTenantContext 从 ctx 强制提取并校验租户上下文（P1-4 fail-closed）。
// 无租户上下文或 tenantID<=0 一律返回错误，禁止跨租户回退到全局无过滤查询。
func requireBPMNTenantContext(ctx context.Context) (int, error) {
	if ctx == nil {
		return 0, fmt.Errorf("BPMN 缺少请求上下文")
	}
	tenantID, _ := ctx.Value(bpmn.BPMNTenantIDContextKey).(int)
	if tenantID <= 0 {
		return 0, fmt.Errorf("BPMN 缺少有效租户上下文（tenantID=%d），已 fail-closed", tenantID)
	}
	return tenantID, nil
}
