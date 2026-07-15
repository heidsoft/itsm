# Label Schema

> **目的**：统一 Issue / PR 标签语义，便于维护者分类、过滤和统计。
>
> **生效范围**：itsm-backend、itsm-frontend、itsm-cli、itsm-skill、itsm-agent、itsm-ai-service、itsm-rag、docs。
>
> **最后更新**：2026-06-28

---

## 1. 标签分类总览

| 类别 | 命名空间 | 数量 | 来源 | 用途 |
|:---|:---|---:|:---|:---|
| 业务域（Area） | `area/*` | 13 | 维护者手动 | 标识修改的文件/模块 |
| 改动类型（Kind） | `kind/*` | 8 | 维护者手动 | 标识 PR 性质 |
| 优先级（Priority） | `priority/*` | 4 | 维护者手动 | 标识优先级 |
| 状态（Status） | `status/*` | 7 | 维护者手动 | 标识生命周期状态 |
| 大小（Size） | `size/*` | 4 | 维护者手动 | 标识 PR 代码规模 |
| 兼容性 | `breaking-change` | 1 | 维护者手动 | 标识破坏性变更 |
| 依赖 | `dependencies` | 1 | dependabot 自动 | Dependabot PR 专用 |
| 受托人 | `needs-triage`、`needs-review`、`needs-design`、`needs-docs` | 4 | 维护者手动 | 标识待处理维度 |
| 安全 | `security` | 1 | 维护者手动 | 安全相关 |
| 贡献者 | `first-time-contributor` | 1 | 维护者手动 | 新人识别 |
| Pin | `pinned` | 1 | 维护者手动 | 标识需长期保留的问题 |

**总计：43 个标签**

---

## 2. area/* — 业务域

由维护者根据改动文件路径应用。

| 标签 | 触发条件 | 责任人 |
|:---|:---|:---|
| `area/backend` | 修改 `itsm-backend/controller/**`、`service/**`、`middleware/**`、`dto/**`、`cache/**`、`router/**` | Backend Lead |
| `area/backend-entity` | 修改 `itsm-backend/ent/schema/**`、`ent/**` | Backend Lead |
| `area/backend-test` | 修改 `itsm-backend/**/*_test.go` | Backend Lead |
| `area/frontend` | 修改 `itsm-frontend/src/**` | Frontend Lead |
| `area/frontend-test` | 修改 `itsm-frontend/tests/**`、`**/*.test.{ts,tsx}` | Frontend Lead |
| `area/ai` | 修改 `itsm-ai-service/**`、`itsm-rag/**` | AI Lead |
| `area/cli` | 修改 `itsm-cli/**` | Tooling Lead |
| `area/agent` | 修改 `itsm-agent/**` | Tooling Lead |
| `area/skill` | 修改 `itsm-skill/**` | Tooling Lead |
| `area/infra` | 修改 `docker-compose*.yml`、`Dockerfile`、`nginx/**`、`monitoring/**` | DevOps Lead |
| `area/ci` | 修改 `.github/workflows/**`、`.github/labeler.yml`、`.github/dependabot.yml` | DevOps Lead |
| `area/docs` | 修改 `docs/**`、`README*.md`、`ROADMAP.md`、`CONTRIBUTING.md` | Docs Lead |
| `area/scripts` | 修改 `scripts/**`、`Makefile` | DevOps Lead |

> 一个 PR 可能同时打多个 `area/*`（例如同时改 backend 和 frontend）。

---

## 3. kind/* — 改动类型

维护者可根据 PR title 前缀和改动内容识别。

| 标签 | 标题前缀 / 模式 | 示例 |
|:---|:---|:---|
| `kind/feature` | `feat:`、`feat(`, `feature:` | `feat(ticket): add SLA auto-escalation` |
| `kind/bugfix` | `fix:`、`fix(`, `bugfix:` | `fix(incident): correct P1 calculation` |
| `kind/refactor` | `refactor:`、`refactor(` | `refactor(auth): split JWT verifier` |
| `kind/tests` | `test:`、`test(` | `test(ticket): add BPMN state machine coverage` |
| `kind/docs` | `docs:`、`docs(` | `docs(api): add change module spec` |
| `kind/ci` | `ci:`、`ci(`、`build:` | `ci: add coverage-diff gate` |
| `kind/build` | `build:`、`build(` | `build(deps): bump gin to 1.12.0` |
| `kind/dependencies` | Dependabot 自动添加 | 任何 Dependabot PR |

---

## 4. priority/* — 优先级（手动）

由维护者在 triage 时手动添加。

| 标签 | SLA | 描述 |
|:---|:---|:---|
| `priority/p0` | 4h | 生产环境故障、安全漏洞、数据丢失 |
| `priority/p1` | 1d | 重要功能失效、无变通方案 |
| `priority/p2` | 1w | 一般 bug、性能问题 |
| `priority/p3` | 2w+ | 小优化、nice-to-have |

---

## 5. status/* — 状态（混合）

| 标签 | 来源 | 描述 |
|:---|:---|:---|
| `status/needs-triage` | 维护者 | 等待分类与优先级评估 |
| `status/triaged` | 维护者 | 已分类等待开发 |
| `status/in-progress` | 维护者 / PR 关联 | 正在开发中 |
| `status/blocked` | 维护者 | 等待外部依赖 |
| `status/duplicate` | 维护者 | 与已有 Issue 重复 |
| `status/wontfix` | 维护者 | 不予修复（请说明原因） |
| `status/ready-to-merge` | 维护者 | PR 满足所有检查，等待合并 |

