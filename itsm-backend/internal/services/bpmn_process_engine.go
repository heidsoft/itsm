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

// ProcessDefinitionService 是 dto.ProcessDefinitionCatalog 的别名：ProcessEngine 三个访问器的返回类型
// 必须与域内窄契约完全一致（Go 要求命名类型相同），接口定义因此收敛到 dto。
type ProcessDefinitionService = dto.ProcessDefinitionCatalog

// ProcessInstanceService 是 dto.ProcessInstanceCatalog 的别名：ProcessEngine 三个访问器的返回类型
// 必须与域内窄契约完全一致（Go 要求命名类型相同），接口定义因此收敛到 dto。
type ProcessInstanceService = dto.ProcessInstanceCatalog

// TaskService 是 dto.TaskCatalog 的别名：ProcessEngine 三个访问器的返回类型
// 必须与域内窄契约完全一致（Go 要求命名类型相同），接口定义因此收敛到 dto。
type TaskService = dto.TaskCatalog

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
	// 定时器事件支持
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
	engine.taskService = &bpmnTaskService{client: client, logger: logger, groupResolver: engine.groupResolver, engine: engine}
	engine.auditService = NewBPMNAuditService(client, logger)

	// 注册流程相关的内置函数
	engine.registerProcessFunctions()

	return engine
}

// SetTimerServices injects timer store and scheduler for runtime timer event registration.
// Called after engine construction in app.go wiring phase.
// SetTimerServices 注入 TimerStore 与调度器；同时把 store 传播给流程定义服务，
// 使发布/停用流程定义时能同步 start timer 时间表。
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

// ProcessDefinitionService 返回流程定义服务。
// 返回引擎内已接线的实例：SetTimerServices 注入的 timerStore 只存在于该实例上，
// 新建副本会让发布/停用定义时的 start timer 同步静默失效。
func (e *CustomProcessEngine) ProcessDefinitionService() ProcessDefinitionService {
	if e.processDefinitionService == nil {
		return &bpmnProcessDefinitionService{client: e.client, logger: e.logger}
	}
	return e.processDefinitionService
}

// ProcessInstanceService 返回流程实例服务
func (e *CustomProcessEngine) ProcessInstanceService() ProcessInstanceService {
	if e.processInstanceService == nil {
		return &bpmnProcessInstanceService{client: e.client, logger: e.logger}
	}
	return e.processInstanceService
}

// TaskService 返回任务服务。返回已接线实例，任务完成经由所属引擎复用定时器设施。
func (e *CustomProcessEngine) TaskService() TaskService {
	if e.taskService == nil {
		return &bpmnTaskService{client: e.client, logger: e.logger, groupResolver: e.groupResolver}
	}
	return e.taskService
}

// SetApprovalService 注入 ApprovalService，解决循环依赖
func (e *CustomProcessEngine) SetApprovalService(svc bpmn.ApprovalServiceInterface) {
	e.callbackRegistry.SetApprovalService(svc)
}

// SetApprovalChainResolver 注入审批链求值器，解决循环依赖
func (e *CustomProcessEngine) SetApprovalChainResolver(resolver bpmn.ApprovalChainResolver) {
	e.callbackRegistry.SetApprovalChainResolver(resolver)
}

// requireBPMNTenantContext 从 ctx 强制提取并校验租户上下文。
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
