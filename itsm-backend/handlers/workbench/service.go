package workbench

import (
	"context"

	"itsm-backend/ent"
	"itsm-backend/middleware"
)

// Service provides workbench business logic.
type Service interface {
	// Query returns paginated workbench items for the current user.
	Query(ctx context.Context, query WorkbenchQuery) (*WorkbenchResponse, error)
}

type serviceImpl struct {
	repo Repository
	// client 用于域级权限判定（middleware.AuthorizeResource 需要 ent 客户端读
	// role_permissions）。nil 表示未装配权限判定——此时不做域级过滤，
	// 与装配前的历史行为一致，但生产装配必须注入。
	client *ent.Client
}

// NewService creates a new workbench service.
func NewService(repo Repository) Service {
	return &serviceImpl{repo: repo}
}

// NewServiceWithClient 装配带权限判定的 workbench service。
func NewServiceWithClient(repo Repository, client *ent.Client) Service {
	return &serviceImpl{repo: repo, client: client}
}

// domainPermission 把工作台的 recordType 映射到该域的读权限码。
//
// 工作台是跨域聚合面（工单/事件/问题/变更），路由层只挂 dashboard:read——
// 仪表盘读权限不等于各业务域的读权限。没有 incident:read 的用户此前仍能在
// 工作台看到事件条目，这就是本函数要堵的口子。
//
// 单一源：authz 词表（internal/authz/catalog.go 的资源名），变更域权限码为
// change:read，与其他域同构（<resource>:read）。
//
// 有序切片而非 map：未指定 recordType 时按此顺序展开，map 迭代顺序随机会让
// 同一请求两次返回不同域序，前端分页游标与缓存键都不该依赖这种随机性。
var domainPermissions = []struct {
	Domain   string
	Resource string
}{
	{"ticket", "ticket"},
	{"incident", "incident"},
	{"problem", "problem"},
	{"change", "change"},
}

func (s *serviceImpl) Query(ctx context.Context, query WorkbenchQuery) (*WorkbenchResponse, error) {
	// 未装配权限判定 → 原样委托（历史行为；生产装配必须注入 client）。
	if s.client == nil {
		return s.repo.Query(ctx, query)
	}

	allowed := s.filterDomainsByPermission(ctx, query)
	// 过滤后一个域都不剩 → 直接返回空页，不落库查询（否则会退化成全域查询）。
	//
	// 注意与「未指定 recordType」的区分：后者语义是「全部域」，由
	// filterDomainsByPermission 展开成完整域列表再逐域判定，不会走到这里。
	if len(allowed) == 0 {
		return &WorkbenchResponse{
			Items:      []WorkbenchItem{},
			Total:      0,
			Page:       query.Page,
			PageSize:   query.PageSize,
			TotalPages: 0,
		}, nil
	}
	query.RecordType = allowed
	return s.repo.Query(ctx, query)
}

// filterDomainsByPermission 按访问者持有的域读权限裁剪 recordType。
//
// 规则：
//   - super_admin → 全域放行（middleware.AuthorizeResource 的既定语义）；
//   - 否则逐域判定 <resource>:read，任一角色命中即放行（多角色并集）。
//
// 请求显式指定 recordType 时同样受裁剪约束：显式指定不等于有权看，
// 否则「?recordType=incident」就成了绕过域权限的后门。
//
// 未指定 recordType 时按全部域展开（固定序），逐域判定后得到实际可见域。
func (s *serviceImpl) filterDomainsByPermission(ctx context.Context, query WorkbenchQuery) []string {
	// 请求指定的域（顺序即用户想要的顺序）；未指定则用全部域（固定序）。
	requested := query.RecordType
	if len(requested) == 0 {
		requested = make([]string, 0, len(domainPermissions))
		for _, dp := range domainPermissions {
			requested = append(requested, dp.Domain)
		}
	}

	resourceByDomain := make(map[string]string, len(domainPermissions))
	for _, dp := range domainPermissions {
		resourceByDomain[dp.Domain] = dp.Resource
	}

	allowed := make([]string, 0, len(requested))
	for _, domain := range requested {
		resource, ok := resourceByDomain[domain]
		if !ok {
			// 未知 recordType 不放行：工作台的域词表是受控的，未知值既没有
			// 对应权限码，也可能拼错——放行等于把拼写错误变成「看到全部」。
			continue
		}
		if middleware.AuthorizeResource(ctx, s.client, query.Roles, resource, "read", query.TenantID) {
			allowed = append(allowed, domain)
		}
	}
	return allowed
}
