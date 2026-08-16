package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"itsm-backend/ent"
	"itsm-backend/ent/approvalchain"
	"itsm-backend/ent/cabmember"
	"itsm-backend/ent/schema"
	"itsm-backend/ent/user"
	"itsm-backend/service/approver"
	"itsm-backend/service/bpmn"

	"go.uber.org/zap"
)

// ============================================================
// 审批链 fallback 求值引擎
// 解决审计标记的「ApprovalChain 仅 CRUD，无量级/层级/会签/或签/fallback 求值」。
// 设计：
//   - 层级(level)：steps 按 level 升序分组，逐级顺序审批（层级 = 顺序审批）。
//   - 会签/或签：同层内多个 step 合并为候选审批人集合；
//   ApprovalType=parallel|all 表示会签(AND，阈值默认=审批人数)，
//   ApprovalType=serial|or|"" 表示或签/单人(阈值=1)。
//   - fallback：必需层解析不到任何审批人时触发兜底策略
//   (block 失败关闭 / auto_approve / escalate / auto_reject)。
//   - 租户隔离：所有审批人解析均按 TenantID 过滤，显式 ApproverID 也校验租户归属，
//   直接堵住「无审批人时自审批」「跨租户注入」类缺陷。
//   默认 fallback 为 block（失败关闭），绝不默认自审批。
// ============================================================

// 兜底策略
const (
	FallbackBlock       = "block"        // 失败关闭：阻塞，需人工介入（默认）
	FallbackAutoApprove = "auto_approve" // 自动通过该层
	FallbackEscalate    = "escalate"     // 升级到兜底审批人(FallbackApproverID)/角色(FallbackRole)
	FallbackAutoReject  = "auto_reject"  // 自动拒绝
)

// ApprovalEvalContext 审批链求值上下文
type ApprovalEvalContext struct {
	TenantID    int
	EntityType  string // ticket, change, service_request
	RequesterID int
	Priority    string
	Amount      float64
}

// ApprovalLevelEval 单层（一个 level）求值结果
type ApprovalLevelEval struct {
	Level             int
	Required          bool
	ApprovalType      string // serial | parallel | all | or
	Threshold         int    // 需要的批准人数
	ApproverIDs       []int
	ApproverNames     []string
	FallbackTriggered bool
	FallbackAction    string
	Status            string // pending | satisfied | blocked
}

// ApprovalChainEvaluation 整体求值结果
type ApprovalChainEvaluation struct {
	ChainID      int
	Levels       []ApprovalLevelEval
	Passed       bool // 所有必需层均满足（含 fallback 解决）
	PendingLevel int  // 下一个待审批层（1-based），0=已完成/无待办
	Blocked      bool // 触发 block/auto_reject，需人工干预
}

