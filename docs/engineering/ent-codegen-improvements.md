# Ent Codegen 后续改进建议

> 基线：2026-11-21 只读审查结论（详见 `ent-codegen-analysis.md`、`ent-codegen-flow.md`）。
> 原则：**不重新设计生成机制、不回退到全量原地写入、不为提速牺牲完整性**。以下建议是在现有「隔离暂存 + 差分同步」架构上的增量加固。
> 每项包含：问题 → 原因 → 建议 → 改动范围 → 验收标准。

---

## P0 — 可能影响生成正确性或数据安全

### P0-1 CI 门禁只覆盖 Schema 路径，生成器 / 工具链变更漏检

- **问题**：`.github/workflows/backend-ci.yml:381` 的变更检测只匹配 `itsm-backend/ent/schema/**`。修改 `itsm-backend/go.mod`（`entgo.io/ent` 版本）、`cmd/entgen/main.go`（Package/IDType/配置）、`scripts/entgen.sh` 或 `ent/` 下手写文件（`ent/tx_exec.go` 等）时，`entgen-drift` job 的 `steps.schema.outputs.changed=false`，**完全不跑** `make entgen-check`。结果是：升级 ent 版本后生成代码仍是旧版本产物、或生成器配置变化后产物不同步，都能合并进主干。
- **原因**：门禁设计假设「只有 Schema 变才会导致生成物变化」，但当前机制的版本绑定（`scripts/entgen.sh:37-48` 从 go.mod 取版本 + `cmd/entgen/main.go` 的 cksum）和固定配置（`cmd/entgen/main.go:33-39`）本身也是生成物内容的一部分。
- **建议**：
  1. 把 `backend-ci.yml:381` 的 pathspec 扩展为多路径：`itsm-backend/ent/schema/**`、`itsm-backend/go.mod`、`itsm-backend/go.sum`、`itsm-backend/cmd/entgen/**`、`itsm-backend/scripts/entgen.sh`、`itsm-backend/ent/generate.go`。
  2. 增设**定时全量一致性校验**（如每周一次 `make entgen-check`，不受路径过滤限制），兜住任何遗漏路径或历史漂移。
  3. `entgen-drift` job 输出 `steps.schema.outputs.changed` 时同时打印触发路径，便于审计。
- **改动范围**：`.github/workflows/backend-ci.yml`（`entgen-drift` job 的 Detect schema change 步骤、新增 scheduled trigger 或独立 nightly job）。
- **验收标准**：① 在仅改 `go.mod` 的 ent 版本、不改 Schema 的测试 PR 上，`entgen-drift` 被触发且 `make entgen-check` 执行；② 分别篡改 `cmd/entgen/main.go`、`scripts/entgen.sh` 的测试 PR 均触发校验；③ 定时 job 在空变更周成功跑完 `make entgen-check`；④ CI 日志可见触发路径。

### P0-2 写回 `ent/` 无锁，并发生成可能产生瞬态不一致

- **问题**：`scripts/entgen.sh:68-71` 用 `mktemp -d` 保证**生成阶段**各会话隔离，但 `:149-156` 的**同步阶段**没有锁。两个并发 `make entgen`（本地两终端、或两个 CI job、或并行 worktree 共享同一 `ent/`）会各自判定差异并同时 `copyfile` / `unlink` 同一批文件；D 类删除与 A/M 写入交错时，可能出现「刚写入的文件被另一进程按旧清单删除」的瞬态窗口。当前内容确定（实测 staged 与 committed `client.go` SHA-256 一致）使最终态通常幂等，但过程不保证。
- **原因**：脚本只在暂存层做了并发隔离，未把「差分判定 → 写回」视为临界区。
- **建议**：在 Python 同步段前后加 flock 临界区，例如：
  ```bash
  exec 9>"${CACHE_DIR}/.sync.lock"
  flock 9
  # ... 内嵌 Python 同步（差分判定 + 写回）...
  flock -u 9
  ```
  使「比较 + 写回」原子化；拿到锁后**重新读取** `ent/` 当前状态再判定差异（避免用过期的 diff 清单写回）。macOS 自带 `flock` 缺失时可用 `mkdir` 锁目录 + trap 清理的可移植实现。
