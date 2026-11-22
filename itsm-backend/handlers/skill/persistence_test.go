package skill_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"itsm-backend/handlers/skill"
	"itsm-backend/middleware"
)

// Skill 元数据持久化回归锁（AI-Native 计划批次 C）。
//
// 生产 DDL（迁移 023）的 JSON 列是 JSONB；测试库用 TEXT 列等价建表——
// 本组测试验证的是 repository 的 **SQL 语义**（幂等 upsert / 恢复 / 删除 /
// 租户归属），列类型 JSONB vs TEXT 的差异由 PostgreSQL 迁移与验收保证
// （JSONB 列接受字符串参数写入，两种库的读写语句完全一致）。

func newDefinitionTestDB(t *testing.T) *sql.DB {
	t.Helper()
	// 文件库 + WAL + busy_timeout：并发用例需要多个真实连接；内存共享缓存
	// 会在写锁竞争时直接返回 SQLITE_LOCKED，掩盖被测的幂等语义。
	dsn := "file:" + filepath.Join(t.TempDir(), "skilldef") +
		"?_busy_timeout=10000&_journal_mode=WAL&_synchronous=NORMAL"
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	db.SetMaxOpenConns(16)
	t.Cleanup(func() { _ = db.Close() })

	createDefinitionTable(t, db)
	return db
}

// createDefinitionTable 建与迁移 023 等价的测试表（JSONB 在 SQLite 里是 TEXT）。
// 独立的竞态注入用例也要用同一张表，故把 DDL 抽出来共享。
func createDefinitionTable(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
CREATE TABLE skill_definitions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL,
    version TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL DEFAULT '',
    tags TEXT NOT NULL DEFAULT '[]',
    capabilities TEXT NOT NULL DEFAULT '[]',
    required_permissions TEXT NOT NULL DEFAULT '[]',
    input_schema TEXT NOT NULL DEFAULT '{}',
    output_schema TEXT NOT NULL DEFAULT '{}',
    executor TEXT NOT NULL DEFAULT '{}',
    provider TEXT NOT NULL DEFAULT '',
    author TEXT NOT NULL DEFAULT '',
    checksum TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    tenant_id INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, code)
)`)
	require.NoError(t, err)
}

// newBuiltinLikeSkill 构造一个注册到 owner=0（全局）的技能替身——
// CustomSkill 是包内唯一的公开 dto.Skill 实现，Register 即全局语义。
func newBuiltinLikeSkill(code, version string) *skill.CustomSkill {
	return skill.NewCustomSkill(skill.CustomSkillConfig{
		Code:                code,
		Version:             version,
		Title:               code,
		RequiredPermissions: []string{"marketplace:read"},
		Executor:            map[string]interface{}{"target": "echo"},
	})
}

// setupRoutesWithStore 构造注入持久化 store 的测试路由（复用批次 B 抽出的
// setupRoutes；认证上下文 tenant_id=1 / user_id=42 / role=admin）。
func setupRoutesWithStore(reg *skill.SkillRegistry, store skill.DefinitionStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	middleware.PermissionConfig.Mode = middleware.PermissionConfigModeHardcodeOnly
	r := gin.New()
	h := skill.NewHandler(reg, zap.NewNop().Sugar(), nil)
	h.SetDefinitionStore(store)
	return setupRoutes(r, h)
}

func countDefinitionRows(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM skill_definitions`).Scan(&n))
	return n
}

