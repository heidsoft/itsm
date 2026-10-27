package global_search

import (
	"context"

	"itsm-backend/ent"
	"itsm-backend/ent/change"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/knowledgearticle"
	"itsm-backend/ent/problem"
	"itsm-backend/ent/ticket"
	"itsm-backend/middleware"
)

const perTypeLimit = 10

// Service owns the tenant-scoped global-search use case.
type Service struct {
	client *ent.Client
}

func NewService(client *ent.Client) *Service { return &Service{client: client} }

// searchDomain 把搜索结果类型映射到该域的读权限码资源名。
//
// 全局搜索是跨域聚合面（工单/事件/问题/变更/知识），访问者只能搜到
// 自己有权读的域——搜索结果会带标题与描述正文，不是有无所谓的元数据。
// 单一源：authz 词表资源名，变更域为 change、知识域为 knowledge。
type searchDomain struct {
	Type     string
	Resource string
}

// searchDomains 固定序展开（不用 map 迭代，避免结果顺序随机）。
var searchDomains = []searchDomain{
	{Type: "ticket", Resource: "ticket"},
	{Type: "incident", Resource: "incident"},
	{Type: "problem", Resource: "problem"},
	{Type: "change", Resource: "change"},
	{Type: "knowledge", Resource: "knowledge"},
}

// allowedDomains 返回访问者有权搜索的域集合。
//
// roles 为全部生效角色（主角色 + 附加角色），判定复用统一入口
// middleware.AuthorizeResource——与路由层 RequirePermission 同一套语义
// （多角色并集、super_admin 直通、DBOnly fail-closed）。
func (s *Service) allowedDomains(ctx context.Context, tenantID int, roles []string) map[string]bool {
	allowed := make(map[string]bool, len(searchDomains))
	for _, d := range searchDomains {
		if middleware.AuthorizeResource(ctx, s.client, roles, d.Resource, "read", tenantID) {
			allowed[d.Type] = true
		}
	}
	return allowed
}

// Search 在调用者有权读的域内搜索。roles 为访问者的全部生效角色。
func (s *Service) Search(ctx context.Context, tenantID int, keyword string, roles []string) (*SearchResponse, error) {
	allowed := s.allowedDomains(ctx, tenantID, roles)
	results := make([]*SearchResult, 0)

	if allowed["ticket"] {
		tickets, err := s.client.Ticket.Query().Where(ticket.TenantID(tenantID), ticket.Or(
			ticket.TitleContainsFold(keyword), ticket.DescriptionContainsFold(keyword), ticket.TicketNumberContainsFold(keyword),
		)).Limit(perTypeLimit).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range tickets {
			results = append(results, &SearchResult{ID: item.ID, Type: "ticket", Title: item.Title, Description: item.Description, Status: item.Status, TicketNumber: item.TicketNumber})
		}
	}

	if allowed["incident"] {
		incidents, err := s.client.Incident.Query().Where(incident.TenantID(tenantID), incident.Or(
			incident.TitleContainsFold(keyword), incident.DescriptionContainsFold(keyword), incident.IncidentNumberContainsFold(keyword),
		)).Limit(perTypeLimit).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range incidents {
			results = append(results, &SearchResult{ID: item.ID, Type: "incident", Title: item.Title, Description: item.Description, Status: item.Status, TicketNumber: item.IncidentNumber})
		}
	}

	if allowed["problem"] {
		problems, err := s.client.Problem.Query().Where(problem.TenantID(tenantID), problem.Or(
			problem.TitleContainsFold(keyword), problem.DescriptionContainsFold(keyword),
		)).Limit(perTypeLimit).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range problems {
			results = append(results, &SearchResult{ID: item.ID, Type: "problem", Title: item.Title, Description: item.Description, Status: item.Status})
		}
	}

	if allowed["change"] {
		changes, err := s.client.Change.Query().Where(change.TenantID(tenantID), change.Or(
			change.TitleContainsFold(keyword), change.DescriptionContainsFold(keyword),
		)).Limit(perTypeLimit).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range changes {
			results = append(results, &SearchResult{ID: item.ID, Type: "change", Title: item.Title, Description: item.Description, Status: string(item.Status)})
		}
	}

	if allowed["knowledge"] {
		// 只搜已发布文章：草稿与评审中的内容尚未对外可见，
		// 搜索结果会带正文片段，等于绕过评审门禁泄出未发布内容。
		articles, err := s.client.KnowledgeArticle.Query().Where(
			knowledgearticle.TenantID(tenantID),
			knowledgearticle.DeletedAtIsNil(),
			knowledgearticle.IsPublished(true),
			knowledgearticle.Or(
				knowledgearticle.TitleContainsFold(keyword), knowledgearticle.ContentContainsFold(keyword),
			),
		).Limit(perTypeLimit).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, item := range articles {
			results = append(results, &SearchResult{ID: item.ID, Type: "knowledge", Title: item.Title, Description: item.Content, Status: "published"})
		}
	}

	return &SearchResponse{Results: results, Total: len(results)}, nil
}