- **改动范围**：`itsm-backend/scripts/entgen.sh`（`ENTGEN_BIN`/`STAGE_DIR` 定义后至 Python 段结束）。
- **验收标准**：① 两个并发 `make entgen` 进程同时运行，`ent/` 最终与单次生成结果 SHA-256 全量一致；② 第二个进程等待锁而非报错；③ 进程被 kill 后锁可释放（trap）；④ 单个进程执行行为不变。

### P0-3 产物完整性校验只查 3 个锚点文件，无法发现部分生成缺失

- **问题**：`scripts/entgen.sh:81-86` 只检查 `client.go`、`mutation.go`、`migrate/schema.go` 是否存在。若生成器因 Schema 局部错误、模板异常或进程异常退出但恰好写出这三个文件，其余 974 个文件可能缺失或截断，仍会被同步进 `ent/`，编译期才暴露为 `undefined: xxx`。
- **原因**：完整性判据是「3 个锚点存在」，而非「生成文件集合完整且语法正确」。
- **建议**：在 Python 差分段之前增加三项校验（只读、失败快速）：
  1. **文件数下限**：`len(staged) >= ent/schema/*.go 数 × 4`（当前 137 × 4 = 548，实测 977；阈值取经验下限即可，用于抓「只写出少量文件」）。
  2. **头校验已有**（`:108-113`），保留并扩展为「每个文件非空且以 ent 头开头」。
  3. **每个 Schema 都有对应生成文件**：对 `ent/schema/<name>.go`，检查 staged 中存在 `<name>.go` 与 `<name>/where.go`（命名规则实测：`sla_state.go` → `slastate.go` 需做下划线去除映射，按实际转换实现）。
- **改动范围**：`itsm-backend/scripts/entgen.sh` 的 Python 内嵌段（`:99-113` 前后）。
- **验收标准**：① 人为删除 staged 中某一个实体的 `*_query.go` 后脚本 exit 1 且 `ent/` 未写；② 正常生成仍通过；③ 错误信息指明缺失的具体 Schema / 文件。

---

## P1 — 影响开发效率和 CI 稳定性

### P1-1 生成链缺少编译 / 测试闭环，错误定位滞后

- **问题**：`scripts/entgen.sh:88-157` 只做生成、差分与同步，**不执行 `go build` / `go vet` / `go test`**。CI 虽有独立的 build（`:231`）与 test（`:309`）job，但它们与 `entgen-drift` 并行，没有「生成 → 编译 → 测试 → 一致性」的因果门禁。生成物语法错误或类型不匹配要到 CI 其他 job 才暴露，且无法区分是生成问题还是业务代码问题。
- **原因**：改造 focus 在写入安全，未把验证纳入脚本职责。
- **建议**：
  1. 在 `scripts/entgen.sh` 同步成功后可选执行 `go build ./...`（受 `ENTGEN_VERIFY=1` 开关控制，默认开；`--check` 模式不写盘时改为在临时 GOPATH/暂存副本上校验，或明确跳过并打印提示）。
  2. 在 `backend-ci.yml` 的 `entgen-drift` job 中，`make entgen-check` 通过后追加 `go build ./...`、`go vet ./...`、关键包 `go test`（与现有 test job 去重，避免翻倍耗时；可用 `needs` 或复用 artifact）。
  3. 失败时打印「Schema 变更 → 生成 → 编译」的关联信息（哪个 Schema 变了、哪些文件被同步）。
- **改动范围**：`itsm-backend/scripts/entgen.sh`、`Makefile`（可新增 `entgen-verify` target）、`.github/workflows/backend-ci.yml`。
- **验收标准**：① 人为制造生成物/业务代码不匹配时，`make entgen` 在脚本内即报编译错误；② CI `entgen-drift` job 包含 build/vet 步骤且失败时定位到具体文件；③ 正常路径总耗时增加可接受（构建一次约分钟级）。

### P1-2 显式 gofmt / 格式一致性保证

- **问题**：脚本**没有显式 gofmt**（`:88-157`）。当前 ent v0.14.6 内置模板输出恰好是 gofmt 形态（实测 `ent/client.go`），但这是隐式约定：升级 ent 版本或未来引入自定义模板后，若输出非 gofmt，CI 的格式门禁（仓库有 `fmt-backend` / gofumpt 检查）会失败，且难以定位是生成还是手写问题。
- **原因**：把「entc 默认格式化」当成永久保证。
- **建议**：在 Python 同步前，对 staged 文件执行 `gofmt -w`（或复用 CI 同版本 `gofumpt`，与 `Makefile:275` 的 `fmt-backend` 对齐），确保任何模板版本的输出都规范化；`--check` 模式下比较 gofmt 前后差异，若生成物未格式化则视为漂移。
- **改动范围**：`itsm-backend/scripts/entgen.sh`（生成后、差分前插入格式化步骤），`Makefile` 的 `fmt-backend` target 可复用其工具版本。
- **验收标准**：① 人为让 staged 文件非 gofmt（测试夹具）后脚本自动格式化并同步；② `--check` 对未格式化生成物返回非零；③ 正常生成 `gofmt -l ent/` 输出为空。