// TestSyncBuiltinSkills_Idempotent 幂等 seed：两次 Sync 行数不变且
// 第二次 changed=0；checksum 演进（技能代码变更）→ 更新行而不是新增行。
func TestSyncBuiltinSkills_Idempotent(t *testing.T) {
	db := newDefinitionTestDB(t)
	store := skill.NewDBDefinitionStore(db)
	reg := skill.NewSkillRegistry()

	builtin := newBuiltinLikeSkill("ai.demo", "v1")
	require.NoError(t, reg.Register(builtin))

	ctx := context.Background()
	synced, changed, err := skill.SyncBuiltinSkillsToStore(ctx, reg, store)
	require.NoError(t, err)
	assert.Equal(t, 1, synced)
	assert.Equal(t, 1, changed)
	assert.Equal(t, 1, countDefinitionRows(t, db))

	// 第二次 Sync：checksum 一致 → 不写、不新增。
	synced, changed, err = skill.SyncBuiltinSkillsToStore(ctx, reg, store)
	require.NoError(t, err)
	assert.Equal(t, 1, synced)
	assert.Equal(t, 0, changed, "checksum 未变时不得产生写入")
	assert.Equal(t, 1, countDefinitionRows(t, db), "幂等 seed 不得新增行")

	// checksum 演进：代码变更 → checksum 变 → UPDATE 同一行。
	updated := newBuiltinLikeSkill("ai.demo", "v2")
	require.NoError(t, reg.Unregister("ai.demo"))
	require.NoError(t, reg.Register(updated))
	_, changed, err = skill.SyncBuiltinSkillsToStore(ctx, reg, store)
	require.NoError(t, err)
	assert.Equal(t, 1, changed, "checksum 变化必须更新既有行")
	assert.Equal(t, 1, countDefinitionRows(t, db), "演进不得新增行")

	var version string
	require.NoError(t, db.QueryRow(`SELECT version FROM skill_definitions WHERE code='ai.demo' AND tenant_id=0`).Scan(&version))
	assert.Equal(t, "v2", version)
}

// TestCreate_PersistsAndRestoresAcrossRestart 注册（双写）→ 用全新空注册表
// 从 DB 恢复 → 技能可用。这是「重启不丢自定义技能」的核心契约。
func TestCreate_PersistsAndRestoresAcrossRestart(t *testing.T) {
	db := newDefinitionTestDB(t)
	store := skill.NewDBDefinitionStore(db)
	reg := skill.NewSkillRegistry()

	r := setupRoutesWithStore(reg, store)

	create := skill.SkillUpsertRequest{
		Code:                "user.persist",
		Version:             "v1",
		RequiredPermissions: []string{"marketplace:read"},
		Executor:            map[string]interface{}{"target": "echo"},
	}
	w := doRequest(t, r, http.MethodPost, "/api/v1/admin/skills", create)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 1, countDefinitionRows(t, db), "注册必须双写持久层")

	// 「重启」：全新空注册表从 DB 恢复。
	freshReg := skill.NewSkillRegistry()
	restored, err := skill.RestoreCustomSkillsFromStore(context.Background(), freshReg, store)
	require.NoError(t, err)
	assert.Equal(t, 1, restored)

	got, err := freshReg.GetScoped(1, "user.persist")
	require.NoError(t, err, "重启后自定义技能必须可被租户取到")
	assert.Equal(t, "user.persist", got.Code())
}

// TestDelete_RemovesPersistedRow 删除必须双写：DB 行消失后重启不再恢复
// （删了又「复活」与删除意图矛盾）。
func TestDelete_RemovesPersistedRow(t *testing.T) {
	db := newDefinitionTestDB(t)
	store := skill.NewDBDefinitionStore(db)
	reg := skill.NewSkillRegistry()

	r := setupRoutesWithStore(reg, store)

	create := skill.SkillUpsertRequest{
		Code:                "user.doomed",
		Version:             "v1",
		RequiredPermissions: []string{"marketplace:read"},
		Executor:            map[string]interface{}{"target": "echo"},
	}
	require.Equal(t, http.StatusOK, doRequest(t, r, http.MethodPost, "/api/v1/admin/skills", create).Code)
	w := doRequest(t, r, http.MethodDelete, "/api/v1/admin/skills/user.doomed", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 0, countDefinitionRows(t, db), "删除必须移除持久层行")

	freshReg := skill.NewSkillRegistry()
	restored, err := skill.RestoreCustomSkillsFromStore(context.Background(), freshReg, store)
	require.NoError(t, err)
	assert.Equal(t, 0, restored, "已删除的技能重启后不得恢复")
}

