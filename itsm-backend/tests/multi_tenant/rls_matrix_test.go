// Package multi_tenant —— 跨租户隔离回归矩阵。
//
// ## 事实基线（2026-10-18 实测，勿凭文档臆测）
//
//  1. PostgreSQL 行级安全（RLS）当前处于**试点**阶段：只有 `changes` 与
//     `vectors` 两张表启用了 `tenant_isolation` 策略
//     （database/rls/migrations/002_pilot_policies.sql），其余 tenant-owned 表
//     尚未启用（该迁移注释写明「其他 100 张业务表将在 R2 里程碑批量启用」）。
//  2. 因此当前覆盖全部租户资源的主力是**应用层 tenant 谓词**：
//     - repository 的 Get/List/Delete 方法携带 tenantID 参数；
//     - service.TenantAwareRepository 的 GuardDelete/GuardUpdate 助手；
//     - middleware.ResolveRequestTenantID 从认证上下文解析有效租户（fail-closed）。
//
// ## 本文件验证什么
//
// 用**真实生产代码**（repository/service 实现）验证应用层隔离，运行在 SQLite，
// 随 `go test ./...` 在 CI 默认执行。覆盖：tickets / incidents / problems /
// changes / service_requests / service_catalogs / knowledge_articles /
// sla_definitions / approval_chains / users / teams / ai conversations。
//
// 每个用例断言三件事：
//   - 同租户读取成功（正控制，证明探针本身可用）；
//   - 跨租户读取失败，且错误形态与「不存在」一致（不泄漏存在性）；
//   - tenantID=0（缺租户上下文）fail closed。
//
// ## 本文件不覆盖什么
//
// configuration_items / ci_relationships / sla_alert_rules / notifications /
// assets / releases 当前没有带 tenantID 的 repository 读取方法（这些域的生产
// 入口不在 handlers/<domain>/repository 形态），未纳入本矩阵。补测需先按其真实
// 生产入口接线，不能靠本矩阵假装覆盖。
//
// 数据库级 RLS 由 build tag `integration_rls` 的真实 PostgreSQL 用例覆盖：
//   - database/rls/rls_integration_test.go（changes 策略 + 连接释放重置）
//   - integration/golden_path_test.go（生产策略下的跨租户不可见）
//
// 由 .github/workflows/rls-golden-path.yml 在 Postgres 服务容器上运行。
// MSP/CSP 的租户切换决策由 middleware/msp_tenant_resolver_test.go 覆盖。
//
// ## 历史（为什么重写）
//
// 本文件旧版本以 17 个空函数体 + 9 个纯注释用例声称「17 张表 RLS 全绿」，
// 实际零断言；且 SQLite 根本无法执行 RLS。该测试在 CI 中恒绿，制造了虚假的
// 隔离保证。现替换为下面的真实断言，并显式记录未覆盖范围。
package multi_tenant

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	entschema "itsm-backend/ent/schema"
	entuser "itsm-backend/ent/user"
	"itsm-backend/handlers/ai"
	"itsm-backend/handlers/change"
	"itsm-backend/handlers/common"
	"itsm-backend/handlers/incident"
	"itsm-backend/handlers/knowledge"
	"itsm-backend/handlers/problem"
	"itsm-backend/handlers/service_catalog"
	"itsm-backend/handlers/service_request"
	"itsm-backend/handlers/sla"
	ticketrepo "itsm-backend/repository/ticket"
	"itsm-backend/service"
)

var matrixDBCounter int64

// matrixEnv 是一次矩阵测试的共享夹具：两个租户 + 一个 A 租户用户 + 内存库。
type matrixEnv struct {
	client  *ent.Client
	tenantA *ent.Tenant
	tenantB *ent.Tenant
	userA   *ent.User
	logger  *zap.SugaredLogger
}

func newMatrixEnv(t *testing.T) *matrixEnv {
	t.Helper()

	dsn := fmt.Sprintf("file:rls_matrix_%d?mode=memory&cache=shared&_fk=1",
		atomic.AddInt64(&matrixDBCounter, 1))
	client := enttest.Open(t, "sqlite3", dsn)
	t.Cleanup(func() { _ = client.Close() })

	tenantA := mustTenant(t, client, "Tenant A", "tenant-a")
	tenantB := mustTenant(t, client, "Tenant B", "tenant-b")
	userA := mustUser(t, client, tenantA.ID, "user-a", "user-a@example.com")

	return &matrixEnv{
		client:  client,
		tenantA: tenantA,
		tenantB: tenantB,
		userA:   userA,
		logger:  zaptest.NewLogger(t).Sugar(),
	}
}

