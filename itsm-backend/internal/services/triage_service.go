package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"itsm-backend/dto"

	"go.uber.org/zap"
)

// TriagePromptVersion 是分诊建议的 prompt/口径版本，随响应与服务端 AI 审计底账
// 一起下发。分词语义、类别枚举或置信度归一化规则变更时必须递增，
// 采纳率等评估指标才能按版本对齐（与 email-intake-v1 / incident-rca-v1 同一契约）。
const TriagePromptVersion = "triage-v1"

// guidanceTriageClient defines the interface for Guidance sidecar calls
// This enables dependency injection for testing
type guidanceTriageClient interface {
	Triage(ctx context.Context, title, description string, tenantID int) (*GuidanceTriageResponse, error)
}

// normalizeResult ensures triage result has valid category, priority, and confidence values
func (t *TriageService) normalizeResult(result TriageResult) TriageResult {
	// Validate and normalize category
	if result.Category == "" || !isValidCategory(result.Category) {
		result.Category = "general"
		// Reset assignee when category is invalid — the original assignee
		// is associated with an unrecognized category and is unreliable.
		result.AssigneeID = 0
	}
	// Validate and normalize priority
	if result.Priority == "" || !isValidPriority(result.Priority) {
		result.Priority = "medium"
	}
	// Validate and normalize confidence
	if result.Confidence == 0 {
		result.Confidence = 0.6
	}
	if result.Confidence < 0 || result.Confidence > 1 {
		result.Confidence = 0.6
	}
	// Set default assignee based on category when not already assigned
	if result.AssigneeID == 0 {
		if assigneeID, ok := t.defaultAssignees[result.Category]; ok {
			result.AssigneeID = assigneeID
		}
	}
	return result
}

// TriageResult 值类型唯一实现在 dto 包；此处保留同名别名以保持 service 包公开 API。
type TriageResult = dto.TriageResult

// modelBackedPaths 是「真的有一次模型推理」的链路标识。
// 其余链路（keyword 启发式、guidance-fallback 等）一律视为降级。
var modelBackedPaths = map[string]bool{
	"llm":      true,
	"guidance": true,
}

// withTrace 给分诊结果补溯源字段（决策链路 + 口径版本 + 是否降级）。
func (t *TriageService) withTrace(r TriageResult, model string) TriageResult {
	r.Model = model
	r.PromptVersion = TriagePromptVersion
	if !modelBackedPaths[strings.ToLower(strings.TrimSpace(model))] {
		r.Degraded = true
		if r.DegradedReason == "" {
			r.DegradedReason = "no model inference on path " + model
		}
	}
	return r
}

// guidanceDegradedMarkers 是 sidecar 响应 method 里表示「这不是模型给的」的标记。
//
// 用「黑名单包含」而不是「白名单等于」：sidecar 未来新增正常模型链路（如
// guidance-minimax）时不该被误判成降级；反过来，只要 method 里带这些词，
// 就一定是降级。空 method 表示老版本 sidecar 没这个字段，按 guidance 处理。
var guidanceDegradedMarkers = []string{"fallback", "keyword", "heuristic", "degraded", "error", "mock", "stub"}

// isDegradedGuidanceMethod 判断 sidecar 返回的 method 是否标记了降级。
func isDegradedGuidanceMethod(method string) bool {
	m := strings.ToLower(strings.TrimSpace(method))
	if m == "" {
		return false
	}
	for _, marker := range guidanceDegradedMarkers {
		if strings.Contains(m, marker) {
			return true
		}
	}
	return false
}

// guidanceResultDegraded 综合 sidecar 的显式降级位与 method 约定做判定。
// 两者取或：新版 sidecar 可能只置 degraded=true 而不改 method，老版只有 method。
func guidanceResultDegraded(r *GuidanceTriageResponse) bool {
	if r == nil {
		return false
	}
	return r.Degraded || isDegradedGuidanceMethod(r.Method)
}