// TestCreate_TenantOwnershipIsolated 租户归属：tenant 1 的自定义技能对
// tenant 2 不可见（运行时 + 持久层行都归属正确租户）。
func TestCreate_TenantOwnershipIsolated(t *testing.T) {
	db := newDefinitionTestDB(t)
	store := skill.NewDBDefinitionStore(db)
	reg := skill.NewSkillRegistry()

	r := setupRoutesWithStore(reg, store)

	create := skill.SkillUpsertRequest{
		Code:                "user.private",
		Version:             "v1",
		RequiredPermissions: []string{"marketplace:read"},
		Executor:            map[string]interface{}{"target": "echo"},
	}
	require.Equal(t, http.StatusOK, doRequest(t, r, http.MethodPost, "/api/v1/admin/skills", create).Code)

	// 持久层行归属 tenant 1。
	var tenantID int
	require.NoError(t, db.QueryRow(`SELECT tenant_id FROM skill_definitions WHERE code='user.private'`).Scan(&tenantID))
	assert.Equal(t, 1, tenantID)

	// 运行时：tenant 2 不可见。
	_, err := reg.GetScoped(2, "user.private")
	assert.Error(t, err, "tenant 2 不得看到 tenant 1 的自定义技能")

	// 恢复后 tenant 2 依然不可见。
	freshReg := skill.NewSkillRegistry()
	_, err = skill.RestoreCustomSkillsFromStore(context.Background(), freshReg, store)
	require.NoError(t, err)
	_, err = freshReg.GetScoped(2, "user.private")
	assert.Error(t, err, "重启恢复后租户隔离必须保持")
}

// TestUpsertDefinition_StoresJSONFields JSON 字段（tags/permissions/executor）
// 的写读往返。
func TestUpsertDefinition_StoresJSONFields(t *testing.T) {
	db := newDefinitionTestDB(t)
	store := skill.NewDBDefinitionStore(db)

	row := skill.SkillDefinitionRow{
		Code: "ai.roundtrip", Version: "v1", Title: "RoundTrip",
		Description: "d", Category: "pilot",
		Tags:                []string{"a", "b"},
		Capabilities:        []string{"c"},
		RequiredPermissions: []string{"ai:read"},
		InputSchema:         map[string]interface{}{"type": "object"},
		OutputSchema:        map[string]interface{}{"type": "object"},
		Executor:            map[string]interface{}{"target": "echo"},
		Provider:            "p", Author: "a", Checksum: "sha256:x",
		Enabled: true, TenantID: 0,
	}
	_, _, err := store.UpsertDefinition(context.Background(), row)
	require.NoError(t, err)

	rows, err := store.LoadGlobalDefinitions(context.Background())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, []string{"a", "b"}, rows[0].Tags)
	assert.Equal(t, []string{"ai:read"}, rows[0].RequiredPermissions)
	raw, _ := json.Marshal(rows[0].Executor)
	assert.True(t, strings.Contains(string(raw), "echo"), fmt.Sprintf("executor 往返失真: %s", raw))
}

// concurrentUpsertRow 构造并发回归用的最小合法行；checksum 与 version 绑定，
// 便于终局断言「落库行不会把不同版本的字段混在一起」。
func concurrentUpsertRow(code, version, checksum string, tenantID int64) skill.SkillDefinitionRow {
	return skill.SkillDefinitionRow{
		Code: code, Version: version, Title: code,
		Tags:                []string{"seed"},
		Capabilities:        []string{},
		RequiredPermissions: []string{"marketplace:read"},
		InputSchema:         map[string]interface{}{},
		OutputSchema:        map[string]interface{}{},
		Executor:            map[string]interface{}{"target": "echo"},
		Checksum:            checksum,
		Enabled:             true,
		TenantID:            tenantID,
	}
}

type upsertResult struct {
	err     error
	changed bool
}

