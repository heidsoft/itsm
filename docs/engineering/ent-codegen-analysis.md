# Ent Schema 代码生成机制审查与工程化评估

> 审查日期：2026-11-21 ｜ 审查范围：`itsm-backend` 及仓库根 Makefile / `.github/workflows`
> 审查性质：只读审查。本次未修改任何生成代码、Schema、模板或生成产物。
> 证据规则：每个结论都附文件路径 / 函数名 / 命令；无法从代码确认的一律标注「待验证」。

---

## 0. 一句话结论

当前实际采用的是 **「全图生成 + 隔离暂存 + 按 SHA-256 内容差分的原子同步」** 模式（`entc.Generate("./ent/schema", cfg)` 每次仍扫描全部 137 个 Schema 生成整张图到临时目录，再只把内容真正变化的文件写回 `ent/`），**不是模块化生成、不是增量生成、不是分阶段生成、也没有目录隔离**；AI 的改造解决的是「写入安全 / 重复编译 / diff 不可审阅」，**没有解决生成耗时与内存峰值本身**。

---

## 1. 当前 Ent Codegen 架构

### 1.1 生成入口清单（含真实性与是否生效判定）

| 入口 | 文件 / 命令 | 是否真正执行生成 | 状态 |
|---|---|---|---|
| **主入口（当前）** | `Makefile:264-268` 的 `entgen` / `entgen-check` → `bash itsm-backend/scripts/entgen.sh [--check]` | 是，间接 | **生效，推荐** |
| **生成器实现** | `itsm-backend/cmd/entgen/main.go:40` → `entc.Generate("./ent/schema", cfg)`（`//go:build entgen`） | 是，实际生成点 | **生效**，由脚本按版本+源码哈希预编译 |
| 旧入口（保留兼容） | `itsm-backend/ent/generate.go:3` → `//go:generate go run -mod=mod entgo.io/ent/cmd/ent generate ./schema` | 是，原地写入 | 仍在仓库，但**非当前流程**；`make entgen` 不调用它 |
| CI 一致性校验 | `.github/workflows/backend-ci.yml:357-392` 的 `entgen-drift` job → `make entgen-check` | 是（以 check 模式） | 生效，**仅在 Schema 路径变化时触发** |
| 迁移执行（非 codegen） | `itsm-backend/cmd/migrate/main.go:1`（`//go:build migrate`） | 否，执行 SQL 迁移 | 与 codegen 无关，勿混 |

判定依据：`grep -rn "entc.Generate\|entc.LoadGraph"` 全仓只有 `cmd/entgen/main.go:40` 一处（`entc.LoadGraph` **无任何调用**）；`grep -rn "go:generate"` 中与 ent 相关的只有 `ent/generate.go:3`；Makefile 中 `entgen:` / `entgen-check:` 两个 target 分别位于 264 / 267 行。

### 1.2 Schema 目录结构

- `itsm-backend/ent/schema/*.go`：**137 个 `.go` 文件，全部 `package schema`，平铺在单一目录**（`find ent/schema -name '*.go' | wc -l` = 137；`grep -h '^package' ent/schema/*.go | sort -u` = `package schema`）。
- 无子目录、无按域拆分、无独立包（`find ent/schema -type d` 只有目录本身）。
- `ent/mixin/tenant_timestamps.go` 是 `ent/` 下唯一的 mixin 包文件，且只被 1 个 Schema 引用（`grep -rln "ent/mixin" ent/schema/*.go` → 仅 `sla_state.go`）。
- `ent/` 顶层 981 个非 schema 的 `.go`（含生成物与手写文件如 `ent/tx_exec.go`）。

### 1.3 生成产物规模（实测）

| 指标 | 数值 | 来源 |
|---|---|---|
| 生成文件数 | 977（暂存目录全部为 `.go`） | `find .cache/entgen/runs/stage-0sLwup -type f | wc -l` |
| 生成总行数 | 746,991 | `find stage-0sLwup -name '*.go' -exec cat {} + | wc -l` |
| `mutation.go` 行数 | 190,667 | `wc -l stage-0sLwup/mutation.go` |
| `client.go` / `migrate/schema.go` | 946 KB / 282 KB | `ls -lh ent/client.go ent/migrate/schema.go` |
| 生成器版本 | `entgo.io/ent v0.14.6`，Go `1.26.10` | `itsm-backend/go.mod:3,11` |

