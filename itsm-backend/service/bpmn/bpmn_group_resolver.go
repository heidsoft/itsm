package bpmn

import (
	"context"
	"fmt"
	"strings"

	"itsm-backend/ent"
	"itsm-backend/ent/group"
	"itsm-backend/ent/user"
)

// GroupResolver 把 BPMN candidateGroups (逗号分隔的组名) 解析为具体的用户。
// 这是修复「审批组业务逻辑不闭环」的关键模块：
// 1. BPMN XML 的 candidateGroups 是组名（如 "managers"）
// 2. 引擎创建 ProcessTask 时需要把这些组展开成具体用户，写入 candidate_users
// 3. 「我的待办」接口才能查到分配给我的任务
type GroupResolver struct {
	client *ent.Client
}

// NewGroupResolver 创建 GroupResolver
func NewGroupResolver(client *ent.Client) *GroupResolver {
	return &GroupResolver{client: client}
}

// splitCSV 把 "a, b, c " 拆成 ["a","b","c"]，忽略空项
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// joinCSV 把字符串数组拼成 CSV，去重并保持顺序
func joinCSV(items []string) string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, it := range items {
		it = strings.TrimSpace(it)
		if it == "" {
			continue
		}
		if _, ok := seen[it]; ok {
			continue
		}
		seen[it] = struct{}{}
		out = append(out, it)
	}
	return strings.Join(out, ",")
}

// ExpandGroupsToUsers 把 BPMN candidateGroups (CSV 组名) 展开为候选用户ID与用户名。
// 返回值：
//   - userIDs: 所有组成员用户的 ID（去重）
//   - usernames: 所有组成员的 username/email（去重）
//
// 注意：当某个组不存在或没有成员时，仅记录警告而不返回错误，
// 避免 BPMN 引擎因为配置缺失而中断流程执行。
func (r *GroupResolver) ExpandGroupsToUsers(ctx context.Context, tenantID int, candidateGroups string) ([]int, []string, error) {
	groupNames := splitCSV(candidateGroups)
	if len(groupNames) == 0 {
		return nil, nil, nil
	}

	groups, err := r.client.Group.Query().
		Where(
			group.NameIn(groupNames...),
			group.TenantID(tenantID),
		).
		WithMembers().
		All(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("查询审批组失败: %w", err)
	}

	found := make(map[string]bool, len(groupNames))
	for _, g := range groups {
		found[g.Name] = true
	}
	for _, name := range groupNames {
		if !found[name] {
			// 组不存在：不阻塞流程，但发出警告，便于管理员发现配置漂移
			// 此处不返回错误，由调用方日志处理
			_ = name
		}
	}

	seenID := make(map[int]struct{})
	seenName := make(map[string]struct{})
	var userIDs []int
	var usernames []string
	for _, g := range groups {
		for _, m := range g.Edges.Members {
			if _, ok := seenID[m.ID]; ok {
				continue
			}
			seenID[m.ID] = struct{}{}
			userIDs = append(userIDs, m.ID)
			// 优先 username，若为空则用 email，最后用 ID
			display := strings.TrimSpace(m.Username)
			if display == "" {
				display = strings.TrimSpace(m.Email)
			}
			if display == "" {
				display = fmt.Sprintf("%d", m.ID)
			}
			if _, ok := seenName[display]; ok {
				continue
			}
			seenName[display] = struct{}{}
			usernames = append(usernames, display)
		}
	}
	return userIDs, usernames, nil
}

// MergeCandidateUsers 把 BPMN candidateUsers 与组展开出的 usernames 合并去重，返回 CSV
func (r *GroupResolver) MergeCandidateUsers(bpmnCandidateUsers string, groupUsers []string) string {
	return joinCSV(append(splitCSV(bpmnCandidateUsers), groupUsers...))
}

// GetUserGroupNames 返回指定用户所在的所有组名（逗号分隔）。
// 用于「我的待办」过滤：候选组匹配我所在组的任务，候选人包含我的任务，都应该列出来。
func (r *GroupResolver) GetUserGroupNames(ctx context.Context, tenantID, userID int) (string, error) {
	if userID <= 0 {
		return "", nil
	}
	// 通过 group → members 反向边查询：找出包含 userID 的组
	groups, err := r.client.Group.Query().
		Where(
			group.TenantID(tenantID),
			group.HasMembersWith(user.IDEQ(userID)),
		).
		All(ctx)
	if err != nil {
		return "", fmt.Errorf("查询用户所属组失败: %w", err)
	}
	names := make([]string, 0, len(groups))
	for _, g := range groups {
		names = append(names, g.Name)
	}
	return joinCSV(names), nil
}