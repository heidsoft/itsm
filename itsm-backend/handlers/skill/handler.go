// Package skill — Sprint C Skill Registry v1: 技能管理与市场 API.
//
// 本包提供 SkillRegistry 的 HTTP 入口：
//
//   - GET    /api/v1/skills                       列出全部 Skill（市场发现）
//   - GET    /api/v1/skills/:code                 拉取单个 Skill 的可序列化视图
//   - POST   /api/v1/admin/skills                 注册（运行时）自定义 Skill
//   - PUT    /api/v1/admin/skills/:code           更新版本号 / 描述 / 权限
//   - POST   /api/v1/admin/skills/:code/promote   将 Skill 从 pilot 晋升为 ga
//   - DELETE /api/v1/admin/skills/:code           禁用 / 卸载 Skill
//   - POST   /api/v1/skills/:code/invoke          统一调用入口（pilot）
//
// 所有写操作需要 skill:write 权限，所有读操作需要 skill:read。
package skill

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"itsm-backend/common"
	"itsm-backend/middleware"
	"itsm-backend/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 持有 SkillRegistry 与 Service 依赖。
//
// 设计要点：
//   - 仅持有 service.SkillRegistry 引用；运行时注册表本身是 thread-safe，无需额外锁。
//   - 写操作（POST/PUT/DELETE）通过 admin 路径与 skill:write 权限保护；
//     读操作（GET）通过 /skills 与 /skills/:code 路径与 skill:read 权限保护。
//   - 调用入口（POST /:code/invoke）需要单独的 ai:read 权限——因为 Skill 调用时
//     会执行 AI 推理或 RAG，与"列出元数据"权限不同。
type Handler struct {
	registry *service.SkillRegistry
	logger   *zap.SugaredLogger
}

// NewHandler 构造技能管理 Handler。
func NewHandler(registry *service.SkillRegistry, logger *zap.SugaredLogger) *Handler {
	if registry == nil {
		registry = service.NewSkillRegistry()
	}
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Handler{registry: registry, logger: logger}
}

// ----------------------------------------------------------------------------
// DTOs
// ----------------------------------------------------------------------------

// SkillUpsertRequest 注册 / 更新 Skill 的请求体（admin）。
//
// Description / Capabilities / Tags / RequiredPermissions 字段均可选——
// 留空表示保留既有值（PUT 语义）或使用 BaseSkill 默认值（POST 语义）。
type SkillUpsertRequest struct {
	Code                string                 `json:"code" binding:"required"`
	Version             string                 `json:"version"`
	Title               string                 `json:"title"`
	Description         string                 `json:"description"`
	LongDescription     string                 `json:"longDescription"`
	Category            string                 `json:"category"`
	Tags                []string               `json:"tags"`
	Capabilities        []string               `json:"capabilities"`
	RequiredPermissions []string               `json:"requiredPermissions"`
	Provider            string                 `json:"provider"`
	Author              string                 `json:"author"`
	InputSchema         map[string]interface{} `json:"inputSchema"`
	OutputSchema        map[string]interface{} `json:"outputSchema"`
	Executor            map[string]interface{} `json:"executor"`
}

// SkillInvokeRequest 统一调用入口的请求体。
//
// 调用方传入 Skill 执行所需的参数；具体字段由各 Skill 的 manifest.InputSchema 决定。
type SkillInvokeRequest struct {
	Input map[string]interface{} `json:"input"`
}

// SkillInvokeResponse 统一调用入口的响应体，附带延迟与指标追踪状态。
type SkillInvokeResponse struct {
	Code              string      `json:"code"`
	Output            interface{} `json:"output"`
	LatencyMs         int64       `json:"latencyMs"`
	MetricsTracked    bool        `json:"metricsTracked"`
	IsPilot           bool        `json:"isPilot"`
}

// ----------------------------------------------------------------------------
// Routes
// ----------------------------------------------------------------------------

// RegisterRoutes 会把技能管理路由挂到传入的 tenant-scoped router group。
//
// 调用方应在已通过 JWT + tenant 中间件的 group 上调用本方法。
//
// - /api/v1/skills（GET）         → skill:read
// - /api/v1/skills/:code（GET）   → skill:read
// - /api/v1/admin/skills/:code/invoke（POST） → ai:read（便于调用方复用 ai 权限）
// - /api/v1/admin/skills（POST/PUT/DELETE）  → skill:write
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	if rg == nil {
		return
	}

	// 读路径
	markets := rg.Group("/skills")
	markets.Use(middleware.RequirePermission("skill", "read"))
	{
		markets.GET("", h.List)
		markets.GET("/:code", h.Get)
	}

	// 写路径（admin 子路径）
	admin := rg.Group("/admin/skills")
	admin.Use(middleware.RequirePermission("skill", "write"))
	{
		admin.POST("", h.Create)
		admin.PUT("/:code", h.Update)
		admin.POST("/:code/promote", h.Promote)
		admin.DELETE("/:code", h.Delete)
	}

	// 统一调用入口（admin 子路径，沿用 ai:read 权限即可）。
	// 这里放在 admin/skills/... 之下，便于后端审计所有"运行时调用"。
	invoke := rg.Group("/admin/skills")
	invoke.Use(middleware.RequirePermission("ai", "read"))
	{
		invoke.POST("/:code/invoke", h.Invoke)
	}
}