### 1.4 自定义模板 / Hooks / Extensions

**未发现**任何 entc 扩展、自定义模板目录、`entc.go` 配置或生成期 hook：

- `cmd/entgen/main.go:33-39` 的 `gen.Config` 只设置 `Package`、`Target`、`IDType` 三个字段，没有 `Templates`、`Hooks`、`Extensions`、`FeatureFlags`。
- 全仓无 `entc.go`（`find . -name entc.go` 无结果）。
- 生成代码中出现的 `hook` 目录（`ent/hook/`）是 ent **运行期** hook，不是 codegen hook。
- Schema 中的 `Annotations`（`ent/schema/user.go:29,33,47,55`，来自 `itsm-backend/migration/pii`）是 **ent 字段 annotation 数据**，不触发额外代码生成路径（生成结果仍是标准 ent 代码）。

---

## 2. 原全量生成机制（改造前）

### 2.1 执行方式

改造前（`docs/delivery/entc-codegen-runbook.md` 由 commit `54bde34f1` 修改前的版本，以及 `ent/generate.go` 保留的入口）：

1. **标准入口**：`cd itsm-backend && go generate ./ent`，等价执行 `ent/generate.go:3` 的 `go run -mod=mod entgo.io/ent/cmd/ent generate ./schema`。特点：每次**重新编译 entc**，`-mod=mod` 可能改写 `go.mod` / `go.sum`（runbook「方式 B」原文：「每次重编译 entc、原地写入，且可能改动 go.mod/go.sum」）。
2. **应急绕行**：`go build -o /tmp/entc entgo.io/ent/cmd/ent`（一次性，约 20 秒 / 22 MB）后 `cd ent && /tmp/entc generate ./schema`——**原地写入 `ent/`**。
3. 三者都是 **entc 全图串行重写**：`entc` 读取 `./schema` 全部 Schema → 构建完整图 → 对每个 entity 跑模板 → 直接覆盖 `ent/` 下全部生成文件。

### 2.2 真正的卡点（不是笼统的「Schema 太多」）

runbook 记录的实测与事故（改造前后同一事实基础）：

- **内存与耗时**：macOS 16GB 实测峰值 **4.1 GB**、耗时 **4 分 40 秒**（132 Schema / 71.7 万行时）；当前 137 Schema / 74.7 万行 / 单 `mutation.go` 19.1 万行。
- **根因定位**：runbook 明确写「生成慢的根因是**全图串行重写**（`entc/gen/graph.go:242` 逐个 node 跑模板），不是 ent 运行时本身；……真正缩小单图规模需要按域拆 client」。 entc 每次都必须把 137 个 Schema 全部加载、校验边关系、再串行套用模板生成 977 个文件，其中 `mutation.go` 单文件近 19 万行（每个实体的全部 mutation 方法叠加）。
- **历史失败的真实成因**（runbook「六、历史背景」）：① 可用内存只剩 590 MB~1.2 GB（Docker + 多 worktree + IDE 占满）；② 习惯性用管道 `| tail`，进程组被杀、输出全丢、报 137，误判为 OOM；③ 错误设置 `GOMEMLIMIT=14GiB` 反而更早被 kill；④ Docker 容器配额 5.8 GB 不够。
- **破坏性后果**（`scripts/entgen.sh:9-11` 注释、commit `54bde34f1` 说明）：原地写入时生成被中断 → `ent/` 留下数百个半成品文件 → 产出 **854 文件不可审阅 diff**，需要 `git checkout -- itsm-backend/ent` 整体回滚。

**结论**：卡住 = 全图串行重写的固有耗时/内存 × 原地写入的不可恢复性 × 错误的环境与调用方式（管道 / GOMEMLIMIT / 低内存）三者叠加；Schema 数量只是放大因子，不是唯一原因。

