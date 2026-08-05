# 动态自定义字段子系统 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把工单模板 / 服务目录项 的"自定义字段"能力，从五处互不相通的 JSON 列，统一成一套共享的 `field_definitions`（字段定义）+ `field_values`（字段值）子系统，工单先跑通完整闭环（定义 + 提交 + 展示），服务目录项接入定义侧。

**Architecture:** 两张新 Ent 表，用代码库已有的 `entity_type`+`entity_id` 多态关联模式（同 `workflowinstance.go`/`approvalchain.go`）挂到不同业务实体上。两个新的共享 service（`FieldDefinitionService`、`FieldValueService`）封装读写逻辑，工单/服务目录的现有 service 文件改成调用它们，而不是自己解析 JSON 列。`field_values` 冗余快照 name/label/顺序，展示时不用反查定义。

**Tech Stack:** Go + Gin + Ent ORM + PostgreSQL（JSON 列，走已有的 `field.JSON` 惯例）；Next.js + TypeScript + Ant Design（前端两处小修）。

## Global Constraints

- 后端 DTO 响应字段一律 camelCase，Ent Schema 字段一律 snake_case，Mapper 负责转换（CLAUDE.md 硬性规则）。
- Controller 不得直接返回 Ent 模型，必须走 DTO。
- 新表必须带 `tenant_id` 并在所有查询里过滤（租户隔离硬性规则）。
- 不需要保留/迁移历史数据（用户已明确），旧列直接删除。
- `ServiceRequest.form_data` 本次不动（见设计文档"非目标"一节，`form_data` 同时承载系统级已知字段，剥离风险高，本次不做）。
- 每个 Task 完成后运行 `cd itsm-backend && gofmt -l .`（必须无输出）+ 该 Task 触及包的 `go test`；本机没有 Go/Node 工具链时，用 `golang:1.25`/`node:22` 容器 + `--network host` + `http_proxy=http://127.0.0.1:10808` 跑（如果本机原生 `go`/`npm` 已装好则直接用，参考本会话此前已经原生装好 Go 1.25.12 于 `/usr/local/go`）。

参考设计文档：`docs/superpowers/specs/2026-08-05-dynamic-custom-fields-design.md`

---

### Task 1: 新增 `field_definitions` / `field_values` Ent Schema

**Files:**
- Create: `itsm-backend/ent/schema/field_definition.go`
- Create: `itsm-backend/ent/schema/field_value.go`
- Test: `itsm-backend/service/field_definition_service_test.go`（本 Task 只验证 Ent CRUD 能跑通，真正的业务逻辑测试在 Task 2）

**Interfaces:**
- Produces: `ent.FieldDefinition`（字段：TenantID, EntityType, EntityID, Name, Label, FieldType, Required, Options, SortOrder, Config, IsActive, CreatedAt, UpdatedAt）、`ent.FieldValue`（字段：TenantID, EntityType, EntityID, FieldDefinitionID *int, FieldName, FieldLabel, SortOrder, Value json.RawMessage, CreatedAt——`json.RawMessage` 底层就是 `[]byte`，`json.Marshal`/`json.Unmarshal` 可以直接传参不用显式转换）。后续所有 Task 都依赖这两个生成的类型。

- [ ] **Step 1: 写 `field_definition.go`**

```go
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FieldDefinition 动态字段定义：谁（entity_type+entity_id）拥有哪些自定义字段。
// entity_type 取值：ticket_template | service_catalog_item
type FieldDefinition struct {
	ent.Schema
}

func (FieldDefinition) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.String("entity_type").Comment("字段定义归属的实体类型: ticket_template | service_catalog_item").NotEmpty(),
		field.Int("entity_id").Comment("归属实体ID（模板ID 或 服务目录项ID）").Positive(),
		field.String("name").Comment("字段key，如 office_location").NotEmpty(),
		field.String("label").Comment("显示名，如 办公地点").NotEmpty(),
		field.String("field_type").Comment("字段类型: text|textarea|number|date|select|multiselect|boolean|file").NotEmpty(),
		field.Bool("required").Comment("是否必填").Default(false),
		field.JSON("options", []interface{}{}).Comment("select/multiselect 的选项列表 [{label,value}]").Optional(),
		field.Int("sort_order").Comment("显示顺序").Default(0),
		field.JSON("config", map[string]interface{}{}).Comment("预留：校验规则/默认值/显隐条件，v1 不使用").Optional(),
		field.Bool("is_active").Comment("是否启用").Default(true),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
		field.Time("updated_at").Comment("更新时间").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (FieldDefinition) Edges() []ent.Edge { return nil }

func (FieldDefinition) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "entity_type", "entity_id", "sort_order"),
		index.Fields("tenant_id", "entity_type", "entity_id", "name").Unique(),
	}
}
```

- [ ] **Step 2: 写 `field_value.go`**

```go
package schema

import (
	"encoding/json"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// FieldValue 动态字段提交的值，挂在具体的工单/服务请求实例上。
// entity_type 取值：ticket | service_request
// 冗余快照 field_name/field_label/sort_order：定义被改名/删除不影响历史值的展示。
type FieldValue struct {
	ent.Schema
}

func (FieldValue) Fields() []ent.Field {
	return []ent.Field{
		field.Int("tenant_id").Comment("租户ID").Positive(),
		field.String("entity_type").Comment("值归属的实体类型: ticket | service_request").NotEmpty(),
		field.Int("entity_id").Comment("归属实体ID（工单ID 或 服务请求ID）").Positive(),
		field.Int("field_definition_id").Comment("指回 field_definitions，可空，定义被删不影响历史值").Optional().Nillable(),
		field.String("field_name").Comment("提交时快照的字段名").NotEmpty(),
		field.String("field_label").Comment("提交时快照的显示名").NotEmpty(),
		field.Int("sort_order").Comment("提交时快照的顺序").Default(0),
		field.JSON("value", json.RawMessage{}).Comment("字段值，JSON 编码，原始类型（数字/字符串/布尔/数组）。用 json.RawMessage 而不是 []byte——Ent 生成的 assignValues 会对 []byte 目标做 encoding/json 的 base64 特判，导致普通 JSON 值读不回来；json.RawMessage 自带 Marshal/Unmarshal 方法绕过这个坑，同时仍然是 Postgres JSONB 列（不是 field.Bytes 那样的 BYTEA，保留以后建 GIN 索引的可能性）。").Optional(),
		field.Time("created_at").Comment("创建时间").Default(time.Now),
	}
}

func (FieldValue) Edges() []ent.Edge { return nil }

func (FieldValue) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("tenant_id", "entity_type", "entity_id"),
	}
}
```

- [ ] **Step 3: 生成 Ent 代码**

Run: `cd itsm-backend && go generate ./ent`
Expected: 成功退出，`git status` 能看到 `ent/fielddefinition/`、`ent/fieldvalue/`、`ent/field_definition*.go`、`ent/field_value*.go` 等新生成文件，以及 `ent/migrate/schema.go`、`ent/mutation.go`、`ent/runtime/runtime.go`、`ent/client.go` 被修改。

- [ ] **Step 4: 验证编译**

Run: `cd itsm-backend && go build ./...`
Expected: 无错误退出（新表此时还没有任何业务代码引用，纯 schema 新增不应该破坏现有编译）。

- [ ] **Step 5: 写一条 round-trip 冒烟测试，验证表结构可用**

```go
// itsm-backend/service/field_definition_service_test.go
package service

import (
	"context"
	"testing"

	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldDefinitionSchema_RoundTrip(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:field_definition_schema?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	created, err := client.FieldDefinition.Create().
		SetTenantID(1).
		SetEntityType("ticket_template").
		SetEntityID(4).
		SetName("office_location").
		SetLabel("办公地点").
		SetFieldType("text").
		SetRequired(true).
		SetSortOrder(0).
		Save(ctx)
	require.NoError(t, err)
	assert.Equal(t, "office_location", created.Name)

	fetched, err := client.FieldDefinition.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "办公地点", fetched.Label)
}
```

- [ ] **Step 6: 跑测试**

Run: `cd itsm-backend && go test ./service/... -run TestFieldDefinitionSchema_RoundTrip -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
cd itsm-backend
gofmt -l . # 确认无输出
git add ent/schema/field_definition.go ent/schema/field_value.go ent/ service/field_definition_service_test.go
git commit -m "feat(backend): add field_definitions/field_values Ent schema for shared dynamic fields subsystem"
```

---

### Task 2: 共享 Service —— `FieldDefinitionService` + `FieldValueService`

**Files:**
- Create: `itsm-backend/service/field_definition_service.go`
- Create: `itsm-backend/service/field_value_service.go`
- Test: `itsm-backend/service/field_definition_service_test.go`（追加）
- Test: `itsm-backend/service/field_value_service_test.go`

**Interfaces:**
- Consumes: `ent.Client`（Task 1 生成的 `client.FieldDefinition`/`client.FieldValue` builder）
- Produces（后续 Task 3/4/7 都会调用）:
  - `type FieldDefinitionInput struct { Name, Label, FieldType string; Required bool; Options []interface{}; SortOrder int }`
  - `func NewFieldDefinitionService(client *ent.Client) *FieldDefinitionService`
  - `func (s *FieldDefinitionService) ReplaceDefinitions(ctx context.Context, tenantID int, entityType string, entityID int, defs []FieldDefinitionInput) ([]*ent.FieldDefinition, error)`
  - `func (s *FieldDefinitionService) ListDefinitions(ctx context.Context, tenantID int, entityType string, entityID int) ([]*ent.FieldDefinition, error)`
  - `func (s *FieldDefinitionService) DeleteDefinitions(ctx context.Context, tenantID int, entityType string, entityID int) error`
  - `type FieldValueDTO struct { Name, Label string; Value interface{} }`
  - `func NewFieldValueService(client *ent.Client) *FieldValueService`
  - `func (s *FieldValueService) CreateValues(ctx context.Context, tenantID int, defEntityType string, defEntityID int, valueEntityType string, valueEntityID int, values map[string]interface{}) error`
  - `func (s *FieldValueService) ListValues(ctx context.Context, tenantID int, entityType string, entityID int) ([]FieldValueDTO, error)`

- [ ] **Step 1: 写 `FieldDefinitionService` 的失败测试**

```go
// 追加到 itsm-backend/service/field_definition_service_test.go
func TestFieldDefinitionService_ReplaceDefinitions(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:field_definition_replace?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewFieldDefinitionService(client)

	defs, err := svc.ReplaceDefinitions(ctx, 1, "ticket_template", 4, []FieldDefinitionInput{
		{Name: "office_location", Label: "办公地点", FieldType: "text", Required: true, SortOrder: 0},
		{Name: "device_count", Label: "设备数量", FieldType: "number", Required: false, SortOrder: 1},
	})
	require.NoError(t, err)
	require.Len(t, defs, 2)
	assert.Equal(t, "office_location", defs[0].Name)
	assert.Equal(t, "device_count", defs[1].Name)

	// 再次 Replace 应该完全替换掉旧的，而不是追加
	defs2, err := svc.ReplaceDefinitions(ctx, 1, "ticket_template", 4, []FieldDefinitionInput{
		{Name: "device_count", Label: "设备数量", FieldType: "number", SortOrder: 0},
	})
	require.NoError(t, err)
	require.Len(t, defs2, 1)

	listed, err := svc.ListDefinitions(ctx, 1, "ticket_template", 4)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, "device_count", listed[0].Name)
}

func TestFieldDefinitionService_TenantIsolation(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:field_definition_tenant?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewFieldDefinitionService(client)

	_, err := svc.ReplaceDefinitions(ctx, 1, "ticket_template", 4, []FieldDefinitionInput{
		{Name: "office_location", Label: "办公地点", FieldType: "text"},
	})
	require.NoError(t, err)

	// 租户 2 查租户 1 的模板定义，必须查不到
	listed, err := svc.ListDefinitions(ctx, 2, "ticket_template", 4)
	require.NoError(t, err)
	assert.Empty(t, listed)
}
```

- [ ] **Step 2: 跑测试确认失败（编译失败，因为 `FieldDefinitionService` 还不存在）**

Run: `cd itsm-backend && go test ./service/... -run TestFieldDefinitionService -v`
Expected: FAIL，报错 `undefined: NewFieldDefinitionService`

- [ ] **Step 3: 实现 `field_definition_service.go`**

