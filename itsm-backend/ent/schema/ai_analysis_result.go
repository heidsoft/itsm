package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// AIAnalysisResult stores AI analysis results for tickets and incidents.
type AIAnalysisResult struct {
	ent.Schema
}

func (AIAnalysisResult) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id"),
		field.Int("user_id").Optional().Comment("分析发起人"),
		field.String("analysis_type").Comment("triage | summary | rca | deep_analytics | trend_prediction | incident_impact"),
		field.Int("ticket_id").Optional().Comment("关联工单ID"),
		field.Int("incident_id").Optional().Comment("关联事件ID"),
		field.String("ticket_number").Optional().Comment("工单编号快照"),
		field.String("ticket_title").Optional().Comment("工单标题快照"),
		field.Text("request_prompt").Comment("原始分析请求 prompt"),
		field.Text("result_json").Comment("分析结果 JSON 完整存储"),
		field.String("model").Optional().Comment("调用的 LLM 模型"),
		field.Int("latency_ms").Optional().Comment("LLM 响应耗时 ms"),
		field.Int("total_tokens").Optional().Comment("消耗 token 数"),
		field.Float("cost_usd").Optional().Comment("估算费用 USD"),
		field.Float("confidence_score").Optional().Comment("置信度 0-1"),
		field.Bool("degraded").Default(false).Comment("是否降级"),
		field.Time("created_at").Default(time.Now),
	}
}

func (AIAnalysisResult) Edges() []ent.Edge {
	return nil // tenant_id/user_id 直接存字段，不依赖 edge
}
