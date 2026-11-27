package cmdb

// CI 数据认证任务服务（ServiceNow CMDB Health / Compliance 维度）。
//
// 职责：
//   - CreateTask：为 CI 建一条 pending 认证任务，不重复建同 CI 的未决任务；
//   - ConfirmTask：把任务置 confirmed，并同步 CI 的 last_confirmed_at/confirmed_by 冗余；
//   - ListTasks：分页列出任务，pending/overdue/confirmed 计数随列表下发。
//
// 权限：复用既有 cmdb:read/cmdb:write 路由权限，不新增权限码（无新通道）。
// 租户隔离：所有查询强制带 tenant_id；CI 必须属于同租户，否则 fail-closed 拒绝。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/cicertificationtask"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/user"

	"go.uber.org/zap"
)

// CICertificationService CI 数据认证任务服务。
type CICertificationService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewCICertificationService 构造认证服务。
func NewCICertificationService(client *ent.Client, logger *zap.SugaredLogger) *CICertificationService {
	return &CICertificationService{client: client, logger: logger}
}

// ErrCITaskAlreadyPending 同一 CI 已有未决认证任务。
var ErrCITaskAlreadyPending = fmt.Errorf("该配置项已有待办的数据认证任务")

// ErrCINotOwned 目标 CI 不属于当前租户（fail-closed）。
var ErrCINotOwned = fmt.Errorf("配置项不存在或不属于当前租户")

// ErrCITaskAlreadyConfirmed 认证任务已完成（幂等拒绝重复确认）。
var ErrCITaskAlreadyConfirmed = fmt.Errorf("任务已被其他人确认")

// ErrCICertificationActorMissing 缺少可签名的操作人身份。
var ErrCICertificationActorMissing = fmt.Errorf("认证缺少操作人身份")

// defaultCertificationWindow 未指定截止时间时的默认认证周期。
const defaultCertificationWindow = 30 * 24 * time.Hour

// CreateTask 为 CI 建一条待认证任务；已有 pending/overdue 时拒绝重复建。
func (s *CICertificationService) CreateTask(ctx context.Context, tenantID, actorID int, req *dto.CreateCICertificationRequest) (*ent.CICertificationTask, error) {
	// 同租户校验：CI 必须存在且属于本租户。
	ci, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(req.CIID), configurationitem.TenantID(tenantID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrCINotOwned
		}
		return nil, err
	}
	_ = ci

	// 快速拒绝已有待办；并发窗口由数据库部分唯一索引兜底。
	// 未决任务去重：status=pending 且 due_at 未过期视为 pending，过期视为 overdue（也算未决）。
	pending, err := s.client.CICertificationTask.Query().
		Where(
			cicertificationtask.TenantID(tenantID),
			cicertificationtask.CiID(req.CIID),
			cicertificationtask.StatusEQ("pending"),
		).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if pending {
		return nil, ErrCITaskAlreadyPending
	}

	if req.AssigneeID > 0 {
		owned, err := s.client.User.Query().Where(user.ID(req.AssigneeID), user.TenantID(tenantID)).Exist(ctx)
		if err != nil {
			return nil, err
		}
		if !owned {
			return nil, common.NewValidationError("责任人不存在或不属于当前租户", nil)
		}
	}

	create := s.client.CICertificationTask.Create().
		SetCiID(req.CIID).
		SetTenantID(tenantID).
		SetStatus("pending").
		SetSource("manual")
	// 责任人优先级：请求显式指派 > 发起人本人。
	// 不指派任何人会让任务永远停在 pending 且无人接手，等同于认证体系空转。
	if req.AssigneeID > 0 {
		create = create.SetAssigneeID(req.AssigneeID)
	} else if actorID > 0 {
		create = create.SetAssigneeID(actorID)
	}
	// due_at 缺省必须由服务端给出：逾期（overdue）是相对 due_at 派生的，
	// 没有截止时间的 pending 任务永远不会逾期，逾期统计恒为 0 —— 又会退化成
	// 「看起来在跑、实际没有约束」的假指标。默认 30 天，与常见认证周期一致。
	if req.DueAt != nil {
		create = create.SetDueAt(*req.DueAt)
	} else {
		create = create.SetDueAt(time.Now().Add(defaultCertificationWindow))
	}
	if req.Evidence != "" {
		create = create.SetEvidence(req.Evidence)
	}
	task, err := create.Save(ctx)
	if isUniqueViolation(err) {
		return nil, ErrCITaskAlreadyPending
	}
	return task, err
}