```go
package service

import (
	"context"
	"fmt"

	"itsm-backend/ent"
	"itsm-backend/ent/fielddefinition"
)

// FieldDefinitionService 动态字段定义的共享服务，工单模板、服务目录项共用。
type FieldDefinitionService struct {
	client *ent.Client
}

func NewFieldDefinitionService(client *ent.Client) *FieldDefinitionService {
	return &FieldDefinitionService{client: client}
}

// FieldDefinitionInput 创建/替换字段定义时的输入。
type FieldDefinitionInput struct {
	Name      string
	Label     string
	FieldType string
	Required  bool
	Options   []interface{}
	SortOrder int
}

// ReplaceDefinitions 删除 (tenantID, entityType, entityID) 下所有既有字段定义，
// 按 defs 的顺序重新插入。字段值（field_values）已经快照了 name/label/顺序，
// 不依赖这里的行 ID 存续，所以用"删除重建"而不是逐字段 diff。
// 删除+插入包在一个事务里：field_definitions 上有 (tenant_id, entity_type, entity_id, name)
// 唯一约束，如果 defs 里有重名字段导致某次 insert 撞约束失败，没有事务的话旧定义已经被删掉、
// 新定义插了一半——比"什么都没做"更糟。事务保证要么全部生效，要么整体回滚。
func (s *FieldDefinitionService) ReplaceDefinitions(ctx context.Context, tenantID int, entityType string, entityID int, defs []FieldDefinitionInput) ([]*ent.FieldDefinition, error) {
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, err
	}

	_, err = tx.FieldDefinition.Delete().
		Where(
			fielddefinition.TenantID(tenantID),
			fielddefinition.EntityType(entityType),
			fielddefinition.EntityID(entityID),
		).
		Exec(ctx)
	if err != nil {
		return nil, rollback(tx, err)
	}

	result := make([]*ent.FieldDefinition, 0, len(defs))
	for i, d := range defs {
		sortOrder := d.SortOrder
		if sortOrder == 0 {
			sortOrder = i
		}
		options := d.Options
		if options == nil {
			options = []interface{}{}
		}
		created, err := tx.FieldDefinition.Create().
			SetTenantID(tenantID).
			SetEntityType(entityType).
			SetEntityID(entityID).
			SetName(d.Name).
			SetLabel(d.Label).
			SetFieldType(d.FieldType).
			SetRequired(d.Required).
			SetOptions(options).
			SetSortOrder(sortOrder).
			Save(ctx)
		if err != nil {
			return nil, rollback(tx, err)
		}
		result = append(result, created)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

// rollback aborts tx and wraps the rollback error (if any) around the original cause.
func rollback(tx *ent.Tx, cause error) error {
	if rbErr := tx.Rollback(); rbErr != nil {
		return fmt.Errorf("%w (rollback also failed: %v)", cause, rbErr)
	}
	return cause
}

// ListDefinitions 按 sort_order 返回 (tenantID, entityType, entityID) 下的字段定义。
func (s *FieldDefinitionService) ListDefinitions(ctx context.Context, tenantID int, entityType string, entityID int) ([]*ent.FieldDefinition, error) {
	return s.client.FieldDefinition.Query().
		Where(
			fielddefinition.TenantID(tenantID),
			fielddefinition.EntityType(entityType),
			fielddefinition.EntityID(entityID),
			fielddefinition.IsActive(true),
		).
		Order(ent.Asc(fielddefinition.FieldSortOrder)).
		All(ctx)
}

// DeleteDefinitions 删除 (tenantID, entityType, entityID) 下所有字段定义。
func (s *FieldDefinitionService) DeleteDefinitions(ctx context.Context, tenantID int, entityType string, entityID int) error {
	_, err := s.client.FieldDefinition.Delete().
		Where(
			fielddefinition.TenantID(tenantID),
			fielddefinition.EntityType(entityType),
			fielddefinition.EntityID(entityID),
		).
		Exec(ctx)
	return err
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd itsm-backend && go test ./service/... -run TestFieldDefinitionService -v`
Expected: PASS

- [ ] **Step 5: 写 `FieldValueService` 的失败测试**

```go
// itsm-backend/service/field_value_service_test.go
package service

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldValueService_CreateAndListValues(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:field_value_create?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	defSvc := NewFieldDefinitionService(client)
	valSvc := NewFieldValueService(client)

	_, err := defSvc.ReplaceDefinitions(ctx, 1, "ticket_template", 4, []FieldDefinitionInput{
		{Name: "office_location", Label: "办公地点", FieldType: "text", SortOrder: 0},
		{Name: "device_count", Label: "设备数量", FieldType: "number", SortOrder: 1},
	})
	require.NoError(t, err)

	err = valSvc.CreateValues(ctx, 1, "ticket_template", 4, "ticket", 100, map[string]interface{}{
		"office_location": "北京",
		"device_count":    float64(2),
		"unknown_field":   "should be ignored",
	})
	require.NoError(t, err)

	values, err := valSvc.ListValues(ctx, 1, "ticket", 100)
	require.NoError(t, err)
	require.Len(t, values, 2) // unknown_field 被忽略，不匹配任何定义
	assert.Equal(t, "office_location", values[0].Name)
	assert.Equal(t, "办公地点", values[0].Label)
	assert.Equal(t, "北京", values[0].Value)
	assert.Equal(t, "device_count", values[1].Name)
	assert.Equal(t, float64(2), values[1].Value)
}

func TestFieldValueService_ListValues_EmptyWhenNoValues(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:field_value_empty?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	valSvc := NewFieldValueService(client)

	values, err := valSvc.ListValues(ctx, 1, "ticket", 999)
	require.NoError(t, err)
	assert.Empty(t, values)
}

func TestFieldValueService_CreateValues_SurvivesDefinitionDeletion(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:field_value_survive?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	defSvc := NewFieldDefinitionService(client)
	valSvc := NewFieldValueService(client)

	_, err := defSvc.ReplaceDefinitions(ctx, 1, "ticket_template", 4, []FieldDefinitionInput{
		{Name: "office_location", Label: "办公地点", FieldType: "text"},
	})
	require.NoError(t, err)
	require.NoError(t, valSvc.CreateValues(ctx, 1, "ticket_template", 4, "ticket", 100, map[string]interface{}{
		"office_location": "北京",
	}))

	// 模板字段定义被删除/改名后（这里模拟改名：Replace 成一个新 name）
	_, err = defSvc.ReplaceDefinitions(ctx, 1, "ticket_template", 4, []FieldDefinitionInput{
		{Name: "office_location_v2", Label: "办公地点(新)", FieldType: "text"},
	})
	require.NoError(t, err)

	// 老工单的历史值展示不受影响
	values, err := valSvc.ListValues(ctx, 1, "ticket", 100)
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.Equal(t, "office_location", values[0].Name)
	assert.Equal(t, "办公地点", values[0].Label)
	assert.Equal(t, "北京", values[0].Value)
}
```

- [ ] **Step 6: 跑测试确认失败**

Run: `cd itsm-backend && go test ./service/... -run TestFieldValueService -v`
Expected: FAIL，报错 `undefined: NewFieldValueService`

- [ ] **Step 7: 实现 `field_value_service.go`**

```go
package service

import (
	"context"
	"encoding/json"

	"itsm-backend/ent"
	"itsm-backend/ent/fielddefinition"
	"itsm-backend/ent/fieldvalue"
)

// FieldValueService 动态字段值的共享服务，工单先接入，服务请求以后接入。
type FieldValueService struct {
	client *ent.Client
}

func NewFieldValueService(client *ent.Client) *FieldValueService {
	return &FieldValueService{client: client}
}

// CreateValues 把提交的 values（fieldName -> 原始值）跟 (defEntityType, defEntityID) 下的
// 字段定义匹配，快照 name/label/顺序后写入 field_values，挂在 (valueEntityType, valueEntityID) 上。
// values 里不匹配任何字段定义的 key 会被忽略（例如 presetTypeId 这类路由元数据）。
// 多条 insert 包在一个事务里：中途某一条失败（比如瞬时 DB 错误）不应该留下"插了一半"的
// 半成品提交记录——field_values 代表的是一次完整的表单提交，要么整体成功要么整体不落库。
func (s *FieldValueService) CreateValues(ctx context.Context, tenantID int, defEntityType string, defEntityID int, valueEntityType string, valueEntityID int, values map[string]interface{}) error {
	if len(values) == 0 {
		return nil
	}
	defs, err := s.client.FieldDefinition.Query().
		Where(
			fielddefinition.TenantID(tenantID),
			fielddefinition.EntityType(defEntityType),
			fielddefinition.EntityID(defEntityID),
			fielddefinition.IsActive(true),
		).
		Order(ent.Asc(fielddefinition.FieldSortOrder)).
		All(ctx)
	if err != nil {
		return err
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}

	for _, def := range defs {
		raw, ok := values[def.Name]
		if !ok {
			continue
		}
		encoded, err := json.Marshal(raw)
		if err != nil {
			return rollback(tx, err)
		}
		defID := def.ID
		_, err = tx.FieldValue.Create().
			SetTenantID(tenantID).
			SetEntityType(valueEntityType).
			SetEntityID(valueEntityID).
			SetFieldDefinitionID(defID).
			SetFieldName(def.Name).
			SetFieldLabel(def.Label).
			SetSortOrder(def.SortOrder).
			SetValue(encoded).
			Save(ctx)
		if err != nil {
			return rollback(tx, err)
		}
	}
	return tx.Commit()
}

// FieldValueDTO 展示用的已解析字段值。
type FieldValueDTO struct {
	Name  string      `json:"name"`
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

// ListValues 按提交时快照的顺序返回 (tenantID, entityType, entityID) 的字段值。
func (s *FieldValueService) ListValues(ctx context.Context, tenantID int, entityType string, entityID int) ([]FieldValueDTO, error) {
	rows, err := s.client.FieldValue.Query().
		Where(
			fieldvalue.TenantID(tenantID),
			fieldvalue.EntityType(entityType),
			fieldvalue.EntityID(entityID),
		).
		Order(ent.Asc(fieldvalue.FieldSortOrder)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]FieldValueDTO, 0, len(rows))
	for _, row := range rows {
		var value interface{}
		if len(row.Value) > 0 {
			if err := json.Unmarshal(row.Value, &value); err != nil {
				continue // 损坏的值跳过展示，不阻塞整个响应
			}
		}
		result = append(result, FieldValueDTO{
			Name:  row.FieldName,
			Label: row.FieldLabel,
			Value: value,
		})
	}
	return result, nil
}
```

- [ ] **Step 8: 跑测试确认通过**

Run: `cd itsm-backend && go test ./service/... -run "TestFieldValueService|TestFieldDefinitionService" -v`
Expected: PASS（全部 6 个测试用例）

- [ ] **Step 9: Commit**

```bash
cd itsm-backend
gofmt -l .
git add service/field_definition_service.go service/field_value_service.go service/field_definition_service_test.go service/field_value_service_test.go
git commit -m "feat(backend): add shared FieldDefinitionService/FieldValueService for dynamic fields"
```

---

### Task 3: 工单模板字段定义迁移到 `field_definitions`

**Files:**
- Modify: `itsm-backend/service/ticket_template_service.go`
- Modify: `itsm-backend/service/ticket_service.go:2034`（`toTicketTemplateDTO`）
- Modify: `itsm-backend/ent/schema/tickettemplate.go`（删除 `form_fields`）
- Test: `itsm-backend/service/ticket_template_service_test.go`（如果不存在则新建）

**Interfaces:**
- Consumes: Task 2 的 `FieldDefinitionService.ReplaceDefinitions/ListDefinitions/DeleteDefinitions`
- Produces: `toTicketTemplateDTO` 现在返回的 `dto.TicketTemplate.Fields` 数组形状不变（`[{name,label,type,required,options}]`），下游（controller、前端）不用改。

- [ ] **Step 1: 写失败测试，断言模板创建后字段定义存进了 field_definitions 而不是 form_fields 列**

```go
// itsm-backend/service/ticket_template_service_test.go
package service

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"
	"itsm-backend/ent/fielddefinition"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTicketTemplateService_CreateTemplate_WritesFieldDefinitions(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_template_fields?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewTicketTemplateService(client)

	fieldSvc := NewFieldDefinitionService(client)

	template, err := svc.CreateTemplate(ctx, &CreateTemplateRequest{
		Name:     "网络接入申请",
		Category: "网络",
		TenantID: 1,
		Fields: []FieldDefinitionInput{
			{Name: "office_location", Label: "办公地点", FieldType: "text", Required: true, SortOrder: 0},
			{Name: "device_count", Label: "设备数量", FieldType: "number", SortOrder: 1},
		},
	})
	require.NoError(t, err)

	defs, err := fieldSvc.ListDefinitions(ctx, 1, "ticket_template", template.ID)
	require.NoError(t, err)
	require.Len(t, defs, 2)
	assert.Equal(t, "office_location", defs[0].Name)
	assert.Equal(t, "device_count", defs[1].Name)
}

func TestTicketTemplateService_UpdateTemplate_ReplacesFieldDefinitions(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_template_update_fields?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewTicketTemplateService(client)
	fieldSvc := NewFieldDefinitionService(client)

	template, err := svc.CreateTemplate(ctx, &CreateTemplateRequest{
		Name: "模板", Category: "cat", TenantID: 1,
		Fields: []FieldDefinitionInput{{Name: "a", Label: "A", FieldType: "text"}},
	})
	require.NoError(t, err)

	_, err = svc.UpdateTemplate(ctx, template.ID, &UpdateTemplateRequest{
		Fields: []FieldDefinitionInput{{Name: "b", Label: "B", FieldType: "text"}},
	}, 1)
	require.NoError(t, err)

	defs, err := fieldSvc.ListDefinitions(ctx, 1, "ticket_template", template.ID)
	require.NoError(t, err)
	require.Len(t, defs, 1)
	assert.Equal(t, "b", defs[0].Name)
}

func TestTicketTemplateService_DeleteTemplate_DeletesFieldDefinitions(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_template_delete_fields?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewTicketTemplateService(client)
	fieldSvc := NewFieldDefinitionService(client)

	template, err := svc.CreateTemplate(ctx, &CreateTemplateRequest{
		Name: "模板", Category: "cat", TenantID: 1,
		Fields: []FieldDefinitionInput{{Name: "a", Label: "A", FieldType: "text"}},
	})
	require.NoError(t, err)
	require.NoError(t, svc.DeleteTemplate(ctx, template.ID, 1))

	defs, err := fieldSvc.ListDefinitions(ctx, 1, "ticket_template", template.ID)
	require.NoError(t, err)
	assert.Empty(t, defs)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd itsm-backend && go test ./service/... -run TestTicketTemplateService_CreateTemplate_WritesFieldDefinitions -v`