---

## 3. 当前工程化生成机制（改造后）

### 3.1 改造来源

唯一一次相关提交：`54bde34f1 build(entgen): 隔离暂存与差分同步生成代码，失败不写 ent/`（2026-11-19，作者 heidsoft）。该 commit 改动 7 个文件：`.github/workflows/backend-ci.yml`、`CHANGELOG.md`、`Makefile`、`README.md`、`docs/delivery/entc-codegen-runbook.md`、`itsm-backend/cmd/entgen/main.go`、`itsm-backend/scripts/entgen.sh`；**未改动任何 `ent/` 下生成产物**（commit stat 可证）。

### 3.2 逐项回答审查问题

**Q1 原全量如何执行？** 见 §2.1：`go generate ./ent` 或 `/tmp/entc generate ./schema`，每次全图重写、原地覆盖。

**Q2 现在采用什么模式？** 全图生成 + 隔离暂存 + 内容寻址差分的原子同步。证据链：`scripts/entgen.sh:71` 建 `stage-XXXXXX` → `:75` 生成到该目录 → `:88-157` Python 做 SHA-256 比较后只 `copyfile`/`unlink` 变化的文件。

**Q3 是模块化 / 增量 / 分阶段 / 自定义 Pipeline 吗？** **都不是**。生成侧仍是单次 `entc.Generate("./ent/schema", cfg)`（`cmd/entgen/main.go:40`），无 `FeatureFlags`、无分批、无 per-schema 生成；「增量」仅体现在**写回侧按内容差分**（`scripts/entgen.sh:116-127`）。

**Q4 是否通过 Schema 目录隔离实现独立生成？** **否**。`ent/schema/` 是唯一平铺目录，137 个文件同一 package；`cmd/entgen/main.go:40` 硬编码 `"./ent/schema"`。`verify-11FyAG` 等 `.cache` 目录里的 `ent/` 副本是**验证用临时产物**（`sync_check.py`、tamper 测试），不是生产入口。

**Q5 是否存在跨模块依赖？** **存在且密集**：
- Edge：104 / 137 个 Schema 含 `edge.To`/`edge.From`，其中 74 个含反向 `edge.From`（`grep -rl` 计数）。示例 `ent/schema/ticket.go:144-169`：`Ticket` 通过 `edge.To` 关联 14+ 实体（TicketComment/TicketAttachment/TicketTag/ApprovalRecord/…），并通过 `edge.From(...User.Type).Ref(...)` 等建立 4 条反向边。跨实体引用计数前列：`ticket` 11、`tickets` 10、`user` 9、`tags` 7、`sla_definition` / `conversation` / `children` 各 5。
- Mixin：仅 `ent/mixin/tenant_timestamps.go`，只被 `ent/schema/sla_state.go` 引用；`ent/schema/ticket_workflow_record.go:19` 的 `Mixin()` 返回 `nil`（注释说明刻意不用 CreateTime mixin，因为 DB 列名是 `created_at`）。
- 外部类型 / 自定义字段：`field.JSON` 用于 76 个 Schema、`field.Enum` 用于 11 个；`ent/schema/user.go:11,29,33,47,55` 从 `itsm-backend/migration/pii` 引入 annotation。字段类型集合：`Bool/Bytes/Enum/Float/Int/JSON/String/Text/Time`（`grep -rho 'field\.[A-Z]...' | sort -u`）。
- **因此不能把 Schema 按目录拆分后独立生成**：任何 per-domain 子集都会缺失边对端实体、`predicate`、反向遍历方法，导致生成不完整或不一致。

**Q6 如何判断哪些 Schema 变了？** 分两层：
- **CI 层**：`.github/workflows/backend-ci.yml:373-385`，`git diff --name-only "$BASE" HEAD -- 'itsm-backend/ent/schema/**' | grep -q .`，设置 `steps.schema.outputs.changed`，只有 `true` 才跑 `make entgen-check`（`:388-392`）。注释明确「Only schema-changing PRs pay the ~5 minute generation cost」。
- **生成本身**：**不检测**。`cmd/entgen/main.go:40` 对 `./ent/schema` 无条件全量生成；`scripts/entgen.sh` 只在生成完成后比较内容差异。