// ----------------------------------------------------------------------------
// Handlers
// ----------------------------------------------------------------------------

// List GET /api/v1/skills
//
// 可选 query：
//   - tag：按 tag 过滤
//   - category：按 category 过滤（ga / pilot / experimental）
//   - status：按 status 过滤（active / disabled）
//   - q：按 code/name/description 模糊匹配
func (h *Handler) List(c *gin.Context) {
	tag := strings.TrimSpace(c.Query("tag"))
	category := strings.TrimSpace(c.Query("category"))
	status := strings.TrimSpace(c.Query("status"))
	q := strings.TrimSpace(strings.ToLower(c.Query("q")))

	entries := h.registry.ListEntries()
	if tag != "" {
		entries = h.registry.ListEntriesByTag(tag)
	}

	filtered := make([]service.SkillEntry, 0, len(entries))
	for _, e := range entries {
		if category != "" && !strings.EqualFold(e.Category, category) {
			continue
		}
		if status != "" && !strings.EqualFold(e.Status, status) {
			continue
		}
		if q != "" {
			hay := strings.ToLower(strings.Join([]string{e.Code, e.Name, e.Description, e.LongDescription}, " "))
			if !strings.Contains(hay, q) {
				continue
			}
		}
		filtered = append(filtered, e)
	}

	common.Success(c, gin.H{
		"items": filtered,
		"total": len(filtered),
	})
}

// Get GET /api/v1/skills/:code
func (h *Handler) Get(c *gin.Context) {
	code := c.Param("code")
	entry, err := h.registry.GetEntry(code)
	if err != nil {
		common.Fail(c, common.NotFoundCode, "skill not found: "+code)
		return
	}
	common.Success(c, entry)
}

// Create POST /api/v1/admin/skills
//
// 行为：
//  1. 校验必填字段（code/version/requiredPermissions）；
//  2. 构造一个最小可用的 CustomSkill（运行时注册，不持久化）；
//  3. 注册到 SkillRegistry；code 冲突时返回 409 ConflictCode。
func (h *Handler) Create(c *gin.Context) {
	var req SkillUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}
	if req.Code == "" {
		common.Fail(c, common.ParamErrorCode, "code is required")
		return
	}
	if req.Version == "" {
		common.Fail(c, common.ParamErrorCode, "version is required")
		return
	}
	if len(req.RequiredPermissions) == 0 {
		common.Fail(c, common.ParamErrorCode, "requiredPermissions must be declared (at least one)")
		return
	}

	category := req.Category
	if category == "" {
		category = "pilot"
	}
	tags := req.Tags
	if len(tags) == 0 {
		tags = []string{"custom"}
	}

	skill := NewCustomSkill(CustomSkillConfig{
		Code:                req.Code,
		Version:             req.Version,
		Title:               req.Title,
		Description:         req.Description,
		LongDescription:     req.LongDescription,
		Category:            category,
		Tags:                tags,
		Capabilities:        req.Capabilities,
		RequiredPermissions: req.RequiredPermissions,
		Provider:            req.Provider,
		Author:              req.Author,
		InputSchema:         req.InputSchema,
		OutputSchema:        req.OutputSchema,
		Executor:            req.Executor,
	})

	if err := h.registry.Register(skill); err != nil {
		if errors.Is(err, service.ErrSkillAlreadyRegistered) {
			common.Fail(c, common.ConflictCode, "skill already registered: "+req.Code)
			return
		}
		common.Fail(c, common.InternalErrorCode, err.Error())
		return
	}
	h.logger.Infow("skill registered", "code", req.Code, "version", req.Version, "category", category)

	entry, err := h.registry.GetEntry(req.Code)
	if err != nil {
		common.Success(c, gin.H{"code": req.Code, "registered": true})
		return
	}
	common.Success(c, entry)
}

