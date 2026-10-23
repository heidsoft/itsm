package ticket

import (
	"testing"
	"time"

	repoTicket "itsm-backend/repository/ticket"

	"github.com/stretchr/testify/assert"
)

// productionSvc 创建结果转 handler 实体时曾漏拷 TicketTypeID，
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

// ticketToResponse 曾丢弃 SLA 截止时间，GET /api/v1/tickets/:id 响应
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

	got := buildTicketResponse(ticket)

	assert.Same(t, &response, got.SLAResponseDeadline)
	assert.Same(t, &resolution, got.SLAResolutionDeadline)

	// 未绑定 SLA 时两个字段保持 nil，被 omitempty 吞掉是预期契约。
	empty := buildTicketResponse(&Ticket{ID: 1, TenantID: 1})
	assert.Nil(t, empty.SLAResponseDeadline)
	assert.Nil(t, empty.SLAResolutionDeadline)
}

// resolve/close 会把解决方案与生命周期时间戳写入数据库，但 buildTicketResponse
// 曾经漏拷这四个字段：`itsm flow run ticket-lifecycle` 在 assert `resolution`
// 时读不到任何值（现象与「解决方案没保存」完全一致），详情页也看不到解决时间。
// 这是 API 契约缺陷，不是测试数据问题。
func TestBuildTicketResponseKeepsResolutionAndLifecycleTimestamps(t *testing.T) {
	resolvedAt := time.Date(2026, 10, 22, 14, 0, 0, 0, time.UTC)
	closedAt := resolvedAt.Add(30 * time.Minute)
	firstResponseAt := resolvedAt.Add(-15 * time.Minute)
	resolution := "重启 worker 并清理连接池"

	ticket := &Ticket{
		ID:              7,
		TenantID:        1,
		Resolution:      &resolution,
		ResolvedAt:      &resolvedAt,
		ClosedAt:        &closedAt,
		FirstResponseAt: &firstResponseAt,
	}

	got := buildTicketResponse(ticket)

	assert.Equal(t, resolution, got.Resolution, "resolution 必须透出，否则调用方无法确认解决方案")
	assert.Same(t, &resolvedAt, got.ResolvedAt)
	assert.Same(t, &closedAt, got.ClosedAt)
	assert.Same(t, &firstResponseAt, got.FirstResponseAt)

	// 未解决工单保持空值，由 omitempty 隐藏。
	unresolved := buildTicketResponse(&Ticket{ID: 8, TenantID: 1})
	assert.Empty(t, unresolved.Resolution)
	assert.Nil(t, unresolved.ResolvedAt)
	assert.Nil(t, unresolved.ClosedAt)
}