**Q7 如何决定哪些代码需要重新生成？** 生成侧：全部。写回侧：`scripts/entgen.sh:99-127` 的 Python 内嵌脚本——
- `list_files()` 枚举 staged 与 `ent/` 全部文件；`digest()` 用 `hashlib.sha256`；
- 逐 staged 文件比较 `ent/` 对应路径，产生 `("A"|"M", rel)`（`:116-119`）；
- 对 `ent/` 独有文件，跳过 `generate.go` 与 `schema/`，只把**带 ent 头** `// Code generated by ent, DO NOT EDIT.` 的标为 `("D", rel)`（`:121-127`），从而保护手写 helper（`mixin/`、`migrate/*_test.go`、SQL 迁移）。

**Q8 是否依赖统一模板 / 版本？** 版本由 `itsm-backend/go.mod:11` 的 `entgo.io/ent v0.14.6` 锁定；脚本 `:37-48` 从 go.mod 解析版本，并以 `cksum(cmd/entgen/main.go)` 作为源码哈希，生成缓存二进制名 `entgen-v0.14.6-<hash>`，变化时自动重建。模板使用 ent v0.14.6 内置模板（无自定义模板目录）。**注意**：`gofmt` 未显式执行，但 entc 内置模板已格式化（实测 `ent/client.go` 为标准 gofmt 形态）。

**Q9 是否存在生成缓存？** 有两类，均**非内容级生成缓存**：
- **二进制缓存**：`.cache/entgen/bin/entgen-v0.14.6`（与 `entgen-v0.14.6-1238035955`，同名不同哈希，21 MB Mach-O）按版本+源码哈希命中（`scripts/entgen.sh:42-48`）。
- **暂存运行目录缓存**：`.cache/entgen/runs/stage-0sLwup`、`tampered-GEKmlv`、`verify-11FyAG`，脚本 `:68-71` 明确「Nothing is ever bulk-deleted」，每次 `mktemp -d` 新建；`.cache/entgen/` 由 `.gitignore:298` 忽略（当前 147 MB）。这些目录**不参与下一次生成的输入**。
- 无「哪些文件可跳过生成」的缓存，每次仍走完整 4~5 分钟生成。

**Q10 如何保证本地结果一致？** 已具备：版本由 go.mod 锁定、生成器源码哈希绑定、固定 `gen.Config.Package = "itsm-backend/ent"`（避免 `ent generate --target` 把 import 改写成暂存包，`cmd/entgen/main.go:5-11,34`）、固定 `IDType = field.TypeInt`（`:38`）、SHA-256 精确比较、CI `--check` 门禁。实测一致性证据：`.cache/entgen/runs/stage-0sLwup/client.go` 与 `ent/client.go` 的 SHA-256 完全相同（`27e1c6c9…`），说明确定性成立。

### 3.3 真实执行流程（应用模式）

1. `make entgen` → `bash itsm-backend/scripts/entgen.sh`（`Makefile:264-265`）。
2. 解析模式：默认 `apply`，`--check` 为校验（`scripts/entgen.sh:14-26`）。
3. 从 `go.mod` 取 ent 版本；`cksum` 取 `cmd/entgen/main.go` 哈希；若缓存二进制不存在则 `go build -tags entgen -o … ./cmd/entgen`（`:37-48`）。
4. 内存预检：`available_mb()` 用 `vm_stat`（macOS）或 `free -m`（Linux）；默认阈值 `ENTGEN_MIN_FREE_MB=6000`，可用 `ENTGEN_SKIP_MEM_CHECK=1` 跳过（`:50-66`）。
5. `STAGE_DIR=$(mktemp -d runs/stage-XXXXXX)`（`:71`）。
6. 运行生成器：`entgen -target "$STAGE_DIR"` → `entc.Generate("./ent/schema", cfg)`（`cmd/entgen/main.go:40`），输出重定向到 `.cache/entgen/last-run.log`（`:75`）。
7. 产物完整性检查：必须存在 `client.go`、`mutation.go`、`migrate/schema.go`（`:81-86`）。
8. 内嵌 Python：校验所有 staged 文件头、SHA-256 比较、列出 A/M/D、`mutation.go` 超 25 万行告警（预算探针）。
9. `--check` 模式：有差异 `exit 1`，否则 `exit 0`，**不写盘**（`:146-147`）。`apply` 模式：按差异 `copyfile` / `unlink`，打印 `[entgen] synced into ent/`（`:149-156`）。