func mustTenant(t *testing.T, client *ent.Client, name, code string) *ent.Tenant {
	t.Helper()
	tn, err := client.Tenant.Create().
		SetName(name).
		SetCode(code).
		SetDomain(code + ".example.com").
		SetStatus("active").
		Save(context.Background())
	require.NoError(t, err)
	return tn
}

func mustUser(t *testing.T, client *ent.Client, tenantID int, username, email string) *ent.User {
	t.Helper()
	u, err := client.User.Create().
		SetUsername(username).
		SetEmail(email).
		SetName(username).
		SetPasswordHash("hashed").
		SetRole(entuser.Role("agent")).
		SetActive(true).
		SetTenantID(tenantID).
		Save(context.Background())
	require.NoError(t, err)
	return u
}

// tenantIsolationCase 描述一个「读路径」隔离用例。
type tenantIsolationCase struct {
	name string
	// seed 在 tenantA 下播种一行，返回其主键。
	seed func(ctx context.Context, env *matrixEnv) int
	// probe 调用真实 repository/service 的 tenant 作用域读取；返回 error 表示不可见。
	probe func(ctx context.Context, env *matrixEnv, id, tenantID int) error
}

// digitsOnly 用于把错误文案里的主键数字归一化，从而比较「不存在」与「跨租户」
// 两种错误的形态，避免因回显请求 ID 造成误判。
var digitsOnly = regexp.MustCompile(`\d+`)

func normalizeErr(err error) string {
	return digitsOnly.ReplaceAllString(err.Error(), "#")
}

