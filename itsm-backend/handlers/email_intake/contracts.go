package email_intake

import (
	"context"

	"itsm-backend/dto"
)

// EmailIncidentCreator 是邮件报障自动/确认建单路径对事件单域的窄契约。
//
// 编排器只调用 CreateFromEmail，且入参与返回值都是 dto 值类型，因此域包
// 不再需要持有共享层的具体 IncidentService；具体指针隐式满足本接口，
// 组合根与测试夹具的装配调用点无需改动。
type EmailIncidentCreator interface {
	CreateFromEmail(ctx context.Context, tenantID int, command dto.EmailIncidentCommand) (*dto.IncidentResponse, error)
}
