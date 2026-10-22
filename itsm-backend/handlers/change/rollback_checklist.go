package change

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"itsm-backend/common"
)

var (
	// ErrRollbackStepsUnresolved 回滚清单里还有未确认的步骤（→ 422 可行动校验）。
	// 语义：回滚没走完清单就不许把变更记成「已回滚」——否则审计看到的
	// rolled_back 可能是「回滚到一半」，PIR 据此判断「影响已复原」就是假结论。
	ErrRollbackStepsUnresolved = errors.New("rollback checklist has unresolved steps")
	// ErrRollbackStepNotFound 步骤不存在，或不属于该变更/租户（→ 404，
	// 跨租户统一 404 不暴露可枚举信号）。
	ErrRollbackStepNotFound = errors.New("rollback step not found")
	// ErrRollbackChecklistLocked 变更已收尾，回滚清单不可再改（→ 409）。
	// 回滚清单是审计事实：已回滚/已完成/已关闭之后回头改写它，
	// 等于让「当时到底做了什么」变成事后可编辑的说法。
	ErrRollbackChecklistLocked = errors.New("rollback checklist is locked")
	// ErrRollbackStepAlreadyResolved 该步骤已被别人确认（→ 409 并发冲突）。
	ErrRollbackStepAlreadyResolved = errors.New("rollback step already resolved")
)

// unprocessable 构造 422/4220 且保留哨兵错误（errors.Is 可判定）。
//
// 不用 common.NewValidationError：它是 400/1002。回滚清单未确认属于
// 「请求本身合法、但当前业务状态不接受」——客户端不该改参数重试，而应先走完清单。
func unprocessable(msg string, cause error) *common.AppError {
	return common.NewAppError(common.ErrCodeValidation, msg, http.StatusUnprocessableEntity, cause)
}

// rollbackConflict 构造 409/4090 且保留哨兵错误。
func rollbackConflict(msg string, cause error) *common.AppError {
	return common.NewAppError(common.ErrCodeConflict, msg, http.StatusConflict, cause)
}

const (
	// rollbackChecklistMaxSteps 单张清单的步数上限：回滚清单是应急时执行的
	// 短清单，超过这个量级说明应该拆成多个变更，而不是一路确认下去。
	rollbackChecklistMaxSteps = 50
	// rollbackStepMaxLen 单步描述长度上限（按 rune 计）。
	rollbackStepMaxLen = 500
)

// rollbackChecklistEditable 判断当前状态是否允许编排/修改回滚清单。
//
// 只允许实施开始之前（draft/pending/approved/scheduled）。
// 刻意**不包含** in_progress 与 failed：那两个状态正是回滚被执行的时候，
// 若此时还能整表覆盖，操作人就可以「清空清单 → 一次状态翻转」绕过
// validateRollbackChecklist 的逐项确认门禁，清单也就成了摆设。
// 执行中发现某步不适用，正确做法是把它标记为 skipped 并写明依据，
// 而不是把计划改成"事后看起来执行过的样子"。
func rollbackChecklistEditable(status string) bool {
	switch status {
	case "draft", "pending", "approved", "scheduled":
		return true
	default:
		return false
	}
}

// rollbackStepActionable 判断当前状态是否允许逐项确认回滚结果。
// 只有正在实施或刚失败时才有「执行回滚」这件事；把确认动作限制在这两个
// 状态，可以避免有人在草稿阶段先勾满清单，然后再靠一次状态翻转"秒回滚"。
func rollbackStepActionable(status string) bool {
	return status == "in_progress" || status == "failed"
}

// GetRollbackSteps 读取回滚清单（只读，任何能读变更的角色都可看）。
func (s *Service) GetRollbackSteps(ctx context.Context, changeID, tenantID int) ([]*RollbackStep, error) {
	// 先确认变更存在且属于本租户：子表没有 RLS，若不校验父对象，
	// 跨租户的 change_id 只会返回空清单（看似"没有清单"），
	// 掩盖了越权探测的成功信号。
	if _, err := s.repo.Get(ctx, changeID, tenantID); err != nil {
		return nil, fmt.Errorf("%w: id=%d", ErrChangeNotFound, changeID)
	}
	return s.repo.ListRollbackSteps(ctx, changeID, tenantID)
}

