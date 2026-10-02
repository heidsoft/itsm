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
	missingKeys []string // 标准五键里没出现的
	aliasKeys   []string // 与 pageSize 并存的分页别名
	hasItemsKey bool
}

// envelopeBaseline 是 2026-10-02 由扫描器实测的「领域名集合键」存量清单（file|struct|jsonKey）。
// 新条目必须经评审才能加入；每收敛一个都应删除条目并配前端契约更新。
var envelopeBaseline = []string{
	"asset_dto.go|AssetListResponse|assets",
	"asset_license_dto.go|LicenseListResponse|licenses",
	"auditlog_dto.go|ListAuditLogsResponse|logs",
	"change_dto.go|ChangeListResponse|changes",
	"cloud_dto.go|CloudAccountListResponse|cloudAccounts",
	"cloud_dto.go|CloudResourceListResponse|cloudResources",
	"cloud_dto.go|CloudServiceListResponse|cloudServices",
	"cmdb_dto.go|ListCIsResponse|cis",
	"knowledge_dto.go|ListParticipantsResponse|participants",
	"menu_dto.go|MenuListResponse|menus",
	"msp_dto.go|MSPAllocationListResponse|allocations",
	"msp_dto.go|MSPCustomerListResponse|customers",
	"msp_dto.go|MSPReportListResponse|reports",
	"notification_dto.go|NotificationListResponse|notifications",
	"problem_dto.go|ListProblemsResponse|problems",
	"release_dto.go|ReleaseListResponse|releases",
	"role_dto.go|RoleListResponse|roles",
	"service_dto.go|ServiceCatalogListResponse|catalogs",
	"tenant_dto.go|TenantListResponse|tenants",
	"ticket_assignment_dto.go|ListAssignmentRulesResponse|rules",
	"ticket_attachment_dto.go|ListTicketAttachmentsResponse|attachments",
	"ticket_automation_rule_dto.go|ListAutomationRulesResponse|rules",
	"ticket_dto.go|ListTicketsResponse|tickets",
	"ticket_notification_dto.go|ListTicketNotificationsResponse|notifications",
	"ticket_workflow_dto.go|TicketCCListResponse|records",
}

// envelopeKeyBaseline 是 2026-10-02 由扫描器实测的「标准分页键不完整」存量清单
// （file|struct|缺失键）。这些信封已经用 items，但漏掉 page/pageSize/totalPages 中的
// 若干项，消费方只能自己猜总页数。
var envelopeKeyBaseline = []string{
	"change_pir_dto.go|ChangePIRListResponse|page,pageSize,totalPages",
	"cmdb_advanced_dto.go|ListResponse|pageSize,totalPages",
	"cmdb_core_dto.go|CIListResponse|pageSize,totalPages",
	"cmdb_core_dto.go|CITagListResponse|pageSize,totalPages",
	"cmdb_core_dto.go|CITypeListResponse|pageSize,totalPages",
	"cmdb_dto.go|CIHistoryListResponse|pageSize,totalPages",
	"ticket_comment_dto.go|ListTicketCommentsResponse|page,pageSize,totalPages",
	"ticket_view_dto.go|ListTicketViewsResponse|page,pageSize,totalPages",
}

// envelopeAliasBaseline 是 2026-10-02 由扫描器实测的「分页别名残留」存量清单
// （file|struct|jsonKey）。这些信封用 `size` 代替 `pageSize`（部分还缺 pageSize/totalPages），
// 属于 AGENTS.md 禁止的双轨；清理时必须同时改请求参数、前端类型与调用点。
var envelopeAliasBaseline = []string{
	"cmdb_advanced_dto.go|ListResponse|size",
	"cmdb_core_dto.go|CIListResponse|size",
	"cmdb_core_dto.go|CITagListResponse|size",
	"cmdb_core_dto.go|CITypeListResponse|size",
	"cmdb_dto.go|CIHistoryListResponse|size",
	"notification_dto.go|NotificationListResponse|size",
	"service_dto.go|ServiceCatalogListResponse|size",
}

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
				for _, key := range canonicalListKeys {
					if !present[key] {
						env.missingKeys = append(env.missingKeys, key)
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
