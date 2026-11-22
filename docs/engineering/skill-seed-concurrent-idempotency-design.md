# 技能种子并发幂等：把 check-then-act 收口为 DB 层 UPSERT

> 状态：设计锁定，本批实施。
> 事实依据：`origin/main@52e9878c5`（v1.6.60），以该树源码为唯一实现事实。
> 关键源码：`handlers/skill/persistence.go:104-160`（`UpsertDefinition`）、
> `handlers/skill/persistence_test.go:95-131`（串行幂等回归）、
> `migration/migrations.go:893-916`（迁移 023 的 `UNIQUE (tenant_id, code)`）、
> `internal/bootstrap/app.go:1139-1142`（失败日志「registry stays in-memory」）。
>
> 本文件是实施前设计；实施后的验证证据写入 CHANGELOG 与本批提交说明，不在此处预告结果。

## 1. 问题（代码事实，不是推测）

串行场景的幂等在 `origin/main` **已经修好**：`TestSyncBuiltinSkills_Idempotent` 锁定
「两次 seed 不增行、第二次 `changed=0`、checksum 演进走 UPDATE」。因此不能再把
「重复 seed 一定报 23505」当作现状——那是过时结论。

真实残余是 `UpsertDefinition` 的 **check-then-act 竞态**：

| 步骤 | 源码 | 窗口 |
|:--|:--|:--|
| ① 查 checksum | `persistence.go:110` `SELECT checksum ...` | 两个调用方可以同时读到 `ErrNoRows` |
| ② 决定动作 | `persistence.go:113-118` / `138` | 两边都选 INSERT |
| ③ 写入 | `persistence.go:146-155` 裸 `INSERT ... RETURNING id` | 一边成功，另一边撞 `UNIQUE (tenant_id, code)` → 23505/SQLite `UNIQUE constraint failed` |

错误被原样返回：内置 seed 路径在 `internal/bootstrap/app.go` 只记一条
`builtin skill seed failed (registry stays in-memory)` 后继续启动；HTTP Create/Update
（`handlers/skill/handler.go:366,479`）则回滚内存注册并返回持久化失败。

危害边界（据源码如实限定）：注册表是内存单源，运行时 `Invoke` 不读 DB，
所以**不会**造成技能整体不可用；危害是多副本同时首启/重启时，撞约束的副本
持久化基线残缺、产生误导性降级日志，以及并发同名自定义技能写返回可避免的 500。
重启后串行 seed 可自愈，但这不取消缺陷。

## 2. 设计原则

1. **幂等放在 DB 层，而不是应用层再查一次**：用唯一约束承载原子性；「再查一次」
   只会把竞态窗口缩小而不会关闭。
2. **保留 `changed` 的真实语义**：checksum 相同必须是 `changed=false`，不能因为
   改成 UPSERT 就每次启动都刷 `updated_at`、把「零变更」报成「有变更」。
3. **冲突键必须带租户**：`UNIQUE (tenant_id, code)` 是物理约束； UPSERT 的冲突目标
   漏掉 `tenant_id` 会把跨租户同名技能合并成一行，属安全隔离事故。
4. **零迁移、零权限码、零运行时语义变化**：不改 DDL、不改已发布迁移 checksum、
   不改内存注册表的单源地位、不改 RBAC/Policy/Audit 判定路径。
5. **双方言可用**：生产 PostgreSQL 的 JSONB 与测试 SQLite 的 TEXT 使用同一条语句。

## 3. 方案

`UpsertDefinition` 保持三段结构，只把第 ③ 段改成单条原子语句：

1. 保留快速路径 `SELECT checksum`。命中且相同 → `(0,false,nil)`，与现有串行语义一致，
   绝大多数启动时连写都不发生。
2. 命中但不同，或未命中 → 序列化 JSON 字段后执行：

```sql
INSERT INTO skill_definitions (code, version, title, description, category,
    tags, capabilities, required_permissions, input_schema, output_schema,
    executor, provider, author, checksum, enabled, tenant_id)
VALUES ($1,...,$16)
ON CONFLICT (tenant_id, code) DO UPDATE SET
    version=$2, title=$3, description=$4, category=$5,
    tags=$6, capabilities=$7, required_permissions=$8,
    input_schema=$9, output_schema=$10, executor=$11,
    provider=$12, author=$13, checksum=$14, enabled=$15,
    updated_at=$17
WHERE skill_definitions.checksum <> EXCLUDED.checksum
RETURNING id
```