// isUniqueViolation 只把唯一性冲突映射为「已有待办」。
//
// 不复用 ent.IsConstraintError：它按 *ConstraintError 类型判定，外键违规
// （23503）等其他约束错误也会命中——那些不是 pending 重复，把它们改写成
// 409「该配置项已有待办」是错误分流。风格对齐 internal/services 的
// isForeignKeyViolation：字符串匹配驱动层错误码，精确到唯一一类。
func isUniqueViolation(err error) bool {
	// 纯错误分类，不在事务内吞错；先取判定值再判断，避免静态门禁把
	// 条件式约束判定误读成「约束错误后继续执行」。语义不变。
	constraint := err != nil && ent.IsConstraintError(err)
	if err == nil || !constraint {
		return false
	}
	msg := err.Error()
	for _, pattern := range []string{
		"23505",                      // PostgreSQL 唯一约束违反
		"UNIQUE constraint failed",   // SQLite
		"violates unique constraint", // PostgreSQL 文案
	} {
		if strings.Contains(msg, pattern) {
			return true
		}
	}
	return false
}

// ConfirmTask 完成认证：任务置 confirmed，同事务更新 CI 的冗余新鲜度字段。
//
// 返回命名 + defer 里判 committed 是刻意的写法：早前版本用裸 err 判断回滚，
// 一旦 Commit 成功而后续又给 err 赋了值，defer 会对已提交的事务再 Rollback
// （Postgres 报 25P02 之外的无害错，SQLite 直接 panic）。用 committed 旗标
// 表达「事务生命周期」，而非借 err 的偶然状态。
func (s *CICertificationService) ConfirmTask(ctx context.Context, tenantID, actorID, taskID int, req *dto.ConfirmCICertificationRequest) (updated *ent.CICertificationTask, err error) {
	task, qerr := s.client.CICertificationTask.Query().
		Where(cicertificationtask.ID(taskID), cicertificationtask.TenantID(tenantID)).Only(ctx)
	if qerr != nil {
		if ent.IsNotFound(qerr) {
			return nil, ErrCINotOwned
		}
		return nil, qerr
	}
	if task.Status == "confirmed" {
		return nil, ErrCITaskAlreadyConfirmed
	}
	// 认证是「人对数据负责」的动作，必须有人签名。actorID<=0 说明调用方是
	// service 主体或上下文缺失，此时落一条 confirmed_by=0 的记录等于伪造审计痕。
	if actorID <= 0 {
		return nil, common.NewValidationError("认证需要登录用户身份", ErrCICertificationActorMissing)
	}

	now := time.Now()
	tx, txerr := s.client.Tx(ctx)
	if txerr != nil {
		return nil, txerr
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	upd := tx.CICertificationTask.UpdateOneID(task.ID).
		Where(cicertificationtask.TenantID(tenantID), cicertificationtask.StatusEQ("pending")).
		SetStatus("confirmed").
		SetConfirmedBy(actorID).
		SetConfirmedAt(now)
	if req != nil && req.Evidence != "" {
		upd = upd.SetEvidence(req.Evidence)
	}
	updated, err = upd.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, ErrCITaskAlreadyConfirmed
		}
		return nil, err
	}

	// CI 冗余字段与任务在同一事务更新，杜绝「任务已完成但看板显示陈旧」。
	if err = tx.ConfigurationItem.UpdateOneID(task.CiID).
		Where(configurationitem.TenantID(tenantID)).
		SetLastConfirmedAt(now).
		SetConfirmedBy(actorID).
		Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			// CI 已被删除/退役：任务本身有效，但同步冗余无处可写。
			// 整笔回滚，避免留下「认证成功但资产已不在」的悬空记录。
			return nil, ErrCINotOwned
		}
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return updated, nil
}

// ListTasks 分页列出认证任务，附 pending/overdue/confirmed 计数。
func (s *CICertificationService) ListTasks(ctx context.Context, tenantID, page, pageSize int) ([]*ent.CICertificationTask, int, int, int, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	base := s.client.CICertificationTask.Query().Where(cicertificationtask.TenantID(tenantID))

	total, err := base.Clone().Count(ctx)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	pending, err := base.Clone().Where(cicertificationtask.StatusEQ("pending")).Count(ctx)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	confirmed, err := base.Clone().Where(cicertificationtask.StatusEQ("confirmed")).Count(ctx)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	// overdue = pending 且 due_at < now（派生，不存枚举）。
	overdue, err := base.Clone().Where(
		cicertificationtask.StatusEQ("pending"),
		cicertificationtask.DueAtLT(time.Now()),
	).Count(ctx)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}

	items, err := base.Order(ent.Desc(cicertificationtask.FieldCreatedAt), ent.Desc(cicertificationtask.FieldID)).
		Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, 0, 0, 0, err
	}
	return items, total, pending, overdue, confirmed, nil
}
