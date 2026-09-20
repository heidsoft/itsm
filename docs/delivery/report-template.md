# 发布报告模板

> Status: current

> **Status**: current. 自 v1.5 起强制。
> **配套**：[`docs/delivery/release-guide.md`](./release-guide.md) §"必须满足的强制字段"。

把以下 frontmatter 与正文骨架完整复制到新报告（建议路径 `docs/delivery/<version>/certification-<date>.md`），**任何强制字段为空或写 "TBD" 都视为不合格**。

---

## Frontmatter（必填）

```markdown
---
title: vX.Y.Z 发布认证证据
status: historical          # historical | current | draft
version: X.Y.Z              # 对应 CHANGELOG 版本
date: 2026-MM-DD            # 当次执行日期
git_sha: abc1234            # 7 位或完整 commit SHA
image_digest: sha256:xxxx   # 后端生产镜像 digest
frontend_image_digest: sha256:yyyy   # 前端生产镜像 digest
database: PostgreSQL 17.10 / itsm_prod
deployment_mode: private    # private | saas | saas_msp
commands:                   # 关键执行命令（建议 ≥3 条）
  - cd itsm-backend && go test ./... -count=1
  - docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
  - ./scripts/smoke-test.sh
unverified_scope:           # 未验证项清单（必须显式列出）
  - 跨 PG 大版本 pg_upgrade 实机演练
  - 大规模租户滚动升级压测
signoff:                    # 签字（缺一不可作为发布凭证）
  release: ""
  security: ""
  dba: ""
---
```

### 字段语义

| 字段 | 含义 | 来源命令 |
|---|---|---|
| `git_sha` | 当次执行的工作树 commit | `git rev-parse --short HEAD` |
| `image_digest` | 推送 GHCR 的后端镜像 digest | `docker inspect --format='{{index .RepoDigests 0}}' ghcr.io/heidsoft/itsm-backend:<tag>` |
| `frontend_image_digest` | 同上，前端 | 同上 |
| `database` | 真实生产 PG 版本 + 库名 | `SELECT version(); SELECT current_database();` |
| `deployment_mode` | 部署模式 | `.env.prod` 中 `DEPLOYMENT_MODE` |
| `commands` | 至少包含 build / test / deploy 三类命令 | CI workflow 实际命令 |
| `unverified_scope` | 显式列出未验证的功能、版本组合、压测规模 | 评审 / 阻断项中已知跳过项 |
| `signoff` | 三类签字：release（发布负责人）/ security（安全）/ dba（数据库） | 签字人 GH handle + 日期 |

---

## 正文骨架

```markdown
# vX.Y.Z 发布认证证据

> **Status**: historical。该文件只证明 YYYY-MM-DD 当次 revision、数据库和容器运行。
> 不证明当前工作树。最新判断参见 [文档状态与事实源](../documentation-governance.md) 与 [部署业务测试](../../output/product-deployment-business-test-YYYY-MM-DD.md)。

本文档归档 vX.Y.Z 的发布证据，覆盖 [初始化发布认证](../initialization-release-certification.md) 中各 P 类阻断项的关闭结果。

## 一、自动化回归（绑定 commit / 日期）

| 验证项 | 命令 | 结果 |
|---|---|---|
| 后端全量测试 | `cd itsm-backend && go test ./... -count=1` | ✅ 34 packages passed (commit abc1234, 2026-MM-DD) |
| 前端类型检查 | `npm run type-check` | ✅ 通过 (commit abc1234, 2026-MM-DD) |
| 前端生产构建 | `npm run build` | ✅ 通过 |
| 前端单元测试 | `npm run test:unit` | ✅ N suites / M passed / 0 failed |

## 二、阻断项关闭证据

逐条列出 P0/P1 阻断项，每条必须包含：

- 阻断项 ID（如 P7 / P8 / P9）
- 关闭证据（命令 + 输出 + commit）
- 关联测试 / 脚本

## 三、生产模式部署验证

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

- 服务健康：postgres / redis / minio / itsm-init / itsm-backend-prod / itsm-frontend
- 健康检查：`GET /api/v1/health` → `{"status":"ok"}`
- 认证 fail-closed：错误密码 → `{"code":2001,"message":"invalid credentials"}`

## 四、已知未验证范围

显式列出 [`unverified_scope`] 字段对应内容，例如：

- 跨两个 PG 大版本的 `pg_upgrade` 实机演练（跟踪中，target v1.6）
- 大规模租户滚动升级压测（跟踪中，target v1.7）

## 五、签字

| 角色 | GH handle | 日期 |
|---|---|---|
| Release Manager |  |  |
| Security Owner |  |  |
| DBA Owner |  |  |

只有阻断项全部关闭并附 commit + 日期证据后，三方可签字；未签字的报告**不得**作为后续发布凭证。
```

---

## 校验

`scripts/docs-gate/check-release-claims.sh` 会：

1. 扫描所有新增 / 修改的 `docs/delivery/**/*.md`；
2. 检查 frontmatter 8 个强制字段（`git_sha`、`image_digest`、`database`、`unverified_scope`、`deployment_mode`、`commands`、`date`、`signoff`）；
3. 扫描正文中的禁用表述（"全部通过"、"立即上线"、"零阻断"、"完美"等），命中且同一段落未出现 commit SHA / 日期时标红。

当前阶段（v1.5）为 advisory，仅在 PR 中以 comment 形式提示；v2.0 起升级为 hard gate，缺失阻断构建。

---

## 反例（不合格写法）

```markdown
# v1.6.0 发布报告

## 结论
本版本全部通过，立即上线，零阻断。

✅ 后端测试通过
✅ 前端构建通过
✅ 数据库迁移成功
```

### 问题

1. 无 frontmatter；缺 8 个强制字段；
2. "全部通过 / 立即上线 / 零阻断" 三个无 revision 锚点的断言；
3. "✅ 后端测试通过" 没有 commit SHA 与日期，无法复现。

### 正例（同内容）

```markdown
---
git_sha: 7d3e9a2
image_digest: sha256:abcdef...
database: PostgreSQL 17.10 / itsm_prod
deployment_mode: private
date: 2026-08-15
unverified_scope:
  - 跨 PG 大版本 pg_upgrade 实机演练
commands:
  - cd itsm-backend && go test ./... -count=1
signoff:
  release: ""
  security: ""
  dba: ""
---

# v1.6.0 发布报告

## 自动化回归（绑定 commit）

| 项目 | 命令 | 结果 |
|---|---|---|
| 后端全量测试 | `cd itsm-backend && go test ./... -count=1` | ✅ 34 packages passed (commit 7d3e9a2, 2026-08-15) |
```

---

## 引用

- [`docs/delivery/release-guide.md`](./release-guide.md)
- [`docs/documentation-governance.md`](../documentation-governance.md)
- [`docs/testing/test-invariants.md`](../testing/test-invariants.md) §6
- `scripts/docs-gate/check-release-claims.sh`