// TriageService provides LLM-powered ticket triage and classification
type TriageService struct {
	gateway          *LLMGateway
	guidanceClient   guidanceTriageClient
	logger           *zap.Logger
	keywordFallback  bool
	defaultAssignees map[string]int
}

// Default assignee IDs by category
var defaultAssignees = map[string]int{
	"database":    101,
	"network":     102,
	"server":      103,
	"application": 104,
	"security":    100,
	"storage":     105,
	"user_access": 106,
	"general":     0,
}

// NewTriageService creates a new LLM-powered triage service
func NewTriageService(gateway *LLMGateway, logger *zap.Logger) *TriageService {
	return &TriageService{
		gateway:          gateway,
		logger:           logger,
		keywordFallback:  true,
		defaultAssignees: defaultAssignees,
	}
}

// NewTriageServiceWithSugaredLogger creates a new LLM-powered triage service with SugaredLogger
func NewTriageServiceWithSugaredLogger(gateway *LLMGateway, logger *zap.SugaredLogger) *TriageService {
	return &TriageService{
		gateway:          gateway,
		logger:           logger.Desugar(),
		keywordFallback:  true,
		defaultAssignees: defaultAssignees,
	}
}

// NewTriageServiceWithGuidance creates a triage service with Guidance sidecar
func NewTriageServiceWithGuidance(gateway *LLMGateway, guidanceClient *GuidanceClient, logger *zap.Logger) *TriageService {
	return &TriageService{
		gateway:          gateway,
		guidanceClient:   guidanceClient,
		logger:           logger,
		keywordFallback:  true,
		defaultAssignees: defaultAssignees,
	}
}

// NewTriageServiceWithGuidanceAndSugaredLogger creates with SugaredLogger
func NewTriageServiceWithGuidanceAndSugaredLogger(gateway *LLMGateway, guidanceClient *GuidanceClient, logger *zap.SugaredLogger) *TriageService {
	return &TriageService{
		gateway:          gateway,
		guidanceClient:   guidanceClient,
		logger:           logger.Desugar(),
		keywordFallback:  true,
		defaultAssignees: defaultAssignees,
	}
}

// NewTriageServiceWithConfig creates a service with custom configuration
func NewTriageServiceWithConfig(gateway *LLMGateway, logger *zap.Logger, fallback bool, assignees map[string]int) *TriageService {
	svc := &TriageService{
		gateway:          gateway,
		logger:           logger,
		keywordFallback:  fallback,
		defaultAssignees: defaultAssignees,
	}
	if assignees != nil {
		svc.defaultAssignees = assignees
	}
	return svc
}

// Suggest 是无租户上下文的分诊入口，仅供离线工具与单元测试使用。
// 生产请求必须使用 SuggestForTenant：缺少租户时不向 guidance 发起租户级查询。
func (t *TriageService) Suggest(ctx context.Context, title, description string) TriageResult {
	return t.suggest(ctx, title, description, 0, false)
}

// SuggestForTenant returns classification suggestions scoped to a tenant.
func (t *TriageService) SuggestForTenant(ctx context.Context, title, description string, tenantID int) TriageResult {
	return t.suggest(ctx, title, description, tenantID, tenantID > 0)
}