Expected: FAIL（编译错误：`CreateTemplateRequest` 还没有 `Fields []FieldDefinitionInput` 字段）

- [ ] **Step 3: 修改 `ticket_template_service.go`**

把 `CreateTemplateRequest`/`UpdateTemplateRequest` 的 `FormFields map[string]interface{}` 换成 `Fields []FieldDefinitionInput`，`CreateTemplate`/`UpdateTemplate`/`DeleteTemplate` 改成调用 `FieldDefinitionService`：

```go
package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/ticket"
	"itsm-backend/ent/tickettemplate"
)

// TicketTemplateService 工单模板服务
type TicketTemplateService struct {
	client  *ent.Client
	fieldSvc *FieldDefinitionService
}

// NewTicketTemplateService 创建工单模板服务实例
func NewTicketTemplateService(client *ent.Client) *TicketTemplateService {
	return &TicketTemplateService{client: client, fieldSvc: NewFieldDefinitionService(client)}
}

// CreateTemplate 创建工单模板
func (s *TicketTemplateService) CreateTemplate(ctx context.Context, req *CreateTemplateRequest) (*ent.TicketTemplate, error) {
	workflowStepsBytes, err := json.Marshal(req.WorkflowSteps)
	if err != nil {
		return nil, err
	}

	template, err := s.client.TicketTemplate.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetCategory(req.Category).
		SetPriority(req.Priority).
		SetWorkflowSteps(workflowStepsBytes).
		SetIsActive(req.IsActive).
		SetTenantID(req.TenantID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := s.fieldSvc.ReplaceDefinitions(ctx, req.TenantID, "ticket_template", template.ID, req.Fields); err != nil {
		return nil, err
	}

	return template, nil
}

// GetTemplate 获取工单模板
func (s *TicketTemplateService) GetTemplate(ctx context.Context, id int, tenantID int) (*ent.TicketTemplate, error) {
	return s.client.TicketTemplate.Query().
		Where(
			tickettemplate.IDEQ(id),
			tickettemplate.TenantIDEQ(tenantID),
		).
		Only(ctx)
}

// ListTemplates 获取工单模板列表
func (s *TicketTemplateService) ListTemplates(ctx context.Context, req *ListTemplatesRequest) ([]*ent.TicketTemplate, int, error) {
	query := s.client.TicketTemplate.Query()

	if req.Category != "" {
		query = query.Where(tickettemplate.Category(req.Category))
	}
	if req.IsActive != nil {
		query = query.Where(tickettemplate.IsActive(*req.IsActive))
	}
	if req.TenantID > 0 {
		query = query.Where(tickettemplate.TenantID(req.TenantID))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	if req.SortBy != "" {
		switch req.SortBy {
		case "name":
			if req.SortOrder == "desc" {
				query = query.Order(ent.Desc(tickettemplate.FieldName))
			} else {
				query = query.Order(ent.Asc(tickettemplate.FieldName))
			}
		case "created_at":
			if req.SortOrder == "desc" {
				query = query.Order(ent.Desc(tickettemplate.FieldCreatedAt))
			} else {
				query = query.Order(ent.Asc(tickettemplate.FieldCreatedAt))
			}
		}
	}

	templates, err := query.All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

// UpdateTemplate 更新工单模板
func (s *TicketTemplateService) UpdateTemplate(ctx context.Context, id int, req *UpdateTemplateRequest, tenantID int) (*ent.TicketTemplate, error) {
	update := s.client.TicketTemplate.UpdateOneID(id).
		Where(tickettemplate.TenantIDEQ(tenantID))

	if req.Name != "" {
		update.SetName(req.Name)
	}
	if req.Description != "" {
		update.SetDescription(req.Description)
	}
	if req.Category != "" {
		update.SetCategory(req.Category)
	}
	if req.Priority != "" {
		update.SetPriority(req.Priority)
	}
	if req.WorkflowSteps != nil {
		workflowStepsBytes, err := json.Marshal(req.WorkflowSteps)
		if err != nil {
			return nil, err
		}
		update.SetWorkflowSteps(workflowStepsBytes)
	}
	if req.IsActive != nil {
		update.SetIsActive(*req.IsActive)
	}
	update.SetUpdatedAt(time.Now())

	updated, err := update.Save(ctx)
	if err != nil {
		return nil, err
	}

	if req.Fields != nil {
		if _, err := s.fieldSvc.ReplaceDefinitions(ctx, tenantID, "ticket_template", id, req.Fields); err != nil {
			return nil, err
		}
	}

	return updated, nil
}

// DeleteTemplate 删除工单模板
func (s *TicketTemplateService) DeleteTemplate(ctx context.Context, id int, tenantID int) error {
	count, err := s.client.Ticket.Query().
		Where(
			ticket.TemplateIDEQ(id),
			ticket.TenantIDEQ(tenantID),
		).
		Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("无法删除正在使用的模板")
	}

	if err := s.client.TicketTemplate.DeleteOneID(id).
		Where(tickettemplate.TenantIDEQ(tenantID)).
		Exec(ctx); err != nil {
		return err
	}

	return s.fieldSvc.DeleteDefinitions(ctx, tenantID, "ticket_template", id)
}

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	Name          string                   `json:"name" binding:"required"`
	Description   string                   `json:"description"`
	Category      string                   `json:"category" binding:"required"`
	Priority      string                   `json:"priority"`
	Fields        []FieldDefinitionInput   `json:"fields"`
	WorkflowSteps []map[string]interface{} `json:"workflowSteps"`
	IsActive      bool                     `json:"isActive"`
	TenantID      int                      `json:"tenantId" binding:"required"`
}

// UpdateTemplateRequest 更新模板请求
type UpdateTemplateRequest struct {
	Name          string                   `json:"name"`
	Description   string                   `json:"description"`
	Category      string                   `json:"category"`
	Priority      string                   `json:"priority"`
	Fields        []FieldDefinitionInput   `json:"fields"`
	WorkflowSteps []map[string]interface{} `json:"workflowSteps"`
	IsActive      *bool                    `json:"isActive"`
}

// ListTemplatesRequest 获取模板列表请求
type ListTemplatesRequest struct {
	Page      int    `json:"page" form:"page"`
	PageSize  int    `json:"pageSize" form:"page_size"`
	Category  string `json:"category" form:"category"`
	IsActive  *bool  `json:"isActive" form:"is_active"`
	TenantID  int    `json:"tenantId" form:"tenant_id"`
	SortBy    string `json:"sortBy" form:"sort_by"`
	SortOrder string `json:"sortOrder" form:"sort_order"`
}
```

- [ ] **Step 4: 把 `ticket_service.go` 里 `CreateTicketTemplate`/`UpdateTicketTemplate`/`toTicketTemplateDTO` 的调用方也改掉**

`itsm-backend/service/ticket_service.go:1844`（`CreateTicketTemplate`）里，把：
```go
formFields := createReq.FormFields
if formFields == nil {
    formFields = make(map[string]interface{})
}
if len(createReq.Fields) > 0 {
    formFields["fields"] = createReq.Fields
}
```
以及 `serviceReq := &CreateTemplateRequest{..., FormFields: formFields, ...}` 这一段，改成直接把 `createReq.Fields`（`dto.TicketTemplate.Fields []map[string]interface{}`）转换成 `[]FieldDefinitionInput`：

```go
func toFieldDefinitionInputs(fields []map[string]interface{}) []FieldDefinitionInput {
	result := make([]FieldDefinitionInput, 0, len(fields))
	for i, f := range fields {
		name, _ := f["name"].(string)
		if name == "" {
			continue
		}
		label, _ := f["label"].(string)
		fieldType, _ := f["type"].(string)
		required, _ := f["required"].(bool)
		var options []interface{}
		if raw, ok := f["options"].([]interface{}); ok {
			options = raw
		}
		result = append(result, FieldDefinitionInput{
			Name:      name,
			Label:     label,
			FieldType: fieldType,
			Required:  required,
			Options:   options,
			SortOrder: i,
		})
	}
	return result
}
```

`CreateTicketTemplate` 改成：

```go
func (s *TicketService) CreateTicketTemplate(ctx context.Context, tenantID int, req interface{}) (interface{}, error) {
	createReq, ok := req.(*dto.TicketTemplate)
	if !ok {
		return nil, fmt.Errorf("无效的请求参数类型")
	}
	if s.client == nil {
		return nil, fmt.Errorf("ent client not available for template")
	}
	priority := strings.TrimSpace(createReq.Priority)
	if priority == "" {
		priority = "medium"
	}

	templateService := NewTicketTemplateService(s.client)
	serviceReq := &CreateTemplateRequest{
		Name:          createReq.Name,
		Description:   createReq.Description,
		Category:      createReq.Category,
		Priority:      priority,
		Fields:        toFieldDefinitionInputs(createReq.Fields),
		WorkflowSteps: createReq.WorkflowSteps,
		IsActive:      true,
		TenantID:      tenantID,
	}
	template, err := templateService.CreateTemplate(ctx, serviceReq)
	if err != nil {
		return nil, err
	}
	return s.toTicketTemplateDTO(ctx, template)
}
```

`UpdateTicketTemplate`（原 `:1884`）同样把 `formFields`/`FormFields` 那段改成 `Fields: toFieldDefinitionInputs(updateReq.Fields)`（`updateReq.Fields == nil` 时传 `nil`，`UpdateTemplate` 已经用 `req.Fields != nil` 判断是否要替换）。

`toTicketTemplateDTO`（原 `:2034`）整个函数替换成：

```go
func (s *TicketService) toTicketTemplateDTO(ctx context.Context, template *ent.TicketTemplate) (*dto.TicketTemplate, error) {
	defs, err := NewFieldDefinitionService(s.client).ListDefinitions(ctx, template.TenantID, "ticket_template", template.ID)
	if err != nil {
		s.logger.Warnw("加载模板字段定义失败", "error", err, "template_id", template.ID)
		defs = nil
	}
	fields := make([]map[string]interface{}, 0, len(defs))
	for _, d := range defs {
		fields = append(fields, map[string]interface{}{
			"name":     d.Name,
			"label":    d.Label,
			"type":     d.FieldType,
			"required": d.Required,
			"options":  d.Options,
		})
	}

	var workflowSteps []map[string]interface{}
	if len(template.WorkflowSteps) > 0 {
		if err := json.Unmarshal(template.WorkflowSteps, &workflowSteps); err != nil {
			s.logger.Warnw("反序列化工作流步骤失败", "error", err, "template_id", template.ID)
			workflowSteps = nil
		}
	}

	return &dto.TicketTemplate{
		ID:            template.ID,
		Name:          template.Name,
		Description:   template.Description,
		Category:      template.Category,
		Priority:      template.Priority,
		Fields:        fields,
		WorkflowSteps: workflowSteps,
		IsActive:      template.IsActive,
		CreatedAt:     template.CreatedAt,
		UpdatedAt:     template.UpdatedAt,
	}, nil
}
```

注意 `toTicketTemplateDTO` 签名多了 `ctx context.Context` 参数——搜索 `ticket_service.go` 里所有调用 `s.toTicketTemplateDTO(` 的地方（`GetTicketTemplates`、`GetTicketTemplate`、`CopyTicketTemplate` 等），都要在调用处补上 `ctx`（它们本来就在有 `ctx` 的方法里，直接传本方法的 `ctx` 参数即可）。`dto.TicketTemplate` struct 可以删掉 `FormFields map[string]interface{}` 字段（不再使用）。

- [ ] **Step 5: 删除 `TicketTemplate.form_fields` Ent 字段**

`itsm-backend/ent/schema/tickettemplate.go` 删除：
```go
field.JSON("form_fields", []byte{}).
    Comment("表单字段定义").
    Optional(),
```

Run: `cd itsm-backend && go generate ./ent`

