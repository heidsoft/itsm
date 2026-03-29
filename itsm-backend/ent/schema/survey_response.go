package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"time"
)

// Answer represents an answer to a survey question
type Answer struct {
	QuestionIndex int         `json:"questionIndex"`
	Value         interface{} `json:"value"`
}

// SurveyResponse holds the schema definition for the SurveyResponse entity.
type SurveyResponse struct {
	ent.Schema
}

// Fields of the SurveyResponse.
func (SurveyResponse) Fields() []ent.Field {
	return []ent.Field{
		field.Int("survey_id").
			Comment("Survey ID"),
		field.Int("ticket_id").
			Comment("Ticket ID associated with the survey").
			Optional(),
		field.Int("respondent_id").
			Comment("User ID of the respondent").
			Optional(),
		field.JSON("answers", []Answer{}).
			Comment("Answers to survey questions"),
		field.Int("score").
			Comment("Overall score (0-10 for NPS, 1-5 for CSAT)").
			Default(0),
		field.String("comment").
			Comment("Additional comment").
			Optional(),
		field.Int("tenant_id").
			Comment("Tenant ID").
			Positive(),
		field.Time("submitted_at").
			Comment("Submission time").
			Default(time.Now),
	}
}

// Edges of the SurveyResponse.
func (SurveyResponse) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("survey", Survey.Type).
			Field("survey_id").
			Ref("responses").
			Unique().
			Required().
			Comment("Associated survey"),
	}
}

// Indexes of the SurveyResponse.
func (SurveyResponse) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("survey_id"),
		index.Fields("ticket_id"),
		index.Fields("tenant_id"),
		index.Fields("respondent_id"),
	}
}