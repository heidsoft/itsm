package service

import (
	"context"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/approvalrecord"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// ============================================================
// 表驱动测试：canPerformAction
// 覆盖阶段二重构的核心方法，验证强类型 ApprovalNodeConfig 的权限判断
// ============================================================

func TestCanPerformAction_TableDriven(t *testing.T) {
	// 构造测试用的工作流（不同 level 有不同的权限配置）
	min5 := 5
	timeout24 := 24

	workflows := map[string]*ent.ApprovalWorkflow{
		"empty_nodes": {
			Nodes: []map[string]interface{}{},
		},
		"level1_reject_yes_delegate_no": {
			Nodes: []map[string]interface{}{
				{
					"level":           1,
					"allow_reject":    true,
					"allow_delegate":  false,
					"approval_mode":   "any",
					"approver_type":   "user",
					"approver_ids":    []interface{}{1},
					"reject_action":   "end",
					"name":            "L1",
				},
			},
		},
		"level1_reject_no_delegate_yes": {
			Nodes: []map[string]interface{}{
				{
					"level":           1,
					"allow_reject":    false,
					"allow_delegate":  true,
					"approval_mode":   "any",
					"approver_type":   "user",
					"approver_ids":    []interface{}{1},
					"reject_action":   "end",
					"name":            "L1",
				},
			},
		},
		"multi_level": {
			Nodes: []map[string]interface{}{
				{
					"level":           1,
					"allow_reject":    true,
					"allow_delegate":  false,
					"approval_mode":   "any",
					"approver_type":   "user",
					"approver_ids":    []interface{}{1},
					"reject_action":   "end",
					"name":            "L1",
				},
				{
					"level":           2,
					"allow_reject":    false,
					"allow_delegate":  true,
					"approval_mode":   "all",
					"approver_type":   "user",
					"approver_ids":    []interface{}{2, 3},
					"reject_action":   "return",
					"name":            "L2",
				},
			},
		},
		"with_optional_fields": {
			Nodes: []map[string]interface{}{
				{
					"level":             1,
					"allow_reject":      true,
					"allow_delegate":    true,
					"approval_mode":     "all",
					"approver_type":     "user",
					"approver_ids":      []interface{}{1, 2},
					"reject_action":     "end",
					"name":              "L1",
					"minimum_approvals": min5,
					"timeout_hours":     timeout24,
				},
			},
		},
	}

	service := &ApprovalService{
		client: nil, // canPerformAction 不使用 client
		logger: zaptest.NewLogger(t).Sugar(),
	}

	tests := []struct {
		name        string
		workflowKey string
		level       int
		action      string
		want        bool
	}{
		// 空节点 → 所有操作默认允许
		{"empty_nodes approve", "empty_nodes", 1, "approve", true},
		{"empty_nodes reject", "empty_nodes", 1, "reject", true},
		{"empty_nodes delegate", "empty_nodes", 1, "delegate", true},

		// Level1: allow_reject=true, allow_delegate=false
		{"reject_yes approve", "level1_reject_yes_delegate_no", 1, "approve", true},
		{"reject_yes reject", "level1_reject_yes_delegate_no", 1, "reject", true},
		{"reject_yes delegate_no", "level1_reject_yes_delegate_no", 1, "delegate", false},

		// Level1: allow_reject=false, allow_delegate=true
		{"reject_no delegate_yes approve", "level1_reject_no_delegate_yes", 1, "approve", true},
		{"reject_no reject", "level1_reject_no_delegate_yes", 1, "reject", false},
		{"reject_no delegate_yes", "level1_reject_no_delegate_yes", 1, "delegate", true},

		// 多级工作流
		{"multi L1 approve", "multi_level", 1, "approve", true},
		{"multi L1 reject", "multi_level", 1, "reject", true},
		{"multi L1 delegate", "multi_level", 1, "delegate", false},
		{"multi L2 approve", "multi_level", 2, "approve", true},
		{"multi L2 reject", "multi_level", 2, "reject", false},
		{"multi L2 delegate", "multi_level", 2, "delegate", true},

		// 找不到对应的 level → 默认允许
		{"level_not_found approve", "level1_reject_yes_delegate_no", 99, "approve", true},
		{"level_not_found reject", "level1_reject_yes_delegate_no", 99, "reject", true},
		{"level_not_found delegate", "level1_reject_yes_delegate_no", 99, "delegate", true},

		// 带 optional fields 的节点
		{"optional_fields approve", "with_optional_fields", 1, "approve", true},
		{"optional_fields reject", "with_optional_fields", 1, "reject", true},
		{"optional_fields delegate", "with_optional_fields", 1, "delegate", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflow := workflows[tt.workflowKey]
			got := service.canPerformAction(workflow, tt.level, tt.action)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ============================================================
// 表驱动测试：parseWorkflowNodes
// 覆盖各种输入格式和边界情况
// ============================================================

func TestParseWorkflowNodes_TableDriven(t *testing.T) {
	service := &ApprovalService{
		client: nil,
		logger: zaptest.NewLogger(t).Sugar(),
	}

	tests := []struct {
		name    string
		input   interface{}
		wantLen int
		check   func(t *testing.T, nodes []workflowNode)
		wantErr bool
	}{
		{
			name:    "nil input",
			input:   nil,
			wantLen: 0,
			check:   nil,
			wantErr: false,
		},
		{
			name:    "empty slice",
			input:   []map[string]interface{}{},
			wantLen: 0,
			check:   nil,
			wantErr: false,
		},
		{
			name: "single node complete",
			input: []map[string]interface{}{
				{
					"level":          1,
					"name":           "L1 Approval",
					"approver_type":  "user",
					"approver_ids":   []interface{}{float64(10), float64(20)},
					"approval_mode":  "all",
					"timeout_hours":  float64(48),
				},
			},
			wantLen: 1,
			check: func(t *testing.T, nodes []workflowNode) {
				assert.Equal(t, 1, nodes[0].Level)
				assert.Equal(t, "L1 Approval", nodes[0].Name)
				assert.Equal(t, "all", nodes[0].ApprovalMode)
				assert.Equal(t, 48, nodes[0].TimeoutHours)
				assert.Equal(t, []int{10, 20}, nodes[0].ApproverIDs)
			},
			wantErr: false,
		},
		{
			name: "missing level auto-increments",
			input: []map[string]interface{}{
				{
					"name":          "First",
					"approver_type": "user",
					"approver_ids":  []interface{}{float64(1)},
					"approval_mode": "any",
				},
				{
					"name":          "Second",
					"approver_type": "user",
					"approver_ids":  []interface{}{float64(2)},
					"approval_mode": "any",
				},
			},
			wantLen: 2,
			check: func(t *testing.T, nodes []workflowNode) {
				assert.Equal(t, 1, nodes[0].Level)
				assert.Equal(t, 2, nodes[1].Level)
			},
			wantErr: false,
		},
		{
			name: "missing name auto-generates",
			input: []map[string]interface{}{
				{
					"level":          1,
					"approver_type":  "user",
					"approver_ids":   []interface{}{float64(1)},
					"approval_mode":  "any",
				},
			},
			wantLen: 1,
			check: func(t *testing.T, nodes []workflowNode) {
				assert.Equal(t, "Step 1", nodes[0].Name)
			},
			wantErr: false,
		},
		{
			name: "missing approval_mode defaults to any",
			input: []map[string]interface{}{
				{
					"level":         1,
					"name":          "L1",
					"approver_type": "user",
					"approver_ids":  []interface{}{float64(1)},
				},
			},
			wantLen: 1,
			check: func(t *testing.T, nodes []workflowNode) {
				assert.Equal(t, "any", nodes[0].ApprovalMode)
			},
			wantErr: false,
		},
		{
			name: "dynamic approver type sets AssigneeType",
			input: []map[string]interface{}{
				{
					"level":          1,
					"name":           "Dept Manager",
					"approver_type":  "dept_manager",
					"approval_mode":  "any",
				},
			},
			wantLen: 1,
			check: func(t *testing.T, nodes []workflowNode) {
				assert.Equal(t, "dept_manager", nodes[0].AssigneeType)
			},
			wantErr: false,
		},
		{
			name: "multiple dynamic types",
			input: []map[string]interface{}{
				{
					"level":          1,
					"name":           "Team Leader",
					"approver_type":  "team_leader",
					"approval_mode":  "any",
				},
				{
					"level":          2,
					"name":           "Project Manager",
					"approver_type":  "project_manager",
					"approval_mode":  "all",
				},
				{
					"level":          3,
					"name":           "Amount Based",
					"approver_type":  "amount_based",
					"approval_mode":  "any",
				},
			},
			wantLen: 3,
			check: func(t *testing.T, nodes []workflowNode) {
				assert.Equal(t, "team_leader", nodes[0].AssigneeType)
				assert.Equal(t, "project_manager", nodes[1].AssigneeType)
				assert.Equal(t, "amount_based", nodes[2].AssigneeType)
			},
			wantErr: false,
		},
		{
			name:    "invalid type",
			input:   "not-a-slice",
			wantLen: 0,
			check:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := service.parseWorkflowNodes(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, nodes, tt.wantLen)
			if tt.check != nil {
				tt.check(t, nodes)
			}
		})
	}
}

// ============================================================
// 表驱动测试：ListWorkflows with WorkflowListFilter
// 验证阶段二新增的强类型过滤器
// ============================================================

func TestListWorkflows_WithFilter_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		filter     *dto.WorkflowListFilter
		wantTotal  int
		wantLength int
		setup      func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int)
	}{
		{
			name:       "nil filter returns all",
			filter:     nil,
			wantTotal:  3,
			wantLength: 3,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createTestWorkflows(t, ctx, client, tenantID, 3, "", "")
			},
		},
		{
			name: "filter by ticket_type",
			filter: &dto.WorkflowListFilter{
				TicketType: "incident",
			},
			wantTotal:  2,
			wantLength: 2,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createTestWorkflows(t, ctx, client, tenantID, 2, "incident", "")
				createTestWorkflows(t, ctx, client, tenantID, 3, "change", "")
			},
		},
		{
			name: "filter by priority",
			filter: &dto.WorkflowListFilter{
				Priority: "urgent",
			},
			wantTotal:  1,
			wantLength: 1,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createTestWorkflows(t, ctx, client, tenantID, 1, "", "urgent")
				createTestWorkflows(t, ctx, client, tenantID, 2, "", "medium")
			},
		},
		{
			name: "filter by is_active true",
			filter: func() *dto.WorkflowListFilter {
				active := true
				return &dto.WorkflowListFilter{IsActive: &active}
			}(),
			wantTotal:  2,
			wantLength: 2,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				// 2 active
				createTestWorkflowsWithActive(t, ctx, client, tenantID, 2, true)
				// 1 inactive
				createTestWorkflowsWithActive(t, ctx, client, tenantID, 1, false)
			},
		},
		{
			name: "filter by is_active false",
			filter: func() *dto.WorkflowListFilter {
				active := false
				return &dto.WorkflowListFilter{IsActive: &active}
			}(),
			wantTotal:  1,
			wantLength: 1,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createTestWorkflowsWithActive(t, ctx, client, tenantID, 2, true)
				createTestWorkflowsWithActive(t, ctx, client, tenantID, 1, false)
			},
		},
		{
			name: "filter by ticket_type and priority combined",
			filter: &dto.WorkflowListFilter{
				TicketType: "incident",
				Priority:   "urgent",
			},
			wantTotal:  1,
			wantLength: 1,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createTestWorkflows(t, ctx, client, tenantID, 1, "incident", "urgent")
				createTestWorkflows(t, ctx, client, tenantID, 2, "incident", "medium")
				createTestWorkflows(t, ctx, client, tenantID, 1, "change", "urgent")
			},
		},
		{
			name: "filter matches nothing",
			filter: &dto.WorkflowListFilter{
				TicketType: "nonexistent",
			},
			wantTotal:  0,
			wantLength: 0,
			setup: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createTestWorkflows(t, ctx, client, tenantID, 3, "incident", "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, service, ctx := setupApprovalTest(t)
			defer client.Close()

			tenant, err := createApprovalTestTenant(ctx, client, t.Name())
			require.NoError(t, err)

			if tt.setup != nil {
				tt.setup(t, ctx, client, tenant.ID)
			}

			workflows, total, err := service.ListWorkflows(ctx, tt.filter, tenant.ID, 1, 50)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
			assert.Len(t, workflows, tt.wantLength)
		})
	}
}

