package skill

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"itsm-backend/dto"
)

// Skill 元数据持久层（AI-Native 计划批次 C）。
//
// 职责边界：
//   - 运行时注册表（SkillRegistry）仍是内存单源：Invoke 走内存，线程安全；
//   - 本层负责把元数据落到 skill_definitions 表：内置技能按 checksum 幂等
//     seed（代码演进 → checksum 变 → 更新行），租户自定义技能跨重启恢复；
//   - DTO 与执行器不经本层持久化执行逻辑——executor 配置随行存储，恢复时
//     交给 NewCustomSkill 重建，执行语义仍由 CustomSkill 实现。
//
// 为什么用原生 SQL 而不是 ent：与 workflow_templates / ai_telemetry 同构
// （迁移建表 + db.ExecContext/QueryContext），表无跨表关系、生命周期独立，
// ent 实体在此没有增量价值。JSON 列以字符串参数读写（PostgreSQL JSONB 列
// 接受字符串参数隐式 cast；SQLite 测试库按 TEXT 处理，语义一致）。

// DefinitionStore 是 skill 元数据持久化所需的窄接口。
type DefinitionStore interface {
	// UpsertDefinition 按 (tenantID, code) 幂等写入；checksum 与库中一致时
	// 不做任何写入并返回 changed=false。返回写入后的行 ID；因并发竞争而
	// 判定无需写入时也返回 (0, false, nil)。
	UpsertDefinition(ctx context.Context, def SkillDefinitionRow) (id int64, changed bool, err error)
	// LoadGlobalDefinitions 读全部全局内置技能行（tenant_id=0）。
	LoadGlobalDefinitions(ctx context.Context) ([]SkillDefinitionRow, error)
	// LoadCustomDefinitions 读全部租户自定义技能行（tenant_id>0）。
	LoadCustomDefinitions(ctx context.Context) ([]SkillDefinitionRow, error)
	// DeleteDefinition 按 (tenantID, code) 删除；返回是否真的删了行。
	DeleteDefinition(ctx context.Context, tenantID int64, code string) (bool, error)
	// GetDefinitionChecksum 按 (tenantID, code) 读 checksum；行不存在返回
	// sql.ErrNoRows 包装错误。
	GetDefinitionChecksum(ctx context.Context, tenantID int64, code string) (string, error)
}

// SkillDefinitionRow 是 skill_definitions 表的行视图（与迁移 DDL 一一对应）。
type SkillDefinitionRow struct {
	Code                string
	Version             string
	Title               string
	Description         string
	Category            string
	Tags                []string
	Capabilities        []string
	RequiredPermissions []string
	InputSchema         map[string]interface{}
	OutputSchema        map[string]interface{}
	Executor            map[string]interface{}
	Provider            string
	Author              string
	Checksum            string
	Enabled             bool
	TenantID            int64
}

// DBDefinitionStore 基于 database/sql 的实现（生产：PostgreSQL；测试：SQLite）。
type DBDefinitionStore struct {
	db *sql.DB
}

// NewDBDefinitionStore 构造持久化 store。
func NewDBDefinitionStore(db *sql.DB) *DBDefinitionStore {
	return &DBDefinitionStore{db: db}
}

const definitionColumns = `code, version, title, description, category,
	tags, capabilities, required_permissions, input_schema, output_schema,
	executor, provider, author, checksum, enabled, tenant_id`

func marshalJSONField(v interface{}) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func mustUnmarshalStrings(raw []byte) []string {
	out := []string{}
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}

