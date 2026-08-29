// Package knowledgeaccess 提供知识库检索的可见性守卫（知识可引用性 L0 权限边界）。
//
// # 背景
//
// RAG 检索此前只按 tenant_id + is_published + deleted_at 过滤，同租户内任何用户
// 都能通过 AI 助手问出本不该看的知识（财务制度、HR 薪酬、高管会议纪要等）。
// 这是「企业内部知识可引用性」的地基：没有它，上层做的分块优化、摘要、
// 问答对（即 GEO 手法）等于给越权内容做了更好的曝光。
//
// # 设计约束与选型
//
// 本项目的 ent 代码生成（entc）在 16GB 环境下必然 OOM，无法给 knowledgearticle
// 新增 visibility/allowed_role_ids 字段。参照 datascope「纯代码层 + 既有字段」
// 的既有模式，此处复用 RBAC 的 permission 表承载分类级可见性：
//
//	resource = "knowledge_category"
//	action   = "read:<分类名>"
//
// permission.resource / permission.action 均为自由字符串，管理员在后台为某个
// 分类配置一条权限记录即完成纳管（opt-in），无需任何 schema 变更或代码生成。
//
// # 安全语义
//
//   - 未纳管的分类：全租户可读（保持向后兼容，现有知识不受影响）
//   - 已纳管的分类：必须持有对应权限才可读；判定失败一律拒绝（fail-closed）
//   - 作者本人：可读自己创作的文章，即使属于受限分类
//   - super_admin / sysadmin：由 HasResourcePermission 内部直通
//   - 上下文缺失（调用方未注入 Viewer）：按匿名处理，仅放行未纳管分类
package knowledgeaccess

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/permission"
	"itsm-backend/middleware"

	"go.uber.org/zap"
)

// CategoryResource 知识分类可见性在 RBAC 中的资源名。
const CategoryResource = "knowledge_category"

// ActionPrefix 知识分类读权限的 action 前缀，完整 action = ActionPrefix + 分类名。
const ActionPrefix = "read:"

// ActionForCategory 生成某分类对应的读权限 action。
func ActionForCategory(category string) string {
	return ActionPrefix + category
}

// Viewer 知识检索的访问者身份。经 context 传递到检索层，
// 避免为传递身份而改动整条 RAG 调用链的签名。
type Viewer struct {
	UserID int
	Role   string
}

type viewerKey struct{}

// WithViewer 把访问者身份注入 context。
func WithViewer(ctx context.Context, v Viewer) context.Context {
	return context.WithValue(ctx, viewerKey{}, v)
}

// ViewerFrom 从 context 取出访问者身份。
// 返回 false 表示调用方未注入，此时调用方必须按匿名（最低权限）处理。
func ViewerFrom(ctx context.Context) (Viewer, bool) {
	v, ok := ctx.Value(viewerKey{}).(Viewer)
	return v, ok
}

// errNoClient 守卫未装配 ent 客户端。
var errNoClient = errors.New("knowledgeaccess: ent client not initialized")

// Guard 知识分类可见性守卫。
type Guard struct {
	client *ent.Client
	logger *zap.SugaredLogger

	mu      sync.RWMutex
	cached  map[int]categorySnapshot // key: tenantID
	ttl     time.Duration
	nowFunc func() time.Time
}

type categorySnapshot struct {
	restricted map[string]bool // 该租户已被纳管（受限）的分类集合
	expiresAt  time.Time
}

// NewGuard 创建守卫。logger 可为 nil。
func NewGuard(client *ent.Client, logger *zap.SugaredLogger) *Guard {
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	return &Guard{
		client:  client,
		logger:  logger,
		cached:  make(map[int]categorySnapshot),
		ttl:     30 * time.Second,
		nowFunc: time.Now,
	}
}