func (t *TriageService) suggest(ctx context.Context, title, description string, tenantID int, useGuidance bool) TriageResult {
	text := strings.ToLower(title + " " + description)

	result := TriageResult{
		Category:    "general",
		Priority:    "medium",
		AssigneeID:  0,
		Confidence:  0.5,
		Explanation: "keyword heuristic",
	}

	// 侧车降级响应先扣在手里：它是「关键词 + 写死置信度」的兜底，价值低于本地
	// 关键词启发式（后者带 category→assignee 映射、中文词表更全）。所以先让 LLM
	// 网关和本地启发式试一遍，只有它们都拿不到结果时才回头用它，并标注 degraded。
	var degradedFromSidecar *GuidanceTriageResponse

	// Priority 1: Use Guidance sidecar for constrained generation
	if useGuidance && t.guidanceClient != nil {
		guidanceResult, err := t.guidanceClient.Triage(ctx, title, description, tenantID)
		if err != nil {
			t.logger.Warn("TriageService: Guidance sidecar failed, falling back", zap.Error(err))
		} else if guidanceResultDegraded(guidanceResult) {
			// 关键：sidecar 把 LLM 故障包装成 HTTP 200 + 关键词结果返回。
			// 直接采信会让 AI 审计底账记下一条「模型决策」，事后无法区分
			// 模型结论与正则凑数。这里先降级为候选，不当最终答案。
			t.logger.Warn("TriageService: Guidance sidecar returned degraded result, trying local paths first",
				zap.String("method", guidanceResult.Method),
				zap.String("category", guidanceResult.Category),
				zap.Float64("confidence", guidanceResult.Confidence))
			degradedFromSidecar = guidanceResult
		} else {
			t.logger.Info("TriageService: Guidance-constrained classification",
				zap.String("category", guidanceResult.Category),
				zap.String("priority", guidanceResult.Priority),
				zap.Float64("confidence", guidanceResult.Confidence),
				zap.String("method", guidanceResult.Method),
				zap.Float64("latency_ms", guidanceResult.LatencyMs))
			// 透传 sidecar 自报的 method（老版本无该字段时回落 "guidance"），
			// 让溯源链路上「谁给的结论」有据可查，而不是后端硬编码一个值。
			method := strings.TrimSpace(guidanceResult.Method)
			if method == "" {
				method = "guidance"
			}
			return t.withTrace(t.normalizeResult(TriageResult{
				Category:     guidanceResult.Category,
				Priority:     guidanceResult.Priority,
				AssigneeID:   guidanceResult.AssigneeID,
				Confidence:   guidanceResult.Confidence,
				Explanation:  guidanceResult.Explanation,
				SuggestedFix: guidanceResult.SuggestedFix,
			}), method)
		}
	}

	// Priority 2: Use LLM gateway directly
	if t.gateway != nil {
		llmResult, err := t.llmClassify(ctx, title, description)
		if err != nil {
			t.logger.Warn("TriageService: LLM classification failed, using fallback", zap.Error(err))
			if t.keywordFallback {
				return t.withTrace(t.keywordBasedSuggest(text), "keyword")
			}
			if degradedFromSidecar != nil {
				r, method := t.sidecarDegradedResult(degradedFromSidecar)
				return t.withTrace(r, method)
			}
			return t.withTrace(result, "keyword")
		}

		// LLM-first: use LLM result if confidence is acceptable
		if llmResult.Confidence < 0.5 && t.keywordFallback {
			keywordResult := t.keywordBasedSuggest(text)
			if keywordResult.Confidence > llmResult.Confidence {
				t.logger.Info("TriageService: keyword result has higher confidence",
					zap.Float64("llm_confidence", llmResult.Confidence),
					zap.Float64("keyword_confidence", keywordResult.Confidence))
				return t.withTrace(keywordResult, "keyword")
			}
		}

		return t.withTrace(llmResult, "llm")
	}

	// Priority 3: Use keyword-based fallback
	if t.keywordFallback {
		return t.withTrace(t.keywordBasedSuggest(text), "keyword")
	}

	// Priority 4: 本地链路全无产出，只能用 sidecar 的降级结果（并如实标注）。
	if degradedFromSidecar != nil {
		r, method := t.sidecarDegradedResult(degradedFromSidecar)
		return t.withTrace(r, method)
	}

	return t.withTrace(result, "keyword")
}