### 3.4 辅助验证痕迹（非生产流程）

`.cache/entgen/runs/` 下的 `sync_check.py`（与脚本内嵌 Python 内容一致）、`tampered-GEKmlv/`（`client.go` 被人为追加 `// tamper for reverse validation`，SHA-256 与 stage 不同）、`verify-11FyAG/`（含 `generate.go` 副本的验证目录）是改造时的**反向验证 / 演练产物**：用于证明 diff 检测能抓到篡改、暂存写坏不会污染 `ent/`。它们不是运行时依赖，也不应被当作生产入口。

---

## 4. Schema 依赖关系分析

### 4.1 为何「按目录拆分即可独立生成」不成立

当前 `entc` 把 137 个 Schema 当作**一张图**处理。每个实体生成物包括：`ent/<entity>.go`、`_create.go`、`_update.go`、`_delete.go`、`_query.go`、`ent/<entity>/where.go`、`predicate/<entity>.go` 片段、`mutation.go` 中的 `<Entity>Mutation`、`client.go` 中的 `<Entity>Client`。

- `mutation.go` 与 `client.go` 是**跨全部实体的单体文件**：任一实体变更都会导致这两个巨型文件重新生成 —— 这是 `mutation.go` 19 万行的由来，也是按 Schema 切分无法做到「只改一个实体只生成一个文件」的根本原因。
- Edge 是双向的：`ticket.go:144-169` 的 `edge.To("comments", TicketComment.Type)` 要求在生成 `Ticket` 时图中存在 `TicketComment`；反之 `TicketComment` 的 `edge.From("ticket", Ticket.Type).Ref("comments")` 要求 `Ticket` 存在。拆成两个独立 `entc.Generate` 图会各自缺对端类型。
- 手写 helper 依赖生成包名：`ent/tx_exec.go` 直接断言 `*entsql.Tx`；所有业务代码 import `itsm-backend/ent` 单包 client。拆 client 意味着全仓调用点改写，超出本次「不重新设计」的范围。

### 4.2 Migration / 其他生成

- **Migration**：`ent/migrate/schema.go` 带 ent 头，是 codegen 产物；数据库实际变更由 `cmd/migrate/main.go`（`//go:build migrate`）执行版本化 SQL（`go run -tags migrate … -up/-down`），**不属于 ent codegen 链**。schema 变更后需要人工补 SQL 迁移（runbook「五、生产环境别忘了 ent migrate」）。
- **GraphQL / OpenAPI**：未发现 ent 驱动的 GraphQL / OpenAPI 生成器。Swagger 走独立入口 `Makefile:270-273` 的 `swag init`（手写注解），与 ent codegen 无调用关系。
- **Ent Client / Mutation / Query / Predicate**：全部由同一次 `entc.Generate` 产出，输出到单一 `ent/` 包；`client.go` / `mutation.go` 为跨实体共享文件。

---

## 5. 生成效率瓶颈

