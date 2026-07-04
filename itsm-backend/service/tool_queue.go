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
	jobs    chan ToolJob
	client  *ent.Client
	tools   *ToolRegistry
	tickets *TicketService
	logger  *zap.SugaredLogger
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
			res, err = q.tickets.UpdateTicket(ctx, ticketID, r, job.TenantID)
		default:
			res, err = q.tools.Execute(ctx, job.TenantID, inv.ToolName, args)
		}
		if err != nil {
			if _, updateErr := q.client.ToolInvocation.UpdateOneID(inv.ID).SetStatus("failed").SetError(err.Error()).Save(ctx); updateErr != nil {
				q.logger.Errorw("Failed to update tool invocation status to failed", "invocation_id", inv.ID, "error", updateErr)
			}
		} else {
			out, _ := json.Marshal(res)
			if _, updateErr := q.client.ToolInvocation.UpdateOneID(inv.ID).SetStatus("done").SetResult(string(out)).Save(ctx); updateErr != nil {
				q.logger.Errorw("Failed to update tool invocation status to done", "invocation_id", inv.ID, "error", updateErr)
			}
		}
		cancel()
	}
}
