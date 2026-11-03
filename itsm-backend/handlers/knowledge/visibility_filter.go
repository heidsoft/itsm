package knowledge

import (
	"context"

	"itsm-backend/internal/knowledgeaccess"
)

// FilterVisibleArticles 返回当前访问者可读的文章子集，保持原顺序。
//
// 跨域反查（问题沉淀知识等）必须复用本函数，否则会绕开知识分类纳管。
// guard 为 nil 时保持旧行为；查询失败由 CanReadCategory 按 fail-closed 拒绝。
func FilterVisibleArticles(ctx context.Context, guard *knowledgeaccess.Guard, tenantID int, articles []*Article) []*Article {
	if guard == nil || len(articles) == 0 {
		return articles
	}
	viewer, _ := knowledgeaccess.ViewerFrom(ctx)
	kept := make([]*Article, 0, len(articles))
	for _, a := range articles {
		if a == nil {
			continue
		}
		if guard.CanReadCategory(ctx, tenantID, viewer, a.Category, a.AuthorID) {
			kept = append(kept, a)
		}
	}
	return kept
}
