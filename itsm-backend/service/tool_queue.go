package service

import (
	"context"
	"encoding/json"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	ticketrepo "itsm-backend/repository/ticket"

	"go.uber.org/zap"
)

type ToolJob struct {
	InvocationID int
	TenantID     int
	RequestID    string
}

type ToolQueue struct {
	jobs        chan ToolJob
	client      *ent.Client
	tools       *ToolRegistry
	tickets     *TicketService
	ticketTypes *TicketTypeService
	logger      *zap.SugaredLogger
}

func NewToolQueue(client *ent.Client, tools *ToolRegistry, capacity int, logger *zap.SugaredLogger) *ToolQueue {
	if capacity <= 0 {
		capacity = 100
	}
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	q := &ToolQueue{jobs: make(chan ToolJob, capacity), client: client, tools: tools, logger: logger}
	go q.worker()
	return q
}

// SetTicketTypeService 注入工单类型服务，供 create_ticket_type 审批通过后执行。
func (q *ToolQueue) SetTicketTypeService(s *TicketTypeService) { q.ticketTypes = s }

func (q *ToolQueue) Enqueue(job ToolJob) {
	select {
	case q.jobs <- job:
	default:
	}
}

func (q *ToolQueue) worker() {
	for job := range q.jobs {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		inv, err := q.client.ToolInvocation.Get(ctx, job.InvocationID)
		if err != nil {
			cancel()
			continue
		}
		var args map[string]interface{}
		_ = json.Unmarshal([]byte(inv.Arguments), &args)
		var res interface{}
		// 优先委派给 ToolRegistry：写工具的参数解析/租户校验/CMDB 本体绑定
		// 只在 ToolRegistry.Execute 里维护一份，避免此处内联实现与之漂移
		// （历史上这里的 create_ticket 就漏了 category/type/ticket_type_id/ci_id）。
		// 仅当注册表未装配对应领域服务时，才回落到下面的内联实现。
		if q.tools != nil && q.tools.canExecuteWriteTool(inv.ToolName) {
			res, err = q.tools.Execute(ctx, job.TenantID, inv.ToolName, q.withInvocationUser(args, inv.UserID))
			q.finalize(ctx, inv.ID, res, err)
			cancel()
			continue
		}
		// execute danger tools via TicketService
		switch inv.ToolName {
		case "create_ticket":
			if q.tickets == nil {
				q.tickets = NewTicketService(&TicketServiceConfig{
					Repository: ticketrepo.NewEntRepository(q.client, zap.NewNop().Sugar()),
					Client:     q.client,
					Logger:     zap.NewNop().Sugar(),
				})
			}
			title, _ := args["title"].(string)
			desc, _ := args["description"].(string)
			priority, _ := args["priority"].(string)
			requesterID := 0
			if v, ok := args["requester_id"].(float64); ok {
				requesterID = int(v)
			}
			r := &dto.CreateTicketRequest{Title: title, Description: desc, Priority: priority, RequesterID: requesterID}
			res, err = q.tickets.CreateTicket(ctx, r, job.TenantID)
			// 回落路径也要补 CMDB 本体绑定，否则审批通过后 ci_id 会被静默丢弃
			if err == nil && q.tools != nil && q.tools.cmdb != nil {
				if v, ok := args["ci_id"].(float64); ok && int(v) > 0 {
					if created, ok := res.(*dto.TicketResponse); ok && created != nil {
						if linkErr := q.tools.cmdb.LinkTicketToCI(ctx, job.TenantID, int(v), created.ID); linkErr != nil {
							q.logger.Warnw("Failed to link ticket to CI after approval",
								"invocation_id", inv.ID, "ticket_id", created.ID, "ci_id", int(v), "error", linkErr)
						}
					}
				}
			}
		case "update_ticket":
			if q.tickets == nil {
				q.tickets = NewTicketService(&TicketServiceConfig{
					Repository: ticketrepo.NewEntRepository(q.client, zap.NewNop().Sugar()),
					Client:     q.client,
					Logger:     zap.NewNop().Sugar(),
				})
			}
			ticketID := 0
			if v, ok := args["ticket_id"].(float64); ok {
				ticketID = int(v)
			}
			status, _ := args["status"].(string)
			assigneeID := 0
			if v, ok := args["assignee_id"].(float64); ok {
				assigneeID = int(v)
			}
			r := &dto.UpdateTicketRequest{Status: status, AssigneeID: assigneeID}
			res, err = q.tickets.UpdateTicket(ctx, ticketID, r, job.TenantID, 0, "") // 0=系统操作，跳过 DataScope
		case "create_ticket_type":
			if q.ticketTypes == nil {
				q.ticketTypes = NewTicketTypeService(q.client, zap.NewNop().Sugar())
			}
			code, _ := args["code"].(string)
			name, _ := args["name"].(string)
			desc, _ := args["description"].(string)
			defaultPriority, _ := args["default_priority"].(string)
			if defaultPriority == "" {
				defaultPriority = "medium"
			}
			icon, _ := args["icon"].(string)
			color, _ := args["color"].(string)
			r := &dto.CreateTicketTypeRequest{
				Code:            code,
				Name:            name,
				Description:     desc,
				DefaultPriority: defaultPriority,
				Icon:            icon,
				Color:           color,
			}
			res, err = q.ticketTypes.CreateTicketType(ctx, r, job.TenantID, inv.UserID)
		default:
			res, err = q.tools.Execute(ctx, job.TenantID, inv.ToolName, args)
		}
		q.finalize(ctx, inv.ID, res, err)
		cancel()
	}
}

// finalize 把工具执行结果写回 ToolInvocation（成功/失败两态）。
func (q *ToolQueue) finalize(ctx context.Context, invocationID int, res interface{}, err error) {
	if err != nil {
		if _, updateErr := q.client.ToolInvocation.UpdateOneID(invocationID).SetStatus("failed").SetError(err.Error()).Save(ctx); updateErr != nil {
			q.logger.Errorw("Failed to update tool invocation status to failed", "invocation_id", invocationID, "error", updateErr)
		}
		return
	}
	out, _ := json.Marshal(res)
	if _, updateErr := q.client.ToolInvocation.UpdateOneID(invocationID).SetStatus("done").SetResult(string(out)).Save(ctx); updateErr != nil {
		q.logger.Errorw("Failed to update tool invocation status to done", "invocation_id", invocationID, "error", updateErr)
	}
}

// withInvocationUser 把发起审批的用户ID回填进参数，让 ToolRegistry 能正确归属
// requester_id / 创建人（LLM 生成的参数里通常不含用户ID）。
func (q *ToolQueue) withInvocationUser(args map[string]interface{}, userID int) map[string]interface{} {
	if userID <= 0 {
		return args
	}
	merged := make(map[string]interface{}, len(args)+1)
	for k, v := range args {
		merged[k] = v
	}
	if _, ok := merged["user_id"]; !ok {
		merged["user_id"] = float64(userID)
	}
	return merged
}
