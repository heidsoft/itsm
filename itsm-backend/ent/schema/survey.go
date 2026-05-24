package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// SurveyQuestion represents a survey question structure
type SurveyQuestion struct {
	Question string   `json:"question"`
	Type     string   `json:"type"` // rating, text, choice
	Options  []string `json:"options,omitempty"`
	Required bool     `json:"required"`
}

// Survey holds the schema definition for the Survey entity.
type Survey struct {
	ent.Schema
}

// Fields of the Survey.
func (Survey) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			Comment("Survey title").
			NotEmpty(),
		field.String("description").
			Comment("Survey description").
			Optional(),
		field.String("survey_type").
			Comment("Survey type: NPS, CSAT, CES").
			Default("CSAT"),
		field.Bool("is_active").
			Comment("Whether survey is active").
			Default(true),
		field.Time("start_date").
			Comment("Survey start date").
			Optional(),
		field.Time("end_date").
			Comment("Survey end date").
			Optional(),
		field.JSON("questions", []SurveyQuestion{}).
			Comment("Survey questions"),
		field.Int("tenant_id").
			Comment("Tenant ID").
			Positive(),
		field.Time("created_at").
			Comment("Created time").
			Default(time.Now),
		field.Time("updated_at").
			Comment("Updated time").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Survey.
func (Survey) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("responses", SurveyResponse.Type).
			Comment("Survey responses"),
	}
}

// Indexes of the Survey.
func (Survey) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id"),
		index.Fields("survey_type"),
		index.Fields("is_active"),
	}
}