// sidecarDegradedResult 把 sidecar 的降级响应转成本地结果，并保留其自报的 method。
// 只在本地链路（LLM 网关 / 关键词启发式）都拿不到结果时使用；返回的 method
// 会被 withTrace 判定为非模型链路，从而置 Degraded=true。
func (t *TriageService) sidecarDegradedResult(degraded *GuidanceTriageResponse) (TriageResult, string) {
	method := strings.TrimSpace(degraded.Method)
	if method == "" {
		method = "guidance-fallback"
	}
	r := t.normalizeResult(TriageResult{
		Category:     degraded.Category,
		Priority:     degraded.Priority,
		AssigneeID:   degraded.AssigneeID,
		Confidence:   degraded.Confidence,
		Explanation:  degraded.Explanation,
		SuggestedFix: degraded.SuggestedFix,
	})
	r.DegradedReason = "guidance sidecar returned " + method
	return r, method
}

// llmClassify uses LLM for intelligent ticket classification
// Uses Guidance-style constrained prompting to ensure valid JSON output
func (t *TriageService) llmClassify(ctx context.Context, title, description string) (TriageResult, error) {
	// Guidance-style constrained prompt: explicit schema with strict enum values
	// This guides the model to output valid JSON matching our schema
	prompt := fmt.Sprintf(`You are an expert IT service management triage assistant.
Your task is to classify IT tickets into exactly ONE of these categories:
- database (MySQL, PostgreSQL, MongoDB, Redis, Oracle, SQL issues)
- network (WiFi, router, switch, firewall, connectivity issues)
- server (CPU, memory, disk, Linux, Windows Server, hardware)
- application (software, app, API, deployment, code bugs)
- security (vulnerability, attack, permission, authentication issues)
- storage (disk space, backup, snapshot, storage capacity)
- user_access (account, login, password, access control)
- general (everything else)

Priority levels (choose ONE):
- critical (system down, security breach, data loss)
- high (major feature broken, many users impacted)
- medium (degraded functionality, some users impacted)
- low (minor issue, cosmetic problems)

Title: %s

Description:
%s

IMPORTANT: Respond with ONLY valid JSON in this exact format, no other text:
{"category": "EXACT_CATEGORY_FROM_LIST", "priority": "EXACT_PRIORITY_FROM_LIST", "confidence": 0.0-1.0, "explanation": "brief reason", "suggestedFix": "brief solution or null"}`, title, description)

	messages := []LLMMessage{
		{Role: "system", Content: "You are an IT service management triage assistant. Always output valid JSON."},
		{Role: "user", Content: prompt},
	}

	resp, err := t.gateway.Chat(ctx, "", messages)
	if err != nil {
		return TriageResult{}, fmt.Errorf("LLM classification failed: %w", err)
	}
	// 调用成功后立刻取真实模型名：本次调用没指定 model，实际用的是 provider
	// 配置的那一个，只有网关能报出来。放在解析之前，避免解析失败时丢掉归属。
	effectiveModel := t.gateway.EffectiveModel("")

	resp = strings.TrimSpace(resp)
	resp = strings.TrimPrefix(resp, "```json")
	resp = strings.TrimPrefix(resp, "```")
	resp = strings.TrimSuffix(resp, "```")
	resp = strings.TrimSpace(resp)

	var classification TriageResult
	if err := json.Unmarshal([]byte(resp), &classification); err != nil {
		// Fallback: try to find JSON in response
		start := strings.Index(resp, "{")
		end := strings.LastIndex(resp, "}")
		if start >= 0 && end >= start {
			if err := json.Unmarshal([]byte(resp[start:end+1]), &classification); err != nil {
				return TriageResult{}, fmt.Errorf("failed to parse LLM response: %w", err)
			}
		} else {
			return TriageResult{}, fmt.Errorf("no JSON found in LLM response")
		}
	}

	// LLM assignee IDs are not trustworthy — always reset to category default
	classification.AssigneeID = 0

	// Go 的 JSON 解码忽略键的大小写但不忽略下划线，LLM 返回 snake_case 的 suggested_fix 时
	// camelCase 字段会是空串，因此 camelCase 为空时再从 snake_case 键回填一次。
	if classification.SuggestedFix == "" {
		var snake struct {
			SuggestedFix string `json:"suggested_fix"`
		}
		_ = json.Unmarshal([]byte(resp), &snake)
		classification.SuggestedFix = snake.SuggestedFix
	}

	// 归属本次决策的真实模型名（可能被 LLM 自己吐出的 JSON 覆盖，因此放在解析之后）。
	classification.ModelName = effectiveModel

	// Normalize and validate enum values using shared helper
	return t.normalizeResult(classification), nil
}

