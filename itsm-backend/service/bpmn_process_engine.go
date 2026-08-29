package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/predicate"
	"itsm-backend/ent/processapprovaldecision"
	"itsm-backend/ent/processdefinition"
	"itsm-backend/ent/processdeployment"
	"itsm-backend/ent/processexecutionhistory"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
	"itsm-backend/ent/ticketassignmentrule"
	"itsm-backend/ent/user"
	"itsm-backend/ent/workflowtask"
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

// StartProcess 启动流程实例
func (e *CustomProcessEngine) StartProcess(ctx context.Context, processDefinitionKey string, businessKey string, variables map[string]interface{}) (*ent.ProcessInstance, error) {
	// P1-4：租户上下文必须显式有效（>0），禁止缺上下文时全局查流程定义。
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}

	// 2. 获取流程定义（固定使用当前租户过滤，不再存在"无过滤跨租户"分支）
	definition, err := e.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(processDefinitionKey),
			processdefinition.IsActive(true),
			processdefinition.IsLatest(true),
			processdefinition.TenantID(tenantID),
		).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	// 3. 解析BPMN
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
	if err := e.executeStep(ctx, e.client, instance, process, startEvent.ID, variables); err != nil {
		return nil, err
	}

	// 6. 记录审计日志 - 流程启动
	// 从context中获取用户信息
	userID := 0
	userName := ""
	if u, ok := ctx.Value("user").(*ent.User); ok {
		userID = u.ID
		userName = u.Name
	}
	if err := e.auditService.RecordProcessStarted(ctx, instance, userID, userName, variables); err != nil {
		e.logger.Warnw("audit record failed", "error", err)
	}

	return instance, nil
}

// CompleteTask 完成任务。
// 整个「置完成 → 合并变量 → 推进流程 → 记录审批决策」包进单个 ent.Tx，
// 任一步骤失败整体回滚，彻底消除并发下的半成品状态（P2 事务原子化）。
func (e *CustomProcessEngine) CompleteTask(ctx context.Context, taskID string, variables map[string]interface{}) error {
	// P1-4：租户上下文必须显式有效（>0），禁止缺上下文时全局查任务
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return err
	}

	tx, err := e.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	txc := tx.Client()
	// 异常时回滚，保证不会留下半提交状态
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	// 1. 获取任务（固定按当前租户过滤，不存在"无过滤跨租户"分支）
	task, err := txc.ProcessTask.Query().
		Where(processtask.TaskID(taskID), processtask.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("获取任务失败: %w", err)
	}
	if err := e.authorizeTaskActor(ctx, task); err != nil {
		_ = tx.Rollback()
		return err
	}

	// 2. 获取流程实例 - 使用任务中存储的ProcessInstanceID (ent自动生成的ID)
	instance, err := txc.ProcessInstance.Get(ctx, task.ProcessInstanceID)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("获取流程实例失败: %w", err)
	}

	// 3. 获取流程定义并解析
	// A running instance is immutable with respect to its deployed definition.
	// Looking up the latest definition here can silently move an old instance
	// onto a newly published graph halfway through execution.
	definition, err := txc.ProcessDefinition.Query().
		Where(
			processdefinition.ID(instance.ProcessDefinitionID),
			processdefinition.TenantID(instance.TenantID),
		).
		Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("获取流程定义失败: %w", err)
	}

	bpmnDefinitions, err := e.parser.ParseXML(definition.BpmnXML)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("解析BPMN失败: %w", err)
	}
	process := bpmnDefinitions.Processes[0]

	// 4. 更新当前任务状态
	if task.Status == "completed" || task.Status == "cancelled" {
		_ = tx.Rollback()
		return fmt.Errorf("任务已结束，不能重复完成")
	}

	updated, err := txc.ProcessTask.Update().
		Where(
			processtask.ID(task.ID),
			processtask.TenantID(instance.TenantID),
			processtask.StatusNEQ("completed"),
			processtask.StatusNEQ("cancelled"),
		).
		SetStatus("completed").
		SetCompletedTime(time.Now()).
		SetTaskVariables(variables).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("更新任务状态失败: %w", err)
	}
	if updated != 1 {
		_ = tx.Rollback()
		return fmt.Errorf("任务已被处理，请刷新后重试")
	}

	// 5. 在事务内合并变量（无并发写者，直接合并即可）
	instance, err = e.mergeVariablesInTx(ctx, txc, instance.ID, variables)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("合并实例变量失败: %w", err)
	}

	// 6. 执行流程推进（从当前UserTask继续）。审批拒绝策略属于节点
	// 运行语义，不能只停留在设计器配置中。
	rejectStrategy, _ := task.TaskVariables["rejectStrategy"].(string)
	approvalAction, _ := variables["approvalAction"].(string)
	if approvalAction == "reject" && rejectStrategy == "terminate" {
		if _, err = txc.ProcessInstance.UpdateOneID(instance.ID).
			SetStatus("terminated").SetEndTime(time.Now()).Save(ctx); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("终止被拒绝流程失败: %w", err)
		}
		if _, err = txc.ProcessTask.Update().Where(
			processtask.ProcessInstanceID(instance.ID), processtask.TenantID(instance.TenantID),
			processtask.StatusNEQ("completed"), processtask.StatusNEQ("cancelled"),
		).SetStatus("cancelled").SetCompletedTime(time.Now()).Save(ctx); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("取消被拒绝流程的剩余任务失败: %w", err)
		}
	} else if err := e.executeStep(ctx, txc, instance, process, task.TaskDefinitionKey, instance.Variables); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := e.recordApprovalDecision(ctx, txc, instance, task, variables); err != nil {
		_ = tx.Rollback()
		return err
	}

	// 7. 提交事务；任一步骤失败已在上方回滚
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	// 8. 记录审计日志 - 任务完成（事务外，仅审计，失败不阻断）
	userID := 0
	userName := ""
	if u, ok := ctx.Value("user").(*ent.User); ok {
		userID = u.ID
		userName = u.Name
	}
	variablesBefore := task.TaskVariables
	if err := e.auditService.RecordTaskCompleted(ctx, task, userID, userName, variablesBefore, variables); err != nil {
		e.logger.Warnw("audit record failed", "error", err)
	}

	return nil
}

// mergeVariablesInTx 在事务内合并流程实例变量并返回更新后的实例。
// 与 mergeVariablesWithOptimisticLock 不同：调用方已持有事务，无需再开事务或重试。
func (e *CustomProcessEngine) mergeVariablesInTx(ctx context.Context, txc *ent.Client, instanceID int, newVars map[string]interface{}) (*ent.ProcessInstance, error) {
	inst, err := txc.ProcessInstance.Get(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("查询流程实例失败: %w", err)
	}
	merged := make(map[string]interface{})
	for k, v := range inst.Variables {
		merged[k] = v
	}
	for k, v := range newVars {
		merged[k] = v
	}
	updated, err := txc.ProcessInstance.UpdateOneID(instanceID).
		SetVariables(merged).
		SetVersion(inst.Version + 1).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新实例变量失败: %w", err)
	}
	return updated, nil
}

func (e *CustomProcessEngine) recordApprovalDecision(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, task *ent.ProcessTask, variables map[string]interface{}) error {
	action, _ := variables["approvalAction"].(string)
	if action == "" {
		return nil
	}
	decision, _ := variables["approvalResult"].(string)
	comment, _ := variables["approvalComment"].(string)
	actorID, _ := ctx.Value(bpmn.BPMNUserIDContextKey).(int)
	if actorID <= 0 {
		return fmt.Errorf("审批决策缺少认证操作人")
	}
	actorName := ""
	if actor, err := txc.User.Get(ctx, actorID); err == nil {
		actorName = actor.Name
	}
	businessType := fmt.Sprint(instance.Variables["business_type"])
	businessID := fmt.Sprint(instance.Variables["business_id"])
	_, err := txc.ProcessApprovalDecision.Create().
		SetProcessInstanceID(instance.ID).SetProcessTaskID(task.ID).
		SetProcessInstanceKey(instance.ProcessInstanceID).SetTaskID(task.TaskID).
		SetProcessDefinitionKey(instance.ProcessDefinitionKey).SetNodeKey(task.TaskDefinitionKey).
		SetBusinessType(businessType).SetBusinessID(businessID).
		SetActorID(actorID).SetActorName(actorName).SetAction(action).SetDecision(decision).
		SetComment(comment).SetVariablesSnapshot(variables).SetTenantID(instance.TenantID).Save(ctx)
	if err != nil {
		return fmt.Errorf("记录审批决策失败: %w", err)
	}
	return nil
}

// authorizeTaskActor ensures that task actions are performed by the assigned
// user or an explicitly resolved candidate. System/internal calls without an
// authenticated actor keep their existing behavior.
func (e *CustomProcessEngine) authorizeTaskActor(ctx context.Context, task *ent.ProcessTask) error {
	userID, _ := ctx.Value(bpmn.BPMNUserIDContextKey).(int)
	if userID <= 0 {
		return nil
	}
	actor, err := e.client.User.Query().Where(user.ID(userID)).Only(ctx)
	if err != nil {
		return fmt.Errorf("审批用户不存在: %w", err)
	}
	allowed := func(csv string) bool {
		for _, candidate := range strings.Split(csv, ",") {
			candidate = strings.TrimSpace(candidate)
			if candidate == strconv.Itoa(userID) || candidate == actor.Username {
				return true
			}
		}
		return false
	}
	if allowed(task.Assignee) || allowed(task.CandidateUsers) {
		return nil
	}
	return fmt.Errorf("当前用户不是该任务的审批人或候选人")
}