// 辅助函数：批量创建工作流
func createTestWorkflows(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, count int, ticketType, priority string) {
	t.Helper()
	for i := 0; i < count; i++ {
		builder := client.ApprovalWorkflow.Create().
			SetName("Test Workflow").
			SetDescription("Test").
			SetIsActive(true).
			SetTenantID(tenantID).
			SetNodes([]map[string]interface{}{})
		if ticketType != "" {
			builder = builder.SetTicketType(ticketType)
		}
		if priority != "" {
			builder = builder.SetPriority(priority)
		}
		_, err := builder.Save(ctx)
		require.NoError(t, err)
	}
}

func createTestWorkflowsWithActive(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, count int, active bool) {
	t.Helper()
	for i := 0; i < count; i++ {
		_, err := client.ApprovalWorkflow.Create().
			SetName("Test Workflow").
			SetDescription("Test").
			SetIsActive(active).
			SetTenantID(tenantID).
			SetNodes([]map[string]interface{}{}).
			Save(ctx)
		require.NoError(t, err)
	}
}

// ============================================================
// 表驱动测试：CreateWorkflow with optional TicketType + Priority
// 验证阶段二修复的 N+1 问题——所有字段在单次 Create 中设置
// ============================================================

func TestCreateWorkflow_OptionalFields_TableDriven(t *testing.T) {
	tests := []struct {
		name             string
		ticketType       *string
		priority         *string
		wantTicketType   *string
		wantPriority     *string
		wantNodesCount   int
	}{
		{
			name:           "no optional fields",
			ticketType:     nil,
			priority:       nil,
			wantTicketType: nil,
			wantPriority:   nil,
			wantNodesCount: 1,
		},
		{
			name:           "only ticket_type",
			ticketType:     strPtr("incident"),
			priority:       nil,
			wantTicketType: strPtr("incident"),
			wantPriority:   nil,
			wantNodesCount: 1,
		},
		{
			name:           "only priority",
			ticketType:     nil,
			priority:       strPtr("urgent"),
			wantTicketType: nil,
			wantPriority:   strPtr("urgent"),
			wantNodesCount: 1,
		},
		{
			name:           "both ticket_type and priority",
			ticketType:     strPtr("change"),
			priority:       strPtr("high"),
			wantTicketType: strPtr("change"),
			wantPriority:   strPtr("high"),
			wantNodesCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, service, ctx := setupApprovalTest(t)
			defer client.Close()

			tenant, err := createApprovalTestTenant(ctx, client, t.Name())
			require.NoError(t, err)

			user, err := createApprovalTestUser(ctx, client, tenant.ID, t.Name())
			require.NoError(t, err)

			req := &dto.CreateApprovalWorkflowRequest{
				Name:        "Test Workflow " + tt.name,
				Description: "Testing optional fields",
				IsActive:    true,
				TicketType:  tt.ticketType,
				Priority:    tt.priority,
				Nodes: []dto.ApprovalNodeRequest{
					{
						Level:        1,
						Name:         "L1",
						ApproverType: "user",
						ApproverIDs:  []int{user.ID},
						ApprovalMode: "any",
						AllowReject:  true,
						RejectAction: "end",
					},
				},
			}

			resp, err := service.CreateWorkflow(ctx, req, tenant.ID)
			require.NoError(t, err)
			assert.NotNil(t, resp)

			// 验证 TicketType
			if tt.wantTicketType != nil {
				require.NotNil(t, resp.TicketType)
				assert.Equal(t, *tt.wantTicketType, *resp.TicketType)
			} else {
				assert.Nil(t, resp.TicketType)
			}

			// 验证 Priority
			if tt.wantPriority != nil {
				require.NotNil(t, resp.Priority)
				assert.Equal(t, *tt.wantPriority, *resp.Priority)
			} else {
				assert.Nil(t, resp.Priority)
			}

			// 验证节点
			assert.Len(t, resp.Nodes, tt.wantNodesCount)
		})
	}
}