// isValidCategory checks if category is a valid enum value
func isValidCategory(cat string) bool {
	validCategories := []string{"database", "network", "server", "application", "security", "storage", "user_access", "general"}
	for _, v := range validCategories {
		if cat == v {
			return true
		}
	}
	return false
}

// isValidPriority checks if priority is a valid enum value
func isValidPriority(pri string) bool {
	validPriorities := []string{"critical", "high", "medium", "low"}
	for _, v := range validPriorities {
		if pri == v {
			return true
		}
	}
	return false
}

// keywordBasedSuggest provides fallback classification using keywords
func (t *TriageService) keywordBasedSuggest(text string) TriageResult {
	result := TriageResult{
		Category:    "general",
		Priority:    "medium",
		AssigneeID:  0,
		Confidence:  0.55,
		Explanation: "keyword heuristic",
	}

	// Category detection based on keywords
	if containsAny(text, "database", "db", "mysql", "postgresql", "oracle", "sql", "mongodb", "redis", "数据库", "慢查询", "连接池") {
		result.Category = "database"
		result.AssigneeID = t.defaultAssignees["database"]
		result.Confidence = 0.7
	} else if containsAny(text, "network", "wifi", "switch", "router", "firewall", "网络", "无法上网", "交换机", "路由器", "防火墙", "丢包") {
		result.Category = "network"
		result.AssigneeID = t.defaultAssignees["network"]
		result.Confidence = 0.68
	} else if containsAny(text, "storage", "disk space", "space", "backup", "snapshot", "存储", "磁盘空间", "空间不足", "备份", "快照", "容量") {
		result.Category = "storage"
		result.AssigneeID = t.defaultAssignees["storage"]
		result.Confidence = 0.66
	} else if containsAny(text, "server", "cpu", "memory", "disk", "linux", "windows server", "服务器", "主机", "内存", "磁盘", "操作系统") {
		result.Category = "server"
		result.AssigneeID = t.defaultAssignees["server"]
		result.Confidence = 0.65
	} else if containsAny(text, "application", "software", "app", "api", "deploy", "应用", "软件", "接口", "部署", "发布", "报错", "服务异常") {
		result.Category = "application"
		result.AssigneeID = t.defaultAssignees["application"]
		result.Confidence = 0.62
	} else if containsAny(text, "security", "vulnerability", "attack", "permission", "authentication", "安全", "漏洞", "攻击", "入侵", "恶意", "认证绕过") {
		result.Category = "security"
		result.AssigneeID = t.defaultAssignees["security"]
		result.Confidence = 0.75
		result.Priority = "critical"
	} else if containsAny(text, "user", "account", "login", "password", "access", "用户", "账号", "账户", "登录", "密码", "权限", "访问") {
		result.Category = "user_access"
		result.AssigneeID = t.defaultAssignees["user_access"]
		result.Confidence = 0.6
	}

	// Priority escalation
	if containsAny(text, "down", "unavailable", "urgent", "critical", "outage", "宕机", "不可用", "紧急", "严重", "中断", "故障") {
		if result.Priority != "critical" {
			result.Priority = "high"
		}
		result.Confidence += 0.1
	}

	if containsAny(text, "impact", "affect", "users", "business", "影响", "多个用户", "大量用户", "所有用户", "全员", "业务") {
		if result.Priority == "medium" {
			result.Priority = "high"
		}
	}

	return result
}