// TestTenantScope_RepositoryMatrix 逐个实体验证应用层 tenant 谓词生效。
func TestTenantScope_RepositoryMatrix(t *testing.T) {
	env := newMatrixEnv(t)

	cases := []tenantIsolationCase{
		{
			name: "tickets",
			seed: func(ctx context.Context, env *matrixEnv) int {
				row, err := env.client.Ticket.Create().
					SetTitle("matrix ticket").
					SetDescription("tenant isolation matrix").
					SetPriority("medium").
					SetStatus("open").
					SetTicketNumber(fmt.Sprintf("TKT-MX-%d", time.Now().UnixNano())).
					SetRequesterID(env.userA.ID).
					SetTenantID(env.tenantA.ID).
					Save(ctx)
				require.NoError(t, err)
				return row.ID
			},
			probe: func(ctx context.Context, env *matrixEnv, id, tenantID int) error {
				_, err := ticketrepo.NewEntRepository(env.client, env.logger).GetByID(ctx, id, tenantID)
				return err
			},
		},
		{
			name: "incidents",
			seed: func(ctx context.Context, env *matrixEnv) int {
				row, err := env.client.Incident.Create().
					SetTitle("matrix incident").
					SetDescription("tenant isolation matrix").
					SetPriority("high").
					SetStatus("open").
					SetIncidentNumber(fmt.Sprintf("INC-MX-%d", time.Now().UnixNano())).
					SetReporterID(env.userA.ID).
					SetTenantID(env.tenantA.ID).
					Save(ctx)
				require.NoError(t, err)
				return row.ID
			},
			probe: func(ctx context.Context, env *matrixEnv, id, tenantID int) error {
				_, err := incident.NewEntRepository(env.client).Get(ctx, id, tenantID)
				return err
			},
		},
		{
			name: "problems",
			seed: func(ctx context.Context, env *matrixEnv) int {
				row, err := env.client.Problem.Create().
					SetTitle("matrix problem").
					SetDescription("tenant isolation matrix").
					SetPriority("high").
					SetStatus("open").
					SetCategory("infrastructure").
					SetImpact("medium").
					SetCreatedBy(env.userA.ID).
					SetTenantID(env.tenantA.ID).
					Save(ctx)
				require.NoError(t, err)
				return row.ID
			},
			probe: func(ctx context.Context, env *matrixEnv, id, tenantID int) error {
				_, err := problem.NewEntRepository(env.client).Get(ctx, id, tenantID)
				return err
			},
		},
		{
			name: "changes",
			seed: func(ctx context.Context, env *matrixEnv) int {
				row, err := env.client.Change.Create().
					SetTitle("matrix change").
					SetDescription("tenant isolation matrix").
					SetType("normal").
					SetPriority("medium").
					SetStatus("draft").
					SetRiskLevel("medium").
					SetCreatedBy(env.userA.ID).
					SetTenantID(env.tenantA.ID).
					Save(ctx)
				require.NoError(t, err)
				return row.ID
			},
			probe: func(ctx context.Context, env *matrixEnv, id, tenantID int) error {
				// db 参数仅供审批链/统计子仓库使用，Get 只走 ent client。
				_, err := change.NewEntRepository(env.client, nil).Get(ctx, id, tenantID)
				return err
			},
		},
		{
			name: "service_requests",
			seed: func(ctx context.Context, env *matrixEnv) int {
				catalog, err := env.client.ServiceCatalog.Create().
					SetName("catalog for request").
					SetStatus("active").
					SetTenantID(env.tenantA.ID).
					Save(ctx)
				require.NoError(t, err)

				row, err := env.client.ServiceRequest.Create().
					SetTitle("matrix service request").
					SetStatus("pending").
					SetCatalogID(catalog.ID).
					SetRequesterID(env.userA.ID).
					SetTenantID(env.tenantA.ID).
					Save(ctx)
				require.NoError(t, err)
				return row.ID
			},
			probe: func(ctx context.Context, env *matrixEnv, id, tenantID int) error {
				_, err := service_request.NewEntRepository(env.client).Get(ctx, id, tenantID)
				return err
			},
		},
		{
			name: "service_catalogs",
			seed: func(ctx context.Context, env *matrixEnv) int {
				row, err := env.client.ServiceCatalog.Create().
					SetName("matrix catalog").
					SetDescription("tenant isolation matrix").
					SetStatus("active").
					SetTenantID(env.tenantA.ID).
					Save(ctx)
				require.NoError(t, err)
				return row.ID
			},
			probe: func(ctx context.Context, env *matrixEnv, id, tenantID int) error {
				// 注意参数顺序：(tenantID, id)
				_, err := service_catalog.NewEntRepository(env.client).Get(ctx, tenantID, id)
				return err
			},
		},
		{
			name: "knowledge_articles",
			seed: func(ctx context.Context, env *matrixEnv) int {
				row, err := env.client.KnowledgeArticle.Create().
					SetTitle("matrix article").
					SetContent("tenant isolation matrix").
					SetAuthorID(env.userA.ID).
					SetTenantID(env.tenantA.ID).
					Save(ctx)
				require.NoError(t, err)
				return row.ID
			},
			probe: func(ctx context.Context, env *matrixEnv, id, tenantID int) error {
				_, err := knowledge.NewEntRepository(env.client).Get(ctx, id, tenantID)
				return err
			},
		},
		{
			name: "sla_definitions",
			seed: func(ctx context.Context, env *matrixEnv) int {
				row, err := env.client.SLADefinition.Create().
					SetName("matrix sla").
					SetDescription("tenant isolation matrix").
					SetPriority("high").
					SetResponseTime(30).
					SetResolutionTime(240).
					SetTenantID(env.tenantA.ID).
					Save(ctx)
				require.NoError(t, err)
				return row.ID
			},
			probe: func(ctx context.Context, env *matrixEnv, id, tenantID int) error {
				_, err := sla.NewEntRepository(env.client).GetDefinition(ctx, id, tenantID)
				return err
			},
		},
		{
			name: "approval_chains",
			seed: func(ctx context.Context, env *matrixEnv) int {
				row, err := env.client.ApprovalChain.Create().
					SetName("matrix approval chain").
					SetEntityType("ticket").
					SetChain([]entschema.ApprovalChainStep{}).
					SetStatus("active").
					SetTenantID(env.tenantA.ID).
					Save(ctx)
				require.NoError(t, err)
				return row.ID
			},
			probe: func(ctx context.Context, env *matrixEnv, id, tenantID int) error {
				_, err := service.NewApprovalChainService(env.client, env.logger).
					GetApprovalChain(ctx, id, tenantID)
				return err
			},
		},
		{
			name: "users",
			seed: func(ctx context.Context, env *matrixEnv) int {
				return env.userA.ID
			},
			probe: func(ctx context.Context, env *matrixEnv, id, tenantID int) error {
				_, err := common.NewEntRepository(env.client).GetUserByID(ctx, id, tenantID)
				return err
			},
		},
		{
			// 矩阵目标中的 “groups” 在当前模型里由 team 承载。
			name: "teams",
			seed: func(ctx context.Context, env *matrixEnv) int {
				row, err := env.client.Team.Create().
					SetName("matrix team").
					SetCode("MX-TEAM").
					SetDescription("tenant isolation matrix").
					SetTenantID(env.tenantA.ID).
					Save(ctx)
				require.NoError(t, err)
				return row.ID
			},
			probe: func(ctx context.Context, env *matrixEnv, id, tenantID int) error {
				_, err := common.NewEntRepository(env.client).GetTeam(ctx, id, tenantID)
				return err
			},
		},
		{
			name: "ai_conversations",
			seed: func(ctx context.Context, env *matrixEnv) int {
				row, err := env.client.Conversation.Create().
					SetTitle("matrix conversation").
					SetUserID(env.userA.ID).
					SetTenantID(env.tenantA.ID).
					Save(ctx)
				require.NoError(t, err)
				return row.ID
			},
			probe: func(ctx context.Context, env *matrixEnv, id, tenantID int) error {
				_, err := ai.NewEntRepository(env.client).GetConversation(ctx, id, tenantID)
				return err
			},
		},
	}

	require.NotEmpty(t, cases, "矩阵不能为空")

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			id := tc.seed(ctx, env)
			require.NotZero(t, id, "seed 必须返回有效主键")

			// 1) 正控制：同租户读取必须成功，证明探针本身可用。
			require.NoError(t, tc.probe(ctx, env, id, env.tenantA.ID),
				"同租户读取必须可见")

			// 2) 跨租户读取必须失败（而不是静默返回零值）。
			errCrossTenant := tc.probe(ctx, env, id, env.tenantB.ID)
			require.Error(t, errCrossTenant, "跨租户读取必须返回错误，不得静默成功")

			// 3) 不泄漏存在性：跨租户错误形态必须与「本租户下的不存在」一致。
			errMissing := tc.probe(ctx, env, id+1_000_000, env.tenantA.ID)
			require.Error(t, errMissing, "不存在的资源必须返回错误")
			require.Equal(t, normalizeErr(errMissing), normalizeErr(errCrossTenant),
				"跨租户与不存在的错误形态必须不可区分，避免存在性泄漏")

			// 4) 缺租户上下文（tenantID=0）必须 fail closed。
			require.Error(t, tc.probe(ctx, env, id, 0),
				"tenantID=0 必须 fail closed")
		})
	}
}