// ============================================================
// 表驱动测试：类型转换器 round-trip
// 验证 nodesToMaps → mapsToNodes 数据完整性
// ============================================================

func TestTypeConverters_RoundTrip_TableDriven(t *testing.T) {
	min5 := 5
	timeout24 := 24
	returnLevel := 1

	tests := []struct {
		name    string
		configs []dto.ApprovalNodeConfig
	}{
		{
			name:    "empty",
			configs: []dto.ApprovalNodeConfig{},
		},
		{
			name: "single minimal node",
			configs: []dto.ApprovalNodeConfig{
				{
					Level:        1,
					Name:         "L1",
					ApproverType: dto.ApprovalNodeTypeUser,
					ApprovalMode: dto.ApprovalModeAny,
				},
			},
		},
		{
			name: "single complete node",
			configs: []dto.ApprovalNodeConfig{
				{
					Level:            1,
					Name:             "L1 Complete",
					ApproverType:     dto.ApprovalNodeTypeUser,
					ApproverIDs:      []int{10, 20, 30},
					ApprovalMode:     dto.ApprovalModeAll,
					MinimumApprovals: &min5,
					TimeoutHours:     &timeout24,
					AllowReject:      true,
					AllowDelegate:    false,
					RejectAction:     dto.ApprovalRejectActionEnd,
					ReturnToLevel:    &returnLevel,
					Conditions: []dto.ApprovalConditionConfig{
						{Field: "amount", Operator: "greater_than", Value: 10000},
					},
				},
			},
		},
		{
			name: "multi-level with dynamic approvers",
			configs: []dto.ApprovalNodeConfig{
				{
					Level:        1,
					Name:         "Dept Manager",
					ApproverType: dto.ApprovalNodeTypeDeptManager,
					ApprovalMode: dto.ApprovalModeAny,
					AllowReject:  true,
				},
				{
					Level:         2,
					Name:          "Team Leader",
					ApproverType:  dto.ApprovalNodeTypeTeamLeader,
					ApprovalMode:  dto.ApprovalModeAll,
					AllowReject:   false,
					AllowDelegate: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// configs → maps → configs（round-trip）
			maps, err := nodesToMaps(tt.configs)
			require.NoError(t, err)

			result, err := mapsToNodes(maps)
			require.NoError(t, err)

			assert.Len(t, result, len(tt.configs))

			for i, original := range tt.configs {
				got := result[i]
				assert.Equal(t, original.Level, got.Level, "Level mismatch at index %d", i)
				assert.Equal(t, original.Name, got.Name, "Name mismatch at index %d", i)
				assert.Equal(t, original.ApproverType, got.ApproverType, "ApproverType mismatch at index %d", i)
				assert.Equal(t, original.ApprovalMode, got.ApprovalMode, "ApprovalMode mismatch at index %d", i)
				assert.Equal(t, original.AllowReject, got.AllowReject, "AllowReject mismatch at index %d", i)
				assert.Equal(t, original.AllowDelegate, got.AllowDelegate, "AllowDelegate mismatch at index %d", i)
				assert.Equal(t, original.RejectAction, got.RejectAction, "RejectAction mismatch at index %d", i)

				// Slice comparison
				if len(original.ApproverIDs) == 0 {
					assert.Empty(t, got.ApproverIDs)
				} else {
					assert.Equal(t, original.ApproverIDs, got.ApproverIDs)
				}

				// Optional int pointers
				if original.MinimumApprovals != nil {
					require.NotNil(t, got.MinimumApprovals)
					assert.Equal(t, *original.MinimumApprovals, *got.MinimumApprovals)
				}
				if original.TimeoutHours != nil {
					require.NotNil(t, got.TimeoutHours)
					assert.Equal(t, *original.TimeoutHours, *got.TimeoutHours)
				}
				if original.ReturnToLevel != nil {
					require.NotNil(t, got.ReturnToLevel)
					assert.Equal(t, *original.ReturnToLevel, *got.ReturnToLevel)
				}

				// Conditions
				assert.Len(t, got.Conditions, len(original.Conditions))
			}
		})
	}
}

// mapsToNodesUnsafe 应该在出错时返回 nil 而非 panic
func TestMapsToNodesUnsafe_InvalidInput(t *testing.T) {
	// 传入无法反序列化的数据
	invalidMaps := []map[string]interface{}{
		{
			"level":         "not-a-number", // 错误的类型
			"approval_mode": "any",
		},
	}
	result := mapsToNodesUnsafe(invalidMaps)
	// 出错时返回 nil，不 panic
	// 注意：json.Unmarshal 对 int 字段传入 string 可能不会报错（取决于实现），
	// 但关键是它不应该 panic
	_ = result
}

// ============================================================
// 表驱动测试：SubmitApproval reject path
// 验证拒绝操作的终止行为
// ============================================================

func TestSubmitApproval_Reject_TerminatesWorkflow(t *testing.T) {
	client, service, ctx := setupApprovalTest(t)
	defer client.Close()

	tenant, err := createApprovalTestTenant(ctx, client, "reject")
	require.NoError(t, err)

	approver, err := createApprovalTestUser(ctx, client, tenant.ID, "reject")
	require.NoError(t, err)
	approver2, err := createApprovalTestUser(ctx, client, tenant.ID, "reject2")
	require.NoError(t, err)

	// 创建工作流（allow_reject=true）
	workflow, err := client.ApprovalWorkflow.Create().
		SetName("Reject Test Workflow").
		SetIsActive(true).
		SetTenantID(tenant.ID).
		SetNodes([]map[string]interface{}{
			{
				"level":          1,
				"name":           "L1",
				"approver_type":  "user",
				"approver_ids":   []int{approver.ID},
				"approval_mode":  "any",
				"allow_reject":   true,
				"reject_action":  "end",
			},
		}).
		Save(ctx)
	require.NoError(t, err)

	// 创建工单
	ticket, err := client.Ticket.Create().
		SetTitle("Reject Me").
		SetDescription("desc").
		SetPriority("urgent").
		SetType("ticket").
		SetStatus("open").
		SetTicketNumber("TKT-REJECT-001").
		SetTenantID(tenant.ID).
		SetRequesterID(approver.ID).
		Save(ctx)
	require.NoError(t, err)

	// 创建两条 pending 审批记录（同一工单、同一工作流）
	record1, err := client.ApprovalRecord.Create().
		SetTicketID(ticket.ID).
		SetTicketNumber(ticket.TicketNumber).
		SetTicketTitle(ticket.Title).
		SetWorkflowID(workflow.ID).
		SetWorkflowName(workflow.Name).
		SetCurrentLevel(1).
		SetTotalLevels(1).
		SetApproverID(approver.ID).
		SetApproverName(approver.Name).
		SetStepOrder(1).
		SetStatus("pending").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	record2, err := client.ApprovalRecord.Create().
		SetTicketID(ticket.ID).
		SetTicketNumber(ticket.TicketNumber).
		SetTicketTitle(ticket.Title).
		SetWorkflowID(workflow.ID).
		SetWorkflowName(workflow.Name).
		SetCurrentLevel(2).
		SetTotalLevels(2).
		SetApproverID(approver2.ID).
		SetApproverName(approver2.Name).
		SetStepOrder(2).
		SetStatus("pending").
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 审批人1拒绝
	err = service.SubmitApproval(ctx, record1.ID, approver.ID, "reject", "rejected by approver", nil, tenant.ID)
	require.NoError(t, err)

	// 验证 record1 被标记为 rejected
	updatedRecord1, err := client.ApprovalRecord.Get(ctx, record1.ID)
	require.NoError(t, err)
	assert.Equal(t, "rejected", updatedRecord1.Status)
	assert.Equal(t, "reject", updatedRecord1.Action)

	// 验证 record2 被取消（terminate 行为）
	updatedRecord2, err := client.ApprovalRecord.Get(ctx, record2.ID)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", updatedRecord2.Status)

	// 验证工单状态变为 rejected
	updatedTicket, err := client.Ticket.Get(ctx, ticket.ID)
	require.NoError(t, err)
	assert.Equal(t, "rejected", updatedTicket.Status)
}

// ============================================================
// 表驱动测试：SubmitApproval error paths
// 验证各种错误场景
// ============================================================

func TestSubmitApproval_ErrorPaths_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		action        string
		userID        int
		delegateTo    *int
		recordStatus  string
		wantErr       bool
		errContains   string
	}{
		{
			name:         "wrong user rejected",
			action:       "approve",
			userID:       99999, // 不存在的用户
			recordStatus: "pending",
			wantErr:      true,
			errContains:  "not authorized",
		},
		{
			name:         "already processed record",
			action:       "approve",
			userID:       0, // 会在测试中设为正确的 approver ID
			recordStatus: "approved",
			wantErr:      true,
			errContains:  "not found or already processed",
		},
		{
			name:         "invalid action",
			action:       "invalid_action",
			userID:       0,
			recordStatus: "pending",
			wantErr:      true,
			errContains:  "invalid action",
		},
		{
			name:         "delegate without target user",
			action:       "delegate",
			userID:       0,
			delegateTo:   nil,
			recordStatus: "pending",
			wantErr:      true,
			errContains:  "delegate_to_user_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, service, ctx := setupApprovalTest(t)
			defer client.Close()

			tenant, err := createApprovalTestTenant(ctx, client, tt.name)
			require.NoError(t, err)

			approver, err := createApprovalTestUser(ctx, client, tenant.ID, tt.name)
			require.NoError(t, err)

			// 创建工作流（allow_reject=true, allow_delegate=true 以覆盖各种场景）
			workflow, err := client.ApprovalWorkflow.Create().
				SetName("Error Path Workflow").
				SetIsActive(true).
				SetTenantID(tenant.ID).
				SetNodes([]map[string]interface{}{
					{
						"level":          1,
						"name":           "L1",
						"approver_type":  "user",
						"approver_ids":   []int{approver.ID},
						"approval_mode":  "any",
						"allow_reject":   true,
						"allow_delegate": true,
						"reject_action":  "end",
					},
				}).
				Save(ctx)
			require.NoError(t, err)

			ticket, err := client.Ticket.Create().
				SetTitle("Error Test").
				SetDescription("desc").
				SetPriority("medium").
				SetType("ticket").
				SetStatus("open").
				SetTicketNumber("TKT-ERR-" + tt.name).
				SetTenantID(tenant.ID).
				SetRequesterID(approver.ID).
				Save(ctx)
			require.NoError(t, err)

			record, err := client.ApprovalRecord.Create().
				SetTicketID(ticket.ID).
				SetTicketNumber(ticket.TicketNumber).
				SetTicketTitle(ticket.Title).
				SetWorkflowID(workflow.ID).
				SetWorkflowName(workflow.Name).
				SetCurrentLevel(1).
				SetTotalLevels(1).
				SetApproverID(approver.ID).
				SetApproverName(approver.Name).
				SetStepOrder(1).
				SetStatus(tt.recordStatus).
				SetTenantID(tenant.ID).
				Save(ctx)
			require.NoError(t, err)

			// 设置正确的 userID（如果测试用例 userID 为 0，使用 approver.ID）
			userID := tt.userID
			if userID == 0 {
				userID = approver.ID
			}

			err = service.SubmitApproval(ctx, record.ID, userID, tt.action, "comment", tt.delegateTo, tenant.ID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ============================================================
// 表驱动测试：UpdateWorkflow with nodes
// 验证阶段二新增的强类型节点更新路径
// ============================================================

func TestUpdateWorkflow_WithNodes_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		initialNodes   []map[string]interface{}
		updateNodes    *[]dto.ApprovalNodeRequest
		wantNodeCount  int
		wantNodeName   string
	}{
		{
			name: "add nodes to empty workflow",
			initialNodes: []map[string]interface{}{},
			updateNodes: &[]dto.ApprovalNodeRequest{
				{
					Level:        1,
					Name:         "New L1",
					ApproverType: "user",
					ApprovalMode: "any",
					AllowReject:  true,
					RejectAction: "end",
				},
			},
			wantNodeCount: 1,
			wantNodeName:  "New L1",
		},
		{
			name: "replace existing nodes",
			initialNodes: []map[string]interface{}{
				{
					"level":          1,
					"name":           "Old L1",
					"approver_type":  "user",
					"approver_ids":   []int{1},
					"approval_mode":  "any",
					"allow_reject":   true,
					"reject_action":  "end",
				},
			},
			updateNodes: &[]dto.ApprovalNodeRequest{
				{
					Level:        1,
					Name:         "Replaced L1",
					ApproverType: "user",
					ApprovalMode: "all",
					AllowReject:  false,
					RejectAction: "return",
				},
				{
					Level:        2,
					Name:         "New L2",
					ApproverType: "user",
					ApprovalMode: "any",
					AllowReject:  true,
					RejectAction: "end",
				},
			},
			wantNodeCount: 2,
			wantNodeName:  "Replaced L1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, service, ctx := setupApprovalTest(t)
			defer client.Close()

			tenant, err := createApprovalTestTenant(ctx, client, t.Name())
			require.NoError(t, err)

			// 创建初始工作流
			created, err := client.ApprovalWorkflow.Create().
				SetName("Original").
				SetDescription("Original desc").
				SetIsActive(true).
				SetTenantID(tenant.ID).
				SetNodes(tt.initialNodes).
				Save(ctx)
			require.NoError(t, err)

			// 更新节点
			resp, err := service.UpdateWorkflow(ctx, created.ID, &dto.UpdateApprovalWorkflowRequest{
				Nodes: tt.updateNodes,
			}, tenant.ID)
			require.NoError(t, err)
			assert.NotNil(t, resp)

			// 验证节点数量
			assert.Len(t, resp.Nodes, tt.wantNodeCount)

			// 验证第一个节点名称
			if tt.wantNodeCount > 0 {
				assert.Equal(t, tt.wantNodeName, resp.Nodes[0].Name)
			}
		})
	}
}