- [ ] **Step 6: 全量编译 + 跑测试**

Run: `cd itsm-backend && go build ./... && go test ./service/... -run TicketTemplate -v`
Expected: 编译通过（如果有其它文件还引用 `template.FormFields`/`CreateTemplateRequest.FormFields`，编译器会报出来，逐个改成上面的新字段），测试全部 PASS（包括 Step 1 里的 3 个测试）

- [ ] **Step 7: Commit**

```bash
cd itsm-backend
gofmt -l .
git add ent/schema/tickettemplate.go ent/ service/ticket_template_service.go service/ticket_template_service_test.go service/ticket_service.go dto/ticket_dto.go
git commit -m "refactor(backend): migrate ticket template field definitions to field_definitions table"
```

---

### Task 4: 工单字段值迁移到 `field_values`（撤销 `Ticket.custom_field_values`），合并 Ticket→Response Mapper

**Files:**
- Modify: `itsm-backend/ent/schema/ticket.go`（删除 `custom_field_values`）
- Modify: `itsm-backend/repository/ticket/model.go`（删除 `CustomFieldValues` 字段）
- Modify: `itsm-backend/repository/ticket/repository_impl.go`（删除相关读写）
- Modify: `itsm-backend/service/ticket_service.go`（`CreateTicket`、`extractCustomFieldValues`、`toEntTicket`、`toTicketResponse` → 导出为 `ToTicketResponse` + 新增 `ToTicketResponseWithCustomFields`）
- Modify: `itsm-backend/controller/ticket_controller.go`（`ticketToResponse`/`ticketListToResponse` 改成薄封装，14 处调用点改签名）
- Modify: `itsm-backend/dto/ticket_dto.go`（`TicketResponse.CustomFieldValues` 类型、新增 `CustomFieldValueResponse`）、`itsm-backend/dto/mappers.go`（删除死代码 `ToTicketResponse`/`ToTicketResponseWithUsers`/`ToTicketResponseList`）
- Test: `itsm-backend/repository/ticket/repository_test.go`、`itsm-backend/service/ticket_service_test.go`

**Interfaces:**
- Consumes: Task 2 的 `FieldValueService.CreateValues/ListValues`
- Produces: `service.ToTicketResponse(ctx, t) *dto.TicketResponse`（列表路径，不查字段值）、`service.ToTicketResponseWithCustomFields(ctx, client, t) *dto.TicketResponse`（详情/创建路径，查一次字段值）——工单响应转换的唯一入口，取代原来 controller/service/dto 三处各自维护的转换函数。`TicketResponse.CustomFieldValues` 现在是 `[]dto.CustomFieldValueResponse`（JSON: `customFields: [{name,label,value}]`），而不是 `map[string]interface{}`——这是一个响应形状变化，Task 8/9 的前端改造依赖这个新形状。

- [ ] **Step 1: 撤销上一版实现——先删 `Ticket.custom_field_values` 相关代码**

`itsm-backend/ent/schema/ticket.go` 删除：
```go
field.JSON("custom_field_values", map[string]interface{}{}).
    Comment("工单创建时提交的自定义字段值（key 为模板字段 name）").
    Optional(),
```

`itsm-backend/repository/ticket/model.go`：
- `Ticket` struct 删除 `CustomFieldValues map[string]interface{}` 字段
- `CreateParams` struct 删除 `CustomFieldValues map[string]interface{}` 字段

`itsm-backend/repository/ticket/repository_impl.go`：
- `Create()` 里删除：
```go
if len(params.CustomFieldValues) > 0 {
    builder.SetCustomFieldValues(params.CustomFieldValues)
}
```
- `toDomainModel()` 里删除：
```go
if len(e.CustomFieldValues) > 0 {
    t.CustomFieldValues = e.CustomFieldValues
}
```

`itsm-backend/service/ticket_service.go`：
- `CreateTicket()` 的 `params := &ticket.CreateParams{...}` 里删除 `CustomFieldValues: extractCustomFieldValues(req.FormFields),` 这一行（`extractCustomFieldValues` 函数本身保留，Step 3 会复用它）
- `toEntTicket()` 里删除 `entTicket.CustomFieldValues = t.CustomFieldValues`
- `toTicketResponse()`（Step 9 会把它导出成包级函数 `ToTicketResponse`，此时还是私有方法，先原地删）里删除：
```go
if len(t.CustomFieldValues) > 0 {
    resp.CustomFieldValues = t.CustomFieldValues
}
```

`itsm-backend/dto/ticket_dto.go`：`TicketResponse` 的 `CustomFieldValues map[string]interface{}` 字段先保留（Step 4 会改成新类型）。

`itsm-backend/controller/ticket_controller.go` 的 `ticketToResponse()` 里同样删除 `if len(t.CustomFieldValues) > 0 { resp.CustomFieldValues = t.CustomFieldValues }`（Step 10 会整体重写这个函数，这里先删掉引用避免编译失败）。

Run: `cd itsm-backend && go generate ./ent && go build ./...`
Expected: 应该能编译通过（这一步纯删除，没有新增依赖）

- [ ] **Step 2: 写失败测试，断言创建工单后字段值进了 field_values**

```go
// 追加到 itsm-backend/service/ticket_service_test.go
func TestTicketService_CreateTicketPersistsCustomFieldValues(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_create_custom_fields_v2?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant := createTicketAssociationTenant(t, ctx, client, "create-custom-fields-v2")
	requester := createTicketAssociationUser(t, ctx, client, tenant.ID, "create-custom-fields-v2-requester")
	template, err := client.TicketTemplate.Create().
		SetName("网络接入").SetCategory("网络").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	_, err = NewFieldDefinitionService(client).ReplaceDefinitions(ctx, tenant.ID, "ticket_template", template.ID, []FieldDefinitionInput{
		{Name: "office_location", Label: "办公地点", FieldType: "text", SortOrder: 0},
	})
	require.NoError(t, err)

	service := NewTicketServiceForTest(client, zaptest.NewLogger(t).Sugar())
	created, err := service.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title: "网络接入申请", Description: "测试", Priority: "medium",
		RequesterID: requester.ID, TemplateID: &template.ID,
		FormFields: map[string]interface{}{
			"values": map[string]interface{}{"office_location": "北京"},
		},
	}, tenant.ID)
	require.NoError(t, err)

	values, err := NewFieldValueService(client).ListValues(ctx, tenant.ID, "ticket", created.ID)
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.Equal(t, "office_location", values[0].Name)
	assert.Equal(t, "北京", values[0].Value)
}
```

- [ ] **Step 3: 跑测试确认失败**

Run: `cd itsm-backend && go test ./service/... -run TestTicketService_CreateTicketPersistsCustomFieldValues -v`
Expected: FAIL（`field_values` 表里没有数据，因为 `CreateTicket` 还没有调用 `FieldValueService`）

- [ ] **Step 4: 在 `CreateTicket` 里接入 `FieldValueService`**

`itsm-backend/service/ticket_service.go` 的 `CreateTicket()`，在 `tkt, err := s.repo.Create(ctx, params, tenantID)` 成功之后（大约原 `:156` 附近，SLA 计算之前）加：

```go
if req.TemplateID != nil {
    if fieldValues := extractCustomFieldValues(req.FormFields); len(fieldValues) > 0 {
        if err := NewFieldValueService(s.client).CreateValues(ctx, tenantID, "ticket_template", *req.TemplateID, "ticket", tkt.ID, fieldValues); err != nil {
            s.logger.Warnw("Failed to persist custom field values", "error", err, "ticket_id", tkt.ID)
        }
    }
}
```

（用 `Warnw` 而不是让整个创建失败——字段值写入失败不应该阻塞工单创建本身，跟现有 SLA 计算失败的处理方式一致。）

- [ ] **Step 5: 跑测试确认通过**

Run: `cd itsm-backend && go test ./service/... -run TestTicketService_CreateTicketPersistsCustomFieldValues -v`
Expected: PASS

- [ ] **Step 6: 改 `TicketResponse.CustomFieldValues` 类型 + 读取逻辑**

`itsm-backend/dto/ticket_dto.go` 的 `TicketResponse`：
```go
CustomFieldValues []service.FieldValueDTO `json:"customFields,omitempty"`
```
（需要在文件顶部 import `"itsm-backend/service"`——检查是否会跟 `dto` 包已有 import 冲突产生循环依赖：`dto` 目前不 import `service`，`service` 大量 import `dto`，如果 `dto` 反过来 import `service` 会成环。**改成在 `service` 包内构造 `TicketResponse.CustomFieldValues`，`dto.TicketResponse` 里这个字段类型改成不依赖 `service` 包的等价 struct**：

```go
// dto/ticket_dto.go 新增
type CustomFieldValueResponse struct {
	Name  string      `json:"name"`
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}
```

`TicketResponse.CustomFieldValues` 类型改成 `[]CustomFieldValueResponse`。

- [ ] **Step 7: 写失败测试，断言字段值出现在响应里，且 controller 和 service 两条路径结果一致**

```go
// 追加到 itsm-backend/service/ticket_service_test.go
func TestToTicketResponse_IncludesCustomFieldValuesOrdered(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:to_ticket_response_fields?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant := createTicketAssociationTenant(t, ctx, client, "to-response-fields")
	requester := createTicketAssociationUser(t, ctx, client, tenant.ID, "to-response-fields-requester")
	template, err := client.TicketTemplate.Create().
		SetName("t").SetCategory("c").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	_, err = NewFieldDefinitionService(client).ReplaceDefinitions(ctx, tenant.ID, "ticket_template", template.ID, []FieldDefinitionInput{
		{Name: "office_location", Label: "办公地点", FieldType: "text", SortOrder: 0},
		{Name: "device_count", Label: "设备数量", FieldType: "number", SortOrder: 1},
	})
	require.NoError(t, err)

	svc := NewTicketServiceForTest(client, zaptest.NewLogger(t).Sugar())
	created, err := svc.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title: "t", Description: "d", Priority: "medium", RequesterID: requester.ID, TemplateID: &template.ID,
		FormFields: map[string]interface{}{"values": map[string]interface{}{
			"office_location": "北京", "device_count": float64(2),
		}},
	}, tenant.ID)
	require.NoError(t, err)

	resp := ToTicketResponse(ctx, created)
	require.Len(t, resp.CustomFieldValues, 2)
	assert.Equal(t, "office_location", resp.CustomFieldValues[0].Name)
	assert.Equal(t, "办公地点", resp.CustomFieldValues[0].Label)
	assert.Equal(t, "device_count", resp.CustomFieldValues[1].Name)
}
```

（这里直接测 `ToTicketResponse` 而不是分别测 controller/service 两条路径——Step 3 之后 controller 那条路径只是薄封装，行为由这一个函数保证。）

- [ ] **Step 8: 跑测试确认失败**

Run: `cd itsm-backend && go test ./service/... -run TestToTicketResponse_IncludesCustomFieldValuesOrdered -v`
Expected: FAIL（`undefined: ToTicketResponse`，目前还是私有方法 `s.toTicketResponse`）

- [ ] **Step 9: 把 `toTicketResponse` 改成导出的包级函数，接上 field_values**

`itsm-backend/service/ticket_service.go`，把原来的方法：
```go
func (s *TicketService) toTicketResponse(t *ticket.Ticket) *dto.TicketResponse {
```
改成包级函数（不再需要 `s` 接收者，需要 `client *ent.Client` 和 `ctx`）：

```go
// ToTicketResponse 是工单领域模型转 DTO 响应的唯一入口，创建/详情/列表所有路径都应该调用它。
func ToTicketResponse(ctx context.Context, t *ticket.Ticket) *dto.TicketResponse {
	if t == nil {
		return nil
	}
	resp := &dto.TicketResponse{
		ID:           t.ID,
		TicketNumber: t.TicketNumber,
		Title:        t.Title,
		Description:  t.Description,
		Status:       string(t.Status),
		Priority:     string(t.Priority),
		Type:         string(t.Type),
		RequesterID:  t.RequesterID,
		TenantID:     t.TenantID,
		Version:      t.Version,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}

	if t.AssigneeID != nil {
		resp.AssigneeID = *t.AssigneeID
	}
	if t.CategoryID != nil {
		resp.CategoryID = *t.CategoryID
	}
	if t.DepartmentID != nil {
		resp.DepartmentID = *t.DepartmentID
	}
	if t.ParentTicketID != nil {
		resp.ParentTicketID = *t.ParentTicketID
	}
	resp.TemplateID = t.TemplateID
	if t.Resolution != nil {
		resp.Resolution = *t.Resolution
	}
	resp.ResolvedAt = t.ResolvedAt
	resp.ClosedAt = t.ClosedAt
	resp.FirstResponseAt = t.FirstResponseAt
	resp.SLAResponseDeadline = t.SLAResponseDeadline
	resp.SLAResolutionDeadline = t.SLAResolutionDeadline

	return resp
}

// ToTicketResponseWithCustomFields 在 ToTicketResponse 基础上额外查一次 field_values。
// 只用于单条工单详情/创建响应，列表接口不调用（避免 N+1）。
func ToTicketResponseWithCustomFields(ctx context.Context, client *ent.Client, t *ticket.Ticket) *dto.TicketResponse {
	resp := ToTicketResponse(ctx, t)
	if resp == nil || client == nil {
		return resp
	}
	values, err := NewFieldValueService(client).ListValues(ctx, t.TenantID, "ticket", t.ID)
	if err != nil || len(values) == 0 {
		return resp
	}
	resp.CustomFieldValues = make([]dto.CustomFieldValueResponse, 0, len(values))
	for _, v := range values {
		resp.CustomFieldValues = append(resp.CustomFieldValues, dto.CustomFieldValueResponse{
			Name: v.Name, Label: v.Label, Value: v.Value,
		})
	}
	return resp
}
```