// runConcurrentUpserts 用起跑闸门把 N 个调用同时放出去；不依赖 sleep。
func runConcurrentUpserts(t *testing.T, n int, row func(i int) skill.SkillDefinitionRow, store *skill.DBDefinitionStore) []upsertResult {
	t.Helper()
	ctx := context.Background()
	start := make(chan struct{})
	results := make(chan upsertResult, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			_, changed, err := store.UpsertDefinition(ctx, row(idx))
			results <- upsertResult{err: err, changed: changed}
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)
	out := []upsertResult{}
	for r := range results {
		out = append(out, r)
	}
	require.Len(t, out, n)
	return out
}

// TestUpsertDefinition_ConcurrentSameChecksum 锁住多副本首启的核心竞态：
// 旧实现「SELECT → INSERT」会让输家撞 UNIQUE，seed 失败并降级内存注册表。
func TestUpsertDefinition_ConcurrentSameChecksum(t *testing.T) {
	db := newDefinitionTestDB(t)
	store := skill.NewDBDefinitionStore(db)
	row := concurrentUpsertRow("ai.concurrent", "v1", "sha256:stable", 0)

	results := runConcurrentUpserts(t, 8, func(int) skill.SkillDefinitionRow { return row }, store)
	changedCount := 0
	for _, r := range results {
		require.NoError(t, r.err, "并发同 checksum 不得产生唯一约束错误")
		if r.changed {
			changedCount++
		}
	}
	assert.LessOrEqual(t, changedCount, 1, "只有第一个落库的调用可以报告 changed")
	assert.Equal(t, 1, countDefinitionRows(t, db), "并发 seed 不得新增重复行")

	var gotChecksum string
	require.NoError(t, db.QueryRow(`SELECT checksum FROM skill_definitions`).Scan(&gotChecksum))
	assert.Equal(t, "sha256:stable", gotChecksum)
}

// TestUpsertDefinition_ConcurrentChecksumEvolution 锁住 checksum 演进：
// 同一 code 的两个版本同时写入，最终只剩一行且字段组合自洽。
func TestUpsertDefinition_ConcurrentChecksumEvolution(t *testing.T) {
	db := newDefinitionTestDB(t)
	store := skill.NewDBDefinitionStore(db)
	versions := map[string]string{"sha256:v1": "v1", "sha256:v2": "v2"}

	results := runConcurrentUpserts(t, 6, func(i int) skill.SkillDefinitionRow {
		if i%2 == 0 {
			return concurrentUpsertRow("ai.evolve", "v1", "sha256:v1", 0)
		}
		return concurrentUpsertRow("ai.evolve", "v2", "sha256:v2", 0)
	}, store)
	for _, r := range results {
		require.NoError(t, r.err, "checksum 演进的并发写不得报错")
	}
	assert.Equal(t, 1, countDefinitionRows(t, db), "演进只更新既有行")

	var checksum, version string
	require.NoError(t, db.QueryRow(`SELECT checksum, version FROM skill_definitions`).Scan(&checksum, &version))
	assert.Equal(t, versions[checksum], version, "checksum 与 version 必须同行自洽")
}

// TestUpsertDefinition_ConcurrentTenantsKeepIsolation 锁住 UPSERT 的冲突键：
// 漏掉 tenant_id 会把两个租户的同名技能合并成一行，属跨租户数据污染。
func TestUpsertDefinition_ConcurrentTenantsKeepIsolation(t *testing.T) {
	db := newDefinitionTestDB(t)
	store := skill.NewDBDefinitionStore(db)

	results := runConcurrentUpserts(t, 6, func(i int) skill.SkillDefinitionRow {
		tenantID := int64(1 + i%2)
		return concurrentUpsertRow("tenant.shared.code", "v1", "sha256:same", tenantID)
	}, store)
	for _, r := range results {
		require.NoError(t, r.err, "跨租户同 code 不得互相冲突")
	}
	assert.Equal(t, 2, countDefinitionRows(t, db), "两个租户必须各有一行")

	rows, err := db.Query(`SELECT tenant_id FROM skill_definitions ORDER BY tenant_id`)
	require.NoError(t, err)
	defer rows.Close()
	tenants := []int{}
	for rows.Next() {
		var id int
		require.NoError(t, rows.Scan(&id))
		tenants = append(tenants, id)
	}
	assert.Equal(t, []int{1, 2}, tenants)
}
