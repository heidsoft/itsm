package service

import (
	"context"
	"encoding/json"
	"reflect"

	"go.uber.org/zap"
)

// ApprovalChainShadowComparator 影子双轨比较器。
//
// 旧路径（直接调用 ResolveApprovalPlan）仍为真实执行路径；
// 新路径（经 ApprovalChainResolverAdapter）影子执行，结果只写日志不影响用户。
// 每次调用对比两条路径的输出，差异记录到审计日志。
// 差异率 < 0.1% 持续 3 天后可安全切换到新路径。
type ApprovalChainShadowComparator struct {
	adapter *ApprovalChainResolverAdapter
	logger  *zap.SugaredLogger
}

// NewApprovalChainShadowComparator 创建影子比较器。
func NewApprovalChainShadowComparator(adapter *ApprovalChainResolverAdapter, logger *zap.SugaredLogger) *ApprovalChainShadowComparator {
	return &ApprovalChainShadowComparator{adapter: adapter, logger: logger}
}

// ShadowResult 记录影子路径的执行结果。
type ShadowResult struct {
	ChainID     int
	Passed      bool
	PendingLevel int
	Blocked     bool
	LevelsJSON  []byte
	Err         error
}

// Compare 对比直接路径结果与影子路径结果，差异写日志。
// directPlan 为直接路径的 ResolveApprovalPlan 返回值（可为 nil 表示直接路径报错）。
// directErr 为直接路径的错误。
func (c *ApprovalChainShadowComparator) Compare(
	ctx context.Context,
	tenantID int,
	entityType string,
	requesterID int,
	priority string,
	amount float64,
	approvals map[int][]int,
	directPlan *ApprovalChainEvaluation,
	directErr error,
) ShadowResult {
	shadowChainID, shadowLevelsJSON, shadowPassed, shadowPendingLevel, shadowBlocked, shadowErr :=
		c.adapter.ResolveApprovalPlanRaw(ctx, tenantID, entityType, requesterID, priority, amount, approvals)

	result := ShadowResult{
		ChainID:     shadowChainID,
		Passed:      shadowPassed,
		PendingLevel: shadowPendingLevel,
		Blocked:     shadowBlocked,
		LevelsJSON:  shadowLevelsJSON,
		Err:         shadowErr,
	}

	c.logDifferences(tenantID, entityType, requesterID, directPlan, directErr, &result)
	return result
}

func (c *ApprovalChainShadowComparator) logDifferences(
	tenantID int,
	entityType string,
	requesterID int,
	directPlan *ApprovalChainEvaluation,
	directErr error,
	shadow *ShadowResult,
) {
	baseFields := []interface{}{
		"tenant_id", tenantID,
		"entity_type", entityType,
		"requester_id", requesterID,
	}

	if directErr != nil && shadow.Err != nil {
		c.logger.Debugw("shadow: both paths error", append(baseFields,
			"direct_err", directErr.Error(),
			"shadow_err", shadow.Err.Error(),
		)...)
		return
	}
	if directErr != nil && shadow.Err == nil {
		c.logger.Warnw("shadow: divergence — direct error but shadow success", append(baseFields,
			"direct_err", directErr.Error(),
			"shadow_chain_id", shadow.ChainID,
			"shadow_passed", shadow.Passed,
			"shadow_blocked", shadow.Blocked,
		)...)
		return
	}
	if directErr == nil && shadow.Err != nil {
		c.logger.Warnw("shadow: divergence — direct success but shadow error", append(baseFields,
			"shadow_err", shadow.Err.Error(),
			"direct_chain_id", directPlan.ChainID,
			"direct_passed", directPlan.Passed,
			"direct_blocked", directPlan.Blocked,
		)...)
		return
	}

	var diffs []string
	if directPlan.ChainID != shadow.ChainID {
		diffs = append(diffs, "chain_id")
	}
	if directPlan.Passed != shadow.Passed {
		diffs = append(diffs, "passed")
	}
	if directPlan.PendingLevel != shadow.PendingLevel {
		diffs = append(diffs, "pending_level")
	}
	if directPlan.Blocked != shadow.Blocked {
		diffs = append(diffs, "blocked")
	}

	if len(shadow.LevelsJSON) > 0 {
		var shadowLevels []ApprovalLevelEval
		if err := json.Unmarshal(shadow.LevelsJSON, &shadowLevels); err == nil {
			if !reflect.DeepEqual(directPlan.Levels, shadowLevels) {
				diffs = append(diffs, "levels")
			}
		}
	}

	if len(diffs) > 0 {
		c.logger.Warnw("shadow: divergence detected", append(baseFields,
			"diff_fields", diffs,
			"direct_chain_id", directPlan.ChainID,
			"shadow_chain_id", shadow.ChainID,
			"direct_passed", directPlan.Passed,
			"shadow_passed", shadow.Passed,
			"direct_blocked", directPlan.Blocked,
			"shadow_blocked", shadow.Blocked,
		)...)
	} else {
		c.logger.Debugw("shadow: paths match", append(baseFields,
			"chain_id", directPlan.ChainID,
			"passed", directPlan.Passed,
		)...)
	}
}
