package skill_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
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
	dsn := "file:" + filepath.Join(t.TempDir(), "skilldef") +
		"?mode=memory&cache=shared"
	db, err := sql.Open("sqlite3", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
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
	return db
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
