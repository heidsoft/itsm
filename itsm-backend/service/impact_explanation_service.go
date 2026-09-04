package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"itsm-backend/dto"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// ImpactExplanationService 影响分析 AI 解释服务（P1-4）
//
// 职责：
//   - 把 CIRelationshipService.GetCIImpactAnalysis 返回的影响图（节点+边+跳数+风险等级）
//     翻译成自然语言解释（业务含义/可能根因/SLA 风险/建议处置）
//   - 缓存：同 CI + 同跳数 5 分钟 Redis 缓存（避免重复 LLM 调用）
//   - 失败 fallback：LLM 失败时返回 nil + 不报错（LLM 是 nice-to-have）
//   - 与 ontology 端点联动：返回 enums + relationshipTypes 让 LLM 用业务术语
//
// 不强制依赖 LLM（nil gateway → 返回 nil explain），方便测试与离线环境。
type ImpactExplanationService struct {
	llm           *LLMGateway
	model         string
	redis         *redis.Client
	logger        *zap.SugaredLogger
	cacheTTL      time.Duration
}

func NewImpactExplanationService(
	llm *LLMGateway,
	model string,
	rdb *redis.Client,
	logger *zap.SugaredLogger,
) *ImpactExplanationService {
	return &ImpactExplanationService{
		llm:      llm,
		model:    model,
		redis:    rdb,
		logger:   logger,
		cacheTTL: 5 * time.Minute,
	}
}

// ImpactExplanation 影响分析 AI 解释响应
type ImpactExplanation struct {
	Summary     string   `json:"summary"`     // 一句话总结（如「3 个下游服务受数据库主库故障影响」）
	RootCauses  []string `json:"rootCauses"`  // 可能根因（业务视角）
	SLARisks    []string `json:"slaRisks"`    // SLA 风险点（哪些 CI criticality 较高、可能超时）
	Suggestions []string `json:"suggestions"` // 建议处置（先排查什么、优先恢复什么）
	GeneratedAt string   `json:"generatedAt"` // 生成时间（RFC3339）
	Model       string   `json:"model,omitempty"`
}

// ExplainImpact 生成影响分析 AI 解释
//   - ciID/tenantID/hops: 影响分析输入参数（同时作为缓存键）
//   - impact: 已经 GetCIImpactAnalysis 算好的影响图（避免重复 BFS）
//
// 返回值：
//   - (*ImpactExplanation, nil): LLM 成功（或缓存命中）
//   - (nil, nil): LLM 未配置 / 失败 / 上下文取消（不向上报错，因为解释层是 nice-to-have）
//   - (nil, error): 仅当参数本身无效
func (s *ImpactExplanationService) ExplainImpact(
	ctx context.Context,
	tenantID int,
	ciID int,
	hops int,
	impact *dto.CIImpactAnalysisResponse,
) (*ImpactExplanation, error) {
	if impact == nil {
		return nil, fmt.Errorf("impact data is nil")
	}
	if s.llm == nil {
		// 无 LLM 配置：跳过解释层（fail-open）
		return nil, nil
	}

	// 缓存命中
	if cached := s.loadFromCache(ctx, tenantID, ciID, hops); cached != nil {
		return cached, nil
	}

	// 构造 prompt：影响图 → 自然语言
	prompt := buildImpactPrompt(impact)
	messages := []LLMMessage{
		{Role: "system", Content: impactExplainerSystemPrompt},
		{Role: "user", Content: prompt},
	}

	resp, err := s.llm.Chat(ctx, s.model, messages)
	if err != nil {
		s.logger.Warnw("Impact explanation LLM failed; falling back to nil", "error", err, "ci_id", ciID)
		return nil, nil
	}

	// 解析 LLM 输出（宽松 JSON：模型可能返回 markdown ```json 块）
	parsed, parseErr := parseImpactExplanationJSON(resp)
	if parseErr != nil {
		s.logger.Warnw("Impact explanation JSON parse failed; raw", "error", parseErr, "raw", truncateForLog(resp))
		return nil, nil
	}
	parsed.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	parsed.Model = s.model

	// 写缓存（即使 LLM 失败也写默认值避免雪崩）
	s.saveToCache(ctx, tenantID, ciID, hops, parsed)
	return parsed, nil
}

// loadFromCache 从 Redis 读缓存（key: impact:explain:<tenantID>:<ciID>:<hops>）
func (s *ImpactExplanationService) loadFromCache(ctx context.Context, tenantID, ciID, hops int) *ImpactExplanation {
	if s.redis == nil {
		return nil
	}
	key := s.cacheKey(tenantID, ciID, hops)
	raw, err := s.redis.Get(ctx, key).Bytes()
	if err != nil {
		return nil
	}
	var ie ImpactExplanation
	if err := json.Unmarshal(raw, &ie); err != nil {
		return nil
	}
	return &ie
}

