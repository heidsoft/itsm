package authz

// 权限策略模型：权限码的**运行时**权威定义与 AI 时代的主体边界。
//
// 设计约束（2026-11-14 架构批次）：
//  1. 权限码只在 DB 的 permissions 行里定义一次；Go 侧的 Definitions() 降级为
//     「全新库的一次性引导清单」，任何运行时鉴权路径都不许读它（有守卫锁定）。
//  2. 元数据不新建表、不改 ent schema：以版本化 JSON 信封存进既有的
//     permissions.description。原因是本仓库的 ent 生成器与已提交产物存在版本漂移
//     （重跑会重写 382 个生成文件），在共享工作树里做 schema 变更会污染并行会话。
//     信封是向后兼容的：存量纯文本描述按「描述 + 派生默认策略」解析。
//  3. 风险分级由 action 语义**按规则推导**，不是再维护一张逐码字典；
//     管理员可以在 DB 里逐码覆盖，推导只是零配置时的安全初值。
//  4. AI 默认零权限：派生策略的 AllowedPrincipals 恒为 ["human"]，
//     必须由管理员在运行期显式把 "agent" 加进某个码，AI 才可能执行它。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// principalContextKey 是「这次操作由谁发起」的请求级身份。
//
// 默认 human：人点按钮、人发 HTTP 请求都走既有 RBAC 链路。
// LLM 自主决定调用工具时由 ChatStream / skill 执行路径显式标记 agent，
// 从而进入「DB 运行期策略 + 零权限默认 + 风险分级人工审批」链路。
//
// 刻意不放在 JWT claim 里：同一位登录用户既可能自己点按钮、也可能让 AI 代做，
// 主体类型描述的是**这一次动作的发起者**，不是账号的类型。
type principalContextKey struct{}

// WithPrincipal 标记本次操作的主体类型。
func WithPrincipal(ctx context.Context, principal string) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

// PrincipalFromContext 取主体类型，缺失时按 human 处理（保持既有链路不变）。
func PrincipalFromContext(ctx context.Context) string {
	if ctx == nil {
		return PrincipalHuman
	}
	if v, ok := ctx.Value(principalContextKey{}).(string); ok && v != "" {
		return v
	}
	return PrincipalHuman
}

// 主体类型。human = 登录用户；agent = AI/自动化代理；service = 系统间服务账号。
const (
	PrincipalHuman   = "human"
	PrincipalAgent   = "agent"
	PrincipalService = "service"
)

// AllPrincipals 是允许出现在 AllowedPrincipals 里的闭集。
var AllPrincipals = []string{PrincipalHuman, PrincipalAgent, PrincipalService}

// 风险等级。
const (
	RiskLow      = "low"
	RiskMedium   = "medium"
	RiskHigh     = "high"
	RiskCritical = "critical"
)

// AllRiskLevels 是允许出现在 RiskLevel 里的闭集（按严重度升序）。
var AllRiskLevels = []string{RiskLow, RiskMedium, RiskHigh, RiskCritical}

// Category 分类键。前端用键做 i18n，不用中文标签做过滤键（翻译后过滤会失效）。
const (
	CategoryServiceOps    = "serviceOps"
	CategoryIdentity      = "identity"
	CategoryPlatform      = "platform"
	CategoryAssets        = "assets"
	CategoryAnalytics     = "analytics"
	CategoryAIExtensions  = "aiExtensions"
	CategoryCollaboration = "collaboration"
	CategoryOther         = "other"
)

// AllCategories 是允许出现在 Category 里的闭集。
var AllCategories = []string{
	CategoryServiceOps, CategoryIdentity, CategoryPlatform, CategoryAssets,
	CategoryAnalytics, CategoryAIExtensions, CategoryCollaboration, CategoryOther,
}

// categoryLabels 是给前端兜底展示用的中文标签；不是权限码定义，不参与匹配。
var categoryLabels = map[string]string{
	CategoryServiceOps:    "服务运营",
	CategoryIdentity:      "组织与权限",
	CategoryPlatform:      "平台配置",
	CategoryAssets:        "资产与主数据",
	CategoryAnalytics:     "报表与分析",
	CategoryAIExtensions:  "AI 与扩展",
	CategoryCollaboration: "协作与支持",
	CategoryOther:         "其他",
}

// CategoryLabel 返回分类键的兜底展示名。
func CategoryLabel(key string) string {
	if label, ok := categoryLabels[key]; ok {
		return label
	}
	return categoryLabels[CategoryOther]
}

