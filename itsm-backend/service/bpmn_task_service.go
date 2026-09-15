package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/predicate"
	"itsm-backend/ent/processapprovaldecision"
	"itsm-backend/ent/processinstance"
	"itsm-backend/ent/processtask"
	"itsm-backend/ent/user"
	"itsm-backend/service/bpmn"

	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// bpmn_task_service.go — BPMN 任务管理服务
//
// 职责：用户任务的 CRUD、认领/指派/委托/加签/升级/超时/重试、批量操作、
// 任务统计、会签子任务创建与投票、审批决策查询。
//
// 社区贡献者只需理解此文件即可掌握任务生命周期的全部操作。
// ---------------------------------------------------------------------------

// Request/Response structs

type ListUserTasksRequest struct {
	Assignee        string `json:"assignee"`
	CandidateUsers  string `json:"candidateUsers"`
	CandidateGroups string `json:"candidateGroups"`
	// UserID 为「我的待办」语义：查询"分配给我 OR 我在候选人 OR 我所在组作为候选组"的任务。
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

	// 「我的待办」语义：UserID 透传时，查出"分配给我 OR 我是候选人 OR 我所在组是候选组"的任务。
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

	responses := dto.ToBPMNTaskResponseList(tasks, instanceMap)

	// 批量解析负责人显示名：assignee 存的是数字用户 ID，直接渲染会让审批中心
	// 显示「1」这类裸 ID。这里一次性按去重后的 ID 查用户，避免 N+1。
	assigneeIDs := make([]int, 0, len(responses))
	seenAssignee := make(map[int]bool, len(responses))
	for _, resp := range responses {
		if id, err := strconv.Atoi(strings.TrimSpace(resp.Assignee)); err == nil && id > 0 && !seenAssignee[id] {
			seenAssignee[id] = true
			assigneeIDs = append(assigneeIDs, id)
		}
	}
	if len(assigneeIDs) > 0 {
		users, err := s.client.User.Query().
			Where(user.IDIn(assigneeIDs...)).
			Select(user.FieldID, user.FieldName, user.FieldUsername).
			All(ctx)
		if err != nil {
			return nil, 0, fmt.Errorf("加载任务负责人失败: %w", err)
		}
		nameByID := make(map[int]string, len(users))
		for _, u := range users {
			if u.Name != "" {
				nameByID[u.ID] = u.Name
			} else {
				nameByID[u.ID] = u.Username
			}
		}
		for _, resp := range responses {
			if id, err := strconv.Atoi(strings.TrimSpace(resp.Assignee)); err == nil {
				resp.AssigneeName = nameByID[id]
			}
		}
	}

	return responses, total, nil
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
	if strings.TrimSpace(newAssignee) == "" {
		return fmt.Errorf("委托目标不能为空")
	}
	// 记录委托来源和时间（覆盖已有值，支持多次委托链）
	task.TaskVariables["delegated_from"] = task.Assignee
	task.TaskVariables["delegated_time"] = time.Now().Format(time.RFC3339)

	originalAssignee := task.Assignee

	// 保持任务活跃状态（"assigned"），仅更换负责人；设为 "delegated" 会导致被委托人看不到该任务
	if _, err = s.client.ProcessTask.UpdateOne(task).
		SetAssignee(newAssignee).
		SetStatus("assigned").
		SetTaskVariables(task.TaskVariables).
		Save(ctx); err != nil {
		return err
	}

	actorID, _ := ctx.Value(bpmn.BPMNUserIDContextKey).(int)
	if actorID <= 0 {
		return nil
	}
	instance, ierr := s.client.ProcessInstance.Get(ctx, task.ProcessInstanceID)
	if ierr != nil {
		return nil
	}
	delegatedFrom, _ := strconv.Atoi(originalAssignee)
	if _, err := s.client.ProcessApprovalDecision.Create().
		SetProcessInstanceID(instance.ID).SetProcessTaskID(task.ID).
		SetProcessInstanceKey(instance.ProcessInstanceID).SetTaskID(task.TaskID).
		SetProcessDefinitionKey(instance.ProcessDefinitionKey).SetNodeKey(task.TaskDefinitionKey).
		SetActorID(actorID).SetAction("delegate").SetDecision("delegated").
		SetNillableDelegatedFrom(&delegatedFrom).
		SetTenantID(instance.TenantID).Save(ctx); err != nil {
		// 审计事实失败不得静默：委派本身已生效，仅记录日志供巡检补偿
		s.logger.Errorw("记录委派审计决策失败",
			"task_id", task.TaskID, "process_instance_id", instance.ID, "error", err)
	}
	return nil
}