func mustUnmarshalObject(raw []byte) map[string]interface{} {
	out := map[string]interface{}{}
	if len(raw) == 0 {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	return out
}

// UpsertDefinition 幂等写入：checksum 相同则跳过（changed=false）。
//
// 快速路径的 SELECT 只是避免无谓写入；幂等本身由迁移 023 的唯一约束
// UNIQUE (tenant_id, code) + 单条 ON CONFLICT 语句保证。不能退回
// 「先查再 INSERT」：多副本同时首启时两个调用方都会查到 ErrNoRows，
// 一个 INSERT 成功、另一个撞 23505，seed 失败并降级成内存注册表。
//
// DO UPDATE 的 WHERE 让并发同 payload 的输家匹配不到行：RETURNING 无行
// 即 changed=false，不刷 updated_at。
func (s *DBDefinitionStore) UpsertDefinition(ctx context.Context, def SkillDefinitionRow) (int64, bool, error) {
	if s == nil || s.db == nil {
		return 0, false, fmt.Errorf("definition store not initialized")
	}
	var existingChecksum string
	err := s.db.QueryRowContext(ctx,
		`SELECT checksum FROM skill_definitions WHERE tenant_id=$1 AND code=$2`,
		def.TenantID, def.Code).Scan(&existingChecksum)
	if err == nil && existingChecksum == def.Checksum {
		return 0, false, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, false, fmt.Errorf("lookup skill definition %s: %w", def.Code, err)
	}

	tags, _ := marshalJSONField(def.Tags)
	caps, _ := marshalJSONField(def.Capabilities)
	perms, _ := marshalJSONField(def.RequiredPermissions)
	inSchema, _ := marshalJSONField(def.InputSchema)
	outSchema, _ := marshalJSONField(def.OutputSchema)
	executor, _ := marshalJSONField(def.Executor)

	var id int64
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO skill_definitions (`+definitionColumns+`)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		 ON CONFLICT (tenant_id, code) DO UPDATE SET
		   version=$2, title=$3, description=$4, category=$5,
		   tags=$6, capabilities=$7, required_permissions=$8,
		   input_schema=$9, output_schema=$10, executor=$11,
		   provider=$12, author=$13, checksum=$14, enabled=$15,
		   updated_at=$17
		 WHERE skill_definitions.checksum <> EXCLUDED.checksum
		 RETURNING id`,
		def.Code, def.Version, def.Title, def.Description, def.Category,
		tags, caps, perms, inSchema, outSchema,
		executor, def.Provider, def.Author, def.Checksum, def.Enabled,
		def.TenantID, time.Now()).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		// 并发赢家已用相同 checksum 落库：无需写入，也不是错误。
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("upsert skill definition %s: %w", def.Code, err)
	}
	return id, true, nil
}

func scanDefinitions(rows *sql.Rows) ([]SkillDefinitionRow, error) {
	defer rows.Close()
	out := []SkillDefinitionRow{}
	for rows.Next() {
		var r SkillDefinitionRow
		var tags, caps, perms, inSchema, outSchema, executor []byte
		if err := rows.Scan(&r.Code, &r.Version, &r.Title, &r.Description, &r.Category,
			&tags, &caps, &perms, &inSchema, &outSchema,
			&executor, &r.Provider, &r.Author, &r.Checksum, &r.Enabled, &r.TenantID); err != nil {
			return nil, err
		}
		r.Tags = mustUnmarshalStrings(tags)
		r.Capabilities = mustUnmarshalStrings(caps)
		r.RequiredPermissions = mustUnmarshalStrings(perms)
		r.InputSchema = mustUnmarshalObject(inSchema)
		r.OutputSchema = mustUnmarshalObject(outSchema)
		r.Executor = mustUnmarshalObject(executor)
		out = append(out, r)
	}
	return out, rows.Err()
}

// LoadGlobalDefinitions 读全部内置技能行（tenant_id=0）。
func (s *DBDefinitionStore) LoadGlobalDefinitions(ctx context.Context) ([]SkillDefinitionRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+definitionColumns+` FROM skill_definitions WHERE tenant_id=0 ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("load global skill definitions: %w", err)
	}
	return scanDefinitions(rows)
}

// LoadCustomDefinitions 读全部租户自定义技能行（tenant_id>0）。
func (s *DBDefinitionStore) LoadCustomDefinitions(ctx context.Context) ([]SkillDefinitionRow, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+definitionColumns+` FROM skill_definitions WHERE tenant_id>0 ORDER BY tenant_id, code`)
	if err != nil {
		return nil, fmt.Errorf("load custom skill definitions: %w", err)
	}
	return scanDefinitions(rows)
}