// Update PUT /api/v1/admin/skills/:code
//
// 限制：
//   - 只允许更新现有 Skill 的元数据（version / description / category / permissions）。
//   - 不允许修改 Executor：执行器一旦上线就不应在管线外替换，避免审计漂移。
//   - 当前实现：先 Unregister，然后以新元数据重新 Register。
//     未来 Sprint 可改为原地 patch，已注册的 in-flight 调用通过 defer 兼容。
func (h *Handler) Update(c *gin.Context) {
	code := c.Param("code")
	var req SkillUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.ParamErrorCode, err.Error())
		return
	}

	existing, err := h.registry.Get(code)
	if err != nil {
		common.Fail(c, common.NotFoundCode, "skill not found: "+code)
		return
	}

	existingManifest := existing.Manifest()
	newManifest := existingManifest

	if req.Version != "" {
		newManifest.Version = req.Version
	}
	if req.Title != "" {
		newManifest.Title = req.Title
	} else {
		newManifest.Title = existing.Name()
	}
	if req.Description != "" {
		newManifest.Description = req.Description
	}
	if req.LongDescription != "" {
		newManifest.LongDescription = req.LongDescription
	}
	if req.Category != "" {
		newManifest.Category = req.Category
	}
	if req.Provider != "" {
		newManifest.Provider = req.Provider
	}
	if req.Author != "" {
		newManifest.Author = req.Author
	}
	if len(req.RequiredPermissions) > 0 {
		newManifest.RequiredPermissions = req.RequiredPermissions
	}
	if len(req.Capabilities) > 0 {
		newManifest.Capabilities = req.Capabilities
	}
	if req.InputSchema != nil {
		newManifest.InputSchema = req.InputSchema
	}
	if req.OutputSchema != nil {
		newManifest.OutputSchema = req.OutputSchema
	}
	newManifest.Checksum = newManifest.ComputeChecksum()

	// 当前 builtin skill 是只读的；仅允许更新用户自定义 skill。
	// 通过 BaseSkill 上预留的 capabilities 标识识别：自定义 skill 的 capabilities
	// 通常包含 "custom.executor" 前缀（由 CustomSkill 注入）。
	isCustom := false
	for _, c := range newManifest.Capabilities {
		if strings.HasPrefix(c, "custom.") {
			isCustom = true
			break
		}
	}
	if !isCustom {
		common.Fail(c, common.ForbiddenCode, "builtin skills are immutable; create a custom skill instead")
		return
	}

	// 替换：先 Unregister 再以新元数据 Register。
	if err := h.registry.Unregister(code); err != nil {
		common.Fail(c, common.InternalErrorCode, "unregister failed: "+err.Error())
		return
	}
	updated := NewCustomSkill(manifestToConfig(newManifest))
	if err := h.registry.Register(updated); err != nil {
		common.Fail(c, common.InternalErrorCode, "re-register failed: "+err.Error())
		return
	}
	h.logger.Infow("skill updated", "code", code, "version", newManifest.Version)
	common.Success(c, gin.H{"code": code, "updated": true, "manifest": newManifest})
}

// Promote POST /api/v1/admin/skills/:code/promote
//
// 行为：
//   - 将 Skill 的 category 从 pilot 提升为 ga；
//   - 其它元数据保持不变。
//   - 仅允许自定义 skill promote；builtin Skill 视为不可变。
func (h *Handler) Promote(c *gin.Context) {
	code := c.Param("code")
	existing, err := h.registry.Get(code)
	if err != nil {
		common.Fail(c, common.NotFoundCode, "skill not found: "+code)
		return
	}
	m := existing.Manifest()
	if !categoryIsCustom(m) {
		common.Fail(c, common.ForbiddenCode, "builtin skills cannot be promoted")
		return
	}
	if strings.EqualFold(m.Category, "ga") {
		common.Success(c, gin.H{"code": code, "category": m.Category, "note": "already ga"})
		return
	}
	m.Category = "ga"
	m.Checksum = m.ComputeChecksum()
	if err := h.registry.Unregister(code); err != nil {
		common.Fail(c, common.InternalErrorCode, "unregister failed: "+err.Error())
		return
	}
	updated := NewCustomSkill(manifestToConfig(m))
	if err := h.registry.Register(updated); err != nil {
		common.Fail(c, common.InternalErrorCode, "re-register failed: "+err.Error())
		return
	}
	h.logger.Infow("skill promoted", "code", code)
	common.Success(c, gin.H{"code": code, "category": "ga", "promoted": true})
}