// ReplaceRollbackSteps 用给定描述覆盖整张回滚清单。
//
// 覆盖式写入而非增量增删：清单是有序整体（"先摘流量，再回滚镜像，最后验活"），
// 增量编辑会让顺序与内容在并发下产生中间态，回滚时按错序执行比没有清单更危险。
func (s *Service) ReplaceRollbackSteps(ctx context.Context, changeID, tenantID int, descriptions []string) ([]*RollbackStep, error) {
	c, err := s.repo.Get(ctx, changeID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("%w: id=%d", ErrChangeNotFound, changeID)
	}
	if !rollbackChecklistEditable(c.Status) {
		return nil, rollbackConflict(
			fmt.Sprintf("变更当前状态为 '%s'，回滚清单已锁定（只有 draft/pending/approved/scheduled 可编排）。"+
				"实施开始后清单即成为执行依据：要说明偏差请把对应步骤标记为 skipped 并写明依据，"+
				"而不是改写计划", c.Status),
			ErrRollbackChecklistLocked,
		)
	}
	if len(descriptions) > rollbackChecklistMaxSteps {
		return nil, unprocessable(
			fmt.Sprintf("回滚清单最多 %d 步（当前 %d 步）。超过这个量级说明回滚范围过大，"+
				"建议拆成多个变更分别编排与验收", rollbackChecklistMaxSteps, len(descriptions)),
			ErrRollbackStepsUnresolved,
		)
	}

	steps := make([]*RollbackStep, 0, len(descriptions))
	for i, raw := range descriptions {
		desc := strings.TrimSpace(raw)
		if desc == "" {
			return nil, unprocessable(
				fmt.Sprintf("第 %d 步内容为空。清单里不允许空步骤——空步骤无法执行，"+
					"却会让「清单已走完」的门禁失去意义", i+1),
				ErrRollbackStepsUnresolved,
			)
		}
		if len([]rune(desc)) > rollbackStepMaxLen {
			return nil, unprocessable(
				fmt.Sprintf("第 %d 步超过 %d 字，请拆成更具体的单步动作（一步只做一件事，才便于逐项确认）",
					i+1, rollbackStepMaxLen),
				ErrRollbackStepsUnresolved,
			)
		}
		steps = append(steps, &RollbackStep{ChangeID: changeID, TenantID: tenantID, StepNo: i + 1, Description: desc})
	}
	if err := s.repo.ReplaceRollbackSteps(ctx, changeID, tenantID, steps); err != nil {
		return nil, fmt.Errorf("保存回滚清单失败: %w", err)
	}
	s.logger.Infow("回滚清单已更新", "change_id", changeID, "tenant_id", tenantID, "steps", len(steps))
	return s.repo.ListRollbackSteps(ctx, changeID, tenantID)
}

