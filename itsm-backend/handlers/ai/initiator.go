package ai

// 发起人身份新鲜度：审批通过后执行前，必须重新加载发起人的当前账号状态与角色。
// RoleSnapshot 只作历史解释，不能作为执行授权依据——否则用户被降权/停用后，
// 已排队的调用仍按旧角色执行。
//
// 本文件只负责「解析身份」：权限判定仍统一走 evaluateToolCall →
// AuthorizeResourceForRole / authz.Registry，不在本层复制任何授权逻辑。

import (
	"context"
	"errors"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/user"
	"itsm-backend/internal/authz"
)

var (
	ErrInitiatorInactive = errors.New("initiator is inactive or deleted")
	ErrInitiatorNotFound = errors.New("initiator not found")
	ErrInitiatorTenant   = errors.New("initiator tenant mismatch")
)

// ResolveCurrentInitiator 返回当前生效的角色。
//
// principal=service 时 userID 恒为 0，无人类用户行可查，直接返回调用方传入的
// 载体角色（其权限仍由 Gate 2 RBAC 判定）；human/agent 必须查到在职用户。
func (s *Service) ResolveCurrentInitiator(ctx context.Context, tenantID, userID int, principal, snapshotRole string) (string, error) {
	if principal == authz.PrincipalService {
		// service 的载体角色由 token 校验保证（AuthMiddleware 强制 audit_admin），
		// 这里不能凭 invocation 的旧快照提权：若快照为空则 fail-closed。
		if snapshotRole == "" {
			return "", fmt.Errorf("%w: service principal without carrier role", ErrInitiatorNotFound)
		}
		return snapshotRole, nil
	}
	if userID <= 0 {
		return "", fmt.Errorf("%w: principal %s requires positive user_id", ErrInitiatorNotFound, principal)
	}
	if s.entClient == nil {
		return "", fmt.Errorf("initiator resolution unavailable: ent client not wired")
	}
	u, err := s.entClient.User.Query().
		Where(user.ID(userID), user.TenantIDEQ(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return "", fmt.Errorf("%w: user_id=%d tenant=%d", ErrInitiatorNotFound, userID, tenantID)
		}
		return "", fmt.Errorf("initiator lookup failed: %w", err)
	}
	if !u.Active || u.DeletedAt != nil {
		return "", fmt.Errorf("%w: user_id=%d", ErrInitiatorInactive, userID)
	}
	// 当前主角色为准；快照角色仅作历史解释，不参与本次授权。
	return string(u.Role), nil
}