// DeleteDefinition 按 (tenantID, code) 删除。行不存在返回 (false, nil)——
// 删除语义幂等（内存里有、库里没有的旧数据照样允许删）。
func (s *DBDefinitionStore) DeleteDefinition(ctx context.Context, tenantID int64, code string) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM skill_definitions WHERE tenant_id=$1 AND code=$2`, tenantID, code)
	if err != nil {
		return false, fmt.Errorf("delete skill definition %s: %w", code, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete skill definition %s rows affected: %w", code, err)
	}
	return affected > 0, nil
}

// GetDefinitionChecksum 按 (tenantID, code) 读 checksum。
func (s *DBDefinitionStore) GetDefinitionChecksum(ctx context.Context, tenantID int64, code string) (string, error) {
	var checksum string
	err := s.db.QueryRowContext(ctx,
		`SELECT checksum FROM skill_definitions WHERE tenant_id=$1 AND code=$2`,
		tenantID, code).Scan(&checksum)
	if err != nil {
		return "", fmt.Errorf("get skill definition checksum %s: %w", code, err)
	}
	return checksum, nil
}

// rowFromSkill 把运行时 Skill 的 manifest 摊平成行视图。
// tenantID 由调用方决定（内置=0，自定义=租户 ID）。
func rowFromSkill(tenantID int64, skill dto.Skill) (SkillDefinitionRow, error) {
	m := skill.Manifest()
	if m.Checksum == "" {
		// 防呆：manifest 未算 checksum（理论上 BaseSkill.BuildManifest 会算）
		// 时兜底计算，保证幂等判据可用。
		m.Checksum = m.ComputeChecksum()
	}
	return SkillDefinitionRow{
		Code:                m.Name,
		Version:             m.Version,
		Title:               m.Title,
		Description:         m.Description,
		Category:            m.Category,
		Tags:                m.Tags,
		Capabilities:        m.Capabilities,
		RequiredPermissions: m.RequiredPermissions,
		InputSchema:         asObject(m.InputSchema),
		OutputSchema:        asObject(m.OutputSchema),
		Provider:            m.Provider,
		Author:              m.Author,
		Checksum:            m.Checksum,
		Enabled:             true,
		TenantID:            tenantID,
	}, nil
}

func asObject(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

// SyncBuiltinSkillsToStore 把注册表里的全部全局技能（owner=0）按 checksum
// 幂等 seed 进 skill_definitions。任何一行写失败都返回错误——内置技能的
// 持久化基线不完整比没有更糟（管理面板会展示残缺清单）。
// 只读注册表，不改运行时状态。
func SyncBuiltinSkillsToStore(ctx context.Context, registry *SkillRegistry, store DefinitionStore) (synced, changed int, err error) {
	for _, entry := range registry.ListEntries() {
		if owner, ok := registry.OwnerOf(entry.Code); !ok || owner != 0 {
			continue
		}
		skill, err := registry.Get(entry.Code)
		if err != nil {
			return synced, changed, fmt.Errorf("read builtin skill %s: %w", entry.Code, err)
		}
		row, err := rowFromSkill(0, skill)
		if err != nil {
			return synced, changed, fmt.Errorf("flatten builtin skill %s: %w", entry.Code, err)
		}
		if _, didChange, err := store.UpsertDefinition(ctx, row); err != nil {
			return synced, changed, fmt.Errorf("seed builtin skill %s: %w", entry.Code, err)
		} else if didChange {
			changed++
		}
		synced++
	}
	return synced, changed, nil
}

// RestoreCustomSkillsFromStore 把租户自定义技能行恢复进运行时注册表
// （重启恢复）。行 → CustomSkillConfig → NewCustomSkill → RegisterScoped。
// 重复 code（不应发生，唯一约束保证）按已注册处理不中断。
func RestoreCustomSkillsFromStore(ctx context.Context, registry *SkillRegistry, store DefinitionStore) (restored int, err error) {
	rows, err := store.LoadCustomDefinitions(ctx)
	if err != nil {
		return 0, err
	}
	for _, row := range rows {
		cfg := CustomSkillConfig{
			Code:                row.Code,
			Version:             row.Version,
			Title:               row.Title,
			Description:         row.Description,
			Category:            row.Category,
			Tags:                row.Tags,
			Capabilities:        row.Capabilities,
			RequiredPermissions: row.RequiredPermissions,
			Provider:            row.Provider,
			Author:              row.Author,
			InputSchema:         row.InputSchema,
			OutputSchema:        row.OutputSchema,
			Executor:            row.Executor,
		}
		if len(cfg.Tags) == 0 {
			cfg.Tags = []string{"custom"}
		}
		skill := NewCustomSkill(cfg)
		if err := registry.RegisterScoped(int(row.TenantID), skill); err != nil {
			if strings.Contains(err.Error(), "already registered") {
				continue
			}
			return restored, fmt.Errorf("restore custom skill %s (tenant %d): %w",
				row.Code, row.TenantID, err)
		}
		restored++
	}
	return restored, nil
}