// Delete DELETE /api/v1/admin/skills/:code
//
// 行为：
//   - 将 Skill 标记为 disabled（status=disabled）但保留历史 metrics；
//   - 现有 ai_audit / tool_invocations 历史记录不删除。
//   - builtin Skill 拒绝删除。
func (h *Handler) Delete(c *gin.Context) {
	code := c.Param("code")
	existing, err := h.registry.Get(code)
	if err != nil {
		common.Fail(c, common.NotFoundCode, "skill not found: "+code)
		return
	}
	m := existing.Manifest()
	if !categoryIsCustom(m) {
		common.Fail(c, common.ForbiddenCode, "builtin skills cannot be deleted")
		return
	}
	if err := h.registry.Unregister(code); err != nil {
		common.Fail(c, common.InternalErrorCode, "unregister failed: "+err.Error())
		return
	}
	h.logger.Infow("skill disabled", "code", code)
	common.Success(c, gin.H{"code": code, "disabled": true})
}

// Invoke POST /api/v1/admin/skills/:code/invoke
//
// 统一调用入口。
//
// 行为：
//   - 校验 Skill 存在；
//   - 校验输入通过 Skill.Validate；
//   - 调用 SkillRegistry.InvokeWithMetrics，返回 Output + LatencyMs；
//   - 返回的 metricsTracked 标识是否被 BaseSkill 自动累计。
//
// 错误码：
//   - 404：Skill 不存在
//   - 400：参数错误 / Validate 失败
//   - 500：执行失败
func (h *Handler) Invoke(c *gin.Context) {
	code := c.Param("code")
	var req SkillInvokeRequest
	// 调用方可不带 body；此时按 nil 处理。
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			common.Fail(c, common.ParamErrorCode, err.Error())
			return
		}
	}

	input := req.Input
	if input == nil {
		// 兜底：把整张 body 当作 input map（兼容简化调用）。
		// 解析失败时回退到 nil，忽略（由 Skill.Validate 兜底）。
		var body map[string]interface{}
		if err := json.NewDecoder(c.Request.Body).Decode(&body); err == nil {
			input = body
		}
	}
	if input == nil {
		input = map[string]interface{}{}
	}

	// 注入 tenant_id/user_id（若调用方未传）。这两项是大部分 Skill 的必填字段。
	tenantID := c.GetInt("tenant_id")
	userID := c.GetInt("user_id")
	if _, ok := input["tenantId"]; !ok && tenantID != 0 {
		input["tenantId"] = tenantID
	}
	if _, ok := input["userId"]; !ok && userID != 0 {
		input["userId"] = userID
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	result, err := h.registry.InvokeWithMetrics(ctx, code, input)
	if err != nil {
		failInvoke(c, code, err)
		return
	}

	entry, _ := h.registry.GetEntry(code)
	resp := SkillInvokeResponse{
		Code:           code,
		Output:         result.Output,
		LatencyMs:      result.LatencyMs,
		MetricsTracked: !result.SkippedMetrics,
		IsPilot:        strings.EqualFold(entry.Category, "pilot"),
	}
	common.Success(c, resp)
}

// failInvoke 翻译 SkillRegistry 调用错误为合适的 HTTP 响应码。
func failInvoke(c *gin.Context, code string, err error) {
	switch {
	case errors.Is(err, service.ErrSkillNotFound):
		common.Fail(c, common.NotFoundCode, "skill not found: "+code)
	case errors.Is(err, service.ErrSkillValidation):
		common.Fail(c, common.ParamErrorCode, err.Error())
	default:
		// ErrSkillInvoke 与其它内部错误都映射为 500，但保留 err 链路便于排查。
		common.Fail(c, common.InternalErrorCode, err.Error())
	}
}

// categoryIsCustom 检查 manifest 是否来自自定义 Skill（即可变）。
func categoryIsCustom(m service.SkillManifest) bool {
	for _, c := range m.Capabilities {
		if strings.HasPrefix(c, "custom.") {
			return true
		}
	}
	return false
}

// manifestToConfig 把 SkillManifest 转换为 CustomSkillConfig，便于 re-register。
func manifestToConfig(m service.SkillManifest) CustomSkillConfig {
	return CustomSkillConfig{
		Code:                m.Name,
		Version:             m.Version,
		Title:               m.Title,
		Description:         m.Description,
		LongDescription:     m.LongDescription,
		Category:            m.Category,
		Tags:                m.Tags,
		Capabilities:        m.Capabilities,
		RequiredPermissions: m.RequiredPermissions,
		Provider:            m.Provider,
		Author:              m.Author,
		InputSchema:         asMap(m.InputSchema),
		OutputSchema:        asMap(m.OutputSchema),
		Executor:            nil,
	}
}

// asMap 接受 interface{}（多为 map[string]any），强转为 map[string]interface{}。
// 解析失败时返回 nil，由上游落空。
func asMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	if m, ok := v.(map[string]any); ok {
		out := make(map[string]interface{}, len(m))
		for k, val := range m {
			out[k] = val
		}
		return out
	}
	return nil
}