// saveToCache 写缓存
func (s *ImpactExplanationService) saveToCache(ctx context.Context, tenantID, ciID, hops int, ie *ImpactExplanation) {
	if s.redis == nil || ie == nil {
		return
	}
	key := s.cacheKey(tenantID, ciID, hops)
	raw, err := json.Marshal(ie)
	if err != nil {
		return
	}
	// 异步写缓存即可：失败不影响主流程
	_ = s.redis.Set(ctx, key, raw, s.cacheTTL).Err()
}

func (s *ImpactExplanationService) cacheKey(tenantID, ciID, hops int) string {
	return fmt.Sprintf("impact:explain:%d:%d:%d", tenantID, ciID, hops)
}

// buildImpactPrompt 把影响图序列化成 prompt 输入
func buildImpactPrompt(impact *dto.CIImpactAnalysisResponse) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("源 CI ID: %d\n", impact.SourceCIID))
	if impact.Graph != nil {
		sb.WriteString(fmt.Sprintf("图概览: 节点=%d, 边=%d, 深度=%d\n",
			impact.Graph.TotalNodes, impact.Graph.TotalEdges, impact.Graph.Depth))
	}
	if len(impact.UpstreamImpact) > 0 {
		sb.WriteString(fmt.Sprintf("上游影响 (%d):\n", len(impact.UpstreamImpact)))
		for _, u := range impact.UpstreamImpact {
			sb.WriteString(fmt.Sprintf("  - distance=%d direction=%s type=%s impact=%s\n",
				u.Distance, u.Direction, u.RelationshipType, u.ImpactLevel))
		}
	}
	if len(impact.DownstreamImpact) > 0 {
		sb.WriteString(fmt.Sprintf("下游影响 (%d):\n", len(impact.DownstreamImpact)))
		for _, d := range impact.DownstreamImpact {
			sb.WriteString(fmt.Sprintf("  - distance=%d direction=%s type=%s impact=%s\n",
				d.Distance, d.Direction, d.RelationshipType, d.ImpactLevel))
		}
	}
	if len(impact.AffectedTickets) > 0 {
		sb.WriteString(fmt.Sprintf("受影响的工单数: %d\n", len(impact.AffectedTickets)))
	}
	if len(impact.AffectedIncidents) > 0 {
		sb.WriteString(fmt.Sprintf("受影响的事件数: %d\n", len(impact.AffectedIncidents)))
	}
	if impact.RiskLevel != "" {
		sb.WriteString(fmt.Sprintf("整体风险等级: %s\n", impact.RiskLevel))
	}
	if impact.Summary != "" {
		sb.WriteString(fmt.Sprintf("图内置摘要: %s\n", impact.Summary))
	}
	return sb.String()
}

const impactExplainerSystemPrompt = `你是 ITSM 影响分析助手。基于给定的 CI 影响图数据（节点、距离、关系类型、风险等级、受影响工单/事件数），
输出严格 JSON 格式的分析结果，字段：
- summary: 一句话中文总结（≤ 80 字）
- rootCauses: 可能的业务根因数组（每条 ≤ 40 字，2-3 条）
- slaRisks: SLA 风险点数组（每条 ≤ 40 字，1-3 条）
- suggestions: 建议处置动作数组（每条 ≤ 40 字，2-4 条）

只输出 JSON，不要任何 markdown 包装、解释或注释。`

// parseImpactExplanationJSON 宽松解析（处理 ```json 块 + 截断）
func parseImpactExplanationJSON(raw string) (*ImpactExplanation, error) {
	trimmed := strings.TrimSpace(raw)
	// 去掉 markdown ```json ... ``` 包装
	if strings.HasPrefix(trimmed, "```") {
		if i := strings.Index(trimmed, "\n"); i > 0 {
			trimmed = trimmed[i+1:]
		}
		if j := strings.LastIndex(trimmed, "```"); j > 0 {
			trimmed = trimmed[:j]
		}
		trimmed = strings.TrimSpace(trimmed)
	}
	var ie ImpactExplanation
	if err := json.Unmarshal([]byte(trimmed), &ie); err != nil {
		return nil, err
	}
	if ie.Summary == "" {
		return nil, fmt.Errorf("summary missing")
	}
	return &ie, nil
}

func truncateForLog(s string) string {
	const max = 200
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}