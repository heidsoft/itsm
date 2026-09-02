package vector_store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"itsm-backend/common"
	connectorVector "itsm-backend/connector/vector"
	"itsm-backend/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler 向量存储HTTP处理器
type Handler struct {
	db     *sql.DB
	logger *zap.SugaredLogger
}

// NewHandler creates a new vector store handler
func NewHandler(db *sql.DB, logger *zap.SugaredLogger) *Handler {
	return &Handler{db: db, logger: logger}
}

// VectorStoreStatusResponse 向量存储状态响应（camelCase 契约）
type VectorStoreStatusResponse struct {
	Configured      bool                   `json:"configured"`
	Source          string                 `json:"source"` // env | default
	Backend         string                 `json:"backend"`
	Collection      string                 `json:"collection"`
	FallbackEnabled bool                   `json:"fallbackEnabled"`
	Capability      string                 `json:"capability"` // ready | degraded | unconfigured | error
	Settings        map[string]interface{} `json:"settings"`   // 已脱敏
	VectorCount     int64                  `json:"vectorCount"`
	CheckedAt       time.Time              `json:"checkedAt"`
	Message         string                 `json:"message,omitempty"`
}

var dsnSecretRe = regexp.MustCompile(`://([^@/\s]*?):[^@\s]*@`)

var bareCredsRe = regexp.MustCompile(`^([A-Za-z0-9_\.\-]+):[^@\s]*@`)

// maskSecret 脱敏连接串中的账号密码
func maskSecret(s string) string {
	if s == "" {
		return s
	}
	masked := dsnSecretRe.ReplaceAllString(s, "://${1}:***@")
	if !strings.Contains(s, "://") {
		masked = bareCredsRe.ReplaceAllString(masked, "${1}:***@")
	}
	return masked
}

// maskSettings 对配置 map 中的敏感键脱敏
func maskSettings(cfg map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(cfg))
	for k, v := range cfg {
		lk := strings.ToLower(k)
		if sv, ok := v.(string); ok {
			switch {
			case strings.Contains(lk, "password"), strings.Contains(lk, "secret"),
				strings.Contains(lk, "token"), strings.Contains(lk, "apikey"),
				strings.Contains(lk, "api_key"):
				out[k] = "***"
				continue
			case strings.Contains(lk, "dsn"), strings.Contains(lk, "uri"), strings.Contains(lk, "addr"):
				out[k] = maskSecret(sv)
				continue
			}
		}
		out[k] = v
	}
	return out
}

// probePrimary 按当前配置构建主后端并 Ping
func probePrimary(ctx context.Context, cfg connectorVector.VectorStoreConfig) (time.Duration, error) {
	if strings.EqualFold(cfg.Backend, "keyword") {
		return 0, nil
	}
	store, err := connectorVector.New(cfg.Backend, cfg.Collection, cfg.Config)
	if err != nil {
		return 0, err
	}
	defer func() { _ = store.Close() }()
	start := time.Now()
	if err := store.Ping(ctx); err != nil {
		return time.Since(start), err
	}
	return time.Since(start), nil
}

// GetStatus 返回当前向量存储配置视图（脱敏）与能力状态
// @Summary 获取向量存储状态
// @Description 获取向量存储配置视图与能力状态
// @Tags 向量存储
// @Produce json
// @Success 200 {object} common.Response
// @Router /api/v1/system/vector-store [get]
func (h *Handler) GetStatus(c *gin.Context) {
	raw := strings.TrimSpace(os.Getenv("VECTOR_STORE_CONFIG"))
	configured := raw != ""

	resp := VectorStoreStatusResponse{
		Configured:  configured,
		Source:      "default",
		Backend:     "keyword",
		Collection:  "knowledge_chunks",
		Capability:  "unconfigured",
		Settings:    map[string]interface{}{},
		VectorCount: -1,
		CheckedAt:   time.Now(),
	}
	if configured {
		resp.Source = "env:VECTOR_STORE_CONFIG"
	}

	cfg, err := connectorVector.LoadConfig(raw)
	if err != nil {
		resp.Capability = "error"
		resp.Message = "向量存储配置解析失败，请检查 VECTOR_STORE_CONFIG 内容"
		h.logger.Warnw("vector store config parse failed", "error", err)
		common.Success(c, resp)
		return
	}
	resp.Backend = cfg.Backend
	resp.Collection = cfg.Collection
	resp.FallbackEnabled = cfg.Fallback
	resp.Settings = maskSettings(cfg.Config)

	probeCtx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	latency, pingErr := probePrimary(probeCtx, cfg)
	switch {
	case !configured:
		resp.Capability = "unconfigured"
		resp.Message = "未配置外部向量库，RAG 使用内置关键字检索回退"
	case pingErr != nil && cfg.Fallback:
		resp.Capability = "degraded"
		resp.Message = fmt.Sprintf("主后端不可用，已回退关键字检索: %s", maskSecret(pingErr.Error()))
	case pingErr != nil:
		resp.Capability = "error"
		resp.Message = fmt.Sprintf("主后端不可用且未启用回退: %s", maskSecret(pingErr.Error()))
	default:
		resp.Capability = "ready"
		resp.Message = fmt.Sprintf("%s 连接正常 (%dms)", cfg.Backend, latency.Milliseconds())
	}

	if h.db != nil {
		countCtx, countCancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer countCancel()
		var count int64
		if err := h.db.QueryRowContext(countCtx, `SELECT COUNT(*) FROM vectors`).Scan(&count); err == nil {
			resp.VectorCount = count
		}
	}

	common.Success(c, resp)
}

// VectorStoreTestResponse 连通性测试响应
type VectorStoreTestResponse struct {
	OK        bool      `json:"ok"`
	Backend   string    `json:"backend"`
	LatencyMs int64     `json:"latencyMs"`
	Fallback  bool      `json:"fallback"`
	Message   string    `json:"message"`
	CheckedAt time.Time `json:"checkedAt"`
}

// TestConnection 对当前配置的向量后端执行一次真实 Ping
// @Summary 测试向量存储连接
// @Description 测试向量存储后端连接
// @Tags 向量存储
// @Produce json
// @Success 200 {object} common.Response
// @Router /api/v1/system/vector-store/test [post]
func (h *Handler) TestConnection(c *gin.Context) {
	cfg, err := connectorVector.LoadConfig(os.Getenv("VECTOR_STORE_CONFIG"))
	if err != nil {
		common.Success(c, VectorStoreTestResponse{
			OK: false, Backend: "unknown", Fallback: false,
			Message: "配置解析失败，请检查 VECTOR_STORE_CONFIG", CheckedAt: time.Now(),
		})
		return
	}

	probeCtx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	latency, pingErr := probePrimary(probeCtx, cfg)
	resp := VectorStoreTestResponse{
		OK: pingErr == nil, Backend: cfg.Backend, Fallback: cfg.Fallback,
		LatencyMs: latency.Milliseconds(), CheckedAt: time.Now(),
	}
	if pingErr != nil {
		resp.Message = maskSecret(pingErr.Error())
		h.logger.Warnw("vector store connectivity test failed",
			"backend", cfg.Backend, "error_class", "connectivity")
	} else {
		resp.Message = "连接正常"
	}
	common.Success(c, resp)
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	root := rg.Group("/system/vector-store")
	root.GET("", middleware.RequirePermission("system", "read"), h.GetStatus)
	root.POST("/test", middleware.RequirePermission("system", "write"), h.TestConnection)
}
