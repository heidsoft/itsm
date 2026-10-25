package cmdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/ent"
	"itsm-backend/service/cloud"
)

// 发现差异的人工决策（apply / reject）。
//
// 背景：manual 对账策略下，执行器对「有变化」的资源只落一条 status=pending 的
// discovery_result，不改 CloudResource——这是对的（生产属性变更要人确认），但
// 全仓没有**任何**写入面能让这条差异被应用或驳回：`/cmdb/reconciliation` 只有
// 只读 GET，`/discovery/results` 只有列表。于是对账永远停在「算出了差异」，
// 账合不上，这也是 cmdbReconciliation 维持 Pilot 的真实阻断项。
//
// 语义边界（有意收窄）：
//   - 只有 `action=update` 且 `status=pending` 的差异可被决策。create 在发现时
//     已落库（无需确认）；delete 根本不会产生（缺失只推进 missing_count→stale，
//     防误删）。
//   - 应用先写 CloudResource 的事实，再回写配置项。回写失败只留日志：差异的「应用」
//     针对云上事实，配置项是下游账，不能因为它没记上就把已生效的写入翻成失败。
//   - 应用以 `resource_identity` 定位资源，而不是 result.resource_id：身份哈希
//     是发现的唯一身份口径（含租户/账号/区域/服务/类型），按 resource_id 找会
//     在跨区域重名时命中错的资源。
const (
	DiscoveryResultPending     = "pending"
	DiscoveryResultApplying    = "applying"
	DiscoveryResultApplied     = "applied"
	DiscoveryResultRejected    = "rejected"
	DiscoveryResultApplyFailed = "apply_failed"
)

// ApplyDiscoveryResult 把一条 pending 差异应用到 CloudResource，并回写配置项。
//
// 幂等语义：只有 pending 能被应用；已 applied/rejected 返回 409（重复点击不该
// 产生第二次写入，也不该被当成成功——那会掩盖「这条差异其实早被驳回过」）。
func (s *Service) ApplyDiscoveryResult(ctx context.Context, tenantID, userID, id int) (*DiscoveryResult, error) {
	result, err := s.repo.GetDiscoveryResult(ctx, tenantID, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, common.NewNotFoundError("discovery result")
		}
		return nil, fmt.Errorf("load discovery result: %w", err)
	}
	if err := ensureDecidableDiscoveryResult(result); err != nil {
		return nil, err
	}
	claimed, err := s.repo.TransitionDiscoveryResult(
		ctx, tenantID, id, DiscoveryResultPending, DiscoveryResultApplying, "", "",
	)
	if err != nil {
		return nil, fmt.Errorf("claim discovery result %d: %w", id, err)
	}
	if claimed != 1 {
		return nil, common.NewConflictError("discoveryResult", "差异已被其他操作者处理")
	}

	resource, err := s.repo.GetCloudResourceByIdentity(ctx, tenantID, result.ResourceIdentity)
	if err != nil {
		if ent.IsNotFound(err) {
			s.failApplyingDiscoveryResult(ctx, tenantID, id, "resource_missing", "资源已不存在")
			return nil, common.NewConflictError("discoveryResult",
				"资源已不存在（可能已被删除），请重跑发现后再决策")
		}
		s.failApplyingDiscoveryResult(ctx, tenantID, id, "resource_load_failed", err.Error())
		return nil, fmt.Errorf("load cloud resource by identity: %w", err)
	}

	applyDiscoveredSnapshot(resource, result)
	if _, err := s.repo.UpdateCloudResource(ctx, resource); err != nil {
		s.failApplyingDiscoveryResult(ctx, tenantID, id, "apply_failed", err.Error())
		return nil, fmt.Errorf("apply discovery result %d: %w", id, err)
	}
	projection := s.discoveryRuntime.CIProjector.Project(ctx, resource)

	changed, err := s.repo.TransitionDiscoveryResult(
		ctx, tenantID, id, DiscoveryResultApplying, DiscoveryResultApplied, "", "",
	)
	if err != nil {
		s.logger.Errorw("failed to persist discovery result applied status",
			"error", err, "result_id", id, "tenant_id", tenantID)
		return nil, fmt.Errorf("persist applied discovery result %d: %w", id, err)
	}
	if changed != 1 {
		return nil, common.NewConflictError("discoveryResult", "差异状态已被其他操作者改变")
	}
	result.Status = DiscoveryResultApplied
	result.ErrorCode = ""
	result.ErrorMessage = ""
	result.CIID = projection.CIID
	if projection.CIID > 0 {
		if err := s.repo.UpdateDiscoveryResult(ctx, result); err != nil {
			s.logger.Warnw("failed to attach configuration item to discovery result",
				"error", err, "result_id", id, "ci_id", projection.CIID, "tenant_id", tenantID)
		}
	}
	s.logger.Infow("discovery difference applied",
		"tenant_id", tenantID, "result_id", id, "user_id", userID,
		"resource_identity", result.ResourceIdentity,
		"ci_projection", projection.Status, "ci_id", projection.CIID)
	return result, nil
}

