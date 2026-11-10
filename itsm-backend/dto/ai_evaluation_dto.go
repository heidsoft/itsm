package dto

// AI 评估与审计读面的值类型（原定义在共享层 service/ai_evaluator.go）。
// 域侧只消费数据形状，service 包保留同名别名，字段与 JSON tag 逐字不变。

type AIEvaluationReport struct {
	GeneratedAt           string                `json:"generatedAt"`
	LookbackDays          int                   `json:"lookbackDays"`
	TotalFeedback         int                   `json:"totalFeedback"`
	UsefulRate            float64               `json:"usefulRate"`
	AcceptedRate          float64               `json:"acceptedRate"`
	AvgConfidence         float64               `json:"avgConfidence"`
	HealthScore           float64               `json:"healthScore"`
	HasData               bool                  `json:"hasData"`
	ByScenario            []AIScenarioEval      `json:"byScenario"`
	BySkill               []AISkillEval         `json:"bySkill"`
	ConfidenceCalibration []AICalibrationBucket `json:"confidenceCalibration"`
	Platform              LLMPlatformStats      `json:"platform"`
}

type AIScenarioEval struct {
	Kind          string  `json:"kind"`
	SkillCode     string  `json:"skillCode,omitempty"`
	SkillName     string  `json:"skillName,omitempty"`
	Count         int     `json:"count"`
	UsefulRate    float64 `json:"usefulRate"`
	AcceptedRate  float64 `json:"acceptedRate"`
	AvgConfidence float64 `json:"avgConfidence"`
}

type AISkillEval struct {
	SkillCode     string  `json:"skillCode"`
	SkillName     string  `json:"skillName,omitempty"`
	Count         int     `json:"count"`
	UsefulRate    float64 `json:"usefulRate"`
	AcceptedRate  float64 `json:"acceptedRate"`
	AvgConfidence float64 `json:"avgConfidence"`
	HealthScore   float64 `json:"healthScore"`
	IsPilot       bool    `json:"isPilot"`
}

type AICalibrationBucket struct {
	Bucket           string  `json:"bucket"`
	Midpoint         float64 `json:"midpoint"`
	Count            int     `json:"count"`
	UsefulRate       float64 `json:"usefulRate"`
	CalibrationError float64 `json:"calibrationError"`
}

type LLMPlatformStats struct {
	LLMCallCount int     `json:"llmCallCount"`
	SuccessRate  float64 `json:"successRate"`
	AvgLatencyMs float64 `json:"avgLatencyMs"`
}

type AIAuditEntry struct {
	ID            int64                  `json:"id"`
	CreatedAt     string                 `json:"createdAt"`
	TenantID      int                    `json:"tenantId"`
	UserID        int                    `json:"userId"`
	RequestID     string                 `json:"requestId"`
	Scenario      string                 `json:"scenario"`
	InputRef      string                 `json:"inputRef"`
	PromptVersion string                 `json:"promptVersion"`
	Model         string                 `json:"model"`
	Confidence    float64                `json:"confidence"`
	Accepted      bool                   `json:"accepted"`
	Suggestion    map[string]interface{} `json:"suggestion"`
	Notes         string                 `json:"notes"`
}