### P1-3 CI 内存预检路径与阈值在 Linux runner 上未验证

- **问题**：`available_mb()` 的 Linux 分支用 `free -m | awk '/^Mem:/{printf "%d", $7}'`（`scripts/entgen.sh:54`），取的是**可用内存**列；默认阈值 6000 MB。GitHub ubuntu-latest runner 的可用内存是否满足、以及 entc 在 Linux 上的峰值内存是否同为 4.1 GB，**待验证**。若不满足，`entgen-drift` 会因预检 abort 而假失败，与生成正确性无关。
- **原因**：阈值与探测逻辑针对 macOS（runbook 实测环境）设计，未针对 CI 环境校准。
- **建议**：① 在 CI 环境实测一次完整 `make entgen-check`，记录 `available_mb()` 输出与 entc 峰值；② 在 CI job 中显式设置经验证的 `ENTGEN_MIN_FREE_MB`（或用 `ENTGEN_SKIP_MEM_CHECK=1` 并在 job 级别保证 runner 规格）；③ 把预检结果写入 job 摘要，避免静默 abort。
- **改动范围**：`.github/workflows/backend-ci.yml`（`entgen-drift` job env）、`scripts/entgen.sh:50-66`（日志增强）。
- **验收标准**：① CI 上 `entgen-drift` 连续 3 次成功且日志显示可用内存 > 阈值；② 阈值被明确记录（注释或 job env），不是默认值碰巧通过。

### P1-4 `--check` 仍付完整生成成本

- **问题**：`make entgen-check` 与 `make entgen` 走同一脚本，`--check` 模式仍执行完整 4~5 分钟生成（`scripts/entgen.sh:146-147` 只在最后决定是否写盘）。CI 注释（`:389-391`）已承认这一点。
- **原因**：没有内容级生成缓存（Q9 结论）：二进制缓存只省编译，暂存目录不复用。
- **建议**：在不改变生成完整性的前提下引入**输入指纹缓存**：对 `ent/schema/*.go` + `cmd/entgen/main.go` + `go.mod` 的 ent 版本计算哈希，若与上次成功生成一致且 `ent/` 生成物 SHA-256 清单也一致，则 `--check` 直接返回 0（声明「已知同步」）。指纹落盘在 gitignore 的 `.cache/entgen/` 中；CI 可用 actions/cache 持久化，本地复用缓存。注意：缓存命中必须同时校验输入与输出两侧，任一变化则回退到完整生成。
- **改动范围**：`itsm-backend/scripts/entgen.sh`（新增指纹计算与命中分支）、`.github/workflows/backend-ci.yml`（actions/cache key）。
- **验收标准**：① 连续两次 `--check`（无变更）第二次耗时显著低于完整生成；② 任一输入（Schema / main.go / go.mod）变化后缓存失效并执行完整生成；③ 篡改 `ent/` 任一生成文件后 `--check` 返回非零（输出侧校验生效）。

---

## P2 — 长期架构优化

### P2-1 按域拆 client，缩小单图规模

- **问题**：137 个 Schema 单图全量生成，`mutation.go` 190,667 行（`wc -l stage-0sLwup/mutation.go`），距预算探针 250,000 行仅约 5.9 万行余量（`scripts/entgen.sh:138-144`）。探针只告警不阻断；一旦越过可用内存上限，生成将再次「卡住」。
- **原因**：entc 全图串行模板重写是固有复杂度；`client.go` / `mutation.go` 是跨全部实体的单体文件，无法按 Schema 增量生成。
- **建议**：按业务域（ticket / incident / cmdb / bpmn / knowledge / sla / rbac …）评估多 client 方案。关键前置：① 用 AST 或 ent 图导出工具建立完整依赖图（Edge 对端、Mixin、annotation 外部包、predicate 引用），当前 104 个 Schema 含 Edge、74 个含反向 Edge（`grep -rl` 实测）；② 明确跨域 Edge 的处理策略（保留在单一共享域、或改 ID 外键 + 应用层 join）；③ 评估全仓 `itsm-backend/ent` import 调用点的改写成本。runbook 称此为「option B」，应先出设计评审，**本次审查不实施**。
- **改动范围**：设计文档 + 大量业务代码 import / 类型改写 + 新生成入口；超出当前「不重新设计」边界。
- **验收标准**：① 有完整依赖图与跨域策略文档；② 试点域拆分后该域生成时间/内存显著下降且 `go build ./...`、`go test ./...` 全绿；③ CI 能同时校验多 client 与生成物一致性。