// RejectDiscoveryResult 驳回一条 pending 差异：资源保持现状，差异不再待处理。
//
// 驳回理由写进 error_message（该行状态已是 rejected，此时该字段语义为
// 「决策说明」而非错误）。理由可选：强制填写会让运维在「不值得解释的噪声差异」
// 上随手填个「-」，反而污染记录；真正需要追责的场景由审计中间件记录的
// request_body 兜底。
func (s *Service) RejectDiscoveryResult(ctx context.Context, tenantID, userID, id int, reason string) (*DiscoveryResult, error) {
	result, err := s.repo.GetDiscoveryResult(ctx, tenantID, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, common.NewNotFoundError("discovery result")
		}
		return nil, fmt.Errorf("load discovery result: %w", err)
	}
	if err := ensureDecidableDiscoveryResult(result); err != nil {
		return nil, err
	}

	changed, err := s.repo.TransitionDiscoveryResult(
		ctx, tenantID, id, DiscoveryResultPending, DiscoveryResultRejected, "",
		truncateMessage(strings.TrimSpace(reason), 400),
	)
	if err != nil {
		return nil, fmt.Errorf("persist rejected discovery result %d: %w", id, err)
	}
	if changed != 1 {
		return nil, common.NewConflictError("discoveryResult", "差异已被其他操作者处理")
	}
	result.Status = DiscoveryResultRejected
	result.ErrorCode = ""
	result.ErrorMessage = truncateMessage(strings.TrimSpace(reason), 400)
	s.logger.Infow("discovery difference rejected",
		"tenant_id", tenantID, "result_id", id, "user_id", userID,
		"resource_identity", result.ResourceIdentity)
	return result, nil
}

func (s *Service) failApplyingDiscoveryResult(ctx context.Context, tenantID, id int, code, message string) {
	if _, err := s.repo.TransitionDiscoveryResult(
		ctx, tenantID, id, DiscoveryResultApplying, DiscoveryResultApplyFailed,
		code, truncateMessage(message, 400),
	); err != nil {
		s.logger.Errorw("failed to persist discovery result apply failure",
			"error", err, "result_id", id, "tenant_id", tenantID)
	}
}

func ensureDecidableDiscoveryResult(result *DiscoveryResult) error {
	if result.Status != DiscoveryResultPending {
		return common.NewConflictError("discoveryResult",
			fmt.Sprintf("差异已处理（%s），不能重复决策", result.Status))
	}
	if result.Action != "update" {
		return common.NewConflictError("discoveryResult",
			fmt.Sprintf("只有 update 差异需要人工决策（当前 %s）：create 在发现时已落库，delete 不产生差异", result.Action))
	}
	if strings.TrimSpace(result.ResourceIdentity) == "" {
		return common.NewConflictError("discoveryResult",
			"差异缺少规范身份哈希，无法定位资源，请重跑发现")
	}
	return nil
}

// applyDiscoveredSnapshot 把发现到的资源事实写回 CloudResource。
//
// 字段映射必须与执行器（discovery_job_executor.go 的 applyResource）一致，
// 否则「自动应用」与「人工应用」会产出两种不同状态，对账口径立刻分叉。
// status 需要走 NormalizeStatus：快照里存的是厂商原始状态（Running/运行中），
// CloudResource 存的是归一后的枚举。
func applyDiscoveredSnapshot(resource *CloudResource, result *DiscoveryResult) {
	snap := result.ResourceSnapshot
	if name, ok := snapString(snap, "resourceName"); ok {
		resource.ResourceName = name
	}
	if raw, ok := snapString(snap, "status"); ok {
		resource.Status = cloud.NormalizeStatus(raw)
	}
	if region, ok := snapString(snap, "region"); ok && strings.TrimSpace(region) != "" {
		resource.Region = strings.ToLower(strings.TrimSpace(region))
	}
	if zone, ok := snapString(snap, "zone"); ok {
		resource.Zone = strings.ToLower(strings.TrimSpace(zone))
	}
	if tags, ok := snap["tags"].(map[string]interface{}); ok {
		resource.Tags = stringifyTagMap(tags)
	} else if tags, ok := snap["tags"].(map[string]string); ok {
		resource.Tags = tags
	}
	if extra, ok := snap["extra"].(map[string]interface{}); ok {
		resource.Metadata = extra
	}
	// 内容指纹换成「被发现的那一版」，否则下一轮发现会再次判为 update，
	// 人工应用等于没生效。
	if result.AfterHash != "" {
		resource.SourceFingerprint = result.AfterHash
	}
	resource.MissingCount = 0
	resource.LifecycleState = "active"
	now := time.Now()
	resource.LastSeenAt = &now
}

func snapString(snap map[string]interface{}, key string) (string, bool) {
	if snap == nil {
		return "", false
	}
	raw, ok := snap[key]
	if !ok {
		return "", false
	}
	value, ok := raw.(string)
	if !ok {
		return "", false
	}
	return value, true
}

// stringifyTagMap 把 JSON 反序列化后的 map[string]interface{} 收敛为
// CloudResource.Tags 需要的 map[string]string。标签值在云上可能是数字/布尔，
// 直接丢弃非字符串值会让「应用差异」少写几个标签，所以统一转字符串。
func stringifyTagMap(tags map[string]interface{}) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string]string, len(tags))
	for k, v := range tags {
		switch value := v.(type) {
		case string:
			out[k] = value
		case nil:
			continue
		default:
			out[k] = fmt.Sprint(value)
		}
	}
	return out
}

func truncateMessage(value string, limit int) string {
	trimmed := strings.TrimSpace(value)
	if limit <= 0 || len([]rune(trimmed)) <= limit {
		return trimmed
	}
	runes := []rune(trimmed)
	return string(runes[:limit]) + "…"
}