---

## 6. size/* — PR 大小（自动）

由 labeler workflow 通过 changed files LOC 自动计算。

| 标签 | LOC 范围 | 期望 review 时间 |
|:---|:---|:---|
| `size/XS` | 0–9 | < 15min |
| `size/S` | 10–49 | < 30min |
| `size/M` | 50–199 | < 1h |
| `size/L` | 200–499 | 拆分或延长 review |
| `size/XL` | 500+ | **必须**拆分 |

---

## 7. 破坏性变更（自动）

| 标签 | 触发条件 | 备注 |
|:---|:---|:---|
| `breaking-change` | PR title 包含 `BREAKING CHANGE:` 或 `!:`（conventional commit） | 自动触发 major version bump |

---

## 8. 初始化脚本

在仓库克隆后运行一次以下脚本，创建全部 43 个标签：

```bash
# 幂等：已存在的标签会被跳过
./scripts/init-labels.sh
```

或在 GitHub 网页手动操作：**Issues → Labels → New label**，参考下表的颜色与描述。

### 完整标签定义（颜色按 hex）

#### area/* — 蓝色（#0E8A16 系）

```
area/backend         #1D76DB  Backend Go code
area/backend-entity  #1D76DB  Backend Ent ORM schema
area/backend-test    #1D76DB  Backend Go tests
area/frontend        #5319E7  Frontend Next.js code
area/frontend-test   #5319E7  Frontend tests
area/ai              #FBCA04  AI / RAG / LLM features
area/cli             #0E8A16  itsm-cli tool
area/agent           #0E8A16  itsm-agent worker
area/skill           #0E8A16  itsm-skill plugins
area/infra           #BFD4F2  Docker / Nginx / monitoring
area/ci              #BFD4F2  GitHub Actions / workflows
area/docs            #0075CA  Documentation
area/scripts         #BFD4F2  Helper scripts / Makefile
```

#### kind/* — 紫色（#5319E7 系）

```
kind/feature         #5319E7  New feature
kind/bugfix          #D93F0B  Bug fix
kind/refactor        #5319E7  Code refactor (no behavior change)
kind/tests           #5319E7  Test-only changes
kind/docs            #0075CA  Documentation-only
kind/ci              #BFD4F2  CI / pipeline changes
kind/build           #BFD4F2  Build system / dependency changes
kind/dependencies    #0366D6  Dependabot (auto)
```

#### priority/* — 红色/橙色/黄色

```
priority/p0          #B60205  Critical (4h SLA)
priority/p1          #D93F0B  High (1d SLA)
priority/p2          #FBCA04  Medium (1w SLA)
priority/p3          #0E8A16  Low (2w+ SLA)
```

#### status/* — 灰色 / 彩色混合

```
status/needs-triage      #FFFFFF  Awaiting triage
status/triaged           #0E8A16  Triaged, ready for dev
status/in-progress       #FBCA04  In development
status/blocked           #B60205  Blocked by external dep
status/duplicate         #CCCCCC  Duplicate of existing issue
status/wontfix           #FFFFFF  Will not be addressed
status/ready-to-merge    #0E8A16  Approved, awaiting merge
```

#### size/* — 浅蓝 / 浅紫

```
size/XS  #C5DEF5  Extra small (< 10 LOC)
size/S   #C5DEF5  Small (10–49 LOC)
size/M   #BFD4F2  Medium (50–199 LOC)
size/L   #FBCA04  Large (200–499 LOC, consider splitting)
size/XL  #B60205  Extra large (500+ LOC, MUST split)
```

#### 其它

```
breaking-change     #B60205  Breaking change (semver major bump)
dependencies        #0366D6  Pull requests that update a dependency
needs-triage        #FFFFFF  Needs maintainer triage
needs-review        #1D76DB  Awaiting reviewer
needs-design        #5319E7  Awaiting design proposal
needs-docs          #0075CA  Needs documentation update
security            #B60205  Security issue (private disclosure)
first-time-contributor #7057FF  First contribution from this user
pinned              #FFFFFF  Prevents stale bot from closing
epic                #5319E7  Tracks a multi-PR feature
```

---

## 9. 使用规则

### Issue

- 创建后 1 个工作日内必须有 `status/needs-triage` 或 `status/triaged`
- `status/needs-triage` 标签由维护者在首次 review 时手动打
- 一旦打了 `status/wontfix`、`status/duplicate`，关闭 Issue 前必须在评论中说明理由

### PR

- `area/*`、`kind/*`、`size/*` 由维护者在 review 时按需添加
- `breaking-change` 一旦出现，必须在 PR 描述中说明：
  - 受影响的 API / 配置
  - 升级路径（migration script / 文档）
  - 影响范围（哪些版本会中招）
- 包含 `kind/dependencies` 的 PR 必须通过正常 CI 并由维护者审核，不自动合并

---

## 10. 维护

- 任何新增 label 必须同步更新本文档和 `scripts/init-labels.sh`
- 删除未使用 label 前需在 Discussions 公示 1 周
- 季度 review：维护者集中讨论标签语义是否需要调整

---

**维护者**：见 [CODEOWNERS](../.github/CODEOWNERS)
**相关文档**：[CONTRIBUTING.md](../CONTRIBUTING.md)、[ROADMAP.md](../ROADMAP.md)