// EvaluateApprovalChain 对一条审批链求值。
// approvals 为「level -> 已批准该层的用户ID列表」；传 nil 表示尚未有任何批准（用于生成审批计划）。
func EvaluateApprovalChain(ctx context.Context, client *ent.Client, logger *zap.SugaredLogger, chain *ent.ApprovalChain, evalCtx ApprovalEvalContext, approvals map[int][]int) (*ApprovalChainEvaluation, error) {
	if chain == nil {
		return nil, fmt.Errorf("审批链为空")
	}

	// 按 level 升序分组
	levelMap := map[int][]schema.ApprovalChainStep{}
	var order []int
	for _, step := range chain.Chain {
		lvl := step.Level
		if lvl < 1 {
			lvl = 1
		}
		if _, ok := levelMap[lvl]; !ok {
			order = append(order, lvl)
		}
		levelMap[lvl] = append(levelMap[lvl], step)
	}
	sort.Ints(order)

	result := &ApprovalChainEvaluation{ChainID: chain.ID}
	blocked := false

	for _, lvl := range order {
		steps := levelMap[lvl]

		levelRequired := false
		approvalType := "serial"
		threshold := 0
		fallbackAction := FallbackBlock
		fallbackApproverID := 0
		fallbackRole := ""
		var allApprovers []int
		var allNames []string

		for _, st := range steps {
			if st.IsRequired {
				levelRequired = true
			}
			if st.ApprovalType == "parallel" || st.ApprovalType == "all" {
				approvalType = st.ApprovalType
			}
			if st.Threshold > threshold {
				threshold = st.Threshold
			}
			if st.FallbackAction != "" {
				fallbackAction = st.FallbackAction
			}
			if st.FallbackApproverID > 0 {
				fallbackApproverID = st.FallbackApproverID
			}
			if st.FallbackRole != "" {
				fallbackRole = st.FallbackRole
			}
			ids, names, err := resolveChainApprovers(ctx, client, logger, st, evalCtx)
			if err != nil {
				// 解析失败视为无审批人（触发 fallback），仅记录不中断。
				if logger != nil {
					logger.Warnw("审批链步骤审批人解析失败", "level", lvl, "role", st.Role, "err", err)
				}
				continue
			}
			allApprovers = append(allApprovers, ids...)
			allNames = append(allNames, names...)
		}
		allApprovers = dedupeInts(allApprovers)
		allNames = dedupeStrings(allNames)

		lv := ApprovalLevelEval{
			Level:         lvl,
			Required:      levelRequired,
			ApprovalType:  approvalType,
			Threshold:     threshold,
			ApproverIDs:   allApprovers,
			ApproverNames: allNames,
		}

		switch {
		case len(allApprovers) == 0 && levelRequired:
			// 必需层却无人可审 → 触发 fallback
			lv.FallbackTriggered = true
			lv.FallbackAction = fallbackAction
			switch fallbackAction {
			case FallbackAutoApprove:
				lv.Status = "satisfied"
			case FallbackEscalate:
				escIDs, escNames, eerr := resolveFallbackApprovers(ctx, client, logger, fallbackApproverID, fallbackRole, evalCtx)
				if eerr == nil && len(escIDs) > 0 {
					lv.ApproverIDs = escIDs
					lv.ApproverNames = escNames
					lv.Status = "pending"
				} else {
					lv.Status = "blocked"
					blocked = true
				}
			default: // block / auto_reject / 未知 → 失败关闭
				lv.Status = "blocked"
				blocked = true
			}
		case len(allApprovers) == 0 && !levelRequired:
			// 非必需层且无审批人 → 视为自动通过
			lv.Status = "satisfied"
		default:
			// 计算阈值
			if approvalType == "parallel" || approvalType == "all" {
				if lv.Threshold <= 0 {
					lv.Threshold = len(allApprovers)
				}
			} else {
				lv.Threshold = 1 // serial / or
			}
			approved := 0
			if approvals != nil {
				for _, a := range approvals[lvl] {
					if intInSlice(a, allApprovers) {
						approved++
					}
				}
			}
			if approved >= lv.Threshold {
				lv.Status = "satisfied"
			} else {
				lv.Status = "pending"
			}
		}

		result.Levels = append(result.Levels, lv)
	}

	// 汇总整体状态
	result.Blocked = blocked
	passed := true
	pending := 0
	for _, lv := range result.Levels {
		if lv.Status == "blocked" {
			passed = false
			if pending == 0 {
				pending = lv.Level
			}
			break
		}
		if lv.Status != "satisfied" {
			passed = false
			if pending == 0 {
				pending = lv.Level
			}
		}
	}
	result.Passed = passed
	result.PendingLevel = pending

	return result, nil
}

// Evaluate 是 ApprovalChainService 上的便捷封装。
func (s *ApprovalChainService) Evaluate(ctx context.Context, chain *ent.ApprovalChain, evalCtx ApprovalEvalContext, approvals map[int][]int) (*ApprovalChainEvaluation, error) {
	return EvaluateApprovalChain(ctx, s.client, s.logger, chain, evalCtx, approvals)
}

