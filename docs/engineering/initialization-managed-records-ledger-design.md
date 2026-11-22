# 初始化托管记录账本：从死表到可消费契约

> 状态：设计已锁定，本批实施第一层（登记写入 + 只读消费 + 回归锁）。
> 事实依据：`origin/main@0e8d26950`、`migrations.go:508-534`、`sql_store.go`、`engine.go`、`docs/initialization-data-classification.md`。
> 边界：本批**不**实现 blueprint §112-115 的字段级三方合并（create/update/revoke/conflict/orphan 全套），那是 P1 结构性设计；本批只做"初始化成功 → 托管记录可查询"这一条最小闭环。

## 1. 问题（代码事实）

`initialization_managed_records` 表在 `008_add_initialization_ledger` 迁移中创建（`migrations.go:508-525`），具备唯一键 `(scope_type, scope_id, component, source_key)` 与索引 `idx_init_managed_scope_component`（`:533-534`），并在 `tenant_guard.go:94-98` 登记为 RLS 豁免（自带 scope 维度）。

但全仓核查（2026-11-22，精确符号搜索）：

| 层 | 状态 |
|:---|:---|
| ent schema / generated code | 无 |
| `INSERT` / `UPDATE` / `SELECT` 消费方 | **零** |
| 唯一测试 | `migrator_test.go:131` 仅断言"表被创建" |

同族另外三张表（`initialization_installations` / `_runs` / `_component_attempts`）在 `sql_store.go` 有完整读写，唯独托管记录表空转。这是项目纪律反复定义的**"字段即声明"假闭环**：`initialization-data-classification.md:9` 声称 "managed fields are reconciled"，但没有任何代码记录"哪些稳定键由平台托管、依据哪版 manifest、上次写入值是什么"。

后果：租户无法回答"这条 CI type / 角色 / 菜单是不是平台托管的、能不能改、升级会不会被覆盖"，升级也无法识别"托管记录被本地改过"——而这正是 P0/T0 分级中最关键的运维能力。

## 2. 设计原则

1. **复用现有 DDL，零新增迁移**：表结构已发布且字段完备，只需开始读写。改已发布迁移会触发 checksum 门禁（`migration/superseded.go`），无谓风险。
2. **写入与安装成功同事务原子**：托管登记写在 `componentTransaction.Complete` 内，与 `initialization_installations.status='succeeded'` 同一 `sql.Tx`。组件回滚 → 账本不落；账本写失败 → 组件安装失败（fail-closed），杜绝"安装报成功但账本空"。
3. **幂等**：`ON CONFLICT (scope_type, scope_id, component, source_key) DO UPDATE`，重放只更新 `source_version` / `manifest_checksum` / `last_applied_values` / `updated_at`，不产生重复行。
4. **多租户隔离**：写读全部带 `scope_type` + `scope_id` 谓词；`ListManagedRecords` 强制 scope 参数，禁止全表扫。
5. **fencing 一致**：写入沿用 `Complete` 已有的 fenced 更新（`lease_owner` + `fencing_token` + `status='running'` + `lease_expires_at > clock_timestamp()`），账本不引入第二套并发控制。
6. **不新增权限码 / 路由**：本批只做存储层能力，读面由内部初始化/运维工具经 Go API 消费，不暴露 HTTP（避免 dead-codes 与 precheck 重生成）。

## 3. 数据模型映射

`Plan.Actions[].SourceKey` 是本批的托管稳定键来源（生产 initializer 当前每个组件声明 1 个 reconcile action，`SourceKey = component 名`，见 `initialization_adapter.go:260-265`）。

| 列 | 来源 | 说明 |
|:---|:---|:---|
| `scope_type` / `scope_id` | `t.scope` | 自带租户维度 |
| `component` | `t.plan.Component` | 初始化组件名 |
| `source_key` | `action.SourceKey` | 稳定键；空则跳过 |
| `source_version` | `t.plan.TargetVersion` | 哪版模板应用 |
| `manifest_checksum` | `t.plan.SourceChecksum` | manifest 指纹（判定输入） |
| `ownership_mode` | 常量 `managed` | 平台托管 |
| `managed_fields` | `[]`（默认） | 字段级归属，留待 P1 三方合并；本批不伪造 |
| `last_applied_values` | `result.Summary` 的 JSON | 上次应用结果摘要，运维可审计 |
| `local_modified_at` | NULL | 本批不实现本地修改检测，保持 NULL |
| `deprecated` | false | 本批不实现退役，保持默认 |
| `stable_key_aliases` | `[]`（默认） | 兼容键，留待 P1 |

## 4. 契约（新增 Go API）

```go
// ManagedRecord 托管记录账本行（只读投影）。
type ManagedRecord struct {
    Component, SourceKey, SourceVersion, ManifestChecksum string
    OwnershipMode string
    ManagedFields json.RawMessage
    LastAppliedValues json.RawMessage
    LocalModifiedAt sql.NullTime
    Deprecated bool
    CreatedAt, UpdatedAt time.Time
}

// ListManagedRecords 按 scope 列出托管记录；scope 必填，禁止全表扫。
func (s *SQLStore) ListManagedRecords(ctx context.Context, scope Scope) ([]ManagedRecord, error)
```

写入不暴露独立方法：只在 `(*componentTransaction).Complete` 内部调用私有 `recordManaged`，确保"只有走完完整组件提交路径才能写账本"。

## 5. 验收标准（每条可独立验证）

| # | 判据 | 验证方式 |
|:--|:---|:---|
| V1 | 组件成功后，账本按 scope 可读，字段与 plan/result 一致 | Postgres fixture + 真实 `SQLStore` |
| V2 | 重放同一 plan 幂等：行数不变、checksum/version 更新 | Postgres fixture |
| V3 | 同 component 新 plan 换 source_key → 旧键仍在（accumulate，不误删） | Postgres fixture |
| V4 | 跨 scope（租户/平台）不可见 | Postgres fixture |
| V5 | 账本写入失败 → `Complete` 失败 → 组件安装整体失败、安装行不置 succeeded | Postgres fixture（注入约束冲突） |
| V6 | 空 `Actions` 不写任何账本行 | Postgres fixture |
| V7 | `ListManagedRecords` 空 scope 参数不触发全表扫 | 代码断言 + 测试 |
| R1 | 反向验证：移除 `recordManaged` 调用 → V1/V2 变红 | 手工注入 |

## 6. 明确不做（边界）

- 字段级 `managed_fields` 对账、revoke/conflict/orphan 三态（P1，需单独设计评审）
- `local_modified_at` 本地修改检测（需要业务表触发器/读面对比，超本批范围）
- HTTP 读面 / 前端消费（本批只建存储契约；对外暴露时机随 Agent Control Plane 只读面统一设计）
- 存量数据回填（该表本就无数据，无回填对象）