// TestTenantScope_ListIsolation 验证列表读取同样按租户过滤。
func TestTenantScope_ListIsolation(t *testing.T) {
	env := newMatrixEnv(t)
	ctx := context.Background()

	// tenantB 也放一个用户，确保 B 的列表非空（否则「B 看到 0 条」无法区分
	// 「谓词生效」与「B 本来就没数据」）。
	userB := mustUser(t, env.client, env.tenantB.ID, "user-b", "user-b@example.com")

	commonRepo := common.NewEntRepository(env.client)

	usersA, err := commonRepo.ListUsers(ctx, env.tenantA.ID)
	require.NoError(t, err)
	require.Len(t, usersA, 1, "tenantA 只应看到自己的用户")
	require.Equal(t, env.userA.ID, usersA[0].ID)

	usersB, err := commonRepo.ListUsers(ctx, env.tenantB.ID)
	require.NoError(t, err)
	require.Len(t, usersB, 1, "tenantB 只应看到自己的用户")
	require.Equal(t, userB.ID, usersB[0].ID)

	// 知识库列表
	_, err = env.client.KnowledgeArticle.Create().
		SetTitle("only-in-A").
		SetContent("tenant isolation matrix").
		SetAuthorID(env.userA.ID).
		SetTenantID(env.tenantA.ID).
		Save(ctx)
	require.NoError(t, err)

	knowledgeRepo := knowledge.NewEntRepository(env.client)
	artsA, totalA, err := knowledgeRepo.List(ctx, env.tenantA.ID, 1, 20, "", "", "")
	require.NoError(t, err)
	require.Len(t, artsA, 1)
	require.Equal(t, 1, totalA)

	artsB, totalB, err := knowledgeRepo.List(ctx, env.tenantB.ID, 1, 20, "", "", "")
	require.NoError(t, err)
	require.Empty(t, artsB, "tenantB 不得看到 tenantA 的知识文章")
	require.Zero(t, totalB)

	// SLA 定义列表
	_, err = env.client.SLADefinition.Create().
		SetName("only-in-A").
		SetDescription("tenant isolation matrix").
		SetPriority("high").
		SetResponseTime(30).
		SetResolutionTime(240).
		SetTenantID(env.tenantA.ID).
		Save(ctx)
	require.NoError(t, err)

	slaRepo := sla.NewEntRepository(env.client)
	defsA, defTotalA, err := slaRepo.ListDefinitions(ctx, env.tenantA.ID, 1, 20)
	require.NoError(t, err)
	require.Len(t, defsA, 1)
	require.Equal(t, 1, defTotalA)

	defsB, defTotalB, err := slaRepo.ListDefinitions(ctx, env.tenantB.ID, 1, 20)
	require.NoError(t, err)
	require.Empty(t, defsB, "tenantB 不得看到 tenantA 的 SLA 定义")
	require.Zero(t, defTotalB)
}