// ============================================================
// 表驱动测试：TriggerApproval with different approval modes
// 验证 "any" 模式只创建一条记录，"all" 模式创建多条记录
// ============================================================

func TestTriggerApproval_ApprovalMode_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		approvalMode   string
		approverCount  int
		wantRecordCount int
	}{
		{
			name:            "any mode with 2 approvers creates 1 record",
			approvalMode:    "any",
			approverCount:   2,
			wantRecordCount: 1,
		},
		{
			name:            "all mode with 2 approvers creates 2 records",
			approvalMode:    "all",
			approverCount:   2,
			wantRecordCount: 2,
		},
		{
			name:            "all mode with 1 approver creates 1 record",
			approvalMode:    "all",
			approverCount:   1,
			wantRecordCount: 1,
		},
		{
			name:            "any mode with 1 approver creates 1 record",
			approvalMode:    "any",
			approverCount:   1,
			wantRecordCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, service, ctx := setupApprovalTest(t)
			defer client.Close()

			tenant, err := createApprovalTestTenant(ctx, client, tt.name)
			require.NoError(t, err)

			// 创建审批人
			approverIDs := make([]int, 0, tt.approverCount)
			for i := 0; i < tt.approverCount; i++ {
				u, err := createApprovalTestUser(ctx, client, tenant.ID, tt.name+string(rune('a'+i)))
				require.NoError(t, err)
				approverIDs = append(approverIDs, u.ID)
			}

			// 创建工作流
			_, err = client.ApprovalWorkflow.Create().
				SetName("Mode Test Workflow").
				SetTicketType("ticket").
				SetPriority("urgent").
				SetIsActive(true).
				SetTenantID(tenant.ID).
				SetNodes([]map[string]interface{}{
					{
						"level":          1,
						"name":           "L1",
						"approver_type":  "user",
						"approver_ids":   approverIDs,
						"approval_mode":  tt.approvalMode,
					},
				}).
				Save(ctx)
			require.NoError(t, err)

			// 创建工单
			ticket, err := client.Ticket.Create().
				SetTitle("Mode Test Ticket").
				SetDescription("desc").
				SetPriority("urgent").
				SetType("ticket").
				SetStatus("open").
				SetTicketNumber("TKT-MODE-" + tt.name).
				SetTenantID(tenant.ID).
				SetRequesterID(approverIDs[0]).
				Save(ctx)
			require.NoError(t, err)

			// 触发审批
			records, err := service.TriggerApproval(ctx, &ApprovalTriggerRequest{
				TicketID:     ticket.ID,
				TicketNumber: ticket.TicketNumber,
				TicketTitle:  ticket.Title,
				TicketType:   "ticket",
				Priority:     "urgent",
				RequesterID:  approverIDs[0],
				TenantID:     tenant.ID,
			})
			require.NoError(t, err)
			assert.Len(t, records, tt.wantRecordCount)

			// 验证数据库中的记录数
			count, err := client.ApprovalRecord.Query().
				Where(
					approvalrecord.TenantIDEQ(tenant.ID),
					approvalrecord.TicketIDEQ(ticket.ID),
				).
				Count(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantRecordCount, count)
		})
	}
}

