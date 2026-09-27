package mixin

import (
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// WorkItemCore 提供工作项核心字段：标题、描述。
// 接入 Schema：Ticket, Incident, Problem, Change
//
// 不包含 status：各域默认值不同（open/new/draft/submitted），无法统一。
// 不包含 priority：Ticket 对其建了索引，Ent v0.14.6 不支持 Mixin 字段出现在
// Indexes() 中，因此 priority 由各 Schema 自行调用 PriorityField() 获取。
// title 使用 NotEmpty()；ServiceRequest 的 title 是 Optional()，不接入。
type WorkItemCore struct {
	mixin.Schema
}

func (WorkItemCore) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			Comment("标题").
			NotEmpty(),
		field.Text("description").
			Comment("描述").
			Optional(),
	}
}

// PriorityField 返回带校验的 priority 字段。
// 需要索引 priority 的 Schema 应在自身 Fields() 中调用此函数，
// 而非依赖 Mixin 提供。
func PriorityField() ent.Field {
	return field.String("priority").
		Comment("优先级").
		Validate(func(s string) error {
			valid := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
			if !valid[s] {
				return fmt.Errorf("invalid priority value: %s", s)
			}
			return nil
		}).
		Default("medium")
}