把原来唯一调用点（`GetTicketsResponse` 的列表构建处，原 `:864`）改成 `ToTicketResponse(ctx, t)`（列表不查字段值，保持原样不引入 N+1）。

- [ ] **Step 10: 把 controller 的 `ticketToResponse`/`ticketListToResponse` 改成薄封装**

`itsm-backend/controller/ticket_controller.go`：

```go
// ticketToResponse 工单详情/创建响应——会额外查一次自定义字段值。
func (tc *TicketController) ticketToResponse(ctx context.Context, t *ticket.Ticket) *dto.TicketResponse {
	return service.ToTicketResponseWithCustomFields(ctx, tc.db, t)
}

func (tc *TicketController) ticketListToResponse(ctx context.Context, ts []*ticket.Ticket) []*dto.TicketResponse {
	result := make([]*dto.TicketResponse, 0, len(ts))
	for _, t := range ts {
		if r := service.ToTicketResponse(ctx, t); r != nil {
			result = append(result, r)
		}
	}
	return result
}
```

这两个函数从包级函数变成了 `TicketController` 的方法（因为需要 `tc.db` 这个 `*ent.Client`）——需要检查 `TicketController` struct 是否已经有一个 `*ent.Client` 字段可用（本文件顶部 `db *ent.Client` 字段，`NewTicketController` 构造函数里已经注入，见文件开头的 struct 定义）。所有调用点从 `ticketToResponse(ticket)` 改成 `tc.ticketToResponse(c.Request.Context(), ticket)`，`ticketListToResponse(tickets)` 改成 `tc.ticketListToResponse(c.Request.Context(), tickets)`——一共 14 处调用点（`grep -n "ticketToResponse(\|ticketListToResponse(" itsm-backend/controller/ticket_controller.go` 列出的所有行）。

- [ ] **Step 11: 删除死代码**

`itsm-backend/dto/mappers.go` 删除整个 `ToTicketResponse`、`ToTicketResponseWithUsers`、`ToTicketResponseList` 三个函数（已确认代码库里没有任何调用点）。

- [ ] **Step 12: 跑测试确认通过**

Run: `cd itsm-backend && go build ./... && go test ./service/... ./controller/... ./dto/... -v 2>&1 | tail -60`
Expected: 编译通过，所有测试 PASS（包括 Step 2 里断言 field_values 落库的测试、Step 7 新加的 `TestToTicketResponse_IncludesCustomFieldValuesOrdered`，以及之前所有既有测试不受影响）

- [ ] **Step 13: Commit**

```bash
cd itsm-backend
gofmt -l .
git add ent/schema/ticket.go ent/ repository/ticket/ service/ticket_service.go service/ticket_service_test.go controller/ticket_controller.go dto/ticket_dto.go dto/mappers.go
git commit -m "refactor(backend): migrate ticket custom field values to field_values table, unify ticket response mapper"
```

---

### Task 6: 清理 `TicketType.custom_fields`（确认无前端调用方，直接删除）

**Files:**
- Modify: `itsm-backend/ent/schema/ticket_type.go`
- Modify: `itsm-backend/service/ticket_type_service.go`
- Modify: `itsm-backend/dto/ticket_type_dto.go`
- Test: `itsm-backend/dto/ticket_type_dto_test.go`（如果测试引用了 `CustomFields` 需要同步改）

**Interfaces:**
- 无新增接口——纯删除。已确认 `/api/v1/ticket-types` 虽然注册在 `router/router.go:1755,1763`，但前端 `itsm-frontend/src/` 没有任何调用（`grep -rl "ticket-types" src` 零结果），删除不影响任何已上线功能。

- [ ] **Step 1: 删除 Ent 字段**

`itsm-backend/ent/schema/ticket_type.go` 删除：
```go
field.JSON("custom_fields", map[string]interface{}{}),
```

Run: `cd itsm-backend && go generate ./ent`

- [ ] **Step 2: 删除 service/dto 里的引用**

`itsm-backend/service/ticket_type_service.go` 删除 `toCustomFieldsMap`/`convertCustomFields` 两个函数，以及所有 `.CustomFields`/`SetCustomFields` 相关代码行（`grep -n "CustomFields" itsm-backend/service/ticket_type_service.go` 列出的所有行，共 6 处）。

`itsm-backend/dto/ticket_type_dto.go` 删除 `CustomFields []CustomFieldDefinition`/`CustomFields *[]CustomFieldDefinition` 字段（3 处），以及 `CustomFieldDefinition` 这个 struct 定义本身（如果删完字段后它没有别的引用了）。

- [ ] **Step 3: 编译确认没有遗漏引用**

Run: `cd itsm-backend && go build ./...`
Expected: 编译失败会精确指出所有还在用 `CustomFields`/`CustomFieldDefinition` 的地方（包括测试文件），逐个删掉

- [ ] **Step 4: 跑受影响的测试**

Run: `cd itsm-backend && go test ./service/... ./dto/... -run TicketType -v`
Expected: PASS（如果 `ticket_type_dto_test.go` 里有断言 `CustomFields` 的用例，删掉那个断言分支，不要删整个测试函数——先看一眼这个文件里还测了什么其它字段）

- [ ] **Step 5: Commit**

```bash
cd itsm-backend
gofmt -l .
git add ent/schema/ticket_type.go ent/ service/ticket_type_service.go dto/ticket_type_dto.go dto/ticket_type_dto_test.go
git commit -m "refactor(backend): remove unused TicketType.custom_fields (superseded by field_definitions, zero frontend callers)"
```

---

### Task 7: 服务目录项字段定义迁移到 `field_definitions`

**Files:**
- Modify: `itsm-backend/service/service_catalog_item_service.go`
- Modify: `itsm-backend/dto/service_catalog_dto.go`
- Modify: `itsm-backend/ent/schema/service_catalog_item.go`（删除 `form_schema`）
- Modify: `itsm-backend/ent/schema/servicecatalog.go`（删除 `form_schema`，已确认完全无引用）
- Test: `itsm-backend/service/service_catalog_item_service_test.go`

**Interfaces:**
- Consumes: Task 2 的 `FieldDefinitionService`（entity_type=`"service_catalog_item"`）
- Produces: `dto.ServiceCatalogItemResponse.Fields []map[string]interface{}`（新字段，跟 `dto.TicketTemplate.Fields` 同形状），`FormSchema` 字段删除。这次**不**接服务请求的值提交（见设计文档，`ServiceRequest`↔`ServiceCatalogItem` 目前没有关联，值采集留到后续单独立项）。

- [ ] **Step 1: 写失败测试**

```go
// itsm-backend/service/service_catalog_item_service_test.go
package service

import (
	"context"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestServiceCatalogItemService_CreateAndGet_PersistsFieldDefinitions(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:catalog_item_fields?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant, err := client.Tenant.Create().SetName("t").SetCode("catalog-item-fields").SetDomain("d.test").SetStatus("active").Save(ctx)
	require.NoError(t, err)
	catalog, err := client.ServiceCatalog.Create().SetName("云资源").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)

	svc := NewServiceCatalogItemService(client, zaptest.NewLogger(t).Sugar())
	created, err := svc.CreateServiceCatalogItem(ctx, &dto.CreateServiceCatalogItemRequest{
		CatalogID: catalog.ID,
		Name:      "云主机申请",
		Fields: []FieldDefinitionInput{
			{Name: "cpu_cores", Label: "CPU核数", FieldType: "number", SortOrder: 0},
		},
	}, tenant.ID)
	require.NoError(t, err)
	require.Len(t, created.Fields, 1)
	assert.Equal(t, "cpu_cores", created.Fields[0]["name"])

	fetched, err := svc.GetServiceCatalogItem(ctx, created.ID, tenant.ID)
	require.NoError(t, err)
	require.Len(t, fetched.Fields, 1)
	assert.Equal(t, "CPU核数", fetched.Fields[0]["label"])
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd itsm-backend && go test ./service/... -run TestServiceCatalogItemService_CreateAndGet_PersistsFieldDefinitions -v`
Expected: FAIL（编译失败：`CreateServiceCatalogItemRequest` 还没有 `Fields` 字段）

- [ ] **Step 3: 改 DTO**

`itsm-backend/dto/service_catalog_dto.go`：把三处 `FormSchema map[string]interface{}`/`FormSchema *map[string]interface{}` 分别改成：
- `CreateServiceCatalogItemRequest.Fields []service.FieldDefinitionInput`（注意跟 Task 3 一样要检查 import 环——`dto` 包不能 import `service`；这里改成在 `service_catalog_item_service.go` 里单独定义一个跟 `FieldDefinitionInput` 字段一致的请求专用类型，或者直接把 `Fields []map[string]interface{}` 留在 dto 层，在 service 层转换，跟 Task 3 处理 `dto.TicketTemplate.Fields` 的方式保持一致）：

```go
// dto/service_catalog_dto.go
Fields []map[string]interface{} `json:"fields"`
```

三处（`CreateServiceCatalogItemRequest`、`UpdateServiceCatalogItemRequest` 用 `*[]map[string]interface{}`、`ServiceCatalogItemResponse`）都改成这个形状，删除 `FormSchema`。

- [ ] **Step 4: 改 `service_catalog_item_service.go`**

复用 Task 3 写的 `toFieldDefinitionInputs` 转换函数（同一个 `service` 包，直接调用，不用重复定义）：

```go
// CreateServiceCatalogItem 创建服务目录项
func (s *ServiceCatalogItemService) CreateServiceCatalogItem(ctx context.Context, req *dto.CreateServiceCatalogItemRequest, tenantID int) (*dto.ServiceCatalogItemResponse, error) {
	_, err := s.client.ServiceCatalog.Get(ctx, req.CatalogID)
	if err != nil {
		return nil, fmt.Errorf("service catalog not found: %w", err)
	}

	item, err := s.client.ServiceCatalogItem.Create().
		SetName(req.Name).
		SetDescription(req.Description).
		SetDetails(req.Details).
		SetCategory(req.Category).
		SetIcon(req.Icon).
		SetNillableSLAID(&req.SlaID).
		SetNillableApprovalChainID(&req.ApprovalChainID).
		SetRequiresApproval(req.RequiresApproval).
		SetEstimatedDays(req.EstimatedDays).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create service catalog item", "error", err, "name", req.Name)
		return nil, fmt.Errorf("failed to create service item: %w", err)
	}

	if _, err := NewFieldDefinitionService(s.client).ReplaceDefinitions(ctx, tenantID, "service_catalog_item", item.ID, toFieldDefinitionInputs(req.Fields)); err != nil {
		return nil, fmt.Errorf("failed to save field definitions: %w", err)
	}

	s.logger.Infow("Service catalog item created", "item_id", item.ID, "name", item.Name)
	return s.toItemResponse(ctx, item), nil
}
```

`UpdateServiceCatalogItem` 在现有 `if req.FormSchema != nil { update.SetFormSchema(*req.FormSchema) }` 那块位置改成：
```go
if req.Fields != nil {
    if _, err := NewFieldDefinitionService(s.client).ReplaceDefinitions(ctx, tenantID, "service_catalog_item", id, toFieldDefinitionInputs(*req.Fields)); err != nil {
        return nil, fmt.Errorf("failed to save field definitions: %w", err)
    }
}
```

`DeleteServiceCatalogItem` 现在是软删除（`SetIsActive(false)`），字段定义不用跟着删（软删除的记录还可能需要恢复展示）。

`toItemResponse` 改成需要查字段定义，签名加 `ctx`：
```go
func (s *ServiceCatalogItemService) toItemResponse(ctx context.Context, item *ent.ServiceCatalogItem) *dto.ServiceCatalogItemResponse {
	fields := make([]map[string]interface{}, 0)
	if defs, err := NewFieldDefinitionService(s.client).ListDefinitions(ctx, item.TenantID, "service_catalog_item", item.ID); err == nil {
		for _, d := range defs {
			fields = append(fields, map[string]interface{}{
				"name": d.Name, "label": d.Label, "type": d.FieldType,
				"required": d.Required, "options": d.Options,
			})
		}
	}
	return &dto.ServiceCatalogItemResponse{
		ID: item.ID, Name: item.Name, Description: item.Description, Details: item.Details,
		Category: item.Category, Icon: item.Icon, Fields: fields,
		SlaID: item.SLAID, ApprovalChainID: item.ApprovalChainID, IsActive: item.IsActive,
		RequiresApproval: item.RequiresApproval, EstimatedDays: item.EstimatedDays,
		TenantID: item.TenantID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}
```

