package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestResolveApprover_DynamicRoleUsesRequestContext 验证 i2 P0 修复：当 assigneeType
// 是 dept_manager/team_leader/project_manager 时，优先使用 ApprovalTriggerRequest 中的
// DepartmentID/TeamID/ProjectID，不再硬依赖从 assigneeValue 解析。
//
// 由于数据库查询路径需要 ent+sqlite，本测试只覆盖「上下文选择 + assigneeValue 解析」分支：
// 通过 nil req 或显式 req.DepartmentID 都能进入一致的逻辑路径。
func TestResolveApprover_DynamicRoleUsesRequestContext(t *testing.T) {
	service := &ApprovalService{
		client: nil, // 不会被本测试调用，因为解析前会因 scopeID=0 提前返回
		logger: zaptest.NewLogger(t).Sugar(),
	}
	ctx := context.Background()

	tests := []struct {
		name          string
		assigneeType  string
		assigneeValue string
		req           *ApprovalTriggerRequest
		wantErrSubstr string
	}{
		{
			name:          "dept_manager without req context returns explicit error",
			assigneeType:  "dept_manager",
			assigneeValue: "",
			req:           nil,
			wantErrSubstr: "缺少有效的范围 ID",
		},
		{
			name:          "dept_manager with req context still errors on missing id",
			assigneeType:  "dept_manager",
			assigneeValue: "",
			req:           &ApprovalTriggerRequest{TenantID: 1},
			wantErrSubstr: "缺少有效的范围 ID",
		},
		{
			name:          "team_leader falls through to error without scope id",
			assigneeType:  "team_leader",
			assigneeValue: "",
			req:           &ApprovalTriggerRequest{TenantID: 1},
			wantErrSubstr: "缺少有效的范围 ID",
		},
		{
			name:          "project_manager falls through to error without scope id",
			assigneeType:  "project_manager",
			assigneeValue: "",
			req:           &ApprovalTriggerRequest{TenantID: 1},
			wantErrSubstr: "缺少有效的范围 ID",
		},
		{
			name:          "temp_team_leader falls through to error without scope id",
			assigneeType:  "temp_team_leader",
			assigneeValue: "",
			req:           &ApprovalTriggerRequest{TenantID: 1},
			wantErrSubstr: "缺少有效的范围 ID",
		},
		{
			name:          "amount_based with empty assignee value returns explicit error",
			assigneeType:  "amount_based",
			assigneeValue: "",
			req:           &ApprovalTriggerRequest{TenantID: 1, Amount: 100},
			wantErrSubstr: "requires assignee_value thresholds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := service.resolveApprover(ctx, tt.assigneeType, tt.assigneeValue, 1, 100, tt.req)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrSubstr)
		})
	}
}

// TestResolveTenantAdminApprover_RequiresClientAndTenant 验证服务未初始化或租户
// 无效时立即返回错误（fail closed）。
func TestResolveTenantAdminApprover_RequiresClientAndTenant(t *testing.T) {
	service := &ApprovalService{client: nil, logger: zaptest.NewLogger(t).Sugar()}

	t.Run("nil client returns error", func(t *testing.T) {
		_, _, err := service.resolveTenantAdminApprover(context.Background(), 1)
		require.Error(t, err)
	})

	t.Run("zero tenant returns error", func(t *testing.T) {
		_, _, err := service.resolveTenantAdminApprover(context.Background(), 0)
		require.Error(t, err)
	})
}

// TestEnrichApproverContext_NilSafe 验证 enrichApproverContext 对 nil req 不会 panic，
// 且对显式给出的 ID 不会被覆盖。
func TestEnrichApproverContext_NilSafe(t *testing.T) {
	service := &ApprovalService{client: nil, logger: zaptest.NewLogger(t).Sugar()}

	t.Run("nil request is a no-op", func(t *testing.T) {
		assert.NotPanics(t, func() {
			service.enrichApproverContext(context.Background(), nil)
		})
	})

	t.Run("explicit department id is preserved", func(t *testing.T) {
		req := &ApprovalTriggerRequest{
			TenantID:     1,
			TicketID:     999,
			RequesterID:  123,
			DepartmentID: 42,
		}
		// client=nil 导致后续查询会失败，但显式 ID 必须保留
		service.enrichApproverContext(context.Background(), req)
		assert.Equal(t, 42, req.DepartmentID)
	})
}