| 环节 | 改造前 | 改造后 | 说明 |
|---|---|---|---|
| entc 重新编译 | 每次 `go run -mod=mod`（慢，且可能改 go.mod/go.sum） | 按版本+源码哈希缓存二进制 | **已改善**（`scripts/entgen.sh:42-48`） |
| Schema 加载 / 全图生成 | 全量、串行 | **仍全量、串行** | **未改善**，仍是 4~5 分钟主耗时 |
| 峰值内存 | ~4.1 GB | 同样量级 | **未改善**；改为提前预检 + 优雅失败（`:50-66`） |
| 写回 ent/ | 全量覆盖，中断留半成品 | 仅写 SHA-256 不同的文件，失败/中断零写入 | **显著改善**（`:9-11,116-156`） |
| diff 可审阅性 | 曾出现 854 文件 diff | 通常仅几个文件变化 | **显著改善** |
| 低内存机器可用性 | 直接 OOM / exit 137 | 预检拦截并给出可操作提示 | **改善**，但需调阈值或关 Docker/IDE |

**关键判断**：本次改造**没有减少生成工作量**，减少的是「失败代价、重复编译、无效写入与审阅噪音」。`mutation.go` 190,667 行距 25 万行预算探针仅约 5.9 万行余量（`scripts/entgen.sh:138-144`），探针只是**告警不阻断**，规模问题仍未解决。

---

## 6. 当前方案的优点和风险

### 优点

1. **写入原子性**：失败、中断、内存不足时 `ent/` 一个字节不写（暂存 + 后置同步）；从根上消除半成品事故（`scripts/entgen.sh:9-11,75-79`）。
2. **可重复 / 确定性**：固定 Package 路径避免 `--target` 污染 import（`cmd/entgen/main.go:5-11,34`）；实测 staged 与 committed `client.go` SHA-256 一致。
3. **最小 diff**：SHA-256 差分只同步变化文件，保留手写 helper 不被误删（`:121-127`）。
4. **版本绑定**：生成器二进制名含 ent 版本与源码哈希，依赖升级自动重建（`:37-48`）。
5. **CI 门禁**：`entgen-drift` 只在 Schema 变化的 PR 上跑 `--check`，避免每个 PR 都付 5 分钟（`backend-ci.yml:388-392`）。
6. **预算探针**：`mutation.go` 行数超阈值时提前告警（`:138-144`）。

### 风险（按影响排序）

1. **CI 门禁覆盖面过窄（正确性 / 数据风险）**：`backend-ci.yml:381` 只检测 `itsm-backend/ent/schema/**`。若只改 `go.mod` 的 ent 版本、`cmd/entgen/main.go`、`scripts/entgen.sh` 或手写 `ent/` 文件（如 `tx_exec.go`），CI **不会触发** `entgen-check`，生成代码可能与工具链不同步而仍合并。这是当前最大的 correctness 风险。
2. **扩展性与规模天花板（可维护性）**：仍是单图全量生成，`mutation.go` 已 19 万行、距探针 25 万行余量有限；探针只告警不阻断；一旦越过硬件内存上限，4~5 分钟生成会再次变成「卡住」。按域拆 client 是已知方向（`runbook` 开头与 `scripts/entgen.sh:141-143`），但当前无实施、也无依赖分析产物。
3. **验证链不完整（CI 一致性 / 故障定位）**：`scripts/entgen.sh` **不执行 gofmt / go vet / go build / go test**（`:88-157` 仅做同步与行数探针）；CI 虽有独立 build/test job（`backend-ci.yml:231,309`），但没有「生成 → 编译 → 测试 → 一致性」的闭环门禁。`--check` 失败只打印差异文件列表，无「如何修复」指引；内存预检在 Linux CI 上依赖 `free -m` 路径（`:54`），CI runner 是否满足 6000 MB 阈值 **待验证**；CI 未设 `ENTGEN_MIN_FREE_MB`，若 runner 可用内存不足，`entgen-drift` 可能假失败。此外 `--check` 仍会完整生成一遍（约 5 分钟），CI 成本未省。

### 其他需注意的次要风险

- **并发安全**：每次 `mktemp -d` 独立暂存目录，多会话并行**生成阶段**互不覆盖（`:68-71`）；但**同步阶段**写回 `ent/` 无锁，两个并发 `make entgen` 可能同时判定差异并删除/写入同一文件。因内容确定通常幂等，但 D 类删除与写入交错时理论上可产生瞬态不一致；**待验证**实际影响。
- **生成结果入 Git**：`ent/` 为 tracked（本次 `git status` 未显示 `ent/` 变化），符合「生成产物提交 Git」；但脚本本身不检查 post-sync 的 git diff，需人工 `git status`。
- **旧入口存活**：`ent/generate.go:3` 仍可被 `go generate ./ent` 直接调用并原地写入，若有人误用会绕过所有保护。