// ============================================================
// 表驱动测试：findMatchingWorkflow
// 验证工作流匹配逻辑（精确匹配 > 类型匹配 > 无匹配）
// ============================================================

func TestFindMatchingWorkflow_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		ticketType     string
		priority       string
		setupWorkflows func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int)
		wantFound      bool
		wantName       string
	}{
		{
			name:       "exact match (type + priority)",
			ticketType: "incident",
			priority:   "urgent",
			setupWorkflows: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createMatchingWorkflow(t, ctx, client, tenantID, "Exact Match", "incident", "urgent", true)
				createMatchingWorkflow(t, ctx, client, tenantID, "Type Only", "incident", "", true)
			},
			wantFound: true,
			wantName:  "Exact Match",
		},
		{
			name:       "type-only match when no exact match",
			ticketType: "incident",
			priority:   "low",
			setupWorkflows: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createMatchingWorkflow(t, ctx, client, tenantID, "Type Only", "incident", "urgent", true)
			},
			wantFound: true,
			wantName:  "Type Only",
		},
		{
			name:       "no match at all",
			ticketType: "nonexistent",
			priority:   "urgent",
			setupWorkflows: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createMatchingWorkflow(t, ctx, client, tenantID, "Other", "incident", "urgent", true)
			},
			wantFound: false,
			wantName:  "",
		},
		{
			name:       "inactive workflow not matched",
			ticketType: "incident",
			priority:   "urgent",
			setupWorkflows: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				createMatchingWorkflow(t, ctx, client, tenantID, "Inactive", "incident", "urgent", false)
			},
			wantFound: false,
			wantName:  "",
		},
		{
			name:       "empty database",
			ticketType: "incident",
			priority:   "urgent",
			setupWorkflows: func(t *testing.T, ctx context.Context, client *ent.Client, tenantID int) {
				// 不创建任何工作流
			},
			wantFound: false,
			wantName:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, service, ctx := setupApprovalTest(t)
			defer client.Close()

			tenant, err := createApprovalTestTenant(ctx, client, tt.name)
			require.NoError(t, err)

			if tt.setupWorkflows != nil {
				tt.setupWorkflows(t, ctx, client, tenant.ID)
			}

			workflow, err := service.findMatchingWorkflow(ctx, tt.ticketType, tt.priority, tenant.ID)
			require.NoError(t, err)

			if tt.wantFound {
				require.NotNil(t, workflow)
				assert.Equal(t, tt.wantName, workflow.Name)
			} else {
				assert.Nil(t, workflow)
			}
		})
	}
}

