package contract

// 列表信封收敛棘轮（ratchet）。
//
// AGENTS.md 规定列表响应只能有一套契约：
//   - 集合只在 data.items 下承载，领域名只允许出现在 item 类型里；
//   - 统计字段固定为 total / page / pageSize / totalPages。
//
// 存量 *ListResponse 仍使用 tickets/changes/incidents 等领域名，或漏掉标准分页键，
// 属于待收敛债务：本文件不要求一次性清零，但禁止新增，也不允许基线悄悄增长。
//
// 收敛一个字段的做法：把 DTO 的 json tag 改成 items（并补齐缺失的分页键），同步更新
// Mapper、真实路由测试、前端 src/lib/api 类型与调用点，然后从对应基线删掉这一条。
//
// 跑测命令：cd itsm-backend && go test ./tests/contract/... -run TestListEnvelope

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// canonicalListKeys 是标准列表信封必须出现的 JSON 键。
var canonicalListKeys = []string{"items", "page", "pageSize", "total", "totalPages"}

// pagingKeys 是「这个信封声称自己分页」的键。docs/api-reference.md「不分页的列表」
// 明确允许只返回 {items,total}，并要求**不得伪造** page/pageSize/totalPages 让调用方
// 以为存在分页协议（历史实例：/tickets/templates 曾伪造 page=1、pageSize=len(items)）。
// 因此本棘轮判定的是「声明了分页却只给出一半分页键」，而不是「所有列表都必须分页」。
var pagingKeys = map[string]bool{"page": true, "pageSize": true, "totalPages": true}

// forbiddenPagingKeys 是 AGENTS.md 明确禁止与 pageSize 并存的分页别名。
// 它们是历史双轨的残留：同一响应把同一个事实说两遍，消费方无法判断哪个权威。
var forbiddenPagingKeys = map[string]bool{
	"size":        true,
	"limit":       true,
	"offset":      true,
	"totalCount":  true,
	"total_count": true,
	"page_size":   true,
	"totalPage":   true, // 少一个 s 的旧拼写，5.5 门禁原来单独查它
}

// listEnvelope 是 dto/ 里一个「看起来是列表信封」的结构体。
type listEnvelope struct {
	file        string
	name        string
	domainKeys  []string // 切片字段上除 items 之外的 json tag
	missingKeys []string // 信封声明了分页时：标准五键里还没出现的
	aliasKeys   []string // 与 pageSize 并存的分页别名
	hasItemsKey bool
}

// envelopeBaseline 是 2026-10-02 由扫描器实测的「领域名集合键」存量清单（file|struct|jsonKey）。
// 新条目必须经评审才能加入；每收敛一个都应删除条目并配前端契约更新。
//
// 2026-10-03 E4-6c 移除 problem_dto.go|ListProblemsResponse|problems：GET /api/v1/problems
// 的集合键改为 items，同时把三处各自夹紧的分页规则收给 common.GetPaginationFromQuery 单点。
// 2026-10-03 E4-6e 移除 cmdb_dto.go|ListCIsResponse|cis：实测该结构体在非测试代码里零引用
// （活的 CI 列表信封是 dto.CIListResponse，键已是 items），属只被本基线引用的死声明，按
// E4-3 口径直接删除而不是改名保留。
// 2026-10-03 E4-6f 移除 service_dto.go|ServiceCatalogListResponse|catalogs：四个 URL
// （/service-catalog、/service-catalogs、/service-catalog-services、/service-catalogs/search）
// 共用同一个 h.List/h.Search，实测都做 Count + Offset/Limit，集合键统一为 items。
// 2026-10-04 E4-6g 移除 asset_dto.go|AssetListResponse|assets 与
// asset_license_dto.go|LicenseListResponse|licenses：两个端点都做 Count + Offset/Limit，
// 但许可列表连 page/pageSize/totalPages 都没有，且服务层的 `if page>0 && pageSize>0`
// 让漏写的调用方拿到整表；同时删除零引用的 ListParticipantsResponse 与
// MSPReportListResponse（按 E4-3 口径删除而不是改名保留）。
// 2026-10-04 E4-6h 移除 ticket_assignment_dto.go|ListAssignmentRulesResponse|rules、
// ticket_automation_rule_dto.go|ListAutomationRulesResponse|rules、
// ticket_attachment_dto.go|ListTicketAttachmentsResponse|attachments：三个端点实测都**不分页**
// （service 整表取回、handler 写 Total: len(items)），所以只把集合键收敛为 items 并保留诚实的
// {items,total}，**不补** page/pageSize/totalPages——给不存在的分页协议补键等于伪造契约。
var envelopeBaseline = []string{
	"auditlog_dto.go|ListAuditLogsResponse|logs",
	"change_dto.go|ChangeListResponse|changes",
	"cloud_dto.go|CloudAccountListResponse|cloudAccounts",
	"cloud_dto.go|CloudResourceListResponse|cloudResources",
	"cloud_dto.go|CloudServiceListResponse|cloudServices",
	"menu_dto.go|MenuListResponse|menus",
	"msp_dto.go|MSPAllocationListResponse|allocations",
	"msp_dto.go|MSPCustomerListResponse|customers",
	"release_dto.go|ReleaseListResponse|releases",
	"role_dto.go|RoleListResponse|roles",
	"tenant_dto.go|TenantListResponse|tenants",
	"ticket_dto.go|ListTicketsResponse|tickets",
	"ticket_workflow_dto.go|TicketCCListResponse|records",
}