所有调用 `s.toItemResponse(item)` 的地方（`GetServiceCatalogItem`、`ListServiceCatalogItems`、`UpdateServiceCatalogItem`）都要加上 `ctx` 参数。

- [ ] **Step 5: 跑测试确认通过**

Run: `cd itsm-backend && go test ./service/... -run ServiceCatalogItem -v`
Expected: PASS

- [ ] **Step 6: 删除 `form_schema` Ent 字段**

`itsm-backend/ent/schema/service_catalog_item.go` 删除 `form_schema` 字段块；`itsm-backend/ent/schema/servicecatalog.go` 同样删除 `form_schema` 字段块（已确认零引用，可以直接删，不用先搜索调用方）。

Run: `cd itsm-backend && go generate ./ent && go build ./...`
Expected: 编译通过

- [ ] **Step 7: Commit**

```bash
cd itsm-backend
gofmt -l .
git add ent/schema/service_catalog_item.go ent/schema/servicecatalog.go ent/ service/service_catalog_item_service.go service/service_catalog_item_service_test.go dto/service_catalog_dto.go
git commit -m "refactor(backend): migrate service catalog item field definitions to field_definitions table"
```

---

### Task 8: 前端 —— 修 `getTemplates()` bug，接上创建工单页面的字段输入

**Files:**
- Modify: `itsm-frontend/src/lib/api/ticket-api.ts`
- Modify: `itsm-frontend/src/app/(main)/tickets/templates/page.tsx`
- Modify: `itsm-frontend/src/app/(main)/tickets/create/page.tsx`

**Interfaces:**
- Consumes: 后端新响应形状 `GET /tickets/templates` → `{ templates: [...], page, pageSize }`（Task 3 之后，`fields` 数组里每项仍是 `{name,label,type,required,options}`，形状没变，只是来源换了）

- [ ] **Step 1: 修 `getTemplates()` 类型声明和调用方**

`itsm-frontend/src/lib/api/ticket-api.ts:603-620`：

```ts
// Get ticket templates
static async getTemplates(params?: {
  page?: number;
  pageSize?: number;
  category?: string;
}): Promise<{
  templates: Array<{
    id: number;
    name: string;
    description: string;
    category: string;
    priority: string;
    fields?: Array<Record<string, unknown>>;
    workflowSteps?: Array<Record<string, unknown>>;
    isActive: boolean;
    createdAt: string;
    updatedAt: string;
  }>;
  page: number;
  pageSize: number;
}> {
  return httpClient.get('/api/v1/tickets/templates', params);
}
```

- [ ] **Step 2: 修 `templates/page.tsx` 的读取点**

`itsm-frontend/src/app/(main)/tickets/templates/page.tsx:134-141`，把：
```ts
const response = await TicketApi.getTemplates({...});
const apiTemplates: TicketTemplate[] = (response.items || []).map(...)
```
改成：
```ts
const response = await TicketApi.getTemplates({...});
const apiTemplates: TicketTemplate[] = (response.templates || []).map(...)
```

- [ ] **Step 3: 修 `create/page.tsx` 的读取点**

`itsm-frontend/src/app/(main)/tickets/create/page.tsx:113-116`，把：
```ts
TicketApi.getTemplates({ page: 1, pageSize: 100 })
  .then(res => {
    if (res && res.items && Array.isArray(res.items)) {
```
改成：
```ts
TicketApi.getTemplates({ page: 1, pageSize: 100 })
  .then(res => {
    if (res && res.templates && Array.isArray(res.templates)) {
```
以及紧跟着的 `res.items.map(...)` 改成 `res.templates.map(...)`（同一个 `useEffect` 块内，第 116 行附近）。

- [ ] **Step 4: 类型检查**

Run: `cd itsm-frontend && npx tsc --noEmit`
Expected: 无错误

- [ ] **Step 5: 手动验证（浏览器）**

前置：本机原生跑着 `go run`/`next dev`（参考本会话已经搭好的本机开发环境：后端 `http://localhost:8090`，前端 `http://localhost:3010`）。

1. 打开 `http://localhost:3010/tickets/templates`，新建一个模板，添加两个字段（如"办公地点"=text=必填，"设备数量"=number），保存。
2. 打开 `http://localhost:3010/tickets/create`，确认刚才建的模板出现在类型选择卡片列表里，点选后确认下方出现"办公地点""设备数量"两个输入框。
3. 填写并提交，确认创建成功。

Expected: 模板可选中，两个输入框都渲染出来了（这是本次改动最初要解决的问题——之前完全不可见）。

- [ ] **Step 6: Commit**

```bash
cd itsm-frontend
git add src/lib/api/ticket-api.ts src/app/\(main\)/tickets/templates/page.tsx src/app/\(main\)/tickets/create/page.tsx
git commit -m "fix(frontend): getTemplates() response shape mismatch (templates vs items) blocked custom-field inputs from ever rendering"
```

---

### Task 9: 前端 —— 工单详情页展示改造（去掉"自定义字段"外层，显示 label）

**Files:**
- Modify: `itsm-frontend/src/lib/api/api-config.ts`
- Modify: `itsm-frontend/src/components/ticket/TicketDetail.tsx`

**Interfaces:**
- Consumes: 后端新响应形状 `TicketResponse.customFields: Array<{name,label,value}>`（Task 4 之后）

- [ ] **Step 1: 改前端类型**

`itsm-frontend/src/lib/api/api-config.ts:115`，把：
```ts
customFields?: Record<string, unknown>;
```
改成：
```ts
customFields?: Array<{ name: string; label: string; value: unknown }>;
```

- [ ] **Step 2: 改 `TicketDetail.tsx` 渲染逻辑**

`itsm-frontend/src/components/ticket/TicketDetail.tsx`，把之前加的：
```tsx
{ticket.customFields && Object.keys(ticket.customFields).length > 0 && (
  <Descriptions.Item label="自定义字段" span={2}>
    <Space orientation="vertical" size={4} style={{ width: '100%' }}>
      {Object.entries(ticket.customFields).map(([key, value]) => (
        <div key={key}>
          <Typography.Text type="secondary">{key}：</Typography.Text>
          {String(value)}
        </div>
      ))}
    </Space>
  </Descriptions.Item>
)}
```
改成每个字段独立一个 `Descriptions.Item`，用 `label` 而不是 key，不再包外层"自定义字段"：
```tsx
{ticket.customFields?.map(field => (
  <Descriptions.Item key={field.name} label={field.label}>
    {String(field.value)}
  </Descriptions.Item>
))}
```

- [ ] **Step 3: 类型检查**

Run: `cd itsm-frontend && npx tsc --noEmit`
Expected: 无错误

- [ ] **Step 4: 手动验证**

打开 Task 8 Step 5 里创建的那条工单详情页，确认："办公地点"和"设备数量"分别是独立的一行（跟"标题""状态"等字段并列展示），不再有一个包着两者的"自定义字段"外层。

- [ ] **Step 5: Commit**

```bash
cd itsm-frontend
git add src/lib/api/api-config.ts src/components/ticket/TicketDetail.tsx
git commit -m "feat(frontend): display each custom field as its own row with proper label, not grouped under a generic wrapper"
```

---

### Task 10: 端到端集成测试

**Files:**
- Create: `itsm-backend/tests/integration/dynamic_fields_test.go`

**Interfaces:**
- Consumes: 真实 Gin router（参考 `itsm-backend/tests/integration/` 目录下其它测试文件的 router/server 搭建方式——先看一眼那个目录已有的测试怎么起 test server，跟着同样的模式写，不要发明新的测试脚手架）

- [ ] **Step 1: 看一眼已有集成测试的脚手架**

Run: `ls itsm-backend/tests/integration/*.go` 然后 Read 其中一个已有的测试文件，确认它是怎么初始化 router + test client 的（这是本 Task 唯一需要先做的探索，避免重新发明一套跟现有集成测试不一致的启动方式）。

- [ ] **Step 2: 写集成测试**（具体 HTTP 调用序列，跟随 Step 1 探索到的脚手架写法）

测试序列：
1. `POST /api/v1/tickets/templates` 创建一个带 2 个字段的模板，断言 `200` 且响应里 `fields` 长度为 2。
2. `POST /api/v1/tickets` 用这个模板 ID，`formFields: {presetTypeId, values: {...}}` 提交 2 个字段值，断言 `200`。
3. `GET /api/v1/tickets/:id` 用上一步返回的工单 ID，断言响应体 `customFields` 数组长度为 2，且每项都有非空的 `label`（不是原始 key）。
4. `GET /api/v1/tickets?...` 列表接口，断言列表项**不**带 `customFields`（验证 Task 4 里"列表不查字段值，避免 N+1"这条设计决策真的生效了）。

- [ ] **Step 3: 跑集成测试**

Run: `cd itsm-backend && go test ./tests/integration/... -run TestDynamicFields -v`
Expected: PASS

- [ ] **Step 4: 跑全量后端测试确认没有破坏其它东西**

Run: `cd itsm-backend && go build ./... && go test ./... 2>&1 | tail -80`
Expected: 全部 `ok`，无 `FAIL`

- [ ] **Step 5: Commit**

```bash
cd itsm-backend
gofmt -l .
git add tests/integration/dynamic_fields_test.go
git commit -m "test(backend): add end-to-end integration test for dynamic custom fields (template -> ticket -> detail)"
```

---

---

### Task 11: 全分支 review 修复轮——补 templateId 传递、修模板编辑清空 bug、静态预设的即席字段值

**背景**：全分支 review（Task 1-10 全部完成后）发现 `tickets/create` 页面提交时从来没有带 `templateId`，导致后端 `CreateTicket` 里 `if req.TemplateID != nil` 这个门槛永远不满足——用户通过真实 UI 提交的自定义字段值全部被静默丢弃，Task 10 的集成测试因为直接在请求体里手写死了 `templateId` 没测出这个问题。同时发现 `tickets/templates` 编辑页读取字段用的是 `item.content?.fields`，但后端从来没返回过 `content` 这个字段，导致编辑保存时把字段定义全清空。用户决定这一轮一并解决"静态预设（代码里写死、不对应数据库模板）的自定义字段值现在没地方结构化存"这个问题。

**Files:**
- Modify: `itsm-backend/service/field_value_service.go`（新增 `CreateAdHocValues`）
- Modify: `itsm-backend/service/ticket_service.go`（`CreateTicket` 分支逻辑、新增 `extractAdHocFieldValues`，Warnw 升级 Errorw）
- Modify: `itsm-backend/service/service_catalog_item_service.go`（`toItemResponse` 静默吞错误升级为记日志）
- Modify: `itsm-frontend/src/lib/api/api-config.ts`（`CreateTicketRequest` 加 `templateId`）
- Modify: `itsm-frontend/src/app/(main)/tickets/create/page.tsx`（区分数据库模板 vs 静态预设两条提交路径）
- Modify: `itsm-frontend/src/app/(main)/tickets/templates/page.tsx`（修复编辑读取字段的 bug）
- Test: `itsm-backend/service/field_value_service_test.go`、`itsm-backend/service/ticket_service_test.go`、`itsm-backend/service/ticket_template_service_test.go`

**Interfaces:**
- Consumes: `FieldValueService`（Task 2）、`TicketController`/`create/page.tsx` 现有提交路径
- Produces: `FieldValueService.CreateAdHocValues(ctx, tenantID, valueEntityType, valueEntityID int, fields []AdHocFieldValue) error`——不需要匹配已有 `field_definitions`，直接按调用方给的 name/label 快照写 `field_values`（`field_definition_id` 留空，跟"定义被删"的历史值走的是同一条读取路径，展示层不需要区分）。

- [ ] **Step 1: `field_value_service.go` 新增即席写入方法**

```go
// AdHocFieldValue 是没有对应 field_definitions 行的自描述字段值——
// 用于前端静态预设（代码里写死、不对应数据库模板）提交自定义字段的场景。
type AdHocFieldValue struct {
	Name      string
	Label     string
	SortOrder int
	Value     interface{}
}

// CreateAdHocValues 直接按调用方提供的 name/label 写入 field_values，跳过
// CreateValues 那种"先查 field_definitions 再匹配"的步骤——静态预设没有
// field_definitions 行可以匹配。
func (s *FieldValueService) CreateAdHocValues(ctx context.Context, tenantID int, valueEntityType string, valueEntityID int, fields []AdHocFieldValue) error {
	if len(fields) == 0 {
		return nil
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return err
	}
	for _, f := range fields {
		encoded, err := json.Marshal(f.Value)
		if err != nil {
			return rollback(tx, err)
		}
		_, err = tx.FieldValue.Create().
			SetTenantID(tenantID).
			SetEntityType(valueEntityType).
			SetEntityID(valueEntityID).
			SetFieldName(f.Name).
			SetFieldLabel(f.Label).
			SetSortOrder(f.SortOrder).
			SetValue(encoded).
			Save(ctx)
		if err != nil {
			return rollback(tx, err)
		}
	}
	return tx.Commit()
}
```