---

## 7. 本地开发与 CI 执行方式

### 本地

```bash
# 仓库根目录（推荐）
make entgen          # 改过 ent/schema/** 后运行：隔离生成 + 差分同步
make entgen-check    # 只校验一致性，不写 ent/（CI 同款）
# Schema 未改时 ent/ 应零 diff；随后手动 go build ./... 与 go test（见 runbook 三）
```

可调环境变量（`scripts/entgen.sh:58-59,138`）：`ENTGEN_MIN_FREE_MB`（默认 6000）、`ENTGEN_SKIP_MEM_CHECK=1`、`ENTGEN_MUTATION_BUDGET`（默认 250000）。

### CI

`.github/workflows/backend-ci.yml:357-392`：`entgen-drift` job → checkout（fetch-depth 0）→ setup-go（go-version-file `itsm-backend/go.mod`，cache by go.sum）→ 计算 Schema 路径 diff → 若变化则 `make entgen-check`。无生成后的 build / vet / test 闭环。

### 自动化验证能力盘点

| 能力 | 当前状态 | 证据 |
|---|---|---|
| Schema 变更检测 | 仅 CI 路径 diff | `backend-ci.yml:381` |
| 生成任务依赖分析 | **无** | `cmd/entgen/main.go:40` 单图全量 |
| 生成前后文件变更检测 | 有（SHA-256） | `scripts/entgen.sh:99-136` |
| 生成结果提交 Git | 是（tracked） | `git ls-files ent/`；脚本不自动提交 |
| 生成后 gofmt | 无显式步骤（entc 内置格式） | 脚本无 gofmt；`ent/client.go` 实测已格式化 |
| go test ./... / go vet | **不在生成链**，CI 独立 job | `backend-ci.yml:231,309`；脚本 `:88-157` 无 |
| 生成一致性检查 | 有（`--check`） | `scripts/entgen.sh:14-17,146-147` |
| 生成失败错误定位 | 有日志 + tail -40 | `scripts/entgen.sh:75-79` |
| CI 检测 Schema 与生成代码同步 | 有，但仅 Schema 变化时 | `backend-ci.yml:388-392` |

---

## 8. 最终工程化评估结论

| 维度 | 评级 | 理由 |
|---|---|---|
| 生成效率 | 部分改善 | 缓存二进制 + 内存预检 + 差分写回减少无效工作，但**全图 4~5 分钟生成未变** |
| 正确性 | 较好 | 全图生成天然不漏依赖；产物完整性检查；实测确定性 |
| 依赖管理 | 正确但保守 | 不拆图 → 不丢边/类型；代价是无法缩小单图规模 |
| 可重复性 | 好 | 版本+源码哈希+固定 Package/IDType+SHA-256；实测字节一致 |
| 并发安全 | 部分 | 暂存隔离好；写回 `ent/` 无锁（待验证影响） |
| 可维护性 | 中 | 新增 Schema 只需放文件 + `make entgen`；但单图规模逼近探针阈值 |
| CI 一致性 | 有缺口 | 仅 Schema 路径触发，生成器/工具链变更漏检；无生成后 build/test 闭环 |
| 故障恢复 | 好 | 失败零写入；暂存不删；可安全重跑 |

**总评**：改造是一次**针对写入安全与工程習慣的精准加固**，方向正确、实现简洁、证据充分；但它把「生成慢/内存高」的问题**从「破坏性事故」降级为「可控成本」**，并未消除。当前方案可安全使用，但需在 CI 触发面、验证闭环与规模治理三处补齐，才能称为「可理解、可重复、可验证、适合大规模 Schema」的工程化能力。改进建议见 `docs/engineering/ent-codegen-improvements.md`，现状流程见 `docs/engineering/ent-codegen-flow.md`。