占位符下标与 `definitionColumns` 顺序一一对应（`$1=code`、`$2=version` …、
`$16=tenant_id`、`$17=updated_at`），**不能**照搬旧 UPDATE 语句从 `$1` 重新编号：
实施中第一版就因 `version=$1` 把 code 写进 version 列，被 `TestSyncBuiltinSkills_Idempotent`
当场抓住。这是「演进必须覆盖字段值而不只是行数」的直接理由。

- `WHERE ... <> EXCLUDED.checksum` 让「并发同 payload 的第二个调用」在 DO UPDATE
  阶段被判定为无需更新，`RETURNING` 无行 → `changed=false`，不刷 `updated_at`。
- 未命中路径由 INSERT 成功返回真实 `id`；冲突更新路径也返回真实 `id`
  （旧实现返回 `RowsAffected()`，调用方均未消费，属顺手纠正而非契约变更）。
- 任何真正的 DB 错误仍带 `%w` 透传，供 HTTP 错误分类与启动日志识别；
  不再有可预期的 23505 需要特殊吞掉。

## 4. 明确不做

- 不新增 `skill_definitions` 迁移：唯一约束在迁移 023 已存在，UPSERT 只使用它。
- 不动 `DefinitionStore` 接口签名：所有调用方与 spy 无需修改。
- 不引入分布式锁/事务边界扩展：这里要的是「同一行最终一致」，不是跨行原子性。
- 不把并发不同 checksum 定义成「谁先谁赢」的强语义：两个不同版本同时写入时
  最后一次 UPDATE 生效，两个调用都可能报 `changed=true`；`changed` 是统计提示，
  不是准入判定，任何消费方都不得据此放行副作用。
- 不改 Memory / Agent Runtime / ActionFlow / Policy 域：本批边界只在 Skill Registry 持久化。

## 5. 验收标准（实施后逐条给出证据）

| 编号 | 判据 |
|:--|:--|
| A1 | 确定性竞态用例（测试驱动在 `SELECT checksum` 之后、INSERT 之前由另一连接落同键行）在旧实现下必红；新实现 `err=nil`、`changed=false`、行数=1。另有 8 goroutine 齐发用例：全部 `err=nil`，行数=1，最多一个 `changed=true`。 |
| A2 | 同 code 的 checksum 演进并发写入：无唯一约束错误，行数=1，最终 `version` 与 `checksum` 同行一致。 |
| A3 | 两个租户同 code 并发写：两行并存，租户隔离不被 UPSERT 冲突目标破坏。 |
| A4 | 既有 `TestSyncBuiltinSkills_Idempotent` 不回归（串行零变更仍 `changed=0`）。 |
| A5 | PostgreSQL（JSONB 列、唯一约束同 DDL）下同样的并发用例通过；测试以环境变量显式开启，未配置时 skip，不计入默认通过数。 |
| A6 | `gofmt` 干净、`go vet` 通过、`handlers/skill` 全包测试绿、`docs-gate` 通过。 |
| A7 | 反向验证：临时把实现退回裸 INSERT，确定性用例必须转红且错误含唯一约束冲突；恢复后转绿。证明测的是竞态本身，而不是「8 个协程恰好没撞上」的假绿——反向验证第一版就因纯齐发用例在旧实现上仍 PASS，才补了驱动层注入。 |

## 6. 安全与治理影响

- **多租户**：冲突目标显式含 `tenant_id`，A3 直接锁住跨租户合并回归。
- **RBAC / ABAC / Policy**：无新增权限码，HTTP Create/Update 的既有权限检查不变；
  本批不改变任何判定输入。
- **Audit**：技能定义的 checksum／updated_at 语义保持；`changed=false` 时不再产生写，
  审计上「无变更」与「无写入」重新一致。
- **副作用边界**：本批不涉及 Agent 工具调用；Action Gateway 的唯一执行入口未变。