// ResolveRollbackStep 确认某一步的执行结果（done/skipped/failed，或回到 pending 撤销误操作）。
//
// skipped 与 failed 必须带备注：这两者都是「没按计划执行」，
// 没有理由的跳过/失败在事后审计里与「漏做」无法区分，
// 而漏做恰恰是回滚最危险的形态（以为恢复了、其实没恢复）。
func (s *Service) ResolveRollbackStep(ctx context.Context, changeID, tenantID, stepID, actorID int, status, note string) (*RollbackStep, error) {
	status = strings.TrimSpace(status)
	note = strings.TrimSpace(note)
	if !IsRollbackStepStatusValid(status) {
		return nil, unprocessable(
			fmt.Sprintf("步骤状态 '%s' 不合法，只能是 pending/done/skipped/failed", status),
			ErrRollbackStepsUnresolved,
		)
	}

	c, err := s.repo.Get(ctx, changeID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("%w: id=%d", ErrChangeNotFound, changeID)
	}
	if !rollbackStepActionable(c.Status) {
		return nil, rollbackConflict(
			fmt.Sprintf("变更当前状态为 '%s'，不能确认回滚步骤。"+
				"回滚步骤只能在实施中（in_progress）或实施失败（failed）时逐项确认", c.Status),
			ErrRollbackChecklistLocked,
		)
	}
	if status == RollbackStepSkipped || status == RollbackStepFailed {
		if note == "" {
			return nil, unprocessable(
				fmt.Sprintf("把第该步标记为 '%s' 必须填写备注：跳过要写不适用依据，"+
					"失败要写现象与残留影响——否则审计无法区分「有意跳过」和「漏做」", status),
				ErrRollbackStepsUnresolved,
			)
		}
	}

	// status=pending 表示撤销误点：清空确认人/时间，但保留备注作为线索。
	if status == RollbackStepPending {
		actorID = 0
	}
	ok, err := s.repo.ResolveRollbackStep(ctx, stepID, changeID, tenantID, actorID, status, note, time.Now())
	if err != nil {
		return nil, fmt.Errorf("确认回滚步骤失败: %w", err)
	}
	if !ok {
		// 未命中：要么步骤不存在/跨租户，要么已被别人确认过。
		// 重新读一次清单来区分——把「来晚了」误报成 404 会让操作人以为链接失效。
		steps, listErr := s.repo.ListRollbackSteps(ctx, changeID, tenantID)
		if listErr == nil {
			for _, step := range steps {
				if step.ID == stepID {
					return step, rollbackConflict(
						fmt.Sprintf("第 %d 步已被 '%s' 于 %s 确认为 '%s'，请刷新后查看最新结果",
							step.StepNo, fallbackString(step.ResolvedByName, "其他操作人"),
							formatTimePtr(step.ResolvedAt), step.Status),
						ErrRollbackStepAlreadyResolved,
					)
				}
			}
		}
		return nil, fmt.Errorf("%w: step_id=%d change_id=%d", ErrRollbackStepNotFound, stepID, changeID)
	}

	steps, err := s.repo.ListRollbackSteps(ctx, changeID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("读取回滚清单失败: %w", err)
	}
	for _, step := range steps {
		if step.ID == stepID {
			return step, nil
		}
	}
	return nil, fmt.Errorf("%w: step_id=%d", ErrRollbackStepNotFound, stepID)
}

// validateRollbackChecklist 回滚门禁：清单未走完不许标记 rolled_back。
//
// 设计取舍（为什么不是「必须配清单才能回滚」）：
//   - 强制配清单会拦住"没来得及编排就出事"的应急回滚，反而诱导使用者
//     先去点别的状态绕过门禁；生产库里既有变更也全都没有清单。
//   - 但只要清单存在，就必须逐项确认——这才是「可验证的回退」：
//     计划了几步、每步结果如何、谁在什么时候确认，全部是行级事实。
//
// 读取失败按失败关闭处理（不允许在"读不到清单"时放行回滚）：
// 读失败意味着无法证明清单已走完，而回滚是不可逆动作。
func (s *Service) validateRollbackChecklist(ctx context.Context, changeID, tenantID int) error {
	steps, err := s.repo.ListRollbackSteps(ctx, changeID, tenantID)
	if err != nil {
		return fmt.Errorf("读取回滚清单失败，无法确认回滚是否已逐项执行: %w", err)
	}
	if len(steps) == 0 {
		return nil
	}
	var pending []string
	var failed []string
	for _, step := range steps {
		switch step.Status {
		case RollbackStepPending:
			pending = append(pending, fmt.Sprintf("第%d步", step.StepNo))
		case RollbackStepFailed:
			failed = append(failed, fmt.Sprintf("第%d步", step.StepNo))
		}
	}
	if len(pending) > 0 {
		return unprocessable(
			fmt.Sprintf("回滚清单还有 %d 步未确认（%s）。请逐项确认执行结果后再把变更标记为已回滚："+
				"未走完的清单如果直接置为 rolled_back，审计与 PIR 都会以为影响已完全复原",
				len(pending), strings.Join(pending, "、")),
			ErrRollbackStepsUnresolved,
		)
	}
	if len(failed) > 0 {
		// 有失败步骤时仍然允许收尾，但提示必须显式承认「回滚不完整」：
		// 拦死会让人无法收尾（甚至谎报 completed），放行而无声则等于默许假复原。
		s.logger.Warnw("变更回滚存在失败步骤，按「回滚不完整」收尾",
			"change_id", changeID, "tenant_id", tenantID, "failed_steps", failed)
	}
	return nil
}

func fallbackString(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return "未知时间"
	}
	return t.Format("2006-01-02 15:04")
}