// PermissionPolicy 是权限码的运行期可配策略。管理员通过
// PUT /api/v1/permissions/:id/policy 修改，变更后立即对鉴权与 AI 执行链生效。
type PermissionPolicy struct {
	// RiskLevel 决定 AI 执行该操作时需要的人类确认强度。
	RiskLevel string `json:"riskLevel"`
	// AllowedPrincipals 允许执行该操作的主体类型。默认只有 human = AI 零权限。
	AllowedPrincipals []string `json:"allowedPrincipals"`
	// RequiresApproval 为 true 时，agent 执行该操作必须创建待人工确认的调用。
	RequiresApproval bool `json:"requiresApproval"`
	// MinApprovals 需要多少个**互不相同**的人类审批人（1 或 2）。
	MinApprovals int `json:"minApprovals"`
	// Category / DisplayName / Icon / Color / I18nKey / SortOrder 只影响展示。
	Category    string `json:"category"`
	DisplayName string `json:"displayName,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Color       string `json:"color,omitempty"`
	I18nKey     string `json:"i18nKey,omitempty"`
	SortOrder   int    `json:"sortOrder"`
	// IsActive 关闭后该码对鉴权不可见（保留行以维持历史 role_permissions 引用）。
	IsActive bool `json:"isActive"`
	// IsSystem 标记引导清单自带的码：不允许删除、不允许改 code/resource/action。
	IsSystem bool `json:"isSystem"`
}

// definitionEnvelope 是 permissions.description 的存储格式。
// v=1 是唯一受支持的版本；存量纯文本没有 v 字段，由 DecodeDefinition 兼容。
type definitionEnvelope struct {
	Version     int              `json:"v"`
	Description string           `json:"description,omitempty"`
	Policy      PermissionPolicy `json:"policy"`
}

const definitionEnvelopeVersion = 1

// readOnlyActions 是不产生副作用的动作。只用于**派生**初始风险，
// 不是权限码白名单：管理员仍可在 DB 里把任意码改成任意风险等级。
var readOnlyActions = map[string]bool{
	"read": true, "view": true, "list": true, "get": true, "search": true,
}

// criticalActions 会造成不可逆的数据/权限损失，或触发外部真实资源交付。
var criticalActions = map[string]bool{
	"delete": true, "admin": true, "all": true, "provision": true,
	"override": true, "force-update": true,
}

// highActions 改变流程裁决结果或回滚既有状态，需要双人确认。
var highActions = map[string]bool{
	"approve": true, "rollback": true, "escalate": true, "archive": true,
}

// RiskForAction 按动作语义推导风险等级。这是**规则**而不是逐码表，
// 新增权限码时无需同步任何字典。
func RiskForAction(action string) string {
	a := strings.ToLower(strings.TrimSpace(action))
	switch {
	case readOnlyActions[a]:
		return RiskLow
	case criticalActions[a]:
		return RiskCritical
	case highActions[a]:
		return RiskHigh
	default:
		// 未知动作一律按中风险处理：宁可多一道审批，也不能把新动作
		// 静默归到 low 让 AI 无确认执行。
		return RiskMedium
	}
}

// MinApprovalsFor 返回该风险等级下的最小人类审批人数。
func MinApprovalsFor(risk string) int {
	switch risk {
	case RiskLow:
		return 0
	case RiskMedium:
		return 1
	case RiskHigh, RiskCritical:
		return 2
	default:
		return 2
	}
}

// CategoryForResource 按资源前缀推导 UI 分类键，同样是规则而不是逐码表。
func CategoryForResource(resource string) string {
	r := strings.ToLower(strings.TrimSpace(resource))
	switch {
	case r == "":
		return CategoryOther
	case strings.HasPrefix(r, "ticket"),
		strings.HasPrefix(r, "incident"),
		strings.HasPrefix(r, "problem"),
		strings.HasPrefix(r, "change"),
		strings.HasPrefix(r, "release"),
		strings.HasPrefix(r, "sla"),
		strings.HasPrefix(r, "task"),
		strings.HasPrefix(r, "service_request"),
		strings.HasPrefix(r, "service_catalog"),
		strings.HasPrefix(r, "service"),
		strings.HasPrefix(r, "on_call"),
		strings.HasPrefix(r, "survey"):
		return CategoryServiceOps
	case strings.HasPrefix(r, "user"),
		strings.HasPrefix(r, "role"),
		strings.HasPrefix(r, "group"),
		strings.HasPrefix(r, "permission"),
		strings.HasPrefix(r, "department"),
		strings.HasPrefix(r, "team"),
		strings.HasPrefix(r, "org"),
		strings.HasPrefix(r, "tenant"),
		strings.HasPrefix(r, "sso"),
		strings.HasPrefix(r, "msp"):
		return CategoryIdentity
	case strings.HasPrefix(r, "system"),
		strings.HasPrefix(r, "audit"),
		strings.HasPrefix(r, "bpmn"),
		strings.HasPrefix(r, "workflow"),
		strings.HasPrefix(r, "automation_rule"),
		strings.HasPrefix(r, "assignment_rule"),
		strings.HasPrefix(r, "notification"),
		strings.HasPrefix(r, "email_intake"):
		return CategoryPlatform
	case strings.HasPrefix(r, "cmdb"),
		strings.HasPrefix(r, "asset"),
		strings.HasPrefix(r, "license"),
		strings.HasPrefix(r, "vendor"),
		strings.HasPrefix(r, "support_contract"),
		strings.HasPrefix(r, "customer_master"),
		strings.HasPrefix(r, "ci_relationship"):
		return CategoryAssets
	case strings.HasPrefix(r, "report"),
		strings.HasPrefix(r, "dashboard"),
		strings.HasPrefix(r, "alert"):
		return CategoryAnalytics
	case strings.HasPrefix(r, "ai"),
		strings.HasPrefix(r, "connector"),
		strings.HasPrefix(r, "marketplace"),
		strings.HasPrefix(r, "skill"):
		return CategoryAIExtensions
	case strings.HasPrefix(r, "knowledge"),
		strings.HasPrefix(r, "project"),
		strings.HasPrefix(r, "application"):
		return CategoryCollaboration
	default:
		return CategoryOther
	}
}

// DerivePolicy 生成零配置时的安全初值。
//
// 关键不变量：AllowedPrincipals 恒为 ["human"]。也就是说任何新权限码、
// 任何存量纯文本行，AI 都拿不到；要授权给 AI 必须由管理员显式改 DB。
func DerivePolicy(resource, action string) PermissionPolicy {
	risk := RiskForAction(action)
	return PermissionPolicy{
		RiskLevel:         risk,
		AllowedPrincipals: []string{PrincipalHuman},
		RequiresApproval:  risk != RiskLow,
		MinApprovals:      MinApprovalsFor(risk),
		Category:          CategoryForResource(resource),
		IsActive:          true,
		IsSystem:          true,
	}
}

// Allows 判断该策略是否允许某主体类型执行。
func (p PermissionPolicy) Allows(principal string) bool {
	if !p.IsActive {
		return false
	}
	for _, allowed := range p.AllowedPrincipals {
		if allowed == principal {
			return true
		}
	}
	return false
}

// ApprovalFor 返回该主体执行时需要的人类确认强度。
// human 主体不引入额外审批（保留既有工具级写审批语义），
// agent/service 主体的任何非低风险操作都必须由人确认。
func (p PermissionPolicy) ApprovalFor(principal string) (needsApproval bool, minApprovals int) {
	if principal == PrincipalHuman {
		return false, 0
	}
	if !p.RequiresApproval {
		return false, 0
	}
	min := p.MinApprovals
	if min < 1 {
		min = 1
	}
	return true, min
}

// ErrPolicyInvalid 表示权限运行期策略违反闭集安全不变量。
//
// 策略来自管理员的显式配置，违反不变量属于**调用方输入错误**：HTTP 层必须
// 映射为参数错误（400/1001），而不是内部错误（500/5001）——否则管理端无法
// 区分「填错策略」与「服务故障」，也违反错误语义契约。
var ErrPolicyInvalid = errors.New("invalid permission policy")

// ValidatePolicy 在写入 DB 前、以及从 DB 读出后都做闭集校验。
// 安全不变量在这里强制，而不是指望调用方自觉：
//   - 允许 agent 的非低风险操作不能关掉审批；
//   - 允许 agent 的高/危急风险操作必须双人审批；
//   - 低风险操作不得要求审批（避免无意义的待办堆积）。
//
// 所有违规都包装 ErrPolicyInvalid，供 handler 用 errors.Is 分流。
func ValidatePolicy(p PermissionPolicy) error {
	if err := validatePolicyRules(p); err != nil {
		return fmt.Errorf("%w: %v", ErrPolicyInvalid, err)
	}
	return nil
}

func validatePolicyRules(p PermissionPolicy) error {
	if !containsString(AllRiskLevels, p.RiskLevel) {
		return fmt.Errorf("riskLevel %q 非法，仅支持 %s", p.RiskLevel, strings.Join(AllRiskLevels, "/"))
	}
	if !containsString(AllCategories, p.Category) {
		return fmt.Errorf("category %q 非法，仅支持 %s", p.Category, strings.Join(AllCategories, "/"))
	}
	if len(p.AllowedPrincipals) == 0 {
		return fmt.Errorf("allowedPrincipals 不能为空")
	}
	seen := make(map[string]bool, len(p.AllowedPrincipals))
	for _, principal := range p.AllowedPrincipals {
		if !containsString(AllPrincipals, principal) {
			return fmt.Errorf("principal %q 非法，仅支持 %s", principal, strings.Join(AllPrincipals, "/"))
		}
		if seen[principal] {
			return fmt.Errorf("allowedPrincipals 存在重复项 %q", principal)
		}
		seen[principal] = true
	}
	if p.MinApprovals < 0 || p.MinApprovals > 2 {
		return fmt.Errorf("minApprovals 必须在 0..2 之间，当前 %d", p.MinApprovals)
	}
	if p.RiskLevel == RiskLow && p.RequiresApproval {
		return fmt.Errorf("低风险权限不得强制人工审批")
	}
	agentAllowed := seen[PrincipalAgent] || seen[PrincipalService]
	if agentAllowed && p.RiskLevel != RiskLow && !p.RequiresApproval {
		return fmt.Errorf("允许 AI 执行 %s 风险操作时必须开启人工审批", p.RiskLevel)
	}
	if agentAllowed && (p.RiskLevel == RiskHigh || p.RiskLevel == RiskCritical) && p.MinApprovals < 2 {
		return fmt.Errorf("允许 AI 执行 %s 风险操作时必须配置双人审批", p.RiskLevel)
	}
	return nil
}

func containsString(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// EncodeDefinition 把人类描述与策略打包成写入 permissions.description 的信封。
func EncodeDefinition(description string, policy PermissionPolicy) (string, error) {
	if err := ValidatePolicy(policy); err != nil {
		return "", fmt.Errorf("invalid permission policy: %w", err)
	}
	// 规范化排序，保证同样的策略在 DB 里总是同一串字节（便于 diff / 幂等 seed）。
	policy.AllowedPrincipals = append([]string(nil), policy.AllowedPrincipals...)
	sort.Strings(policy.AllowedPrincipals)
	buf, err := json.Marshal(definitionEnvelope{
		Version:     definitionEnvelopeVersion,
		Description: description,
		Policy:      policy,
	})
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// DecodeDefinition 读取 permissions.description。
//
// 兼容三种历史形态：
//   - 空串        → 无描述 + 派生默认策略；
//   - 纯文本      → 旧版描述 + 派生默认策略（AI 仍为零权限，fail-closed）；
//   - v=1 JSON    → 完整运行期策略。
//
// 未知版本 / 损坏 JSON / 非法策略一律返回错误，由 Registry 走 fail-closed，
// 绝不「解析失败就按宽松默认值放行」。
func DecodeDefinition(raw, resource, action string) (string, PermissionPolicy, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return trimmed, DerivePolicy(resource, action), nil
	}
	var env definitionEnvelope
	if err := json.Unmarshal([]byte(trimmed), &env); err != nil {
		return "", PermissionPolicy{}, fmt.Errorf("权限定义信封不是合法 JSON: %w", err)
	}
	if env.Version != definitionEnvelopeVersion {
		return "", PermissionPolicy{}, fmt.Errorf("不支持的权限定义信封版本 v=%d", env.Version)
	}
	if err := ValidatePolicy(env.Policy); err != nil {
		return "", PermissionPolicy{}, fmt.Errorf("权限定义策略非法: %w", err)
	}
	return env.Description, env.Policy, nil
}

// IsDefinitionEnvelope 判断一行是否已经是运行期策略信封。
// seeder 用它区分「管理员配置过的行」与「待迁移的存量纯文本行」：
// 前者重启时必须原样保留，后者只迁移一次。
func IsDefinitionEnvelope(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if !strings.HasPrefix(trimmed, "{") {
		return false
	}
	var env definitionEnvelope
	if err := json.Unmarshal([]byte(trimmed), &env); err != nil {
		return false
	}
	return env.Version == definitionEnvelopeVersion && ValidatePolicy(env.Policy) == nil
}

// DecodeDefinitionLenient 只给列表/导出等读面使用：解析失败时降级为派生默认策略，
// 但**绝不**用于任何鉴权判定（鉴权必须走 Registry / DecodeDefinition 严格路径）。
func DecodeDefinitionLenient(raw, resource, action string) (string, PermissionPolicy) {
	desc, policy, err := DecodeDefinition(raw, resource, action)
	if err != nil {
		return raw, DerivePolicy(resource, action)
	}
	return desc, policy
}