// TestTenantScope_MutationGuard 验证写路径的租户守卫：
// service.TenantAwareRepository 必须在跨租户时于 mutation 之前中止。
func TestTenantScope_MutationGuard(t *testing.T) {
	env := newMatrixEnv(t)
	ctx := context.Background()

	ticket, err := env.client.Ticket.Create().
		SetTitle("guard ticket").
		SetDescription("tenant mutation guard").
		SetPriority("medium").
		SetStatus("open").
		SetTicketNumber(fmt.Sprintf("TKT-GD-%d", time.Now().UnixNano())).
		SetRequesterID(env.userA.ID).
		SetTenantID(env.tenantA.ID).
		Save(ctx)
	require.NoError(t, err)

	// ValidateTenantAccess 是守卫的核心判定。
	repoB := service.NewTenantAwareRepository(env.client, env.tenantB.ID)
	require.Error(t, repoB.ValidateTenantAccess(ctx, env.tenantA.ID),
		"实体租户与当前租户不一致必须拒绝")
	require.NoError(t, repoB.ValidateTenantAccess(ctx, env.tenantB.ID),
		"实体租户与当前租户一致必须放行")

	for _, g := range []struct {
		name string
		run  func(repo *service.TenantAwareRepository, action service.GuardAction) error
	}{
		{"GuardDelete", func(repo *service.TenantAwareRepository, action service.GuardAction) error {
			return repo.GuardDelete(ctx, repo.LoadTicketTenant, action, ticket.ID)
		}},
		{"GuardUpdate", func(repo *service.TenantAwareRepository, action service.GuardAction) error {
			return repo.GuardUpdate(ctx, repo.LoadTicketTenant, action, ticket.ID)
		}},
	} {
		t.Run(g.name+"_cross_tenant", func(t *testing.T) {
			executed := false
			action := func(context.Context, int) error {
				executed = true
				return nil
			}

			err := g.run(repoB, action)
			require.Error(t, err, "跨租户写操作必须被拒绝")
			require.False(t, executed, "跨租户时 mutation 绝不能执行")

			// 行必须原样存在（未被删除/修改）。
			_, err = env.client.Ticket.Get(ctx, ticket.ID)
			require.NoError(t, err, "跨租户拒绝后资源必须仍然存在")
		})

		t.Run(g.name+"_own_tenant", func(t *testing.T) {
			executed := false
			action := func(context.Context, int) error {
				executed = true
				return nil
			}

			repoA := service.NewTenantAwareRepository(env.client, env.tenantA.ID)
			require.NoError(t, g.run(repoA, action), "同租户写操作必须放行")
			require.True(t, executed, "同租户时 mutation 必须执行")
		})

		t.Run(g.name+"_missing_entity", func(t *testing.T) {
			executed := false
			action := func(context.Context, int) error {
				executed = true
				return nil
			}

			repoA := service.NewTenantAwareRepository(env.client, env.tenantA.ID)
			err := repoA.GuardDelete(ctx, repoA.LoadTicketTenant, action, ticket.ID+1_000_000)
			require.ErrorIs(t, err, service.ErrNotFound,
				"不存在的资源必须映射为 ErrNotFound")
			require.False(t, executed, "资源不存在时 mutation 绝不能执行")
		})
	}

	t.Run("nil loader/action rejected", func(t *testing.T) {
		repoA := service.NewTenantAwareRepository(env.client, env.tenantA.ID)
		require.Error(t, repoA.GuardDelete(ctx, nil, func(context.Context, int) error { return nil }, ticket.ID))
		require.Error(t, repoA.GuardDelete(ctx, repoA.LoadTicketTenant, nil, ticket.ID))
	})
}