// envelopeKeyBaseline 是「声明了分页却漏掉标准分页键」的存量清单（file|struct|缺失键），
// 2026-10-03 按新判定重新实测。判定只看**部分分页**：出现了 page/pageSize/totalPages 里
// 的任一键，就必须凑满五元组。纯 {items,total} 不在这里，因为 docs/api-reference.md
// 「不分页的列表」承认它是合法形状，且明确要求不得伪造分页键。
//
// 因此这 5 条全是 CMDB 一侧「带 page 但用 size 顶替 pageSize、也没有 totalPages」的信封，
// 服务层实测真的做 Offset/Limit（如 service/ci_tag_service.go:102），消费方无法核对页数。
// 2026-10-03 从本清单移除的两条是按实测改判，不是放宽规则：
//   - ticket_view_dto.go|ListTicketViewsResponse、ticket_comment_dto.go|ListTicketCommentsResponse
//     —— service/ticket_view_service.go:28 与 service/ticket_comment_service.go:104 都是
//     无 Limit 的 All(ctx)，handler 里 Total=len(items) 诚实，属合法不分页形状；
//   - change_pir_dto.go|ChangePIRListResponse —— 它**真的分页**（handlers/change/handler.go:957
//     读 page/pageSize，service/pir_service.go:180 做 Offset/Limit），所以按补全五元组收口，
//     而不是降级成「不分页」。
//
// 2026-10-03 E4-6e 把余下 5 条 CMDB 信封（ListResponse[T] / CIListResponse / CITagListResponse /
// CITypeListResponse / CIHistoryListResponse）补成五元组并清零本基线：这些服务实测都做
// Count + Offset/Limit，缺 totalPages 的 page+size 形状让调用方无法核对是否还有下一页。
// 因此本基线现在是空集——CMDB 侧再引入「带分页键但不全」的信封会直接失败。
var envelopeKeyBaseline = []string{}

// envelopeAliasBaseline 是 2026-10-02 由扫描器实测的「分页别名残留」存量清单
// （file|struct|jsonKey）。这些信封用 `size` 代替 `pageSize`（部分还缺 pageSize/totalPages），
// 属于 AGENTS.md 禁止的双轨；清理时必须同时改请求参数、前端类型与调用点。
//
// 2026-10-03 E4-7 移除 notification_dto.go|NotificationListResponse|size：请求侧原本同时
// 接受 `size` 与 `pageSize`，现在只认 `pageSize`（HTTP 入口走 common.GetPaginationFromQuery）。
// 2026-10-03 E4-6e 移除 5 条 CMDB 信封的 size：响应键改 pageSize，请求侧 dto.ListCIRequest 的
// `form:"size"` 一并删除（不保留双接受），前端 9 个 size 发送点同批改发 pageSize。
// 2026-10-03 E4-6f 移除最后一条 service_dto.go|ServiceCatalogListResponse|size：响应 catalogs→items、
// size→pageSize+totalPages，请求侧 dto.GetServiceCatalogsRequest 的 page/size（含
// binding:"max=1000" 这第四套页长真相）整体删除，分页只由 HTTP 入口的
// common.GetPaginationFromQuery 决定；前端 size 发送点同批改发 pageSize。
// 因此本基线现在是空集——任何再用 `size` 顶替 `pageSize` 的信封都会在这里失败。
var envelopeAliasBaseline = []string{}

// TestListEnvelopeRatchet 禁止列表信封再引入领域名集合键（items 之外的第二个集合键）。
func TestListEnvelopeRatchet(t *testing.T) {
	diffRatchet(t,
		"新增领域名列表信封字段，违反 data.items 单一契约，请改用 items 或走评审",
		"以下条目已收敛，请从 envelopeBaseline 删除",
		scanListEnvelopes(t, func(e listEnvelope) []string {
			var out []string
			for _, key := range e.domainKeys {
				out = append(out, e.file+"|"+e.name+"|"+key)
			}
			return out
		}), envelopeBaseline)
}

// TestListEnvelopeKeys 禁止列表信封漏掉标准分页键。
func TestListEnvelopeKeys(t *testing.T) {
	diffRatchet(t,
		"新增分页键不完整的列表信封，请补齐 page/pageSize/totalPages",
		"以下条目已收敛，请从 envelopeKeyBaseline 删除",
		scanListEnvelopes(t, func(e listEnvelope) []string {
			if !e.hasItemsKey || len(e.missingKeys) == 0 {
				return nil
			}
			return []string{e.file + "|" + e.name + "|" + strings.Join(e.missingKeys, ",")}
		}), envelopeKeyBaseline)
}