// DelegateTaskByID 按数据库主键委托任务（与 CompleteTaskByID 对称）。
func (s *bpmnTaskService) DelegateTaskByID(ctx context.Context, id int, newAssignee string) error {
	task, err := s.client.ProcessTask.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("任务不存在: %w", err)
	}
	return s.DelegateTask(ctx, task.TaskID, newAssignee)
}

func (s *bpmnTaskService) AddApproverTask(ctx context.Context, taskID string, newApprover string) error {
	task, err := s.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	if task.TaskVariables == nil {
		task.TaskVariables = make(map[string]interface{})
	}
	allowAdd, _ := task.TaskVariables["allowAddApprover"].(bool)
	if !allowAdd {
		return fmt.Errorf("该审批节点不允许加签")
	}
	if strings.TrimSpace(newApprover) == "" {
		return fmt.Errorf("加签目标不能为空")
	}

	tenantID, err := requireBPMNTenantContext(ctx)
	if err != nil {
		return err
	}

	rootTaskID := task.TaskID
	if task.RootTaskID != "" {
		rootTaskID = task.RootTaskID
	}
	parentTaskID := task.TaskID
	if task.ParentTaskID != "" {
		parentTaskID = task.ParentTaskID
	}

	addedTaskID := fmt.Sprintf("%s_addapprover_%d", task.TaskID, time.Now().UnixNano())
	newTask, err := s.client.ProcessTask.Create().
		SetTaskID(addedTaskID).
		SetProcessInstanceID(task.ProcessInstanceID).
		SetProcessDefinitionKey(task.ProcessDefinitionKey).
		SetTaskDefinitionKey(task.TaskDefinitionKey).
		SetTaskName(task.TaskName).
		SetTaskType("user_task").
		SetAssignee(newApprover).
		SetStatus(common.ProcessTaskStatusAssigned).
		SetPriority(task.Priority).
		SetParentTaskID(parentTaskID).
		SetRootTaskID(rootTaskID).
		SetTaskVariables(map[string]interface{}{
			"taskPurpose":    "approval",
			"added_by":       task.Assignee,
			"added_time":     time.Now().Format(time.RFC3339),
			"source_task_id": task.TaskID,
		}).
		SetTenantID(tenantID).
		SetCreatedTime(time.Now()).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("创建加签任务失败: %w", err)
	}

	actorID, _ := ctx.Value(bpmn.BPMNUserIDContextKey).(int)
	if actorID <= 0 {
		return nil
	}
	instance, ierr := s.client.ProcessInstance.Get(ctx, task.ProcessInstanceID)
	if ierr != nil {
		return nil
	}
	if _, err := s.client.ProcessApprovalDecision.Create().
		SetProcessInstanceID(instance.ID).SetProcessTaskID(newTask.ID).
		SetProcessInstanceKey(instance.ProcessInstanceID).SetTaskID(addedTaskID).
		SetProcessDefinitionKey(instance.ProcessDefinitionKey).SetNodeKey(task.TaskDefinitionKey).
		SetActorID(actorID).SetAction("add_approver").SetDecision("added").
		SetTenantID(instance.TenantID).Save(ctx); err != nil {
		// 审计事实失败不得静默：加签本身已生效，仅记录日志供巡检补偿
		s.logger.Errorw("记录加签审计决策失败",
			"task_id", addedTaskID, "process_instance_id", instance.ID, "error", err)
	}
	return nil
}

func (s *bpmnTaskService) AddApproverTaskByID(ctx context.Context, id int, newApprover string) error {
	task, err := s.client.ProcessTask.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("任务不存在: %w", err)
	}

	return s.AddApproverTask(ctx, task.TaskID, newApprover)
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
		systemCtx := context.WithValue(context.Background(), bpmn.BPMNTenantIDContextKey, tenantID)
		if err := engine.CompleteTask(systemCtx, parentTask.TaskID, map[string]interface{}{"approvalResult": status.Status, "approved": status.Status == "approved"}); err != nil {
			return fmt.Errorf("推进会签父任务失败: %w", err)
		}
	}

	return nil
}