### P2-2 依赖分析产物化，替代「单图隐式正确」

- **问题**：当前不拆图的正确性来自「全图天然包含依赖」，但没有显式的依赖清单。新增 Schema 时若误用跨模块 Mixin / 外部类型，只能在生成或编译时发现问题，且无法回答「改这个 Schema 会影响哪些生成文件」。
- **原因**：无 Schema 级依赖图（Q5/Q7 结论）。
- **建议**：新增只读分析脚本（如 `scripts/ent-deps.py`），解析 `ent/schema/*.go` 的 AST，输出：实体 → Edge 对端、Mixin 引用、外部 annotation 包、字段类型统计、以及 `Schema → 生成文件` 映射；产物提交或作为 CI artifact，供 P2-1 与日常评审使用。
- **改动范围**：新增脚本 + CI 可选步骤；不触碰现有生成代码。
- **验收标准**：① 脚本输出的实体数与 137 一致；② Edge 对端解析覆盖 `ticket.go:144-169` 等已知样例；③ 能正确报告 `sla_state.go` 使用 `ent/mixin`、`user.go` 使用 `migration/pii`。

### P2-3 退役或收敛旧入口 `go generate ./ent`

- **问题**：`ent/generate.go:3` 的 `go:generate` 仍可被任何人直接执行，绕过隔离暂存与差分同步，原地覆盖 `ent/`（runbook「方式 B：不推荐」）。这是当前写入安全的**旁路**。
- **原因**：为兼容旧脚本保留，但无护栏。
- **建议**：评估两条路径：① 在 `ent/generate.go` 的指令注释中加醒目警告并指向 `make entgen`（最小改动）；② 或把 `go:generate` 指令改为调用 `scripts/entgen.sh`（`//go:generate bash ../../scripts/entgen.sh`，需核实相对路径），使旧习惯也走安全路径。**不删除文件**，符合约束。
- **改动范围**：`itsm-backend/ent/generate.go`（一行）、runbook 同步。
- **验收标准**：① `go generate ./ent` 不再原地写入（改为安全路径或有明确拒绝提示）；② `make entgen` 行为不变；③ 文档与实际一致。

---

## 优先级汇总

| 优先级 | 编号 | 一句话 | 主要文件 |
|---|---|---|---|
| P0 | P0-1 | CI 门禁扩到生成器 / 工具链路径 + 定时全量校验 | `backend-ci.yml` |
| P0 | P0-2 | 写回 `ent/` 加 flock 临界区 | `scripts/entgen.sh` |
| P0 | P0-3 | 产物完整性校验从 3 锚点扩展到全集合 | `scripts/entgen.sh` |
| P1 | P1-1 | 生成链补 build/vet/test 闭环与错误定位 | `scripts/entgen.sh`、`Makefile`、`backend-ci.yml` |
| P1 | P1-2 | 显式 gofmt / gofumpt 对齐 CI | `scripts/entgen.sh` |
| P1 | P1-3 | CI 内存预检阈值实测校准 | `backend-ci.yml`、`scripts/entgen.sh` |
| P1 | P1-4 | 输入指纹缓存省 `--check` 完整生成 | `scripts/entgen.sh`、`backend-ci.yml` |
| P2 | P2-1 | 按域拆 client（需设计评审，本审查不实施） | 设计文档 + 全仓 |
| P2 | P2-2 | 依赖分析产物化 | 新增 `scripts/ent-deps.py` |
| P2 | P2-3 | 收敛旧 `go generate ./ent` 旁路 | `ent/generate.go` |

> 执行纪律：以上任何改动均需先走人工确认；不得在未验证依赖关系前拆分 Schema；不得为提速牺牲生成完整性（如用「只生成变化 Schema」替代全图生成）。
