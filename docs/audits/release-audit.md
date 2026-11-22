# Release 与发布说明审计

> 基准 commit：`b3bfd748604041d5082a320ed6d5c447221ac76a`

## 1. 当前生成机制

`.github/workflows/release.yml:179-183` 用 `awk` 从 `CHANGELOG.md` 抽取 `## [X.Y.Z]` 段落；`:185-187` 在段落为空时只打 warning 并设 `use_auto=true`；`:201-202` 把 `body_path` 置空并打开 `generate_release_notes`。

**`release.yml` 全文没有 `go test` / `npm test` / `sha256` / `check-release-claims` / `digest`（grep 零命中）。** 构建 job 只编译（`:61`、`:90`）。
`scripts/docs-gate/check-release-claims.sh:37-55` 定义了禁用语表，但只在 `docs-gate.yml:199` 经 `run-all.sh --strict` 调用，不在发布 workflow 中。
`docs/delivery/release-guide.md:98` 声称该脚本"在发布 workflow 中调用"——与代码不符。

## 2. 发现项

```yaml
id: AUDIT-003
priority: P0
category: release/governance
status: resolved
title: 缺 CHANGELOG 条目时静默降级为 git-log 发布说明，且无测试证据与制品校验
file: .github/workflows/release.yml:185-202
impact: 对外发布说明绕过全部文案与证据规则；公开的 zip 无 checksum（本地 scripts/release.sh:143-148 反而有）
fix: 缺条目改为硬失败或显式 workflow_dispatch opt-in；发布前校验禁用语与必需字段；为两个 zip 生成 .sha256 并写入 GITHUB_STEP_SUMMARY
acceptance: 无对应 CHANGELOG 段的 tag 使 workflow 失败；含禁用语无锚点的段落失败；产物包含两个 .sha256
```

```yaml
id: AUDIT-030
priority: P1
status: confirmed
title: 最近 22 个 tag 没有 CHANGELOG 版本段落，自动生成分支是实际路径
evidence: CHANGELOG.md 最高发布段为 1.6.37（:469），而 git describe 为 v1.6.59；awk 抽取 v1.6.59 为空
contrast: scripts/prepare-release.sh:97-110 在同样情况下硬失败——两个入口规则不一致
fix: CI 增加 tag↔CHANGELOG 一致性检查；补齐或声明 1.6.38-1.6.59 为非公开
```

```yaml
id: AUDIT-031
priority: P1
status: confirmed
title: workflow_dispatch 重发混源：二进制来自分支、镜像来自 tag
file: release.yml:43-44,74-75,127-128,236-239,209,213
evidence: 只有 publish-containers checkout 使用 inputs.tag || github.ref；build-backend/frontend 无 ref；summary 用 github.ref_name；make_latest: true
reproduction: gh workflow run release.yml -f tag=v1.6.12 后比对二进制版本与容器 tag
fix: 增加 resolve job 统一 VERSION/CHECKOUT_REF，三个 job 共用一个 ref；summary 用解析后的版本；dispatch 重发不设 latest
```

```yaml
id: AUDIT-032
priority: P1
status: confirmed
title: 至少 7 处版本真源互相矛盾
evidence: git tag v1.6.59 / CHANGELOG 1.6.37 / Unreleased 中 v1.6.61-63 / .env.prod.example:13 VERSION=1.6.9 / documentation-governance.md:25 基准 v1.6.9 / RELEASE_PROCESS.md:21 v1.6.57 未 GA / router.go:138 与 package.json:3 与 main.go:12 均 1.6.19 / ga_readiness.go:78 为 1.6.8
note: scripts/check-deploy-version.sh 存在但未接入任何 workflow/Makefile，且读取 gitignore 的 .env.prod
fix: 改为 check-version-consistency.sh（只读取受版本控制的输入），加 Jest 测试并接入 docs-gate；允许的滞后项用显式小清单声明
```

```yaml
id: AUDIT-033
priority: P2
status: confirmed
title: CHANGELOG 条目以工程报告口吻书写，缺少用户价值导语
evidence: CHANGELOG.md:16（正则解析器细节）、:18（Gate 2/Gate 2.5、evaluateToolCall）、:20（SHA-256 指纹算法）；正面样例 :30（"用户反馈…中文英文混排"）、:34（"切深色后…文字看不见"）
fix: 要求首句为可观察症状+受益对象，技术细节移入链接文档；长度预算+禁用首句内部标识符的 lint
```

```yaml
id: AUDIT-034
priority: P2
status: confirmed
title: 用户要求的七段式发布说明模板在仓库中没有定义
evidence: 全仓无"本次更新/新增功能/问题修复/兼容性说明/升级建议/已知限制/验证情况"标题；仓库内唯一规则是 documentation-governance.md:59 的用户视角条目
fix: 在 docs/delivery/release-guide.md 定义模板并让 G2 守卫可配置段落集合
```

## 3. PR-04 最小守卫集合

1. `scripts/check-release-notes.sh --tag`：要求 `## [X.Y.Z]` + ISO 日期 + 非空段落；发布 workflow 缺条目改为硬失败（G1）。
2. 用户价值结构检查，按条目棘轮推进（G2）。
3. 文案质量：首句内部标识符/长度预算，先 advisory 后棘轮（G3）。
4. 复用 `check-release-claims.sh` 的禁用语表，在发布 job 直接检查 notes body（G4，不削弱 docs-gate 原有硬检查）。
5. `check-version-consistency.sh` + Jest 测试（G5）。
6. 扩展 `ci-workflow-contract.test.js` 断言 release.yml 的显式 checkout ref 与 notes fallback 策略（G6）。
7. 同步修正 `release-guide.md:98` 与 `report-template.md:113-119` 与实际脚本不一致的描述（G7）。

每个守卫都需要正向 fixture 与能变红的反向 fixture；不删除、不降级任何既有检查。


## 本批已验证修复（基准 bde1814d）

采纳并行会话在发布相关文件上的未提交工作，并补齐文案门禁：
- `scripts/release-notes.sh`：按 tag 精确匹配 `## [X.Y.Z]`，优先取「发布说明」小节，缺失或空内容即 exit 1；旧版本回退读取完整条目。
- `.github/workflows/release.yml`：所有 checkout 统一到解析出的 tag；发布正文固定取 `release-notes.md`，不再启用 `generate_release_notes`；summary 使用解析版本而非 `github.ref_name`；新增 `check-release-notes-quality.sh` 步骤拦截无证据宣传语。
- 回归 `release-notes.test.js` 3 例 + `release-notes-quality.test.js` 3 例（含 8 类禁用表述反向验证）全绿。
- 遗留：RQ-02（1.6.38–1.6.59 无 CHANGELOG 段）与 RQ-07（多 surface 版本漂移）未在本批处理。

## 版本一致性守卫（RQ-07 阶段进展）

`scripts/check-version-consistency.js` 对五个版本表面（示例环境文件、后端兜底版本、GA readiness、前端 package.json、治理文档基线）做棘轮检查。当前五处差异逐项登记在 `scripts/version-drift-baseline.json` 并以 WARN 呈现；任一表面出现未登记的新版本立即 FAIL，`--strict` 可要求完全对齐。四组测试（含 5 个表面逐一反向注入）通过。本批不改动任何版本字面量——选定正式版本属于发布决策。
