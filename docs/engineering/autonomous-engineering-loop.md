# AI-Native 软件工程闭环骨架

> Status: current
> 目标：让 CloudMesh 不仅交付 AI-Native ITSM 产品，也具备一套能持续改进自身的工程系统。
> 本文件只描述**观察、校验与回写**机制，不引入第二套审批引擎，不改变部署/权限策略。

## 1. 为什么需要它

仓库已有 `itsm-agent` 的 issue → worktree → PR → 人工审批链（`itsm-agent/internal/orchestrator/orchestrator.go`），也有会话账本（`scripts/parallel/session-ledger.py`）和发布证据脚本（`scripts/release-evidence.sh`）。但它们都没有把一次自主开发的**决策依据、代码证据、测试退出码、CI 结论、风险与回滚**固化成机器可读、可回写的产物；整改 backlog 仍靠手工改表，报告没有消费者，闭环到"写完报告"为止。

本骨架补齐三件事：

1. **Run Manifest schema**：一次 run 的唯一机器可读事实源；
2. **validator**：在写回前拒绝缺证据/缺退出码/状态非法的清单；
3. **writeback**：把 run 记录 upsert 到 JSONL 账本，并按清单把 backlog 状态回写到表格，默认 dry-run，写入前校验快照摘要防止覆盖他人改动。

## 2. 产物与位置

| 产物 | 路径 | 作用 |
|:--|:--|:--|
| Manifest schema | [`scripts/parallel/run-manifest.schema.json`](../../scripts/parallel/run-manifest.schema.json) | 定义必填字段、状态枚举与证据约束 |
| 工具 | [`scripts/parallel/run-manifest.py`](../../scripts/parallel/run-manifest.py) | `validate` / `writeback` 两个子命令 |
| 测试 | [`scripts/__tests__/test_run_manifest.py`](../../scripts/__tests__/test_run_manifest.py) | 8 个标准库单测/CLI 测试 |
| Run 账本 | `docs/audits/autonomous-runs.jsonl` | 每次 run 一条 JSON 记录，按 `run_id` 幂等 upsert |
| 整改 Backlog | [`docs/audits/remediation-backlog.md`](../audits/remediation-backlog.md) | 状态回写目标，ID 必须已存在 |
| 示例 Manifest | [`docs/audits/run-manifests/cm-auto-20261125-01.json`](../audits/run-manifests/cm-auto-20261125-01.json) | 首个真实 run 的完整证据 |

Schema 使用 JSON Schema 的受支持子集：`type`、`required`、`properties`、`items`、`enum`、`pattern`、`minLength`、`minimum`、`minItems`。不支持的子集不会假装校验；这是有意选择，避免引入外部依赖或制造"假闭环"（参见 `docs/engineering/skill-output-schema-validation-assessment.md`）。

## 3. Manifest 最小契约

必填顶层字段：`schema_version`、`run_id`、`generated_at`、`status`、`repository`、`base_commit`、`branch`、`task`、`tests`、`ci`、`security`、`approvals`、`risks`、`blockers`、`deployment`、`rollback`。

关键约束：

- `status` ∈ `success | partial | blocked`；
- `tests[].exit_code` 必须是整数。**没有真实退出码的运行不要放进 `tests`**，写入 `blockers`；禁止用 0 代替未知结果；
- `ci[].status` ∈ `pass | fail | pending | skipped`；基线既有失败用 `baseline: true` 标注；
- `task.evidence[]` 至少一条 `path + fact`，用 `path`/`lines` 指向代码锚点；
- `security.policy_changed` / `db_migration` / `deploy` 必须为布尔值；高风险操作不能在这里被"合法化"；
- `backlog_updates[]` 的 `status` ∈ `confirmed | suspected | resolved | blocked`，ID 必须已经存在于 backlog；
- 带 `backlog_updates` 且 `--apply` 时，必须提供 `backlog_snapshot_sha256`，且必须与当前文件摘要一致。

## 4. 使用方式

```bash
# 校验清单（只读，退出码 0/1）
python3 scripts/parallel/run-manifest.py validate docs/audits/run-manifests/cm-auto-20261125-01.json

# 预览回写（默认 dry-run，不写文件）
python3 scripts/parallel/run-manifest.py writeback docs/audits/run-manifests/cm-auto-20261125-01.json

# 显式应用回写（要求快照摘要匹配）
python3 scripts/parallel/run-manifest.py writeback docs/audits/run-manifests/cm-auto-20261125-01.json --apply

# 测试
python3 -m unittest discover -s scripts/__tests__ -p 'test_*.py' -v
```

`writeback` 的行为：

- 同一 `run_id` 再次写入时替换旧记录，不产生重复行；
- backlog 状态单元格改为 `` `<status>`（Run `<run_id>`） ``，处置列追加 `Run <run_id>: <note>（<evidence>）`，重复运行不会重复追加；
- 找不到 backlog ID、账本存在非法 JSON、或快照不匹配时整体拒绝，不做部分写入；
- 只写这两个明确目标文件，不修改源码、CI 配置或权限策略。

## 5. 治理边界

- 本工具**不合并 PR、不触发部署、不执行迁移、不修改权限码/策略**；这些仍需既有人工审批链（`AGENTS.md`、`docs/audits/remediation-backlog.md`）。
- 并行开发仍遵守 `AGENTS.md`：独立 worktree、会话账本 `enter` / `diff` / `check --commit`、显式 `git add`、提交前 `gate-at-head.sh` 归因。
- 新增自动化不能绕开"数据库/权限/部署变更需人工审批"的规则；Manifest 中的 `security` 布尔字段只是声明，真正的门禁在既有流程。
- 本轮未把 `writeback` 接入 CI 强制检查：先以可验证的库内工具运行，避免在没有消费者的情况下形成新的假门禁。

## 6. 首个闭环实例

`cm-auto-20261125-01` 针对 `AUDIT-070`（AI/MCP `update_ticket` 写路径绕过行级范围）：

- 证据锚点：`internal/services/tool_registry.go:776` 传 `(0, "")`；`internal/services/ticket_service.go:888-892` 直接放行；
- 红/绿：回退修复 `exit 1` → 恢复修复 `exit 0`；定点 4 包 `exit 0`；`go vet` / `gofumpt` `exit 0`；
- PR #131，Lint/Build/Fresh install/Docs C.6 通过；`internal/services` 包在 CI 中通过；`Test` 整体、Security Scan、GA Gate 因基线问题仍红；
- 未部署、未迁移、未改策略；回滚为 revert 提交。

该实例用本工具把 AUDIT-070 从 `confirmed` 回写为 `resolved`，并在 `autonomous-runs.jsonl` 留下可机器消费的一条记录。它同时暴露了闭环仍未完成的部分：CI 红项归因、Ent Codegen 门禁被跳过、agent 一级身份未实现——这些进入下一轮 backlog，而不是被报告"写完即结束"。
