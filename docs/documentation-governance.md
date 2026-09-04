# 文档状态与事实源

> 最后审查：2026-09-04。本文定义仓库文档的权威层级；历史测试报告不得作为当前功能或发布状态的依据。

## 权威层级

出现冲突时按以下顺序判断：

1. 当前源码、数据库迁移、运行时 API 与当前 CI 结果。
2. 根目录 `README.md`、`ROADMAP.md`、`CHANGELOG.md` 和 `AGENTS.md`。
3. `docs/product/`、`docs/architecture/`、`docs/DEPLOYMENT_OPTIMIZATION.md`、`docs/testing/` 中未标记为 historical/superseded 的规范。
4. `plans/`：计划和设计输入，不证明已经实现。
5. `output/`、`docs/review/`、`docs/test-plan/`：带日期的历史快照和测试证据，只证明当次运行。
6. `docs/archive/`：归档资料，不参与当前设计决策。

GitHub Issue、Project 或旧认证报告中的“完成”不能覆盖源码、运行时或最新测试证据。

## 当前开发基线

| 项目 | 当前判定 |
|---|---|
| 产品版本 | 最新发布标签为 `v1.6.9`；当前工作树可能包含未发布变更，版本号不等于生产门禁已放行 |
| 当前路线 | 以根目录 `ROADMAP.md` 和 `plans/open-source-commercialization-2026q3-blueprint.md` 为准 |
| 最新部署事实 | 以 `output/product-deployment-business-test-2026-08-14.md` 及重新执行后的证据为准 |
| 生产认证 | `docs/initialization-release-certification.md` 与 `docs/release-v1.5.0-certification-evidence.md` 是 2026-07-30 历史快照，不代表当前工作树 |
| 开发登录 | 本地开发可使用 seed 账号；生产必须使用当前发布支持的 bootstrap 流程，禁止引用历史默认密码 |
| 成熟度 | 页面、Schema 或测试文件存在不等于 GA；运行时 capability/readiness 和本次发布证据共同决定 |

## 文档状态标识

新增或修改文档时，在标题后使用以下一种标识：

- `Status: current`：当前规范，可指导实现。
- `Status: draft`：讨论稿，不得作为已完成证明。
- `Status: historical`：带日期的历史证据，只适用于当时 revision/runtime。
- `Status: superseded`：已被指定文档替代。
- `Status: generated`：由脚本生成，不应手工修改。

测试或发布报告必须写明：日期、Git SHA、镜像 digest、部署模式、数据库、执行命令、未验证层级。缺少这些字段时只能作为参考。

## 维护规则

1. 路线图只维护根目录 `ROADMAP.md`；`docs/roadmap.md` 仅保留跳转说明。
2. 生产部署命令只维护 `docs/DEPLOYMENT_OPTIMIZATION.md`；其他文档链接到它，不复制整段命令。
3. 开发默认账号必须明确标注“仅本地开发”；生产文档不得提供固定默认密码。
4. `plans/` 中的 checklist 完成后不能直接写“已发布”，必须链接对应测试/部署证据。
5. `output/` 中报告不回写成当前规范；发现仍有效的问题，应迁移到 roadmap、issue 或 current 文档。
6. 每次发布检查重复版本号、失效链接、默认凭据、过时 Compose 命令及“全部通过/立即上线”等无 revision 声明。

## 变更 → 文档同步触发矩阵

文档必须随代码在同一批次提交内更新（不允许“下个 PR 再补”）。按下表判断哪些文档被触发：

| 代码变更类型 | 必须同步的文档 | 说明 |
|:---|:---|:---|
| 用户可感知的行为修复 / 新能力 | `CHANGELOG.md` `[Unreleased]` | 对外可见的每项变更都要有一条用户视角的条目 |
| API 契约变更（请求/响应/错误码/破坏性变更） | `CHANGELOG.md` + `docs/api/API_REFERENCE.md` + `UPGRADE.md`（若破坏性） | 破坏性变更必须带迁移说明 |
| 领域能力成熟度变化（GA 候选 / Pilot 升降级） | `README.md` 成熟度表 + `docs/product/` 对应文档 | 声明升降级前必须在源码/测试中核实，不凭印象 |
| 新增 / 更名 make 目标、dev 命令、compose 用法 | `README.md`（及引用它的 README.en/ja） | 文档引用的命令必须真实存在（由 docs-gate C.5 自动校验） |
| 路线图状态变化（落地 / 新收敛项 / 发布里程碑） | `ROADMAP.md`（更新 `Last synced` 日期） | 路线图是迭代方向的唯一事实源 |
| 可靠执行 / 架构边界变更 | `docs/architecture/` 对应规范 + `README.md` 架构节 | |
| 依赖版本 / 环境要求变化 | `README.md` 环境要求 + `UPGRADE.md` | |

## 双向驱动循环

文档与代码是同一个迭代闭环的两个方向，不是单向的“代码写完补文档”：

- **代码 → 文档（同步义务）**：每次合并都按下表同步文档；`make docs-gate` 的 C.5 规则自动校验可自动化的一致性信号（make 目标存在性、ROADMAP/CHANGELOG 新鲜度）。发现文档失真时，修复文档的同一提交里也要问：是文档过时了，还是门禁缺规则？
- **文档 → 代码（驱动义务）**：开始一个迭代时，从 `ROADMAP.md` 的收敛项和 `docs/prd/` 出发拆解任务；修复 / 加固类工作完成后，把新获得的能力基线回写进 ROADMAP「已落地」和 README 成熟度表，使下一轮迭代站在最新事实上。成熟度升降级必须引用本次的测试/验收证据，形成“证据 → 文档 → 下一次迭代”的循环。
- **失同步即缺陷**：文档与代码不一致按缺陷处理——在修复代码的同一批次修文档；无法当场同步时，在 CHANGELOG `[Unreleased]` 记一条待办，不允许静默搁置。

## 当前高风险历史材料

- `docs/initialization-release-certification.md`：已被最新部署测试和当前工作树状态取代。
- `docs/release-v1.5.0-certification-evidence.md`：仅证明 2026-07-30 当次运行。
- `docs/v1-ga-readiness.md`：仅用于 v1.0 历史验收，不代表 v1.6.9 或当前工作树。
- `docs/test-plan/`、`docs/review/`、`output/` 中的“全部通过”“立即上线”：均为历史结论。
- `docs/ci/postmortem-v1.0-GA.md` 中的默认 `ADMIN_PASSWORD`：仅为历史 CI 复盘，不得复制到生产配置。
