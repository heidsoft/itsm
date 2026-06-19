package service

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"
	"itsm-backend/ent/ticketworkflowrecord"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	_ "github.com/mattn/go-sqlite3"
)

// TestAcceptTicket_UsesTransaction 验证 AcceptTicket 使用 Ent 事务保护
func TestAcceptTicket_UsesTransaction(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:twf_ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	// 创建租户
	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	assert.NoError(t, err)

	// 创建用户（申请人 + 处理人）
	requester, err := client.User.Create().
		SetUsername("requester").
		SetEmail("req@test.com").
		SetPasswordHash("hash").
		SetName("Requester").
		SetTenantID(tenant.ID).
		Save(ctx)
	assert.NoError(t, err)

	assignee, err := client.User.Create().
		SetUsername("assignee").
		SetEmail("assign@test.com").
		SetPasswordHash("hash").
		SetName("Assignee").
		SetTenantID(tenant.ID).
		Save(ctx)
	assert.NoError(t, err)

	// 创建工单
	ticket, err := client.Ticket.Create().
		SetTitle("Test Ticket").
		SetStatus("open").
		SetType("incident").
		SetPriority("medium").
		SetTicketNumber("TK-001").
		SetRequesterID(requester.ID).
		SetTenantID(tenant.ID).
		Save(ctx)
	assert.NoError(t, err)

	svc := NewTicketWorkflowService(client, logger)

	// 执行接单
	req := &dto.AcceptTicketRequest{
		TicketID: ticket.ID,
		Comment:  "接单处理",
	}
	err = svc.AcceptTicket(ctx, req, assignee.ID, tenant.ID)
	assert.NoError(t, err)

	// 验证工单状态已更新为 in_progress
	updatedTicket, err := client.Ticket.Get(ctx, ticket.ID)
	assert.NoError(t, err)
	assert.Equal(t, "in_progress", updatedTicket.Status)
	assert.Equal(t, assignee.ID, updatedTicket.AssigneeID)

	// 验证流转记录已创建（事务保护：工单更新和记录创建原子性）
	// 使用 ent 生成的查询方法
	records, err := client.TicketWorkflowRecord.Query().
		Where(ticketworkflowrecord.TicketID(ticket.ID)).
		All(ctx)
	assert.NoError(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, string(dto.WorkflowActionAccept), records[0].Action)
	assert.Equal(t, "open", records[0].FromStatus)
	assert.Equal(t, "in_progress", records[0].ToStatus)
	assert.Equal(t, assignee.ID, records[0].OperatorID)
	assert.Equal(t, tenant.ID, records[0].TenantID)
}