// TestRowLevelSecurity_PilotScopeIsPinned 固定 DB 级 RLS 的真实覆盖边界。
//
// 该断言是一道「反过度声明」闸门：DB 级 RLS 当前只试点 changes + vectors。
// 一旦有人扩大或缩小启用范围，本测试会失败，强制同步文档与验收范围，
// 避免再次出现「17 张表 RLS 全绿」式的虚假保证。
func TestRowLevelSecurity_PilotScopeIsPinned(t *testing.T) {
	path := filepath.Join("..", "..", "database", "rls", "migrations", "002_pilot_policies.sql")
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "RLS 试点策略迁移必须存在")

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?m)^ALTER TABLE (\w+) ENABLE ROW LEVEL SECURITY`),
		regexp.MustCompile(`'ALTER TABLE (\w+) ENABLE ROW LEVEL SECURITY'`),
	}

	enabled := map[string]struct{}{}
	for _, re := range patterns {
		for _, m := range re.FindAllStringSubmatch(string(raw), -1) {
			enabled[m[1]] = struct{}{}
		}
	}

	tables := make([]string, 0, len(enabled))
	for name := range enabled {
		tables = append(tables, name)
	}
	sort.Strings(tables)

	require.Equal(t, []string{"changes", "vectors"}, tables,
		"DB 级 RLS 当前仅为试点（changes + vectors）；扩大或调整覆盖时请同步本断言、"+
			"docs/testing/test-invariants.md 与 ROADMAP，不得静默改变验收范围")

	// 试点策略必须使用 app.current_tenant SESSION 变量，且缺省时 fail closed。
	require.Contains(t, string(raw), "NULLIF(current_setting('app.current_tenant', true), '')::bigint",
		"RLS 策略必须以 app.current_tenant 为唯一租户来源，且空值 fail closed")
}

// TestRowLevelSecurity_GuardHelpersExist 确保应用层守卫与解析器未被删改。
//
// 本测试只做存在性/契约固定，实际行为由上面的矩阵与 middleware 测试覆盖。
func TestRowLevelSecurity_GuardHelpersExist(t *testing.T) {
	env := newMatrixEnv(t)

	// TenantAwareRepository 暴露的 per-entity loader 必须覆盖关键实体。
	repo := service.NewTenantAwareRepository(env.client, env.tenantA.ID)
	ctx := context.Background()

	loaders := map[string]service.TenantLoader{
		"ticket":   repo.LoadTicketTenant,
		"incident": repo.LoadIncidentTenant,
		"change":   repo.LoadChangeTenant,
		"problem":  repo.LoadProblemTenant,
		"user":     repo.LoadUserTenant,
	}
	for name, loader := range loaders {
		t.Run("missing_"+name, func(t *testing.T) {
			_, err := loader(ctx, 1_000_000)
			require.ErrorIs(t, err, service.ErrNotFound,
				"不存在的实体必须映射为 ErrNotFound（跨租户探测不泄漏存在性）")
		})
	}

	// 存在的实体应返回其真实租户。
	require.NoError(t, repo.ValidateTenantAccess(ctx, env.tenantA.ID))
	require.Error(t, repo.ValidateTenantAccess(ctx, env.tenantB.ID))
}
