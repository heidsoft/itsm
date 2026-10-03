package ticket

import (
	"testing"
	"time"

	repoTicket "itsm-backend/repository/ticket"

	"github.com/stretchr/testify/assert"
)

// 回归：productionSvc 创建结果转 handler 实体时曾漏拷 TicketTypeID，
// 被 TicketResponse 的 omitempty 吞掉，导致 POST /api/v1/tickets 响应
// 缺少 ticketTypeId（ga-gate 的 ticket-type-full-chain E2E 因此红灯）。
func TestProductionTicketToTicketKeepsTicketTypeID(t *testing.T) {
	ticketTypeID := 13
	created := &repoTicket.Ticket{
		ID:             42,
		TicketNumber:   "TKT-202610-000042",
		Title:          "E2E full chain",
		Description:    "Disposable HTTP/worker regression",
		Status:         repoTicket.Status("new"),
		Priority:       repoTicket.Priority("high"),
		Type:           repoTicket.Type("incident"),
		TicketTypeID:   &ticketTypeID,
		TicketTypeCode: "e2e_pacs",
		TicketTypeName: "PACS 连通性",
		FormFields:     map[string]interface{}{"pacsNode": "node-1"},
		RequesterID:    7,
		TenantID:       1,
		Version:        1,
	}

	got := productionTicketToTicket(created)

	assert.NotNil(t, got.TicketTypeID, "ticketTypeId 不得在转换中丢失")
	assert.Equal(t, ticketTypeID, *got.TicketTypeID)
	assert.Equal(t, "e2e_pacs", got.TicketTypeCode)
	assert.Equal(t, "PACS 连通性", got.TicketTypeName)
	assert.Equal(t, 42, got.ID)
	assert.Equal(t, 7, got.RequesterID)
	assert.Equal(t, 1, got.TenantID)
}

// 回归：ticketToResponse 曾丢弃 SLA 截止时间，GET /api/v1/tickets/:id 响应
// 缺少 slaResponseDeadline/slaResolutionDeadline（ga-gate 的 assertSLA 因此红灯）。
func TestTicketToResponseKeepsSLADeadlines(t *testing.T) {
	response := time.Date(2026, 10, 3, 10, 37, 0, 0, time.UTC)
	resolution := response.Add(210 * time.Minute)
	ticket := &Ticket{
		ID:                    42,
		TenantID:              1,
		SLAResponseDeadline:   &response,
		SLAResolutionDeadline: &resolution,
	}

	got := ticketToResponse(ticket)

	assert.Same(t, &response, got.SLAResponseDeadline)
	assert.Same(t, &resolution, got.SLAResolutionDeadline)

	// 未绑定 SLA 时两个字段保持 nil，被 omitempty 吞掉是预期契约。
	empty := ticketToResponse(&Ticket{ID: 1, TenantID: 1})
	assert.Nil(t, empty.SLAResponseDeadline)
	assert.Nil(t, empty.SLAResolutionDeadline)
}