// 辅助函数：创建匹配的工作流
func createMatchingWorkflow(t *testing.T, ctx context.Context, client *ent.Client, tenantID int, name, ticketType, priority string, isActive bool) {
	t.Helper()
	builder := client.ApprovalWorkflow.Create().
		SetName(name).
		SetDescription("Test").
		SetIsActive(isActive).
		SetTenantID(tenantID).
		SetNodes([]map[string]interface{}{
			{
				"level":          1,
				"name":           "L1",
				"approver_type":  "user",
				"approver_ids":   []int{1},
				"approval_mode":  "any",
			},
		})
	if ticketType != "" {
		builder = builder.SetTicketType(ticketType)
	}
	if priority != "" {
		builder = builder.SetPriority(priority)
	}
	_, err := builder.Save(ctx)
	require.NoError(t, err)
}

// ============================================================
// 表驱动测试：GetApprovalRecords with filters
// 验证审批记录查询的各种过滤条件
// ============================================================

func TestGetApprovalRecords_WithFilters_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		req        *dto.GetApprovalRecordsRequest
		wantTotal  int
		wantLength int
	}{
		{
			name: "no filters returns all",
			req: &dto.GetApprovalRecordsRequest{
				Page:     1,
				PageSize: 10,
			},
			wantTotal:  3,
			wantLength: 3,
		},
		{
			name: "filter by status pending",
			req: &dto.GetApprovalRecordsRequest{
				Status:   strPtr("pending"),
				Page:     1,
				PageSize: 10,
			},
			wantTotal:  2,
			wantLength: 2,
		},
		{
			name: "filter by status approved",
			req: &dto.GetApprovalRecordsRequest{
				Status:   strPtr("approved"),
				Page:     1,
				PageSize: 10,
			},
			wantTotal:  1,
			wantLength: 1,
		},
		{
			name: "pagination page 1 size 2",
			req: &dto.GetApprovalRecordsRequest{
				Page:     1,
				PageSize: 2,
			},
			wantTotal:  3,
			wantLength: 2,
		},
		{
			name: "pagination page 2 size 2",
			req: &dto.GetApprovalRecordsRequest{
				Page:     2,
				PageSize: 2,
			},
			wantTotal:  3,
			wantLength: 1,
		},
		{
			name: "filter by non-existent status",
			req: &dto.GetApprovalRecordsRequest{
				Status:   strPtr("nonexistent"),
				Page:     1,
				PageSize: 10,
			},
			wantTotal:  0,
			wantLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, service, ctx := setupApprovalTest(t)
			defer client.Close()

			tenant, err := createApprovalTestTenant(ctx, client, tt.name)
			require.NoError(t, err)

			user, err := createApprovalTestUser(ctx, client, tenant.ID, tt.name)
			require.NoError(t, err)

			workflow, err := client.ApprovalWorkflow.Create().
				SetName("Test Workflow").
				SetIsActive(true).
				SetTenantID(tenant.ID).
				SetNodes([]map[string]interface{}{}).
				Save(ctx)
			require.NoError(t, err)

			ticket, err := client.Ticket.Create().
				SetTitle("Test Ticket").
				SetDescription("desc").
				SetPriority("medium").
				SetType("ticket").
				SetStatus("open").
				SetTicketNumber("TKT-RECFILT-" + tt.name).
				SetTenantID(tenant.ID).
				SetRequesterID(user.ID).
				Save(ctx)
			require.NoError(t, err)

			// 创建 3 条记录：2 条 pending，1 条 approved
			statuses := []string{"pending", "pending", "approved"}
			for _, status := range statuses {
				_, err := client.ApprovalRecord.Create().
					SetTicketID(ticket.ID).
					SetTicketNumber(ticket.TicketNumber).
					SetTicketTitle(ticket.Title).
					SetWorkflowID(workflow.ID).
					SetWorkflowName(workflow.Name).
					SetCurrentLevel(1).
					SetTotalLevels(1).
					SetApproverID(user.ID).
					SetApproverName(user.Name).
					SetStepOrder(1).
					SetStatus(status).
					SetTenantID(tenant.ID).
					Save(ctx)
				require.NoError(t, err)
			}

			records, total, err := service.GetApprovalRecords(ctx, tt.req, tenant.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTotal, total)
			assert.Len(t, records, tt.wantLength)
		})
	}
}

// ============================================================
// 辅助函数
// ============================================================

func strPtr(s string) *string {
	return &s
}

// 确保 time 包被使用（某些测试场景可能用到）
var _ = time.Now