// executeStep 执行流程步骤
func (e *CustomProcessEngine) executeStep(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, process *BPMNProcess, currentElementID string, variables map[string]interface{}) error {
	outgoingFlows := e.findOutgoingFlows(process, currentElementID)

	if len(outgoingFlows) == 0 {
		if e.isEndEvent(process, currentElementID) {
			return e.completeProcess(ctx, txc, instance)
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

	// 记录排他网关路由决策，使「走哪条分支」可审计（F-4）
	if e.findExclusiveGateway(process, currentElementID) != nil {
		e.recordGatewayHistory(ctx, txc, instance, currentElementID, "exclusive", "fork", []string{targetRef}, variables)
	}

	return e.handleElement(ctx, txc, instance, process, targetRef)
}

func (e *CustomProcessEngine) handleElement(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, process *BPMNProcess, elementID string) error {
	// Find the element name for logging
	elementName := elementID
	if task := e.findUserTask(process, elementID); task != nil {
		elementName = task.Name
	} else if endEvent := e.findEndEvent(process, elementID); endEvent != nil {
		elementName = endEvent.Name
	}

	_, err := txc.ProcessInstance.UpdateOne(instance).
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
		return e.createUserTask(ctx, txc, instance, task)
	} else if endEvent := e.findEndEvent(process, elementID); endEvent != nil {
		e.markElementDone(ctx, txc, instance, elementID)
		return e.completeProcess(ctx, txc, instance)
	} else if gateway := e.findParallelGateway(process, elementID); gateway != nil {
		// 并行网关：分叉激活所有出边；汇聚等待所有入边分支完成（F-1）
		return e.handleParallelGateway(ctx, txc, instance, process, gateway, 0)
	} else if gateway := e.findInclusiveGateway(process, elementID); gateway != nil {
		// 包容网关：分叉激活所有命中条件的出边；汇聚等待所有入边分支完成（F-1）
		return e.handleInclusiveGateway(ctx, txc, instance, process, gateway, 0)
	} else if gateway := e.findExclusiveGateway(process, elementID); gateway != nil {
		e.markElementDone(ctx, txc, instance, elementID)
		return e.executeStep(ctx, txc, instance, process, elementID, instance.Variables)
	} else if serviceTask := e.findServiceTask(process, elementID); serviceTask != nil {
		// 通过 CallbackRegistry 执行真实的服务任务逻辑
		serviceRef := serviceTask.ID
		if serviceTask.Name != "" {
			serviceRef = serviceTask.Name
		}
		// 尝试通过实现类或表达式属性获取服务引用
		if serviceTask.Implementation != "" {
			serviceRef = serviceTask.Implementation
		} else if serviceTask.Class != "" {
			serviceRef = serviceTask.Class
		} else if serviceTask.DelegateExpression != "" {
			serviceRef = serviceTask.DelegateExpression
		} else if serviceTask.OperationRef != "" {
			serviceRef = serviceTask.OperationRef
		}

		// 查找并执行 Callback
		if e.callbackRegistry != nil {
			handler := e.callbackRegistry.GetHandler(serviceRef)
			if handler == nil {
				// 尝试按任务类型匹配
				handler = e.callbackRegistry.GetHandler(serviceTask.GetType())
			}
			if handler != nil {
				e.logger.Infow("执行 ServiceTask 回调", "serviceRef", serviceRef, "elementID", elementID)
				taskVariables := mergeServiceTaskVariables(instance.Variables, serviceTask)
				if _, err := handler.Execute(ctx, nil, taskVariables); err != nil {
					return fmt.Errorf("ServiceTask %s 执行失败: %w", serviceRef, err)
				}
			} else {
				return fmt.Errorf("ServiceTask handler '%s' 未注册，无法执行此自动化步骤", serviceRef)
			}
		}
		e.markElementDone(ctx, txc, instance, elementID)
		return e.executeStep(ctx, txc, instance, process, elementID, instance.Variables)
	}

	e.markElementDone(ctx, txc, instance, elementID)
	return e.executeStep(ctx, txc, instance, process, elementID, instance.Variables)
}

func mergeServiceTaskVariables(instanceVariables map[string]interface{}, task *BPMNServiceTask) map[string]interface{} {
	variables := make(map[string]interface{}, len(instanceVariables)+12)
	for key, value := range instanceVariables {
		variables[key] = value
	}
	if task == nil {
		return variables
	}
	if task.Type != "" {
		variables["type"] = task.Type
	}
	if task.OperationRef != "" {
		variables["operationRef"] = task.OperationRef
	}
	if task.CCType != "" {
		variables["ccType"] = task.CCType
	}
	if task.CCUserIDs != "" {
		variables["ccUserIds"] = task.CCUserIDs
	}
	if task.CCGroupIDs != "" {
		variables["ccGroupIds"] = task.CCGroupIDs
	}
	if task.CCRoleIDs != "" {
		variables["ccRoleIds"] = task.CCRoleIDs
	}
	if task.CCVariable != "" {
		variables["ccVariable"] = task.CCVariable
	}
	if task.CCNotify != "" {
		variables["ccNotify"] = task.CCNotify
	}
	if task.NotifyChannels != "" {
		variables["notifyChannels"] = task.NotifyChannels
	}
	return variables
}

func (e *CustomProcessEngine) createUserTask(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, task *BPMNUserTask) error {
	// 幂等：同实例同节点已存在未结束任务时直接复用，避免 CompleteTask 流程推进失败重试时重复创建任务（F-2）
	if existing, _ := e.client.ProcessTask.Query().
		Where(
			processtask.ProcessInstanceID(instance.ID),
			processtask.TaskDefinitionKey(task.ID),
			processtask.StatusNotIn("completed", "cancelled"),
		).
		First(ctx); existing != nil {
		e.logger.Infow("createUserTask: 复用已存在的活跃任务", "existingTaskID", existing.TaskID, "node", task.ID)
		return nil
	}

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

	// 展开 candidateGroups 为具体用户，合并到 candidate_users。
	// 这样「我的待办」接口才有可能查到分配给我的任务。
	expandedCandidateUsers := task.CandidateUsers
	if e.groupResolver != nil && strings.TrimSpace(task.CandidateGroups) != "" {
		_, groupUsernames, err := e.groupResolver.ExpandGroupsToUsers(ctx, instance.TenantID, task.CandidateGroups)
		if err != nil {
			// 解析失败：记录警告但不阻塞流程，以免审批组配置漂移导致整个流程中断
			e.logger.Warnw(
				"审批组展开失败，继续仅使用 BPMN candidateUsers",
				"taskID", task.ID,
				"candidateGroups", task.CandidateGroups,
				"error", err,
			)
		} else {
			expandedCandidateUsers = e.groupResolver.MergeCandidateUsers(task.CandidateUsers, groupUsernames)
			e.logger.Infow(
				"审批组已展开",
				"taskID", task.ID,
				"candidateGroups", task.CandidateGroups,
				"expandedUsers", groupUsernames,
			)
		}
	}

	// Use instance.ID (auto-generated integer) for the relationship
	taskConfig := map[string]interface{}{
		"taskPurpose": task.TaskPurpose, "approvalMode": task.ApprovalMode,
		"approvalThreshold": task.ApprovalThreshold, "rejectStrategy": task.RejectStrategy,
		"timeoutAction": task.TimeoutAction, "allowDelegate": task.AllowDelegate,
		"allowAddApprover":        task.AllowAddApprover,
		"commentRequiredOnReject": task.CommentRequiredOnReject,
	}
	createdTask, err := txc.ProcessTask.Create().
		SetTaskID(fmt.Sprintf("TASK-%s-%d", task.ID, time.Now().UnixNano())).
		SetProcessInstanceID(instance.ID).
		SetProcessDefinitionKey(instance.ProcessDefinitionKey).
		SetTaskDefinitionKey(task.ID).
		SetTaskName(task.Name).
		SetTaskType("user_task").
		SetStatus("created").
		SetAssignee(assignee).
		SetCandidateUsers(expandedCandidateUsers).
		SetCandidateGroups(task.CandidateGroups).
		SetFormKey(task.FormKey).
		SetTaskVariables(taskConfig).
		SetTenantID(instance.TenantID).
		SetCreatedTime(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("创建用户任务失败: %w", err)
	}
	if task.TaskPurpose == "approval" && task.ApprovalMode != "" && task.ApprovalMode != "single" {
		approvers := splitNonEmptyCSV(expandedCandidateUsers)
		if len(approvers) > 1 {
			threshold := task.ApprovalThreshold
			switch task.ApprovalMode {
			case "any":
				threshold = 1
			case "all", "sequential":
				threshold = len(approvers)
			}
			approvalType := "parallel"
			if task.ApprovalMode == "sequential" {
				approvalType = "serial"
			}
			if _, err := createCounterSignTasksWithClient(ctx, txc, createdTask.TaskID, instance.TenantID, &CounterSignRequest{ApprovalType: approvalType, Approvers: approvers, Threshold: threshold}); err != nil {
				return fmt.Errorf("创建会签任务失败: %w", err)
			}
		}
	}
	e.logger.Infow("User task created with auto-assignment", "taskID", task.ID, "taskName", task.Name, "assignee", assignee)
	return nil
}

func splitNonEmptyCSV(value string) []string {
	var values []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}

// getDefaultAssigntee 根据任务类型和业务逻辑获取默认分配人
// 优先级：1.流程变量显式指定 > 2.数据库规则匹配 > 3.中文关键词兜底（deprecated）
func (e *CustomProcessEngine) getDefaultAssigntee(ctx context.Context, instance *ent.ProcessInstance, task *BPMNUserTask) string {
	taskName := task.Name

	// 第一优先：流程变量中显式指定 assignee
	if instance.Variables != nil {
		if assignee, ok := instance.Variables["assignee"]; ok {
			switch val := assignee.(type) {
			case float64:
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
	}

	// 第二优先：数据库 ticket_assignment_rules 规则匹配
	if assigneeFromRule := e.getAssigneeFromDBRules(ctx, instance, taskName); assigneeFromRule != "" {
		return assigneeFromRule
	}

	// 第三优先：中文关键词兜底（deprecated，未来版本将移除）
	// 审批类任务 - 尝试分配给管理员或安全审批人
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

	// 处理类任务 - 分配给工程师
	if strings.Contains(taskName, "处理") || strings.Contains(taskName, "执行") {
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

	// 默认分配 - 返回第一个活跃用户
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

// getAssigneeFromDBRules 从数据库 ticket_assignment_rules 表查询匹配的分配规则
func (e *CustomProcessEngine) getAssigneeFromDBRules(ctx context.Context, instance *ent.ProcessInstance, taskName string) string {
	// 查询当前租户下所有激活的分配规则，按优先级降序排列
	rules, err := e.client.TicketAssignmentRule.Query().
		Where(
			ticketassignmentrule.TenantID(instance.TenantID),
			ticketassignmentrule.IsActive(true),
		).
		Order(ent.Desc(ticketassignmentrule.FieldPriority)).
		All(ctx)
	if err != nil {
		e.logger.Warnw("查询分配规则失败", "error", err)
		return ""
	}

	// 在内存中匹配规则条件
	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}
		// 检查条件是否匹配任务名称
		if matchRuleConditions(rule.Conditions, taskName) {
			// 从 actions 中提取 assignee
			if assigneeVal, ok := rule.Actions["assignee_id"]; ok {
				switch v := assigneeVal.(type) {
				case float64:
					if v > 0 {
						return strconv.FormatFloat(v, 'f', 0, 64)
					}
				case int:
					if v > 0 {
						return strconv.Itoa(v)
					}
				case string:
					if v != "" && v != "0" {
						return v
					}
				}
			}
		}
	}

	return ""
}

// matchRuleConditions 检查规则条件是否与任务名称匹配
func matchRuleConditions(conditions []map[string]interface{}, taskName string) bool {
	if len(conditions) == 0 {
		return false
	}
	for _, cond := range conditions {
		field, _ := cond["field"].(string)
		operator, _ := cond["operator"].(string)
		value, _ := cond["value"].(string)

		if field != "task_name" {
			continue
		}

		switch operator {
		case "equals":
			if taskName == value {
				return true
			}
		case "contains":
			if strings.Contains(taskName, value) {
				return true
			}
		case "prefix":
			if strings.HasPrefix(taskName, value) {
				return true
			}
		case "suffix":
			if strings.HasSuffix(taskName, value) {
				return true
			}
		}
	}
	return false
}

func (e *CustomProcessEngine) completeProcess(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance) error {
	_, err := txc.ProcessInstance.UpdateOne(instance).
		SetStatus("completed").
		SetEndTime(time.Now()).
		Save(ctx)
	return err
}

// recordGatewayHistory 将网关路由决策写入流程执行历史，使并行/包容/排他网关的
// 分叉（fork）与汇聚等待（join-wait）可审计（F-4）。字段与 bpmn_gateway_engine.go 的
// recordGatewayExecution 同构，GetGatewayExecutionHistory 可按 activity_type=gateway 检索。
// 写历史属于辅助审计，失败仅告警不阻断主流程。
func (e *CustomProcessEngine) recordGatewayHistory(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, gatewayID, gatewayType, eventType string, nextActivities []string, variables map[string]interface{}) {
	detail := map[string]interface{}{
		"gateway_id":      gatewayID,
		"gateway_type":    gatewayType,
		"event":           eventType,
		"next_activities": nextActivities,
		"tenant_id":       instance.TenantID,
	}
	detailBytes, err := json.Marshal(detail)
	if err != nil {
		e.logger.Warnw("recordGatewayHistory 序列化事件详情失败", "error", err)
		detailBytes = []byte("{}")
	}
	_, err = txc.ProcessExecutionHistory.Create().
		SetHistoryID(fmt.Sprintf("HIST-%s-%d", gatewayID, time.Now().UnixNano())).
		SetProcessInstanceID(instance.ID).
		SetProcessDefinitionKey(instance.ProcessDefinitionKey).
		SetActivityID(gatewayID).
		SetActivityType("gateway").
		SetEventType(fmt.Sprintf("%s.%s", gatewayType, eventType)).
		SetEventDetail(string(detailBytes)).
		SetVariables(variables).
		SetTenantID(instance.TenantID).
		SetTimestamp(time.Now()).
		Save(ctx)
	if err != nil {
		e.logger.Warnw("recordGatewayHistory 保存失败", "error", err)
	}
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

	// 兼容模板中常见的 ${...} 包裹语法（expr-lang 无法编译 ${} 前缀）
	expr := strings.TrimSpace(flow.ConditionExpression.Expression)
	if strings.HasPrefix(expr, "${") && strings.HasSuffix(expr, "}") {
		expr = strings.TrimSpace(expr[2 : len(expr)-1])
	}
	if expr == "" {
		return true // 剥离包裹后为空视为无条件
	}

	// 合并变量
	evalVars := make(map[string]interface{})
	for k, v := range e.expressionVars {
		evalVars[k] = v
	}
	for k, v := range variables {
		evalVars[k] = v
	}
	// 兼容模板中 variables['xxx'] 的字典访问语法
	evalVars["variables"] = variables

	// 使用表达式引擎评估条件
	result, err := e.exprEngine.EvaluateCondition(expr, evalVars)
	if err != nil {
		// SEC-002 修复：评估失败时默认拒绝（return false），而非放行
		e.logger.Errorw(
			"条件评估失败，默认拒绝流转",
			"expression", expr,
			"error", err,
		)
		return false
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

func (e *CustomProcessEngine) findParallelGateway(process *BPMNProcess, id string) *BPMNParallelGateway {
	for _, gateway := range process.ParallelGateways {
		if gateway.ID == id {
			return gateway
		}
	}
	return nil
}

func (e *CustomProcessEngine) findInclusiveGateway(process *BPMNProcess, id string) *BPMNInclusiveGateway {
	for _, gateway := range process.InclusiveGateways {
		if gateway.ID == id {
			return gateway
		}
	}
	return nil
}

// findIncomingFlows 返回以 targetRef 为目标的顺序流（即 targetRef 的入边）
func (e *CustomProcessEngine) findIncomingFlows(process *BPMNProcess, targetRef string) []*BPMNSequenceFlow {
	var flows []*BPMNSequenceFlow
	for _, flow := range process.SequenceFlows {
		if flow.TargetRef == targetRef {
			flows = append(flows, flow)
		}
	}
	return flows
}

const maxGatewayForkDepth = 64

// handleParallelGateway 处理并行网关：多入边时作为汇聚节点等待所有分支完成；否则作为分叉节点激活所有出边（F-1）。
func (e *CustomProcessEngine) handleParallelGateway(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, process *BPMNProcess, gateway *BPMNParallelGateway, depth int) error {
	if depth > maxGatewayForkDepth {
		return fmt.Errorf("并行网关分叉层级超过上限，可能存在环路: %s", gateway.ID)
	}
	incoming := e.findIncomingFlows(process, gateway.ID)
	if len(incoming) > 1 && !e.allIncomingBranchesCompleted(ctx, instance, process, gateway.ID) {
		e.logger.Infow("并行网关汇聚等待其余分支完成", "gateway", gateway.ID, "instance", instance.ID)
		e.recordGatewayHistory(ctx, txc, instance, gateway.ID, "parallel", "join-wait", nil, instance.Variables)
		return nil // 仍有分支未结束，等待，不推进
	}
	var next []string
	for _, flow := range e.findOutgoingFlows(process, gateway.ID) {
		next = append(next, flow.TargetRef)
		if err := e.dispatchGatewayOrElement(ctx, txc, instance, process, flow.TargetRef, depth); err != nil {
			return err
		}
	}
	e.markElementDone(ctx, txc, instance, gateway.ID)
	e.recordGatewayHistory(ctx, txc, instance, gateway.ID, "parallel", "fork", next, instance.Variables)
	return nil
}

// handleInclusiveGateway 处理包容网关：汇聚等待所有入边分支完成；分叉时激活所有命中条件的出边（F-1）。
func (e *CustomProcessEngine) handleInclusiveGateway(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, process *BPMNProcess, gateway *BPMNInclusiveGateway, depth int) error {
	if depth > maxGatewayForkDepth {
		return fmt.Errorf("包容网关分叉层级超过上限，可能存在环路: %s", gateway.ID)
	}
	incoming := e.findIncomingFlows(process, gateway.ID)
	if len(incoming) > 1 && !e.allIncomingBranchesCompleted(ctx, instance, process, gateway.ID) {
		e.logger.Infow("包容网关汇聚等待其余分支完成", "gateway", gateway.ID, "instance", instance.ID)
		e.recordGatewayHistory(ctx, txc, instance, gateway.ID, "inclusive", "join-wait", nil, instance.Variables)
		return nil
	}
	var next []string
	matched := false
	for _, flow := range e.findOutgoingFlows(process, gateway.ID) {
		if e.evaluateCondition(flow, instance.Variables) {
			matched = true
			next = append(next, flow.TargetRef)
			if err := e.dispatchGatewayOrElement(ctx, txc, instance, process, flow.TargetRef, depth); err != nil {
				return err
			}
		}
	}
	e.markElementDone(ctx, txc, instance, gateway.ID)
	e.recordGatewayHistory(ctx, txc, instance, gateway.ID, "inclusive", "fork", next, instance.Variables)
	if !matched {
		e.logger.Warnw("包容网关无满足条件的出边，流程在此等待", "gateway", gateway.ID, "instance", instance.ID)
	}
	return nil
}

// dispatchGatewayOrElement 优先分发到嵌套的并行/包容网关（depth+1 用于环路防护），其余交给 handleElement。
func (e *CustomProcessEngine) dispatchGatewayOrElement(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, process *BPMNProcess, elementID string, depth int) error {
	if pg := e.findParallelGateway(process, elementID); pg != nil {
		return e.handleParallelGateway(ctx, txc, instance, process, pg, depth+1)
	}
	if ig := e.findInclusiveGateway(process, elementID); ig != nil {
		return e.handleInclusiveGateway(ctx, txc, instance, process, ig, depth+1)
	}
	return e.handleElement(ctx, txc, instance, process, elementID)
}

// markElementDone 标记某流程元素已执行完成（写入实例变量 _done_ 并持久化）。
// 用于并行/包容网关汇聚时判断「非用户任务源」分支（服务任务、子网关、排他网关、结束事件）
// 是否已真正结束——这是修复「汇聚只认 user-task 源而漏掉其他分支导致提前汇聚/死锁」的关键。
// 直接就地修改 instance.Variables（同一 map 引用会在同一次执行内对汇聚判断可见），
// 并通过 txc 持久化，使后续 CompleteTask 的汇聚判断也能读到。
func (e *CustomProcessEngine) markElementDone(ctx context.Context, txc *ent.Client, instance *ent.ProcessInstance, elementID string) {
	if instance.Variables == nil {
		instance.Variables = map[string]interface{}{}
	}
	done, ok := instance.Variables["_done_"].(map[string]interface{})
	if !ok {
		done = map[string]interface{}{}
		instance.Variables["_done_"] = done
	}
	done[elementID] = true
	if _, err := txc.ProcessInstance.UpdateOneID(instance.ID).SetVariables(instance.Variables).Save(ctx); err != nil {
		e.logger.Warnw("markElementDone 持久化失败", "elementID", elementID, "error", err)
	}
}

// allIncomingBranchesCompleted 判断汇聚网关的所有入边分支是否均已结束。
//   - 以用户任务为源的分支：按 DB 中该任务是否已 completed/cancelled 判断（与历史行为一致）。
//   - 以服务任务/子网关/排他网关/结束事件为源的分支：必须已在 _done_ 中标记为完成。
//     旧实现对这些非用户任务源直接 continue（永远视为完成），会导致提前汇聚或死锁（P1 网关完整性）。
//   - 查询失败时保守返回 false（视为未完成），避免提前汇聚。
func (e *CustomProcessEngine) allIncomingBranchesCompleted(ctx context.Context, instance *ent.ProcessInstance, process *BPMNProcess, gatewayID string) bool {
	done := map[string]interface{}{}
	if instance.Variables != nil {
		if d, ok := instance.Variables["_done_"].(map[string]interface{}); ok {
			done = d
		}
	}
	for _, flow := range e.findIncomingFlows(process, gatewayID) {
		src := flow.SourceRef
		if e.findUserTask(process, src) != nil {
			// 用户任务源：以 DB 实际完成状态为准
			open, err := e.client.ProcessTask.Query().
				Where(
					processtask.ProcessInstanceID(instance.ID),
					processtask.TaskDefinitionKey(src),
					processtask.StatusNotIn("completed", "cancelled"),
				).
				Exist(ctx)
			if err != nil {
				e.logger.Warnw("allIncomingBranchesCompleted 查询失败，保守视为未完成", "error", err)
				return false
			}
			if open {
				return false
			}
			continue
		}
		// 服务任务/子网关/排他网关/结束事件源：必须已记录完成
		if v, ok := done[src]; !ok || v != true {
			e.logger.Infow("并行/包容网关汇聚等待分支完成", "gateway", gatewayID, "waitingSrc", src)
			return false
		}
	}
	return true
}

// DetectStuckInstances 返回运行时间超过 olderThan 仍未结束（running）的流程实例，
// 用于可观测性与卡死检测（P3）。结合日志中的 "并行/包容网关汇聚等待分支完成" 可定位汇聚死锁。
func (e *CustomProcessEngine) DetectStuckInstances(ctx context.Context, tenantID int, olderThan time.Time) ([]*ent.ProcessInstance, error) {
	if tenantID <= 0 {
		return nil, fmt.Errorf("DetectStuckInstances 需要有效的租户上下文")
	}
	return e.client.ProcessInstance.Query().
		Where(
			processinstance.TenantID(tenantID),
			processinstance.Status("running"),
			processinstance.StartTimeLT(olderThan),
		).
		All(ctx)
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
	// P1-4：租户上下文必须显式有效（>0），fail-closed
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return err
	}
	// 1. 获取流程实例（固定按当前租户过滤）
	instance, err := e.client.ProcessInstance.Query().
		Where(processinstance.ProcessInstanceID(processInstanceID), processinstance.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("获取流程实例失败: %w", err)
	}

	// 2. 更新实例状态
	_, err = e.client.ProcessInstance.UpdateOne(instance).
		SetStatus("suspended").
		SetSuspendedTime(time.Now()).
		SetSuspendedReason(reason).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("暂停流程实例失败: %w", err)
	}

	// 3. 记录审计日志
	userID := 0
	userName := ""
	if u, ok := ctx.Value("user").(*ent.User); ok {
		userID = u.ID
		userName = u.Name
	}
	if err := e.auditService.RecordAudit(ctx, &AuditContext{
		ProcessInstanceID:    instance.ID,
		ProcessInstanceKey:   instance.ProcessInstanceID,
		ProcessDefinitionKey: instance.ProcessDefinitionKey,
		ProcessDefinitionID:  instance.ProcessDefinitionID,
		ActivityID:           instance.CurrentActivityID,
		ActivityName:         instance.CurrentActivityName,
		ActivityType:         ActivityTypeUserTask,
		Action:               AuditActionProcessSuspended,
		UserID:               userID,
		UserName:             userName,
		Comment:              reason,
		TenantID:             instance.TenantID,
	}); err != nil {
		e.logger.Warnw("audit record failed", "error", err)
	}

	return nil
}

func (e *CustomProcessEngine) ResumeProcess(ctx context.Context, processInstanceID string) error {
	// P1-4：租户上下文必须显式有效（>0），fail-closed
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return err
	}
	// 1. 获取流程实例（固定按当前租户过滤）
	instance, err := e.client.ProcessInstance.Query().
		Where(processinstance.ProcessInstanceID(processInstanceID), processinstance.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("获取流程实例失败: %w", err)
	}

	// 2. 更新实例状态
	_, err = e.client.ProcessInstance.UpdateOne(instance).
		SetStatus("running").
		Save(ctx)
	if err != nil {
		return fmt.Errorf("恢复流程实例失败: %w", err)
	}

	// 3. 记录审计日志
	userID := 0
	userName := ""
	if u, ok := ctx.Value("user").(*ent.User); ok {
		userID = u.ID
		userName = u.Name
	}
	if err := e.auditService.RecordAudit(ctx, &AuditContext{
		ProcessInstanceID:    instance.ID,
		ProcessInstanceKey:   instance.ProcessInstanceID,
		ProcessDefinitionKey: instance.ProcessDefinitionKey,
		ProcessDefinitionID:  instance.ProcessDefinitionID,
		ActivityID:           instance.CurrentActivityID,
		ActivityName:         instance.CurrentActivityName,
		ActivityType:         ActivityTypeUserTask,
		Action:               AuditActionProcessResumed,
		UserID:               userID,
		UserName:             userName,
		TenantID:             instance.TenantID,
	}); err != nil {
		e.logger.Warnw("audit record failed", "error", err)
	}

	return nil
}

func (e *CustomProcessEngine) TerminateProcess(ctx context.Context, processInstanceID string, reason string) error {
	// P1-4：租户上下文必须显式有效（>0），fail-closed
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return err
	}
	// 1. 获取流程实例（固定按当前租户过滤）
	instance, err := e.client.ProcessInstance.Query().
		Where(processinstance.ProcessInstanceID(processInstanceID), processinstance.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("获取流程实例失败: %w", err)
	}

	// 2. 更新实例状态
	_, err = e.client.ProcessInstance.UpdateOne(instance).
		SetStatus("terminated").
		SetEndTime(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("终止流程实例失败: %w", err)
	}

	// 3. 取消所有进行中的任务
	_, err = e.client.ProcessTask.Update().
		Where(processtask.ProcessInstanceID(instance.ID)).
		Where(processtask.StatusNEQ("completed")).
		Where(processtask.StatusNEQ("cancelled")).
		SetStatus("cancelled").
		SetCompletedTime(time.Now()).
		Save(ctx)
	if err != nil {
		e.logger.Warnw("取消流程任务失败", "error", err)
	}

	// 4. 记录审计日志
	userID := 0
	userName := ""
	if u, ok := ctx.Value("user").(*ent.User); ok {
		userID = u.ID
		userName = u.Name
	}
	if err := e.auditService.RecordAudit(ctx, &AuditContext{
		ProcessInstanceID:    instance.ID,
		ProcessInstanceKey:   instance.ProcessInstanceID,
		ProcessDefinitionKey: instance.ProcessDefinitionKey,
		ProcessDefinitionID:  instance.ProcessDefinitionID,
		ActivityID:           instance.CurrentActivityID,
		ActivityName:         instance.CurrentActivityName,
		ActivityType:         ActivityTypeEndEvent,
		Action:               AuditActionProcessTerminated,
		UserID:               userID,
		UserName:             userName,
		Comment:              reason,
		TenantID:             instance.TenantID,
	}); err != nil {
		e.logger.Warnw("audit record failed", "error", err)
	}

	return nil
}

// Request/Response structs
type CreateProcessDefinitionRequest struct {
	Key              string                 `json:"key" binding:"required"`
	Name             string                 `json:"name" binding:"required"`
	Description      string                 `json:"description"`
	Category         string                 `json:"category"`
	BPMNXML          string                 `json:"bpmnXml" binding:"required"`
	ProcessVariables map[string]interface{} `json:"processVariables"`
	TenantID         int                    `json:"tenantId" binding:"required"`
	Publish          bool                   `json:"publish"`
}

type UpdateProcessDefinitionRequest struct {
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Category         string                 `json:"category"`
	BPMNXML          string                 `json:"bpmnXml"`
	ProcessVariables map[string]interface{} `json:"processVariables"`
	IsActive         *bool                  `json:"isActive"`
}

type ListProcessDefinitionsRequest struct {
	Key      string `json:"key"`
	Category string `json:"category"`
	IsActive *bool  `json:"isActive"`
	TenantID int    `json:"tenantId"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}

type ListProcessInstancesRequest struct {
	ProcessDefinitionKey string `json:"processDefinitionKey"`
	Status               string `json:"status"`
	BusinessKey          string `json:"businessKey"`
	TenantID             int    `json:"tenantId"`
	Page                 int    `json:"page"`
	PageSize             int    `json:"pageSize"`
}

type ListUserTasksRequest struct {
	Assignee        string `json:"assignee"`
	CandidateUsers  string `json:"candidateUsers"`
	CandidateGroups string `json:"candidateGroups"`
	// UserID 为「我的待办」语义：查询“分配给我 OR 我在候选人 OR 我所在组作为候选组”的任务。
	// 传入后：Assignee/CandidateUsers/CandidateGroups 会被忽略（可选透传）。
	UserID int `json:"userId"`
	// AllTasks is an internal authorization decision made by the controller after
	// the task:admin middleware succeeds. It must never be populated from HTTP input.
	AllTasks             bool   `json:"-" form:"-"`
	Status               string `json:"status"`
	ProcessDefinitionKey string `json:"processDefinitionKey"`
	ProcessInstanceID    int    `json:"processInstanceId"`
	TenantID             int    `json:"tenantId"`
	Page                 int    `json:"page"`
	PageSize             int    `json:"pageSize"`
}

type TaskStatisticsRequest struct {
	ProcessDefinitionKey string     `json:"processDefinitionKey"`
	Assignee             string     `json:"assignee"`
	Status               string     `json:"status"`
	TenantID             int        `json:"tenantId"`
	StartDate            *time.Time `json:"startDate"`
	EndDate              *time.Time `json:"endDate"`
}

type TaskStatistics struct {
	TotalTasks        int                    `json:"totalTasks"`
	CompletedTasks    int                    `json:"completedTasks"`
	PendingTasks      int                    `json:"pendingTasks"`
	OverdueTasks      int                    `json:"overdueTasks"`
	AverageCompletion float64                `json:"averageCompletion"`
	StatusBreakdown   map[string]int         `json:"statusBreakdown"`
	AssigneeBreakdown map[string]int         `json:"assigneeBreakdown"`
	TimeDistribution  map[string]interface{} `json:"timeDistribution"`
}

// InstanceStatisticsRequest 实例统计请求
type InstanceStatisticsRequest struct {
	ProcessDefinitionKey string     `json:"processDefinitionKey"`
	Status               string     `json:"status"`
	TenantID             int        `json:"tenantId"`
	StartDate            *time.Time `json:"startDate"`
	EndDate              *time.Time `json:"endDate"`
}

// InstanceStatistics 实例统计
type InstanceStatistics struct {
	Total      int `json:"total"`
	Running    int `json:"running"`
	Completed  int `json:"completed"`
	Suspended  int `json:"suspended"`
	Terminated int `json:"terminated"`
}

// CounterSignStatus 会签状态
type CounterSignStatus struct {
	ParentTaskID string `json:"parentTaskId"`
	Total        int    `json:"total"`
	Completed    int    `json:"completed"`
	Approved     int    `json:"approved"`
	Rejected     int    `json:"rejected"`
	Pending      int    `json:"pending"`
	Status       string `json:"status"` // pending, approved, rejected
}

// CounterSignRequest 会签请求
type CounterSignRequest struct {
	ApprovalType string   `json:"approvalType"` // serial, parallel
	Approvers    []string `json:"approvers"`
	Threshold    int      `json:"threshold"`
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
	if _, err := NewBPMNParser().ParseXML([]byte(req.BPMNXML)); err != nil {
		return nil, fmt.Errorf("BPMN XML 校验失败: %w", err)
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开始流程定义事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	client := tx.Client()
	// 首先检查或创建 ProcessDeployment
	var deployment *ent.ProcessDeployment
	existingDeployments, err := client.ProcessDeployment.Query().
		Where(processdeployment.TenantID(req.TenantID)).
		Order(ent.Desc("created_at")).
		Limit(1).
		All(ctx)

	if err == nil && len(existingDeployments) > 0 {
		deployment = existingDeployments[0]
	} else {
		// 创建新的部署记录
		deployment, err = client.ProcessDeployment.Create().
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
	nextVersion := s.getNextVersionWithClient(ctx, client, req.Key, req.TenantID)

	// 将旧版本标记为非最新
	existing, err := client.ProcessDefinition.Query().
		Where(processdefinition.Key(req.Key)).
		Where(processdefinition.IsLatest(true)).
		Where(processdefinition.TenantID(req.TenantID)).
		First(ctx)

	if err == nil && existing != nil {
		_, err = client.ProcessDefinition.UpdateOne(existing).
			SetIsLatest(false).
			Save(ctx)
		if err != nil {
			return nil, fmt.Errorf("更新旧版本失败: %w", err)
		}
	}

	definition, err := client.ProcessDefinition.Create().
		SetKey(req.Key).
		SetName(req.Name).
		SetDescription(req.Description).
		SetCategory(req.Category).
		SetBpmnXML([]byte(req.BPMNXML)).
		SetProcessVariables(req.ProcessVariables).
		SetVersion(nextVersion).
		SetIsActive(req.Publish).
		SetIsLatest(true).
		SetTenantID(req.TenantID).
		SetDeploymentID(deployment.ID).
		SetDeploymentName(deployment.DeploymentName).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("创建流程定义失败: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交流程定义事务失败: %w", err)
	}
	return definition, nil
}

// getNextVersion 获取下一个版本号（major.minor.0）。约定与部署服务一致：递增 minor、patch 归零、minor 不封顶。
// 旧实现用 Order(Desc("version")) 对字符串版本号做字典序排序，多位数版本（如 1.10.0 vs 1.9.0）会被误判大小；
// 此处改为在 Go 侧解析取最大 minor 后递增，避免字典序陷阱。
func (s *bpmnProcessDefinitionService) getNextVersion(ctx context.Context, key string, tenantID int) string {
	return s.getNextVersionWithClient(ctx, s.client, key, tenantID)
}

func (s *bpmnProcessDefinitionService) getNextVersionWithClient(ctx context.Context, client *ent.Client, key string, tenantID int) string {
	defs, err := client.ProcessDefinition.Query().
		Where(processdefinition.Key(key)).
		Where(processdefinition.TenantID(tenantID)).
		All(ctx)
	if err != nil || len(defs) == 0 {
		return "1.0.0"
	}

	maxMaj, maxMin := 0, -1
	for _, d := range defs {
		maj, min, _ := parseSemver(d.Version)
		if maj > maxMaj || (maj == maxMaj && min > maxMin) {
			maxMaj, maxMin = maj, min
		}
	}
	return fmt.Sprintf("%d.%d.0", maxMaj, maxMin+1)
}

// parseSemver 解析 "major.minor.patch" / "major" / "major.minor" 为三段整数。
func parseSemver(v string) (int, int, int) {
	var maj, min, pat int
	parts := strings.Split(v, ".")
	if len(parts) > 0 {
		fmt.Sscanf(strings.TrimSpace(parts[0]), "%d", &maj)
	}
	if len(parts) > 1 {
		fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &min)
	}
	if len(parts) > 2 {
		fmt.Sscanf(strings.TrimSpace(parts[2]), "%d", &pat)
	}
	return maj, min, pat
}

func (s *bpmnProcessDefinitionService) GetProcessDefinition(ctx context.Context, key string, version string) (*ent.ProcessDefinition, error) {
	// P1-4：租户上下文必须显式有效（>0），fail-closed
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	definition, err := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(key),
			processdefinition.Version(version),
			processdefinition.TenantID(tenantID),
		).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	return definition, nil
}

// GetProcessDefinitionByID 根据ID获取流程定义
func (s *bpmnProcessDefinitionService) GetProcessDefinitionByID(ctx context.Context, id int) (*ent.ProcessDefinition, error) {
	// P1-4：租户上下文必须显式有效（>0），fail-closed
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	definition, err := s.client.ProcessDefinition.Query().
		Where(processdefinition.ID(id), processdefinition.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	return definition, nil
}

func (s *bpmnProcessDefinitionService) GetLatestProcessDefinition(ctx context.Context, key string) (*ent.ProcessDefinition, error) {
	// P1-4：租户上下文必须显式有效（>0），fail-closed
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	definition, err := s.client.ProcessDefinition.Query().
		Where(
			processdefinition.Key(key),
			processdefinition.IsLatest(true),
			processdefinition.TenantID(tenantID),
		).
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

// PublishProcessDefinition atomically persists the draft contents and makes the
// selected immutable version the sole active/latest version for the key.
func (s *bpmnProcessDefinitionService) PublishProcessDefinition(ctx context.Context, key string, version string, req *UpdateProcessDefinitionRequest) (*ent.ProcessDefinition, error) {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.BPMNXML) == "" {
		return nil, fmt.Errorf("发布流程必须包含 BPMN XML")
	}
	if _, err = NewBPMNParser().ParseXML([]byte(req.BPMNXML)); err != nil {
		return nil, fmt.Errorf("BPMN XML 校验失败: %w", err)
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("开始发布事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	definition, err := tx.ProcessDefinition.Query().Where(
		processdefinition.Key(key), processdefinition.Version(version), processdefinition.TenantID(tenantID),
	).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取待发布流程定义失败: %w", err)
	}
	if _, err = tx.ProcessDefinition.Update().Where(
		processdefinition.Key(key), processdefinition.TenantID(tenantID),
	).SetIsActive(false).SetIsLatest(false).Save(ctx); err != nil {
		return nil, fmt.Errorf("停用旧流程版本失败: %w", err)
	}
	update := tx.ProcessDefinition.UpdateOne(definition).
		SetBpmnXML([]byte(req.BPMNXML)).SetIsActive(true).SetIsLatest(true).SetDeployedAt(time.Now())
	if req.Name != "" {
		update.SetName(req.Name)
	}
	if req.Description != "" {
		update.SetDescription(req.Description)
	}
	if req.Category != "" {
		update.SetCategory(req.Category)
	}
	if req.ProcessVariables != nil {
		update.SetProcessVariables(req.ProcessVariables)
	}
	published, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("保存发布版本失败: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交发布事务失败: %w", err)
	}
	return published, nil
}

func (s *bpmnProcessDefinitionService) DeleteProcessDefinition(ctx context.Context, key string, version string) error {
	definition, err := s.GetProcessDefinition(ctx, key, version)
	if err != nil {
		return err
	}

	// 检查是否有运行中的实例
	runningCount, err := s.client.ProcessInstance.
		Query().
		Where(processinstance.ProcessDefinitionID(definition.ID)).
		Count(ctx)
	if err != nil {
		return fmt.Errorf("检查流程实例失败: %w", err)
	}
	if runningCount > 0 {
		return fmt.Errorf("该流程定义有 %d 个运行中的实例，请先关闭后再删除", runningCount)
	}

	return s.client.ProcessDefinition.DeleteOne(definition).Exec(ctx)
}

func (s *bpmnProcessDefinitionService) ListProcessDefinitions(ctx context.Context, req *ListProcessDefinitionsRequest) ([]*ent.ProcessDefinition, int, error) {
	// P1-4：租户上下文必须显式有效（>0）；允许请求级 req.TenantID，但必须与上下文一致
	ctxTenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	tenantID := ctxTenantID
	if req.TenantID > 0 && req.TenantID != ctxTenantID {
		return nil, 0, fmt.Errorf("请求租户 %d 与上下文租户 %d 不一致，已拒绝", req.TenantID, ctxTenantID)
	}

	query := s.client.ProcessDefinition.Query().
		Where(processdefinition.TenantID(tenantID))

	if req.Key != "" {
		query = query.Where(processdefinition.Key(req.Key))
	}
	if req.Category != "" {
		query = query.Where(processdefinition.Category(req.Category))
	}
	if req.IsActive != nil {
		query = query.Where(processdefinition.IsActive(*req.IsActive))
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
	// P1-4：租户上下文必须显式有效（>0），fail-closed
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := strconv.Atoi(processInstanceID)
	if err != nil {
		return nil, fmt.Errorf("无效的流程实例ID: %w", err)
	}
	instance, err := s.client.ProcessInstance.Query().
		Where(processinstance.ID(id), processinstance.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程实例失败: %w", err)
	}

	return instance, nil
}

func (s *bpmnProcessInstanceService) ListProcessInstances(ctx context.Context, req *ListProcessInstancesRequest) ([]*ent.ProcessInstance, int, error) {
	// P1-4：租户上下文必须显式有效（>0）；同时允许请求对象级 req.TenantID 覆盖，前提是与上下文一致
	ctxTenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	tenantID := ctxTenantID
	if req.TenantID > 0 && req.TenantID != ctxTenantID {
		return nil, 0, fmt.Errorf("请求租户 %d 与上下文租户 %d 不一致，已拒绝", req.TenantID, ctxTenantID)
	}

	query := s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID))

	if req.ProcessDefinitionKey != "" {
		query = query.Where(processinstance.ProcessDefinitionKey(req.ProcessDefinitionKey))
	}
	if req.Status != "" {
		query = query.Where(processinstance.Status(req.Status))
	}
	if req.BusinessKey != "" {
		query = query.Where(processinstance.BusinessKey(req.BusinessKey))
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
	// P1-4：租户上下文必须显式有效（>0），fail-closed
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := strconv.Atoi(processInstanceID)
	if err != nil {
		return nil, fmt.Errorf("无效的流程实例ID: %w", err)
	}

	query := s.client.ProcessExecutionHistory.Query().
		Where(
			processexecutionhistory.ProcessInstanceID(id),
			processexecutionhistory.TenantID(tenantID),
		)

	history, err := query.Order(ent.Asc(processexecutionhistory.FieldTimestamp)).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取流程实例历史失败: %w", err)
	}

	return history, nil
}

// GetInstanceStatistics 获取实例统计
func (s *bpmnProcessInstanceService) GetInstanceStatistics(ctx context.Context, req *InstanceStatisticsRequest) (*InstanceStatistics, error) {
	// P1-4：租户上下文必须显式有效（>0）；允许请求级 req.TenantID，但必须与上下文一致
	ctxTenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	tenantID := ctxTenantID
	if req.TenantID > 0 && req.TenantID != ctxTenantID {
		return nil, fmt.Errorf("请求租户 %d 与上下文租户 %d 不一致，已拒绝", req.TenantID, ctxTenantID)
	}

	query := s.client.ProcessInstance.Query().
		Where(processinstance.TenantID(tenantID))

	if req.ProcessDefinitionKey != "" {
		query = query.Where(processinstance.ProcessDefinitionKey(req.ProcessDefinitionKey))
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
	client        *ent.Client
	logger        *zap.SugaredLogger
	groupResolver *bpmn.GroupResolver
}

// GetTask 根据任务ID (BPMN标准task_id字符串)获取任务
func (s *bpmnTaskService) GetTask(ctx context.Context, taskID string) (*ent.ProcessTask, error) {
	// P1-4：租户上下文必须显式有效（>0），fail-closed
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	task, err := s.client.ProcessTask.Query().
		Where(processtask.TaskID(taskID), processtask.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取任务失败: %w", err)
	}

	return task, nil
}

// GetTaskByID 根据数据库自增ID获取任务
func (s *bpmnTaskService) GetTaskByID(ctx context.Context, id int) (*ent.ProcessTask, error) {
	// P1-4：租户上下文必须显式有效（>0），fail-closed
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	task, err := s.client.ProcessTask.Query().
		Where(processtask.ID(id), processtask.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取任务失败: %w", err)
	}

	return task, nil
}

// CompleteTaskByID 根据数据库自增ID完成任务
func (s *bpmnTaskService) CompleteTaskByID(ctx context.Context, id int, variables map[string]interface{}) error {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return err
	}

	engine := NewCustomProcessEngine(s.client, s.logger)
	// 首次读取即按当前租户过滤，避免先全局读取再事后校验造成跨租户任务泄漏。
	task, err := s.client.ProcessTask.Query().
		Where(processtask.ID(id), processtask.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}
	return engine.CompleteTask(ctx, task.TaskID, variables)
}

func (s *bpmnTaskService) ListUserTasks(ctx context.Context, req *ListUserTasksRequest) ([]*ent.ProcessTask, int, error) {
	if req == nil {
		return nil, 0, fmt.Errorf("任务列表请求不能为空")
	}
	if req.TenantID <= 0 {
		return nil, 0, fmt.Errorf("缺少有效租户上下文")
	}
	if !req.AllTasks && req.UserID <= 0 {
		return nil, 0, fmt.Errorf("我的任务查询缺少有效用户上下文")
	}
	s.logger.Debugw("ListUserTasks called", "assignee", req.Assignee, "userID", req.UserID, "tenantID", req.TenantID)
	query := s.client.ProcessTask.Query().Where(processtask.TenantID(req.TenantID))

	// 「我的待办」语义：UserID 透传时，查出“分配给我 OR 我是候选人 OR 我所在组是候选组”的任务。
	// 这样能同时覆盖 assignee / candidate_users / candidate_groups 三种途径。
	if !req.AllTasks {
		tenantID := req.TenantID
		userIDStr := strconv.Itoa(req.UserID)

		// 1. 取得该用户所在的组名（逗号分隔）
		userGroupsCSV := ""
		if s.groupResolver != nil {
			groups, gErr := s.groupResolver.GetUserGroupNames(ctx, tenantID, req.UserID)
			if gErr != nil {
				s.logger.Warnw("查询用户所属组失败", "error", gErr, "userID", req.UserID)
			} else {
				userGroupsCSV = groups
			}
		}

		// 2. OR 复合查询：assignee == me OR candidate_users 包含我 OR candidate_groups 包含我所在组
		orPreds := []predicate.ProcessTask{
			processtask.Assignee(userIDStr),
			processtask.CandidateUsersContains(userIDStr),
		}
		// 同时以 username 形式匹配（process_task.candidate_users 中保存的是 username/email/ID 混合）
		if u, err := s.client.User.Get(ctx, req.UserID); err == nil && u != nil {
			username := strings.TrimSpace(u.Username)
			if username != "" && username != userIDStr {
				orPreds = append(orPreds, processtask.CandidateUsersContains(username))
			}
			email := strings.TrimSpace(u.Email)
			if email != "" && email != userIDStr && email != username {
				orPreds = append(orPreds, processtask.CandidateUsersContains(email))
			}
		}
		for _, group := range strings.Split(userGroupsCSV, ",") {
			if group = strings.TrimSpace(group); group != "" {
				orPreds = append(orPreds, processtask.CandidateGroupsContains(group))
			}
		}
		query = query.Where(processtask.Or(orPreds...))
	} else {
		if req.Assignee != "" {
			query = query.Where(processtask.Assignee(req.Assignee))
		}
		if req.CandidateUsers != "" {
			query = query.Where(processtask.CandidateUsersContains(req.CandidateUsers))
		}
		if req.CandidateGroups != "" {
			query = query.Where(processtask.CandidateGroupsContains(req.CandidateGroups))
		}
	}
	if req.Status != "" {
		query = query.Where(processtask.Status(req.Status))
	}
	if req.ProcessDefinitionKey != "" {
		query = query.Where(processtask.ProcessDefinitionKey(req.ProcessDefinitionKey))
	}
	if req.ProcessInstanceID > 0 {
		query = query.Where(processtask.ProcessInstanceID(req.ProcessInstanceID))
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

// ListUserTaskViews 「我的待办」视图：任务列表附带所属实例的 businessKey 等业务上下文，
// 供审批中心跳转业务单据使用。返回 DTO 而非 Ent 模型。
func (s *bpmnTaskService) ListUserTaskViews(ctx context.Context, req *ListUserTasksRequest) ([]*dto.BPMNTaskResponse, int, error) {
	tasks, total, err := s.ListUserTasks(ctx, req)
	if err != nil {
		return nil, 0, err
	}

	// 批量加载任务所属流程实例，避免 N+1 查询
	instanceIDs := make([]int, 0, len(tasks))
	seen := make(map[int]bool, len(tasks))
	for _, task := range tasks {
		if !seen[task.ProcessInstanceID] {
			seen[task.ProcessInstanceID] = true
			instanceIDs = append(instanceIDs, task.ProcessInstanceID)
		}
	}
	instanceMap := make(map[int]*ent.ProcessInstance, len(instanceIDs))
	if len(instanceIDs) > 0 {
		instances, err := s.client.ProcessInstance.Query().
			Where(processinstance.IDIn(instanceIDs...)).
			All(ctx)
		if err != nil {
			return nil, 0, fmt.Errorf("加载任务所属流程实例失败: %w", err)
		}
		for _, instance := range instances {
			instanceMap[instance.ID] = instance
		}
	}

	return dto.ToBPMNTaskResponseList(tasks, instanceMap), total, nil
}

func (s *bpmnTaskService) ListApprovalDecisions(ctx context.Context, processInstanceKey string) ([]*ent.ProcessApprovalDecision, error) {
	tenantID, _ := ctx.Value(bpmn.BPMNTenantIDContextKey).(int)
	if tenantID <= 0 {
		return nil, fmt.Errorf("缺少租户上下文")
	}
	return s.client.ProcessApprovalDecision.Query().
		Where(
			processapprovaldecision.ProcessInstanceKey(processInstanceKey),
			processapprovaldecision.TenantID(tenantID),
		).
		Order(ent.Asc(processapprovaldecision.FieldCreatedAt)).
		All(ctx)
}

func (s *bpmnTaskService) AssignTask(ctx context.Context, taskID string, assignee string) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	_, err = s.client.ProcessTask.UpdateOne(task).
		SetAssignee(assignee).
		SetStatus(common.ProcessTaskStatusAssigned).
		SetAssignedTime(time.Now()).
		Save(ctx)

	return err
}

// ReassignTask atomically validates the tenant-scoped target, reassigns an
// active BPMN task, and records the operator decision in the BPMN audit log.
func (s *bpmnTaskService) ReassignTask(ctx context.Context, taskID string, newAssigneeID int, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return fmt.Errorf("重新分配原因不能为空")
	}
	tenantID, _ := ctx.Value(bpmn.BPMNTenantIDContextKey).(int)
	actorID, _ := ctx.Value(bpmn.BPMNUserIDContextKey).(int)
	if tenantID <= 0 || actorID <= 0 {
		return fmt.Errorf("缺少有效的租户或操作人上下文")
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开启重新分配事务失败: %w", err)
	}
	defer tx.Rollback()

	task, err := tx.ProcessTask.Query().Where(
		processtask.TaskID(taskID),
		processtask.TenantID(tenantID),
	).Only(ctx)
	if err != nil {
		return fmt.Errorf("任务不存在或不属于当前租户: %w", err)
	}
	if task.Status != common.ProcessTaskStatusCreated && task.Status != common.ProcessTaskStatusAssigned && task.Status != common.ProcessTaskStatusStarted && task.Status != "running" {
		return fmt.Errorf("任务状态 %s 不允许重新分配", task.Status)
	}
	assignee, err := tx.User.Query().Where(
		user.IDEQ(newAssigneeID),
		user.TenantIDEQ(tenantID),
		user.ActiveEQ(true),
	).Only(ctx)
	if err != nil {
		return fmt.Errorf("目标处理人不存在、已停用或不属于当前租户")
	}
	actor, err := tx.User.Query().Where(user.IDEQ(actorID), user.TenantIDEQ(tenantID), user.ActiveEQ(true)).Only(ctx)
	if err != nil {
		return fmt.Errorf("操作人不存在、已停用或不属于当前租户")
	}
	instance, err := tx.ProcessInstance.Get(ctx, task.ProcessInstanceID)
	if err != nil || instance.TenantID != tenantID {
		return fmt.Errorf("任务所属流程实例不存在或租户不一致")
	}
	previousAssignee := task.Assignee
	if _, err = tx.ProcessTask.UpdateOne(task).
		SetAssignee(strconv.Itoa(assignee.ID)).
		SetStatus(common.ProcessTaskStatusAssigned).
		SetAssignedTime(time.Now()).
		Save(ctx); err != nil {
		return fmt.Errorf("重新分配任务失败: %w", err)
	}
	if _, err = tx.ProcessAuditLog.Create().
		SetProcessInstanceID(instance.ID).
		SetProcessInstanceKey(instance.ProcessInstanceID).
		SetProcessDefinitionKey(instance.ProcessDefinitionKey).
		SetProcessDefinitionID(instance.ProcessDefinitionID).
		SetActivityID(task.TaskDefinitionKey).
		SetActivityName(task.TaskName).
		SetActivityType(task.TaskType).
		SetAction(AuditActionTaskReassigned).
		SetUserID(actor.ID).
		SetUserName(actor.Name).
		SetAssigneeID(assignee.ID).
		SetAssigneeName(assignee.Name).
		SetComment(reason).
		SetVariablesBefore(map[string]interface{}{"assignee": previousAssignee}).
		SetVariablesAfter(map[string]interface{}{"assignee": strconv.Itoa(assignee.ID)}).
		SetTenantID(tenantID).
		SetTimestamp(time.Now()).
		SetMetadata(map[string]interface{}{"recoveryAction": "reassign", "taskId": task.TaskID}).
		Save(ctx); err != nil {
		return fmt.Errorf("记录重新分配审计失败: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("提交重新分配事务失败: %w", err)
	}
	return nil
}

// TerminateTask terminates the owning process rather than cancelling a single
// task and leaving an unrecoverable running instance behind.
func (s *bpmnTaskService) TerminateTask(ctx context.Context, taskID string, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return fmt.Errorf("终止原因不能为空")
	}
	tenantID, _ := ctx.Value(bpmn.BPMNTenantIDContextKey).(int)
	actorID, _ := ctx.Value(bpmn.BPMNUserIDContextKey).(int)
	if tenantID <= 0 || actorID <= 0 {
		return fmt.Errorf("缺少有效的租户或操作人上下文")
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开启终止事务失败: %w", err)
	}
	defer tx.Rollback()
	task, err := tx.ProcessTask.Query().Where(processtask.TaskID(taskID), processtask.TenantID(tenantID)).Only(ctx)
	if err != nil {
		return fmt.Errorf("任务不存在或不属于当前租户: %w", err)
	}
	instance, err := tx.ProcessInstance.Get(ctx, task.ProcessInstanceID)
	if err != nil || instance.TenantID != tenantID {
		return fmt.Errorf("任务所属流程实例不存在或租户不一致")
	}
	if instance.Status == "completed" || instance.Status == "terminated" {
		return fmt.Errorf("流程实例已处于终态 %s", instance.Status)
	}
	actor, err := tx.User.Query().Where(user.IDEQ(actorID), user.TenantIDEQ(tenantID), user.ActiveEQ(true)).Only(ctx)
	if err != nil {
		return fmt.Errorf("操作人不存在、已停用或不属于当前租户")
	}
	now := time.Now()
	if _, err = tx.ProcessInstance.UpdateOne(instance).SetStatus("terminated").SetEndTime(now).Save(ctx); err != nil {
		return fmt.Errorf("终止流程实例失败: %w", err)
	}
	if _, err = tx.ProcessTask.Update().Where(
		processtask.ProcessInstanceID(instance.ID),
		processtask.StatusNotIn(common.ProcessTaskStatusCompleted, common.ProcessTaskStatusCancelled),
	).SetStatus(common.ProcessTaskStatusCancelled).SetCompletedTime(now).Save(ctx); err != nil {
		return fmt.Errorf("取消流程未完成任务失败: %w", err)
	}
	if _, err = tx.ProcessAuditLog.Create().
		SetProcessInstanceID(instance.ID).
		SetProcessInstanceKey(instance.ProcessInstanceID).
		SetProcessDefinitionKey(instance.ProcessDefinitionKey).
		SetProcessDefinitionID(instance.ProcessDefinitionID).
		SetActivityID(task.TaskDefinitionKey).
		SetActivityName(task.TaskName).
		SetActivityType(task.TaskType).
		SetAction(AuditActionProcessTerminated).
		SetUserID(actor.ID).
		SetUserName(actor.Name).
		SetComment(reason).
		SetTenantID(tenantID).
		SetTimestamp(now).
		SetMetadata(map[string]interface{}{"recoveryAction": "terminate", "sourceTaskId": task.TaskID}).
		Save(ctx); err != nil {
		return fmt.Errorf("记录终止审计失败: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("提交终止事务失败: %w", err)
	}
	return nil
}

// ClaimTask 认领任务 (根据task_id字符串)
func (s *bpmnTaskService) ClaimTask(ctx context.Context, taskID string, userID string) error {
	currentUserID, err := strconv.Atoi(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	return s.claimTask(ctx, currentUserID, func(client *ent.Client, tenantID int) (*ent.ProcessTask, error) {
		return client.ProcessTask.Query().
			Where(processtask.TaskID(taskID), processtask.TenantID(tenantID)).
			First(ctx)
	})
}

// ClaimTaskByID 认领任务 (根据数据库自增ID)
func (s *bpmnTaskService) ClaimTaskByID(ctx context.Context, id int, userID int) error {
	return s.claimTask(ctx, userID, func(client *ent.Client, tenantID int) (*ent.ProcessTask, error) {
		return client.ProcessTask.Query().
			Where(processtask.ID(id), processtask.TenantID(tenantID)).
			First(ctx)
	})
}

func (s *bpmnTaskService) claimTask(
	ctx context.Context,
	currentUserID int,
	findTask func(*ent.Client, int) (*ent.ProcessTask, error),
) error {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return err
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start claim task transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	task, err := findTask(tx.Client(), tenantID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}
	currentUser := strconv.Itoa(currentUserID)
	if task.Assignee != "" && task.Assignee != "0" && task.Assignee != currentUser {
		return fmt.Errorf("task already assigned")
	}
	if task.Assignee == currentUser {
		return tx.Commit()
	}

	if err = validateTaskCandidate(ctx, tx.Client(), task, tenantID, currentUserID); err != nil {
		return err
	}

	if _, err = tx.ProcessTask.UpdateOne(task).
		SetAssignee(currentUser).
		SetStatus(common.ProcessTaskStatusAssigned).
		SetAssignedTime(time.Now()).
		Save(ctx); err != nil {
		return fmt.Errorf("claim task: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit claim task transaction: %w", err)
	}
	return nil
}

func validateTaskCandidate(ctx context.Context, client *ent.Client, task *ent.ProcessTask, tenantID, currentUserID int) error {
	candidateUsers := splitNonEmptyCSV(task.CandidateUsers)
	candidateGroups := splitNonEmptyCSV(task.CandidateGroups)
	if len(candidateUsers) == 0 && len(candidateGroups) == 0 {
		return nil
	}

	identities := map[string]struct{}{strconv.Itoa(currentUserID): {}}
	actor, err := client.User.Query().
		Where(user.ID(currentUserID), user.TenantID(tenantID)).
		Only(ctx)
	if err == nil {
		identities[strings.TrimSpace(actor.Username)] = struct{}{}
		identities[strings.TrimSpace(actor.Email)] = struct{}{}
	} else if !ent.IsNotFound(err) {
		return fmt.Errorf("get claiming user: %w", err)
	}
	for _, candidate := range candidateUsers {
		if _, ok := identities[candidate]; ok {
			return nil
		}
	}

	if len(candidateGroups) > 0 {
		groupsCSV, groupErr := bpmn.NewGroupResolver(client).GetUserGroupNames(ctx, tenantID, currentUserID)
		if groupErr != nil {
			return fmt.Errorf("get user groups: %w", groupErr)
		}
		groups := make(map[string]struct{})
		for _, groupName := range splitNonEmptyCSV(groupsCSV) {
			groups[groupName] = struct{}{}
		}
		for _, candidateGroup := range candidateGroups {
			if _, ok := groups[candidateGroup]; ok {
				return nil
			}
		}
	}

	return fmt.Errorf("you are not a candidate for this task")
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
	allowDelegate, _ := task.TaskVariables["allowDelegate"].(bool)
	if !allowDelegate {
		return fmt.Errorf("该审批节点不允许委托")
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

func (s *bpmnTaskService) BatchAssignTasks(ctx context.Context, taskIDs []string, assignee string, tenantID int) error {
	if len(taskIDs) == 0 {
		return fmt.Errorf("任务ID列表为空")
	}

	// 租户过滤，防止跨租户批量指派
	_, err := s.client.ProcessTask.Update().
		Where(
			processtask.TaskIDIn(taskIDs...),
			processtask.TenantID(tenantID),
		).
		SetAssignee(assignee).
		SetStatus(common.ProcessTaskStatusAssigned).
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
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	return createCounterSignTasksWithClient(ctx, s.client, parentTaskID, tenantID, req)
}

func createCounterSignTasksWithClient(ctx context.Context, client *ent.Client, parentTaskID string, tenantID int, req *CounterSignRequest) ([]*ent.ProcessTask, error) {
	if req == nil || len(req.Approvers) == 0 {
		return nil, fmt.Errorf("会签审批人不能为空")
	}
	// 获取父任务
	parentTask, err := client.ProcessTask.Query().
		Where(processtask.TaskID(parentTaskID), processtask.TenantID(tenantID)).
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
		status := common.ProcessTaskStatusAssigned
		if req.ApprovalType == "serial" && i > 0 {
			status = "created"
		}
		task, err := client.ProcessTask.Create().
			SetTaskID(taskID).
			SetProcessInstanceID(parentTask.ProcessInstanceID).
			SetProcessDefinitionKey(parentTask.ProcessDefinitionKey).
			SetTaskDefinitionKey(parentTask.TaskDefinitionKey + "_counter").
			SetTaskName(parentTask.TaskName + "_会签").
			SetTaskType("user_task").
			SetAssignee(approver).
			SetStatus(status).
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
	_, err = client.ProcessTask.UpdateOneID(parentTask.ID).
		SetTaskVariables(map[string]interface{}{
			"approval_type": req.ApprovalType,
			"threshold":     threshold,
			"total":         len(req.Approvers),
			"completed":     0,
			"approved":      0,
			"rejected":      0,
		}).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("更新父任务会签配置失败: %w", err)
	}

	return tasks, nil
}

// GetCounterSignStatus 获取会签状态
func (s *bpmnTaskService) GetCounterSignStatus(ctx context.Context, parentTaskID string) (*CounterSignStatus, error) {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return nil, err
	}
	// 获取所有会签子任务
	subTasks, err := s.client.ProcessTask.Query().
		Where(processtask.ParentTaskID(parentTaskID), processtask.TenantID(tenantID)).
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

	threshold := status.Total
	if parent, err := s.client.ProcessTask.Query().Where(processtask.TaskID(parentTaskID), processtask.TenantID(tenantID)).Only(ctx); err == nil {
		if value, ok := numericInt(parent.TaskVariables["threshold"]); ok && value > 0 {
			threshold = value
		}
	}
	if status.Approved >= threshold {
		status.Status = "approved"
	} else if status.Approved+status.Pending < threshold {
		status.Status = "rejected"
	} else {
		status.Status = "pending"
	}

	return status, nil
}

func numericInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

// Vote 投票（完成会签任务）
func (s *bpmnTaskService) Vote(ctx context.Context, taskID string, req *VoteRequest) error {
	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return err
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("开始会签投票事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	task, err := s.client.ProcessTask.Query().
		Where(processtask.TaskID(taskID), processtask.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return fmt.Errorf("获取任务失败: %w", err)
	}
	engineForAuth := NewCustomProcessEngine(s.client, s.logger).(*CustomProcessEngine)
	if err := engineForAuth.authorizeTaskActor(ctx, task); err != nil {
		return err
	}
	if task.Status == "completed" || task.Status == "cancelled" {
		return fmt.Errorf("会签任务已结束")
	}
	if task.ParentTaskID != "" && task.Status != common.ProcessTaskStatusAssigned {
		return fmt.Errorf("会签任务尚未轮到当前审批人")
	}

	// 更新任务状态为完成
	updated, err := tx.ProcessTask.Update().Where(
		processtask.ID(task.ID), processtask.TenantID(tenantID), processtask.StatusEQ(common.ProcessTaskStatusAssigned),
	).
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
	if updated != 1 {
		return fmt.Errorf("会签任务已被处理，请刷新后重试")
	}
	instance, err := tx.ProcessInstance.Query().Where(processinstance.ID(task.ProcessInstanceID), processinstance.TenantID(tenantID)).Only(ctx)
	if err == nil {
		action, decision := "reject", "rejected"
		if req.Approved {
			action, decision = "approve", "approved"
		}
		engine := NewCustomProcessEngine(s.client, s.logger).(*CustomProcessEngine)
		if err := engine.recordApprovalDecision(ctx, tx.Client(), instance, task, map[string]interface{}{"approvalAction": action, "approvalResult": decision, "approvalComment": req.Comment}); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交会签投票失败: %w", err)
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
		Where(processtask.TaskID(parentTaskID), processtask.TenantID(tenantID)).
		First(ctx)
	if err != nil {
		return nil
	}

	vars := parentTask.TaskVariables
	if vars == nil {
		vars = make(map[string]interface{})
	}
	threshold := 1
	if t, ok := numericInt(vars["threshold"]); ok {
		threshold = t
	}
	approvalType := "parallel"
	if at, ok := vars["approval_type"].(string); ok {
		approvalType = at
	}

	if approvalType == "serial" && req.Approved && status.Status == "pending" {
		next, err := s.client.ProcessTask.Query().Where(
			processtask.ParentTaskID(parentTaskID),
			processtask.TenantID(tenantID),
			processtask.Status("created"),
		).Order(ent.Asc(processtask.FieldID)).First(ctx)
		if err == nil {
			_, _ = s.client.ProcessTask.Update().Where(
				processtask.ID(next.ID),
				processtask.TenantID(tenantID),
				processtask.Status("created"),
			).SetStatus(common.ProcessTaskStatusAssigned).Save(ctx)
		}
	}

	// 检查是否达到阈值
	if status.Status == "approved" || status.Status == "rejected" {
		finalVariables := map[string]interface{}{
			"approval_type": approvalType,
			"threshold":     threshold,
			"total":         status.Total,
			"completed":     status.Completed,
			"approved":      status.Approved,
			"rejected":      status.Rejected,
			"final_status":  status.Status,
		}
		// Only one concurrent voter may claim and advance the parent task. Other
		// voters observe the finalizing/completed state and return successfully.
		claimed, claimErr := s.client.ProcessTask.Update().Where(
			processtask.ID(parentTask.ID),
			processtask.TenantID(tenantID),
			processtask.StatusNEQ("completed"),
			processtask.StatusNEQ("cancelled"),
			processtask.StatusNEQ("finalizing"),
		).SetStatus("finalizing").SetTaskVariables(finalVariables).Save(ctx)
		if claimErr != nil {
			return fmt.Errorf("抢占会签父任务失败: %w", claimErr)
		}
		if claimed == 0 {
			return nil
		}
		_, _ = s.client.ProcessTask.Update().
			Where(
				processtask.ParentTaskID(parentTaskID),
				processtask.TenantID(tenantID),
				processtask.StatusNEQ("completed"),
				processtask.StatusNEQ("cancelled"),
			).
			SetStatus("cancelled").SetCompletedTime(time.Now()).Save(ctx)
		engine := NewCustomProcessEngine(s.client, s.logger)
		if err := engine.CompleteTask(ctx, parentTask.TaskID, map[string]interface{}{"approvalResult": status.Status, "approved": status.Status == "approved"}); err != nil {
			return fmt.Errorf("推进会签父任务失败: %w", err)
		}
	}

	return nil
}