（`rollback` 是 Task 2 已经在 `field_definition_service.go` 里定义好的包内共享 helper，同一个 `service` 包直接调用，不用重新定义。`field_definition_id` 不设置——`ent.FieldValue.FieldDefinitionID` 是 `Optional().Nillable()`，不设置就是 NULL，跟"定义后来被删掉"的历史值走完全相同的读取展示路径。）

- [ ] **Step 2: `ticket_service.go` 新增 `extractAdHocFieldValues` + `CreateTicket` 分支**

```go
// extractAdHocFieldValues 解析 formFields["fieldDefs"]（客户端提交的 {name,label} 列表，
// 用于没有 field_definitions 行的静态预设）配合 formFields["values"] 里的实际值，
// 构造成 AdHocFieldValue 列表。fieldDefs 缺失或为空返回 nil。
func extractAdHocFieldValues(formFields map[string]interface{}) []AdHocFieldValue {
	if formFields == nil {
		return nil
	}
	rawDefs, ok := formFields["fieldDefs"].([]interface{})
	if !ok || len(rawDefs) == 0 {
		return nil
	}
	values, _ := formFields["values"].(map[string]interface{})
	if len(values) == 0 {
		return nil
	}
	result := make([]AdHocFieldValue, 0, len(rawDefs))
	for i, raw := range rawDefs {
		defMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := defMap["name"].(string)
		if name == "" {
			continue
		}
		val, ok := values[name]
		if !ok {
			continue
		}
		label, _ := defMap["label"].(string)
		if label == "" {
			label = name
		}
		result = append(result, AdHocFieldValue{Name: name, Label: label, SortOrder: i, Value: val})
	}
	return result
}
```

`CreateTicket()` 里现在这一段：

```go
if req.TemplateID != nil {
    if fieldValues := extractCustomFieldValues(req.FormFields); len(fieldValues) > 0 {
        if err := NewFieldValueService(s.client).CreateValues(ctx, tenantID, "ticket_template", *req.TemplateID, "ticket", tkt.ID, fieldValues); err != nil {
            s.logger.Warnw("Failed to persist custom field values", "error", err, "ticket_id", tkt.ID)
        }
    }
}
```

改成：

```go
if req.TemplateID != nil {
    if fieldValues := extractCustomFieldValues(req.FormFields); len(fieldValues) > 0 {
        if err := NewFieldValueService(s.client).CreateValues(ctx, tenantID, "ticket_template", *req.TemplateID, "ticket", tkt.ID, fieldValues); err != nil {
            s.logger.Errorw("Failed to persist custom field values", "error", err, "ticket_id", tkt.ID, "template_id", *req.TemplateID)
        }
    }
} else if adHocFields := extractAdHocFieldValues(req.FormFields); len(adHocFields) > 0 {
    if err := NewFieldValueService(s.client).CreateAdHocValues(ctx, tenantID, "ticket", tkt.ID, adHocFields); err != nil {
        s.logger.Errorw("Failed to persist ad-hoc custom field values", "error", err, "ticket_id", tkt.ID)
    }
}
```

（`Warnw`→`Errorw`：全分支 review 指出字段值写入失败目前没有任何恢复手段，用户以为提交成功了，日志至少要能定位到是哪张工单哪次提交丢了值。）

- [ ] **Step 3: `ToTicketResponseWithCustomFields` 里静默吞错误升级为记日志**

`itsm-backend/service/ticket_service.go` 的 `ToTicketResponseWithCustomFields`：

```go
values, err := NewFieldValueService(client).ListValues(ctx, t.TenantID, "ticket", t.ID)
if err != nil || len(values) == 0 {
    return resp
}
```

改成：

```go
values, err := NewFieldValueService(client).ListValues(ctx, t.TenantID, "ticket", t.ID)
if err != nil {
    zap.S().Warnw("Failed to load custom field values for ticket response", "error", err, "ticket_id", t.ID)
    return resp
}
if len(values) == 0 {
    return resp
}
```

（这是包级函数没有注入 logger，用 `zap.S()` 全局访问器——CLAUDE.md 本来就要求日志走 `zap.S()`，这是既定用法，不是新引入的模式。需要确认文件顶部已经 `import "go.uber.org/zap"`，没有的话加上。）

- [ ] **Step 4: `service_catalog_item_service.go` 的 `toItemResponse` 同样升级**

```go
fields := make([]map[string]interface{}, 0)
if defs, err := NewFieldDefinitionService(s.client).ListDefinitions(ctx, item.TenantID, "service_catalog_item", item.ID); err == nil {
    for _, d := range defs {
        ...
    }
}
```

改成：

```go
fields := make([]map[string]interface{}, 0)
defs, err := NewFieldDefinitionService(s.client).ListDefinitions(ctx, item.TenantID, "service_catalog_item", item.ID)
if err != nil {
    s.logger.Warnw("Failed to load field definitions for service catalog item", "error", err, "item_id", item.ID)
} else {
    for _, d := range defs {
        ...
    }
}
```

（这个文件的 service struct 本来就有 `logger *zap.SugaredLogger` 字段，直接用 `s.logger`，不用改构造函数。）

- [ ] **Step 5: 前端 `api-config.ts` 加 `templateId`**

`CreateTicketRequest` 加一行：

```ts
export interface CreateTicketRequest {
  title: string;
  description: string;
  priority: string;
  type?: 'incident' | 'service_request' | 'change' | 'problem' | string;
  category?: string;
  categoryId?: number;
  templateId?: number;
  formFields?: Record<string, unknown>;
  assigneeId?: number;
  workflowDefinitionKey?: string;
}
```

- [ ] **Step 6: `create/page.tsx` 提交时区分数据库模板 vs 静态预设**

`handleSubmit` 里构造 `TicketApi.createTicket({...})` 那一段（当前在 `formFields: selectedType ? { presetTypeId: selectedType.id, values: customFieldValues } : undefined,` 附近），改成：

```ts
// 数据库模板的 id 形如 `db_${item.id}`（见本文件 useEffect 里 dbTemplates 的构造逻辑）；
// 静态预设（ticket-type-presets.ts 里代码写死的那些）没有这个前缀，也没有对应的
// field_definitions 行，走即席提交路径。
const isDbTemplate = selectedType?.id.startsWith('db_') ?? false;
const templateId = isDbTemplate ? Number(selectedType!.id.slice(3)) : undefined;

const created = await TicketApi.createTicket({
  title: title,
  description: description,
  priority: priority,
  type: inferTicketType(selectedType),
  category: values.category || (selectedType ? selectedType.category : undefined),
  templateId,
  formFields: selectedType
    ? {
        presetTypeId: selectedType.id,
        values: customFieldValues,
        ...(isDbTemplate
          ? {}
          : {
              fieldDefs: (selectedType.fields || []).map(f => ({
                name: f.name,
                label: f.label,
              })),
            }),
      }
    : undefined,
  workflowDefinitionKey: selectedType?.workflowTemplateId,
});
```

- [ ] **Step 7: `templates/page.tsx` 修编辑读取字段的 bug**

第 170-173 行（当前）：

```ts
customFields:
  (item.content?.fields as CustomField[]) ||
  (item.content?.customFields as CustomField[]) ||
  [],
```

改成：

```ts
customFields: (item.fields as CustomField[]) || [],
```

（后端从来没有返回过 `item.content` 这个字段——`dto.TicketTemplate` 只有顶层 `fields`，`create/page.tsx` 已经在正确读 `item.fields`，这个页面之前读错了，导致编辑保存时把已有字段定义当成空数组提交、被 `ReplaceDefinitions` 整个清空。）

- [ ] **Step 8: 补两个回归测试**

`itsm-backend/service/ticket_template_service_test.go` 追加：

```go
func TestTicketTemplateService_UpdateTemplate_NilFieldsPreservesExisting(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_template_nil_fields?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	svc := NewTicketTemplateService(client)
	fieldSvc := NewFieldDefinitionService(client)

	template, err := svc.CreateTemplate(ctx, &CreateTemplateRequest{
		Name: "模板", Category: "cat", TenantID: 1,
		Fields: []FieldDefinitionInput{{Name: "a", Label: "A", FieldType: "text"}},
	})
	require.NoError(t, err)

	// 只改状态，不碰 Fields——Fields 是 nil，应该被当成"不修改"，不是"清空"。
	isActive := false
	_, err = svc.UpdateTemplate(ctx, template.ID, &UpdateTemplateRequest{
		IsActive: &isActive,
	}, 1)
	require.NoError(t, err)

	defs, err := fieldSvc.ListDefinitions(ctx, 1, "ticket_template", template.ID)
	require.NoError(t, err)
	require.Len(t, defs, 1)
	assert.Equal(t, "a", defs[0].Name)
}
```

`itsm-backend/service/ticket_service_test.go` 追加：

```go
func TestTicketService_CreateTicket_AdHocFieldValuesWithoutTemplate(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_create_adhoc_fields?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant := createTicketAssociationTenant(t, ctx, client, "create-adhoc-fields")
	requester := createTicketAssociationUser(t, ctx, client, tenant.ID, "create-adhoc-fields-requester")
	service := NewTicketServiceForTest(client, zaptest.NewLogger(t).Sugar())

	created, err := service.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title: "K8S扩容", Description: "测试", Priority: "medium", RequesterID: requester.ID,
		// 没有 TemplateID——模拟静态预设
		FormFields: map[string]interface{}{
			"fieldDefs": []interface{}{
				map[string]interface{}{"name": "replicas", "label": "副本数"},
			},
			"values": map[string]interface{}{"replicas": float64(3)},
		},
	}, tenant.ID)
	require.NoError(t, err)

	values, err := NewFieldValueService(client).ListValues(ctx, tenant.ID, "ticket", created.ID)
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.Equal(t, "replicas", values[0].Name)
	assert.Equal(t, "副本数", values[0].Label)
	assert.Equal(t, float64(3), values[0].Value)
}
```

- [ ] **Step 9: 跑测试**

Run: `cd itsm-backend && go build ./... && go test ./service/... -run "TicketTemplateService_UpdateTemplate_NilFields|TicketService_CreateTicket_AdHocFieldValues" -v`
Expected: PASS

Run: `cd itsm-backend && go test ./... 2>&1 | tail -60`
Expected: 全部 `ok`，无 `FAIL`

Run: `cd itsm-frontend && npx tsc --noEmit`
Expected: 0 errors

- [ ] **Step 10: Commit**

```bash
cd itsm-backend
gofmt -l .
git add service/field_value_service.go service/field_value_service_test.go service/ticket_service.go service/ticket_service_test.go service/ticket_template_service_test.go service/service_catalog_item_service.go
git commit -m "fix(backend): support ad-hoc custom field values for templateless tickets, log swallowed errors instead of discarding"
```

```bash
cd itsm-frontend
git add src/lib/api/api-config.ts "src/app/(main)/tickets/create/page.tsx" "src/app/(main)/tickets/templates/page.tsx"
git commit -m "fix(frontend): send templateId on ticket create (was silently discarding all custom field values), fix template-edit page reading a field the backend never returns"
```

---

---

### Task 12: 第二轮修复——`formFields.values` 改数组格式避开全局驼峰转换，修 priority 清空 bug

**背景**：Task 11 的修复经复查（opus）发现只解决了一半。真正的根因比 templateId 更底层：`itsm-frontend/src/lib/api/http-client.ts` 有一个全局逻辑，对**每一个** HTTP 请求体做递归 snake_case→camelCase key 转换，不区分"这是接口契约字段"还是"这是用户数据"。`formFields.values` 这个 map 的 key 就是用户填的自定义字段名（不是接口契约字段），会被这个全局转换悄悄改写——比如 `current_replicas` 变成 `currentReplicas`，导致后端 `extractCustomFieldValues`/`extractAdHocFieldValues` 用原始字段名去 `values[name]` 匹配时永远匹配不上，静默丢弃、没有任何报错。`ticket-type-presets.ts` 里 75 个预设字段名有 47 个带下划线，Task 11 新加的测试之所以能过纯粹是因为凑巧用了个不带下划线的字段名。这个问题同时影响数据库模板路径（`CreateValues`）和新加的即席路径（`CreateAdHocValues`）。

不去动 `http-client.ts` 的全局转换（改动面是整个 App 每个接口调用，风险太大），而是把 `formFields.values` 的线上格式从"字段名作 key 的 map"改成"`{name, value}` 的数组"——字段名变成数组元素里的字符串**值**而不是对象的 **key**，天然绕开这个全局转换（`fieldDefs` 已经是这个形状，从没受影响，这次是照抄这个已验证有效的模式）。

