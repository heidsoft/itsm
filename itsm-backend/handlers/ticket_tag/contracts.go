package ticket_tag

import (
	"context"

	"itsm-backend/dto"
	"itsm-backend/ent"
)

// TagService 是工单标签 handler 真正消费的最小能力面。
// 方法集按整个域包的生产文件对字段的调用并集整理；请求类型与失败哨兵
// 已下沉到 dto 包，因此契约本身不需要引用共享层。
type TagService interface {
	AssignTagsToTicket(ctx context.Context, ticketID int, tagIDs []int, tenantID int) error
	CreateTag(ctx context.Context, req *dto.CreateTagRequest) (*ent.TicketTag, error)
	DeleteTag(ctx context.Context, id int, tenantID int) error
	GetTag(ctx context.Context, id int, tenantID int) (*ent.TicketTag, error)
	ListTags(ctx context.Context, req *dto.ListTagsRequest) ([]*ent.TicketTag, int, error)
	RemoveTagsFromTicket(ctx context.Context, ticketID int, tagIDs []int, tenantID int) error
	ResolveTagIDsByNames(ctx context.Context, names []string, tenantID int, createMissing bool) ([]int, error)
	UpdateTag(ctx context.Context, id int, req *dto.UpdateTagRequest, tenantID int) (*ent.TicketTag, error)
}