// TestListEnvelopePagingAliases 禁止列表信封继续同时返回 pageSize 与 size/limit/offset 等别名。
func TestListEnvelopePagingAliases(t *testing.T) {
	diffRatchet(t,
		"新增分页别名，AGENTS.md 只允许 page/pageSize",
		"以下条目已收敛，请从 envelopeAliasBaseline 删除",
		scanListEnvelopes(t, func(e listEnvelope) []string {
			var out []string
			for _, key := range e.aliasKeys {
				out = append(out, e.file+"|"+e.name+"|"+key)
			}
			return out
		}), envelopeAliasBaseline)
}

// diffRatchet 双向比对：新增违规失败，基线过期同样失败，避免收敛后忘记收口。
func diffRatchet(t *testing.T, addedMsg, staleMsg string, got, baseline []string) {
	t.Helper()

	allowed := make(map[string]struct{}, len(baseline))
	for _, entry := range baseline {
		allowed[entry] = struct{}{}
	}
	seen := make(map[string]struct{}, len(got))
	for _, entry := range got {
		seen[entry] = struct{}{}
	}

	var added, stale []string
	for _, entry := range got {
		if _, ok := allowed[entry]; !ok {
			added = append(added, entry)
		}
	}
	for _, entry := range baseline {
		if _, ok := seen[entry]; !ok {
			stale = append(stale, entry)
		}
	}

	sort.Strings(added)
	sort.Strings(stale)
	if len(added) > 0 {
		t.Errorf("%s：%v", addedMsg, added)
	}
	if len(stale) > 0 {
		t.Errorf("%s：%v", staleMsg, stale)
	}
}

// scanListEnvelopes 扫描 dto/*.go 里的列表信封，并按 project 提取违规条目。
//
// 信封判定条件刻意收紧：类型名必须含 List 且带 total，避免把详情/统计 DTO 内嵌的
// 关联集合（如 ProblemDetailResponse.incidents、ImportUsersResponse.failed）误判成信封违规。
func scanListEnvelopes(t *testing.T, project func(listEnvelope) []string) []string {
	t.Helper()

	files, err := filepath.Glob(filepath.Join("..", "..", "dto", "*.go"))
	if err != nil {
		t.Fatalf("glob dto: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("dto 目录扫描为空，测试路径假设已失效")
	}

	fset := token.NewFileSet()
	var out []string
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		base := filepath.Base(path)
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}
				env := listEnvelope{file: base, name: ts.Name.Name}
				present := make(map[string]bool)
				for _, field := range st.Fields.List {
					tag := jsonTag(field)
					if tag == "" {
						continue
					}
					present[tag] = true
					if tag == "items" {
						env.hasItemsKey = true
					}
					if tag != "items" && tag != "total" && isSliceField(field) {
						env.domainKeys = append(env.domainKeys, tag)
					}
				}
				// 详情/统计 DTO（如 ChangeApprovalSummary、ImportUsersResponse）里内嵌的
				// 关联集合领域名合法，不在收敛范围。
				if !present["total"] || !strings.Contains(ts.Name.Name, "List") {
					continue
				}
				sort.Strings(env.domainKeys)
				var aliases []string
				for key := range present {
					if forbiddenPagingKeys[key] {
						aliases = append(aliases, key)
					}
				}
				sort.Strings(aliases)
				env.aliasKeys = aliases
				// 只有「已经出现过分页键」的信封才要求凑满五元组。
				// 纯 {items,total} 属 docs/api-reference.md「不分页的列表」承认的形状，
				// 给它补 page/pageSize/totalPages 反而是伪造分页契约。
				declaresPaging := false
				for key := range present {
					if pagingKeys[key] {
						declaresPaging = true
						break
					}
				}
				if declaresPaging {
					for _, key := range canonicalListKeys {
						if !present[key] {
							env.missingKeys = append(env.missingKeys, key)
						}
					}
				}
				out = append(out, project(env)...)
			}
		}
	}
	return out
}

func jsonTag(field *ast.Field) string {
	if field.Tag == nil {
		return ""
	}
	lit := strings.Trim(field.Tag.Value, "`")
	tag := reflect.StructTag(lit).Get("json")
	if i := strings.Index(tag, ","); i >= 0 {
		tag = tag[:i]
	}
	if tag == "-" {
		return ""
	}
	return tag
}

func isSliceField(field *ast.Field) bool {
	switch field.Type.(type) {
	case *ast.ArrayType:
		return true
	case *ast.StarExpr:
		inner, ok := field.Type.(*ast.StarExpr).X.(*ast.ArrayType)
		return ok && inner != nil
	default:
		return false
	}
}