同一轮复查还发现 `templates/page.tsx` 的 `priority` 字段有和 Task 11 刚修的 `customFields` 一模一样的清空 bug（`item.content?.priority` 永远不存在、读成默认值 `medium`，且这个字段真的会在保存时写回后端），顺手一起修。

**Files:**
- Modify: `itsm-backend/service/ticket_service.go`（`extractCustomFieldValues`、`extractAdHocFieldValues` 改成优先解析数组格式，保留 map 格式作为向后兼容——现有测试直接用 Go 构造 map 调用 service 层，不能破坏）
- Modify: `itsm-frontend/src/app/(main)/tickets/create/page.tsx`（`customFieldValues` 从对象改成数组）
- Modify: `itsm-frontend/src/app/(main)/tickets/templates/page.tsx`（修 `priority` 清空 bug）
- Test: `itsm-backend/service/ticket_service_test.go`（新增数组格式 + 带下划线字段名的测试）
- Test: `itsm-frontend/src/lib/api/__tests__/http-client.test.ts`（新增一条证明数组形状的自定义字段名能在请求体里存活，反证 map 形状会被破坏）

**Interfaces:**
- Consumes: Task 11 的 `extractAdHocFieldValues`、既有的 `extractCustomFieldValues`
- Produces: `formFields.values` 新的线上形状 `Array<{name: string, value: unknown}>`（前端发送、后端解析都要认这个新形状；后端同时继续兼容旧的 map 形状，因为现有测试和任何直接调后端的调用方还在用它）

- [ ] **Step 1: 写失败测试，断言数组格式里带下划线的字段名能正确落库**

```go
// 追加到 itsm-backend/service/ticket_service_test.go
func TestTicketService_CreateTicket_ValuesArrayFormatSurvivesUnderscoreNames(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_create_values_array?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant := createTicketAssociationTenant(t, ctx, client, "create-values-array")
	requester := createTicketAssociationUser(t, ctx, client, tenant.ID, "create-values-array-requester")
	template, err := client.TicketTemplate.Create().
		SetName("云主机申请").SetCategory("云").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	_, err = NewFieldDefinitionService(client).ReplaceDefinitions(ctx, tenant.ID, "ticket_template", template.ID, []FieldDefinitionInput{
		{Name: "current_replicas", Label: "当前副本数", FieldType: "number", SortOrder: 0},
	})
	require.NoError(t, err)

	service := NewTicketServiceForTest(client, zaptest.NewLogger(t).Sugar())
	created, err := service.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title: "扩容申请", Description: "测试", Priority: "medium",
		RequesterID: requester.ID, TemplateID: &template.ID,
		FormFields: map[string]interface{}{
			// 数组形状——模拟前端发送、经过 http-client.ts 全局驼峰转换后的真实线上形态。
			// 注意：这里故意用数组而不是 map，因为这正是这次要修的 bug 场景。
			"values": []interface{}{
				map[string]interface{}{"name": "current_replicas", "value": float64(5)},
			},
		},
	}, tenant.ID)
	require.NoError(t, err)

	values, err := NewFieldValueService(client).ListValues(ctx, tenant.ID, "ticket", created.ID)
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.Equal(t, "current_replicas", values[0].Name)
	assert.Equal(t, float64(5), values[0].Value)
}

func TestTicketService_CreateTicket_ValuesMapFormatStillWorks(t *testing.T) {
	// 向后兼容：直接调 service 层（不经过前端/http-client）的现有测试和调用方，
	// 传的还是 map 形状，这条要继续通过。
	client := enttest.Open(t, "sqlite3", "file:ticket_create_values_map_compat?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	tenant := createTicketAssociationTenant(t, ctx, client, "create-values-map-compat")
	requester := createTicketAssociationUser(t, ctx, client, tenant.ID, "create-values-map-compat-requester")
	template, err := client.TicketTemplate.Create().
		SetName("模板").SetCategory("c").SetTenantID(tenant.ID).Save(ctx)
	require.NoError(t, err)
	_, err = NewFieldDefinitionService(client).ReplaceDefinitions(ctx, tenant.ID, "ticket_template", template.ID, []FieldDefinitionInput{
		{Name: "office_location", Label: "办公地点", FieldType: "text", SortOrder: 0},
	})
	require.NoError(t, err)

	service := NewTicketServiceForTest(client, zaptest.NewLogger(t).Sugar())
	created, err := service.CreateTicket(ctx, &dto.CreateTicketRequest{
		Title: "t", Description: "d", Priority: "medium",
		RequesterID: requester.ID, TemplateID: &template.ID,
		FormFields: map[string]interface{}{
			"values": map[string]interface{}{"office_location": "北京"},
		},
	}, tenant.ID)
	require.NoError(t, err)

	values, err := NewFieldValueService(client).ListValues(ctx, tenant.ID, "ticket", created.ID)
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.Equal(t, "北京", values[0].Value)
}
```

- [ ] **Step 2: 跑测试确认第一个失败、第二个通过**

Run: `cd itsm-backend && go test ./service/... -run "TestTicketService_CreateTicket_ValuesArrayFormatSurvivesUnderscoreNames|TestTicketService_CreateTicket_ValuesMapFormatStillWorks" -v`
Expected: 第一个 FAIL（`extractCustomFieldValues` 还只认 map，数组格式解析不出东西，`ListValues` 返回空）；第二个 PASS（现有 map 路径没动）

- [ ] **Step 3: 实现——加一个共享的数组解析 helper，`extractCustomFieldValues`/`extractAdHocFieldValues` 都改成优先用它**

`itsm-backend/service/ticket_service.go`，在 `extractCustomFieldValues` 定义之前加：

```go
// parseFieldValuesArray 把 formFields["values"] 解析成 [{name,value}] 数组形状，
// 转成内部用的 map[name]value。数组形状是必须的——字段名作为数组元素里的字符串值
// （而不是对象的 key）传输，这样才能绕开前端 http-client.ts 那个全局的、不区分
// 契约字段和用户数据的 snake_case→camelCase 请求体转换（那个转换会把 map 形状里
// 带下划线的字段名 key 悄悄改写，导致匹配失败、值静默丢失）。
// 解析不出数组形状返回 nil，调用方会退回到兼容 map 形状的旧逻辑。
func parseFieldValuesArray(formFields map[string]interface{}) map[string]interface{} {
	if formFields == nil {
		return nil
	}
	rawValues, ok := formFields["values"].([]interface{})
	if !ok {
		return nil
	}
	result := make(map[string]interface{}, len(rawValues))
	for _, raw := range rawValues {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := entry["name"].(string)
		if name == "" {
			continue
		}
		if val, ok := entry["value"]; ok {
			result[name] = val
		}
	}
	return result
}
```

`extractCustomFieldValues` 现在这样：

```go
func extractCustomFieldValues(formFields map[string]interface{}) map[string]interface{} {
	if formFields == nil {
		return nil
	}
	if values, ok := formFields["values"].(map[string]interface{}); ok {
		return values
	}
	return nil
}
```

改成：

```go
func extractCustomFieldValues(formFields map[string]interface{}) map[string]interface{} {
	if formFields == nil {
		return nil
	}
	if values := parseFieldValuesArray(formFields); values != nil {
		return values
	}
	// 兼容旧的 map 形状——直接调用 service 层的测试/调用方还在用。
	if values, ok := formFields["values"].(map[string]interface{}); ok {
		return values
	}
	return nil
}
```

`extractAdHocFieldValues`（Task 11 加的）里解析 `values` 那一行：

```go
values, _ := formFields["values"].(map[string]interface{})
if len(values) == 0 {
    return nil
}
```

改成：

```go
values := parseFieldValuesArray(formFields)
if len(values) == 0 {
    if mapValues, ok := formFields["values"].(map[string]interface{}); ok {
        values = mapValues
    }
}
if len(values) == 0 {
    return nil
}
```

- [ ] **Step 4: 跑测试确认两个都通过**

Run: `cd itsm-backend && go test ./service/... -run "TestTicketService_CreateTicket_ValuesArrayFormatSurvivesUnderscoreNames|TestTicketService_CreateTicket_ValuesMapFormatStillWorks" -v`
Expected: 两个都 PASS

- [ ] **Step 5: 前端改 `create/page.tsx`，`customFieldValues` 从对象改数组**

`handleSubmit` 里现在这段：

```ts
const customFieldValues: Record<string, unknown> = {};
if (selectedType?.fields && selectedType.fields.length > 0) {
  selectedType.fields.forEach(field => {
    const value = values[field.name];
    if (value !== undefined && value !== null && value !== '') {
      customFieldValues[field.name] = value;
    }
  });
}
```

改成：

```ts
const customFieldValues: Array<{ name: string; value: unknown }> = [];
if (selectedType?.fields && selectedType.fields.length > 0) {
  selectedType.fields.forEach(field => {
    const value = values[field.name];
    if (value !== undefined && value !== null && value !== '') {
      customFieldValues.push({ name: field.name, value });
    }
  });
}
```

（后面构造 `TicketApi.createTicket({...formFields: {..., values: customFieldValues, ...}})` 那段不用改——`customFieldValues` 类型变了，直接传下去就行，Task 11 已经接好的 `fieldDefs`/`templateId` 逻辑不受影响。）

- [ ] **Step 6: 前端改 `templates/page.tsx`，修 `priority` 清空 bug**

第 159 行（当前）：

```ts
priority: (item.content?.priority as string) || 'medium',
```

改成：

```ts
priority: item.priority || 'medium',
```

- [ ] **Step 7: 加一条 http-client 层面的回归测试，证明数组形状能在请求体转换里存活**

`itsm-frontend/src/lib/api/__tests__/http-client.test.ts` 的 `describe('post', ...)` 块里追加：

```ts
it('preserves custom field names inside array-shaped values (map-shaped keys would be corrupted by camelCase normalization)', async () => {
  let capturedBody: string | undefined;
  mockFetch.mockImplementationOnce((_url, init) => {
    capturedBody = init?.body as string;
    return Promise.resolve(
      new Response(JSON.stringify({ code: 0, message: 'ok', data: {} }), { status: 200 })
    );
  });

  await httpClient.post('/api/v1/tickets', {
    formFields: {
      values: [{ name: 'current_replicas', value: 5 }],
    },
  });

  const sentBody = JSON.parse(capturedBody!);
  // 数组元素里的 name 是字符串值，不是 object key，不会被请求体的驼峰转换改写。
  expect(sentBody.formFields.values[0].name).toBe('current_replicas');
  expect(sentBody.formFields.values[0].value).toBe(5);
});
```

（照抄这个文件里已有的 `post` 测试用的 mock 方式——先读一下同一个 `describe('post', ...)` 块里其它测试是怎么 mock `fetch`/构造 `Response` 的，跟那个写法保持一致，不要发明新的 mock 方式。）

- [ ] **Step 8: 跑全部相关测试**

Run: `cd itsm-backend && go build ./... && go test ./... 2>&1 | tail -60`
Expected: 全部 `ok`，无 `FAIL`

Run: `cd itsm-frontend && npx tsc --noEmit && npx jest src/lib/api/__tests__/http-client.test.ts`
Expected: 0 类型错误，新加的 http-client 测试通过

- [ ] **Step 9: Commit**

```bash
cd itsm-backend
gofmt -l .
git add service/ticket_service.go service/ticket_service_test.go
git commit -m "fix(backend): accept formFields.values as array of {name,value} (backward-compat with map), fixing snake_case field names silently dropped by frontend's request-body camelCase normalization"
```

```bash
cd itsm-frontend
git add "src/app/(main)/tickets/create/page.tsx" "src/app/(main)/tickets/templates/page.tsx" src/lib/api/__tests__/http-client.test.ts
git commit -m "fix(frontend): send custom field values as array (not name-keyed map) so underscore field names survive http-client's camelCase normalization; fix template-edit page resetting priority to medium on every save"
```

---

## Self-Review 记录

- **Spec 覆盖检查**：设计文档的"数据模型""API 集成""清理范围""顺手偿还的技术债""测试计划"五节，分别对应 Task 1-2 / Task 3-4-7 / Task 3-4-6-7 / Task 4 / Task 10，`ServiceRequest.form_data` 按已批准的范围收窄不动（未创建对应 Task）。任务编号不连续（没有"Task 5"——已并入 Task 4）是有意为之，`task-brief` 按标题文本匹配、不依赖连续编号，不影响执行。
- **占位符扫描**：全文没有 TBD/待定/"类似 Task N 处理"（Task 3 和 Task 7 的 `toFieldDefinitionInputs` 复用是明确说"同一个 service 包直接调用"，不是省略代码）。
- **类型一致性**：`FieldDefinitionInput`（Task 2 定义）在 Task 3/7 的调用签名一致；`FieldValueDTO`/`CustomFieldValueResponse` 命名在 Task 2/4 之间保持一致（前者是 service 内部类型，后者是暴露给 JSON 的 DTO，Task 4 Step 9 里有明确的类型转换代码，不是疏漏）。