// RestrictedCategories 返回该租户已被纳管的分类集合。
// 通过一次查询拿到全部受限分类，避免每篇文章都查一次权限表。
// 结果带短 TTL 缓存：纳管配置变更频率极低，而 RAG 检索 QPS 相对较高。
func (g *Guard) RestrictedCategories(ctx context.Context, tenantID int) (map[string]bool, error) {
	if g.client == nil {
		return nil, errNoClient
	}

	if snap, ok := g.readCache(tenantID); ok {
		return snap, nil
	}

	// 只按 resource 过滤：ent 未为 action 生成 HasPrefix 谓词（schema 未开该特性），
	// 且该 resource 下记录数极少，前缀判断放在 Go 侧更直接也更可靠。
	perms, err := g.client.Permission.Query().
		Where(
			permission.TenantIDEQ(tenantID),
			permission.ResourceEQ(CategoryResource),
		).
		All(ctx)
	if err != nil {
		// 查询失败时 fail-closed：返回空集合会让受限分类被当作公开放行，
		// 因此这里必须向上抛错，由调用方决定拒绝还是降级。
		return nil, err
	}

	restricted := make(map[string]bool, len(perms))
	for _, p := range perms {
		if !strings.HasPrefix(p.Action, ActionPrefix) {
			continue
		}
		cat := strings.TrimSpace(p.Action[len(ActionPrefix):])
		if cat != "" {
			restricted[cat] = true
		}
	}

	g.writeCache(tenantID, restricted)
	return restricted, nil
}

// CanReadCategory 判定访问者是否可读某分类下的知识。
//
// authorID > 0 且等于访问者 UserID 时放行（作者可读自己的文章）。
// 分类未被纳管时放行（向后兼容）。
// 分类已纳管但无权限、或判定过程出错时拒绝（fail-closed）。
func (g *Guard) CanReadCategory(ctx context.Context, tenantID int, v Viewer, category string, authorID int) bool {
	// 作者本人豁免：即便落在受限分类，作者也能读自己写的文章
	if authorID > 0 && v.UserID > 0 && authorID == v.UserID {
		return true
	}

	if category == "" {
		// 无分类的文章不参与分类级管控
		return true
	}

	restricted, err := g.RestrictedCategories(ctx, tenantID)
	if err != nil {
		g.logger.Warnw("knowledgeaccess: 受限分类查询失败，按 fail-closed 拒绝",
			"tenant_id", tenantID, "category", category, "error", err)
		return false
	}

	if !restricted[category] {
		return true // 未纳管，全租户可读
	}

	// 已纳管：必须持有 knowledge_category:read:<cat> 权限
	if v.Role == "" {
		g.logger.Debugw("knowledgeaccess: 受限分类访问被拒（无角色信息）",
			"tenant_id", tenantID, "category", category, "user_id", v.UserID)
		return false
	}
	ok := middleware.HasResourcePermission(ctx, g.client, v.Role, CategoryResource, ActionForCategory(category), tenantID)
	if !ok {
		g.logger.Debugw("knowledgeaccess: 受限分类访问被拒（权限不足）",
			"tenant_id", tenantID, "category", category, "role", v.Role, "user_id", v.UserID)
	}
	return ok
}

// FilterArticles 批量过滤文章，返回可访问的子集，保持原顺序。
// 用于检索结果返回前的统一收敛。
func (g *Guard) FilterArticles(ctx context.Context, tenantID int, v Viewer, articles []*ent.KnowledgeArticle) ([]*ent.KnowledgeArticle, int) {
	if len(articles) == 0 {
		return articles, 0
	}
	kept := make([]*ent.KnowledgeArticle, 0, len(articles))
	for _, a := range articles {
		if a == nil {
			continue
		}
		if g.CanReadCategory(ctx, tenantID, v, a.Category, a.AuthorID) {
			kept = append(kept, a)
		}
	}
	return kept, len(articles) - len(kept)
}

// Invalidate 清除某租户的受限分类缓存（纳管配置变更后调用）。
func (g *Guard) Invalidate(tenantID int) {
	g.mu.Lock()
	delete(g.cached, tenantID)
	g.mu.Unlock()
}

func (g *Guard) readCache(tenantID int) (map[string]bool, bool) {
	g.mu.RLock()
	snap, ok := g.cached[tenantID]
	g.mu.RUnlock()
	if !ok || g.nowFunc().After(snap.expiresAt) {
		return nil, false
	}
	return snap.restricted, true
}

func (g *Guard) writeCache(tenantID int, restricted map[string]bool) {
	g.mu.Lock()
	g.cached[tenantID] = categorySnapshot{
		restricted: restricted,
		expiresAt:  g.nowFunc().Add(g.ttl),
	}
	g.mu.Unlock()
}
