package dto

// EmailIncidentCommand 是邮件报障自动建单的入参值类型。
// 原定义在共享层 service 包，域侧 orchestrator 只构造并传递它，
// 因此下沉到 dto，service 保留同名别名。

type EmailIncidentCommand struct {
	ConversationID    int
	SupportContractID int
	ReporterUserID    int
	AssignmentGroupID *int
	AssigneeID        *int
	Title             string
	Description       string
	Impact            string
	Urgency           string
	Category          string
	Metadata          map[string]interface{}
	OverrideContract  bool
	OverrideReason    string
}