// containsAny checks if text contains any of the given substrings
func containsAny(text string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(text, sub) {
			return true
		}
	}
	return false
}

// TriageWithMetadata includes timing and method information
type TriageWithMetadata struct {
	Result  TriageResult  `json:"result"`
	Method  string        `json:"method"`
	Latency time.Duration `json:"latencyMs"`
	Model   string        `json:"model,omitempty"`
}

// SuggestWithMetadata returns triage result with additional metadata
func (t *TriageService) SuggestWithMetadata(ctx context.Context, title, description string) (*TriageWithMetadata, error) {
	start := time.Now()

	text := strings.ToLower(title + " " + description)

	result := &TriageWithMetadata{
		Result: t.withTrace(TriageResult{
			Category:    "general",
			Priority:    "medium",
			AssigneeID:  0,
			Confidence:  0.5,
			Explanation: "keyword heuristic",
		}, "keyword"),
		Method:  "fallback",
		Latency: time.Since(start),
	}

	if t.gateway == nil {
		if t.keywordFallback {
			result.Result = t.withTrace(t.keywordBasedSuggest(text), "keyword")
			result.Method = "fallback"
		}
		result.Latency = time.Since(start)
		return result, nil
	}

	llmResult, err := t.llmClassify(ctx, title, description)
	if err != nil {
		t.logger.Warn("TriageService: LLM classification failed", zap.Error(err))
		if t.keywordFallback {
			result.Result = t.withTrace(t.keywordBasedSuggest(text), "keyword")
			result.Method = "fallback"
		}
		result.Latency = time.Since(start)
		return result, nil
	}

	result.Result = t.withTrace(llmResult, "llm")
	result.Method = "llm"
	// 真实模型名，不再是硬编码常量：硬编码会让审计底账在换模型之后继续
	// 指向旧模型。取不到时报 "unknown"（EffectiveModel 的语义），不猜。
	result.Model = llmResult.ModelName
	result.Latency = time.Since(start)

	return result, nil
}

// SuggestAssignee suggests an assignee based on ticket content
func (t *TriageService) SuggestAssignee(ctx context.Context, category, title, description string) (int, string, error) {
	if t.gateway != nil {
		prompt := fmt.Sprintf(`Recommend the best handler for this IT ticket:

Category: %s
Title: %s
Description: %s

Choose from:
- Zhang San (101) - Database expert
- Li Si (102) - Network engineer
- Wang Wu (103) - Server admin
- Zhao Liu (104) - Application developer
- Zhou Qi (100) - Security expert
- Chen Ba (105) - Storage expert

Return only the ID number, or 0 if unsure.`, category, title, description)

		messages := []LLMMessage{
			{Role: "system", Content: "You are an IT service management assistant."},
			{Role: "user", Content: prompt},
		}

		resp, err := t.gateway.Chat(ctx, "", messages)
		if err != nil {
			t.logger.Warn("TriageService: assignee suggestion failed", zap.Error(err))
		} else {
			resp = strings.TrimSpace(resp)
			for _, line := range strings.Split(resp, "\n") {
				line = strings.TrimSpace(line)
				if len(line) <= 3 {
					continue
				}
				for i := len(line) - 1; i >= 0; i-- {
					if line[i] < '0' || line[i] > '9' {
						if i < len(line)-1 {
							if id, err := parseAssigneeID(line[i+1:]); err == nil {
								return id, "", nil
							}
						}
						break
					}
				}
			}
		}
	}

	if assigneeID, ok := t.defaultAssignees[category]; ok {
		return assigneeID, "", nil
	}

	return 0, "", nil
}

func parseAssigneeID(s string) (int, error) {
	var id int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			id = id*10 + int(c-'0')
		} else {
			break
		}
	}
	if id == 0 {
		return 0, fmt.Errorf("no valid ID found")
	}
	return id, nil
}
