package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"itsm-backend/service"
)

// CustomSkill 是 SkillRegistry 在运行期注册用户自定义 Skill 的统一容器。
//
// 它持有：
//   - SkillManifest（来自配置 / DTO）
//   - Optional Executor 字典（用户声明的"如何运行"配置；本期仅作为元数据记录，
//     真正的执行仍由 SkillRegistry.Invoke 转发到具体的 Execute 实现）
//
// 重要约束：自定义 Skill 必须配置 requiredPermissions 中至少一项，否则
// SkillManifest.ValidateForRegistration 会拒绝 Register。
//
// 本期范围：
//   - 用户通过 POST /api/v1/admin/skills 注册自定义 Skill；
//   - CustomSkill 在 Registry 中可查询、可 Promote（pilot→ga）、可 Disable；
//   - Execute() 阶段做以下事情：
//     1) 校验 executor 是否声明 target（"echo" / "noop" / "static"）；
//     2) 按照 target 类型返回约定结果（echo: 原样返回 input；noop: nil；static: 配置里的常量）；
//     3) 其它 target 视为配置错误，返回错误。
type CustomSkill struct {
	*service.BaseSkill
	manifest   service.SkillManifest
	executor   map[string]interface{}
	executorOK bool
}

// CustomSkillConfig 构造 CustomSkill 的输入。
type CustomSkillConfig struct {
	Code                string
	Version             string
	Title               string
	Description         string
	LongDescription     string
	Category            string
	Tags                []string
	Capabilities        []string
	RequiredPermissions []string
	Provider            string
	Author              string
	InputSchema         map[string]interface{}
	OutputSchema        map[string]interface{}
	Executor            map[string]interface{}
}

// NewCustomSkill 构造 CustomSkill。
func NewCustomSkill(cfg CustomSkillConfig) *CustomSkill {
	tags := cfg.Tags
	if len(tags) == 0 {
		tags = []string{"custom"}
	}
	caps := append([]string(nil), cfg.Capabilities...)
	// 强制注入 "custom.executor" 标识，让 Update / Promote / Delete 识别该 Skill 为可变。
	caps = appendIfMissing(caps, "custom.executor")

	perms := cfg.RequiredPermissions
	if len(perms) == 0 {
		perms = []string{"skill:read"}
	}

	category := cfg.Category
	if category == "" {
		category = "pilot"
	}
	provider := cfg.Provider
	if provider == "" {
		provider = "user"
	}
	author := cfg.Author
	if author == "" {
		author = "custom"
	}
	title := cfg.Title
	if title == "" {
		title = cfg.Code
	}
	description := cfg.Description
	if description == "" {
		description = "用户自定义 Skill"
	}

	b := service.NewBaseSkill(
		cfg.Code,
		title,
		cfg.Version,
		category,
		tags,
		perms,
		caps,
	).WithProvider(provider).
		WithAuthor(author).
		WithDescription(description).
		WithLongDescription(cfg.LongDescription)

	m := b.BuildManifest()
	if cfg.InputSchema != nil {
		m.InputSchema = cfg.InputSchema
	}
	if cfg.OutputSchema != nil {
		m.OutputSchema = cfg.OutputSchema
	}
	m.Checksum = m.ComputeChecksum()

	return &CustomSkill{
		BaseSkill:  b,
		manifest:   m,
		executor:   cfg.Executor,
		executorOK: cfg.Executor != nil,
	}
}

// Manifest 返回 SkillManifest。
func (s *CustomSkill) Manifest() service.SkillManifest {
	return s.manifest
}

// Validate 校验输入：仅检查 input 是 JSON object；深入字段由 executor 自行校验。
func (s *CustomSkill) Validate(input interface{}) error {
	if input == nil {
		return nil
	}
	switch input.(type) {
	case map[string]interface{}:
		return nil
	default:
		// 尝试 JSON 公约
		raw, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("custom skill input is not JSON-serializable: %w", err)
		}
		var probe map[string]interface{}
		if err := json.Unmarshal(raw, &probe); err != nil {
			return fmt.Errorf("custom skill input must be a JSON object: %w", err)
		}
		return nil
	}
}

// Execute 按 executor.target 路由：
//   - echo：返回 { echoed: <input> }
//   - noop：返回 { ok: true }
//   - static：返回 { constants: <executor.constants> }（注意 constants 必须是 JSON object）
//   - 其它：返回错误（fail closed）
func (s *CustomSkill) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	if !s.executorOK {
		return map[string]interface{}{"ok": true, "input": input, "note": "no executor configured"}, nil
	}
	// 兼容 context
	_ = ctx

	targetRaw, ok := s.executor["target"]
	if !ok {
		return nil, fmt.Errorf("custom skill executor.target is required")
	}
	target, ok := targetRaw.(string)
	if !ok {
		return nil, fmt.Errorf("custom skill executor.target must be a string")
	}
	target = strings.ToLower(strings.TrimSpace(target))

	switch target {
	case "echo":
		return map[string]interface{}{"echoed": input}, nil
	case "noop":
		return map[string]interface{}{"ok": true}, nil
	case "static":
		constants, _ := s.executor["constants"].(map[string]interface{})
		return map[string]interface{}{"constants": constants}, nil
	default:
		return nil, fmt.Errorf("custom skill executor.target=%q is not supported in this Sprint", target)
	}
}

func appendIfMissing(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}