// FindActiveChainByEntityType 查找租户内某实体类型的激活审批链（最新创建优先）。
// 找不到时返回 (nil, nil)，调用方据此决定走默认审批或拒绝。
func (s *ApprovalChainService) FindActiveChainByEntityType(ctx context.Context, tenantID int, entityType string) (*ent.ApprovalChain, error) {
	chain, err := s.client.ApprovalChain.Query().
		Where(
			approvalchain.TenantIDEQ(tenantID),
			approvalchain.EntityTypeEQ(entityType),
			approvalchain.StatusEQ("active"),
		).
		Order(ent.Desc(approvalchain.FieldCreatedAt)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return chain, nil
}

// ResolveApprovalPlan 查找匹配链并立即求值，返回可直接用于生成审批记录的计划。
// 这是消费审批链求值引擎的统一入口（替换原先「无审批人时自审批 / 硬编码层级」等缺陷路径）。
func (s *ApprovalChainService) ResolveApprovalPlan(ctx context.Context, tenantID int, entityType string, evalCtx ApprovalEvalContext, approvals map[int][]int) (*ApprovalChainEvaluation, error) {
	evalCtx.TenantID = tenantID
	evalCtx.EntityType = entityType
	chain, err := s.FindActiveChainByEntityType(ctx, tenantID, entityType)
	if err != nil {
		return nil, err
	}
	if chain == nil {
		return nil, fmt.Errorf("未找到实体类型 %s 的激活审批链（租户 %d）", entityType, tenantID)
	}
	return s.Evaluate(ctx, chain, evalCtx, approvals)
}

// resolveChainApprovers 把一个 step 解析为「候选审批人(全量，用于会签)」,
// 全程按 tenantID 过滤，显式 ApproverID 也校验租户归属。
func resolveChainApprovers(ctx context.Context, client *ent.Client, logger *zap.SugaredLogger, step schema.ApprovalChainStep, evalCtx ApprovalEvalContext) ([]int, []string, error) {
	if step.ApproverID > 0 {
		u, err := client.User.Query().
			Where(user.ID(step.ApproverID), user.TenantIDEQ(evalCtx.TenantID)).
			Only(ctx)
		if err != nil || u == nil {
			return nil, nil, fmt.Errorf("审批人 %d 不属于租户 %d", step.ApproverID, evalCtx.TenantID)
		}
		return []int{u.ID}, []string{u.Name}, nil
	}
	if step.Role == "" {
		return nil, nil, nil
	}

	assigneeType := step.Role
	assigneeValue := step.Role
	if idx := strings.Index(step.Role, ":"); idx > 0 {
		assigneeType = step.Role[:idx]
		assigneeValue = step.Role[idx+1:]
	}

	switch assigneeType {
	case "role":
		return usersByRole(client, evalCtx.TenantID, assigneeValue)
	case "group":
		return bpmn.NewGroupResolver(client).ExpandGroupsToUsers(ctx, evalCtx.TenantID, assigneeValue)
	case "dept_manager", "team_leader", "project_manager", "temp_team_leader":
		scopeID, err := strconv.Atoi(assigneeValue)
		if err != nil {
			return nil, nil, fmt.Errorf("无效的审批人范围ID: %s", assigneeValue)
		}
		appCtx := &approver.ApproverContext{TenantID: evalCtx.TenantID}
		switch assigneeType {
		case "dept_manager":
			appCtx.DepartmentID = scopeID
		case "team_leader", "temp_team_leader":
			appCtx.TeamID = scopeID
		case "project_manager":
			appCtx.ProjectID = scopeID
		}
		registry := approver.NewResolverRegistry(logger)
		registry.Register(approver.NewDeptManagerResolver())
		registry.Register(approver.NewTeamLeaderResolver())
		registry.Register(approver.NewProjectMgrResolver())
		registry.Register(approver.NewTempTeamResolver())
		infos, err := registry.Resolve(ctx, client, assigneeType, appCtx)
		if err != nil {
			return nil, nil, err
		}
		return approverInfosToIDs(infos)
	case "amount_based":
		thresholds, err := parseAmountThresholds(assigneeValue)
		if err != nil {
			return nil, nil, err
		}
		registry := approver.NewResolverRegistry(logger)
		registry.Register(approver.NewAmountResolver(thresholds))
		infos, err := registry.Resolve(ctx, client, "amount_based", &approver.ApproverContext{
			TenantID: evalCtx.TenantID,
			Amount:   evalCtx.Amount,
		})
		if err != nil {
			return nil, nil, err
		}
		return approverInfosToIDs(infos)
	case "cab":
		// CAB/ECAB 委员会成员，租户隔离、仅活跃成员。使「CAB 必须审批」成为
		// 审批链上普通一步，自动获得引擎的租户隔离 + 会签/或签 + fallback。
		return resolveCABMembers(ctx, client, evalCtx.TenantID, assigneeValue)
	default:
		// 当作纯角色名处理
		return usersByRole(client, evalCtx.TenantID, assigneeType)
	}
}

// resolveCABMembers 按委员会类型（CAB/ECAB）解析活跃成员用户，租户隔离。
func resolveCABMembers(ctx context.Context, client *ent.Client, tenantID int, boardType string) ([]int, []string, error) {
	if boardType != "CAB" && boardType != "ECAB" {
		return nil, nil, fmt.Errorf("无效的 CAB 委员会类型: %s", boardType)
	}
	members, err := client.CABMember.Query().
		Where(cabmember.Type(boardType), cabmember.TenantID(tenantID), cabmember.IsActive(true)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}
	ids := make([]int, 0, len(members))
	names := make([]string, 0, len(members))
	for _, m := range members {
		u, uerr := client.User.Get(ctx, m.UserID)
		if uerr != nil || u == nil {
			continue
		}
		ids = append(ids, u.ID)
		names = append(names, u.Name)
	}
	return ids, names, nil
}

// usersByRole 返回租户内某角色的全部有效用户（会签需要全量）。
func usersByRole(client *ent.Client, tenantID int, role string) ([]int, []string, error) {
	users, err := client.User.Query().
		Where(user.RoleEQ(user.Role(role)), user.TenantIDEQ(tenantID), user.Active(true)).
		All(context.Background())
	if err != nil {
		return nil, nil, err
	}
	ids := make([]int, 0, len(users))
	names := make([]string, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
		names = append(names, u.Name)
	}
	return ids, names, nil
}

// resolveFallbackApprovers escalate 时解析兜底审批人。
func resolveFallbackApprovers(ctx context.Context, client *ent.Client, logger *zap.SugaredLogger, fallbackApproverID int, fallbackRole string, evalCtx ApprovalEvalContext) ([]int, []string, error) {
	if fallbackApproverID > 0 {
		u, err := client.User.Query().
			Where(user.ID(fallbackApproverID), user.TenantIDEQ(evalCtx.TenantID)).
			Only(ctx)
		if err == nil && u != nil {
			return []int{u.ID}, []string{u.Name}, nil
		}
	}
	if fallbackRole != "" {
		return resolveChainApprovers(ctx, client, logger, schema.ApprovalChainStep{Role: fallbackRole}, evalCtx)
	}
	return nil, nil, fmt.Errorf("无可用兜底审批人")
}

// approverInfosToIDs 把 ApproverInfo 列表转换为 ID/Name 切片。
func approverInfosToIDs(infos []*approver.ApproverInfo) ([]int, []string, error) {
	ids := make([]int, 0, len(infos))
	names := make([]string, 0, len(infos))
	for _, a := range infos {
		ids = append(ids, a.UserID)
		names = append(names, a.UserName)
	}
	return ids, names, nil
}

// ---- 小工具 ----

func dedupeInts(in []int) []int {
	if len(in) == 0 {
		return in
	}
	seen := make(map[int]struct{}, len(in))
	out := make([]int, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

func dedupeStrings(in []string) []string {
	if len(in) == 0 {
		return in
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

func intInSlice(v int, list []int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
