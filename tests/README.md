# Tests 目录说明

> Status: current（口径纠正于 2026-11-14，基于 HEAD `0de5fd68b`）
> 权威层级遵循 [`docs/documentation-governance.md`](../docs/documentation-governance.md)：
> 计划与历史报告不证明功能已实现，以当前源码、迁移、运行时与**当前 CI 结果**为准。

## 1. 先读这一段：本目录不在当前 CI 链路上

仓库当前被 CI 实际执行的测试是：

| 层 | 位置 | Runner | 实际 CI / 本地入口 | 规模 |
|:--|:--|:--|:--|:--|
| 后端单元 / 集成 | `itsm-backend/**/*_test.go` | `go test ./...` | `backend-ci.yml`、`make test-backend` | 802 文件 |
| 后端 RLS 专项 | `itsm-backend/database/rls/`、`integration/` | `go test -tags integration_rls` | `rls-golden-path.yml`（独立 workflow） | 专项 |
| 前端单元 | `itsm-frontend/src/**/*.test.ts(x)` | `jest`（默认收 `src/lib/**` 覆盖率并设 80% 阈值） | `frontend-ci.yml`、`make test-frontend` | 251 文件 |
| 前端 E2E | `itsm-frontend/tests/e2e/` | `playwright test`（`testDir: './tests/e2e'`） | **仅 `ga-gate.yml` 跑 `ticket-type-full-chain.spec.ts` 1 个 spec**；其余靠 `make test-e2e` 本地执行 | 67 spec |
| 门禁脚本自测 | `scripts/__tests__/` | `node --test scripts/__tests__/*.test.js` | `docs-gate.yml` 的 `Script Guard Tests` job（整目录）、`make test-scripts` | 12 文件 / 90 用例 |
| 整栈集成 | `scripts/smoke-test.sh` | curl + E2E | `ga-gate.yml`（起 compose 全栈） | 1 job |

**口径纠正（2026-11-14 复核）**：`frontend-ci.yml` 只跑 lint / type-check / Jest / build，
**没有任何 playwright 步骤**；`ga-gate.yml` 的 playwright 步骤也只跑 1 个 TicketType 全链 spec。
因此 67 个 E2E spec 中的绝大多数**不在任何 CI job 中执行**，不得据此宣称「E2E 全绿」。
`make test` = 后端 + 前端单元 + 门禁脚本自测；**不含** E2E、RLS 专项与整栈集成。
`make test-e2e` 需要已启动的整栈，属于本地/人工验收，不是发布门禁证据。

**本地复现远端 CI（2026-11-14 新增）**：`make ci-parity` 用 CI 同版本 gofumpt 与同一条
`swag` 命令检查格式欠账和 OpenAPI 漂移，`make ci-parity-strict` 有差异即失败。
它检查**当前工作树**（含未提交 WIP），而 CI 检查**提交态**；有 WIP 改过注解时，
发布前须在纯 HEAD 隔离树再复核一次。远端 Actions 的四类历史红灯
（gofumpt、Swagger 漂移、两个过期 Jest 断言、star-history 直推保护分支）
由 `scripts/__tests__/github-ci-parity.test.js` 锁住，见 `output/github-ci-review-2026-11-14.md`。

**根 `tests/` 目录：`.github/workflows/` 中对它零引用，根目录没有 `package.json`，
也没有根级 `playwright.config.*`。** 因此本目录下的 pytest 用例、`.spec.ts` 与 `.cjs`
浏览器脚本目前**不会被任何 CI job 或 npm script 执行**。

本目录定位为「离线/本地验证框架与历史用例」，可手动运行，但其结果：
- 不构成发布门禁证据；
- 不得用于宣称「测试已通过」或「功能已实现」；
- 新增测试用例请优先加到上面四层活跃链路。

## 2. 目录结构

```
tests/
├── api/          # API 客户端封装 + test_api_cases.py / test_complete_api.py
├── ui/           # UI 客户端封装 + test_ui_cases.py / test_frontend_ui.py / test_frontend_complete.py
├── database/     # 数据库客户端（__init__.py）
├── fixtures/     # 测试数据生成器
├── e2e/          # test_navigation.py / test_tickets_full.py + requirements.txt
├── reports/      # pytest-html / junit 输出（本地产物，已被 gitignore）
├── config.ini        # 本地环境配置
├── pytest.ini        # pytest 配置（注意 addopts 依赖 pytest-html）
├── requirements.txt  # Python 依赖
├── run_tests.py      # 本地运行入口
├── *.cjs / *.mjs     # 一次性浏览器验证脚本（无 runner，见 §4）
└── *.spec.ts         # Playwright 风格用例（无根级 config，见 §4）
```

## 3. 本地手动运行

```bash
cd tests
python3 -m pip install -r requirements.txt
playwright install chromium          # UI 用例需要
python3 run_tests.py --type all      # 或 --type api / --type ui / --markers smoke
```

`pytest.ini` 的 `addopts` 含 `--html=reports/report.html --self-contained-html`，
未安装 `pytest-html` 时会直接报错；仅跑纯单元测试可用 `pytest -o addopts=''`。

### 配置与凭据

通过环境变量提供连接信息，**不要在 `config.ini` 或代码里写固定口令**：

```python
import os
base_url = os.environ.get('ITSM_BASE_URL', 'http://127.0.0.1:8090')
username = os.environ['ITSM_TEST_USER']
password = os.environ['ITSM_TEST_PASSWORD']
```

生产环境禁止使用固定默认口令；登录凭据取自部署时的 bootstrap 流程或本地密钥管理。
（治理依据：`docs/documentation-governance.md` 维护规则 3 与 6。）

## 4. 无 runner 的历史用例（待决策，不要误当作已覆盖）

- 3 个 `.spec.ts`：`e2e-ticket-flow.spec.ts`、`prod-button-test.spec.ts`、`ui/authz-roles-groups.spec.ts`
  —— 根级无 playwright config，前端 config 的 `testDir` 也不覆盖本目录。
- 10 个 `.cjs` / 1 个 `.mjs`：含 `browser-e2e-verify.cjs` 与 `-v2.cjs`、
  `screenshot-workflows.cjs` 与 `-v2.cjs`、`verify-designer-fix.cjs` 与 `-v2.cjs` 等
  v1/v2 迭代残留，以及 `inspect-canvas.cjs`、`screenshot-designer.cjs`、
  `playwright-four-domain-journey.cjs`、`browser-test.mjs`。

这些文件是某次人工验证的脚本快照。三条可选去向（需项目侧拍板后统一执行）：

1. **接线**：迁到 `itsm-frontend/tests/e2e/` 并纳入 `playwright.config.ts`，补 CI job；
2. **标注**：保留为 historical 示例，在文件头写明「不参与 CI」；
3. **归档**：移入本地归档并删除，结论迁移到 `docs/testing/` 规范文档。

在决策前，请勿把它们的结果写进 CHANGELOG 或发布报告。

## 4.1 测试用例编写规范（新增 2026-11-14）

以下规则适用于**活跃链路**的新增/修改用例，评审时按此清单核对。

| # | 规则 | 反例 / 依据 |
|:--|:--|:--|
| 1 | **写进有 runner 的层** | 根 `tests/` 的 pytest / `.cjs` / `.spec.ts` 零 CI 引用，新增用例请放到 `itsm-backend/**`、`itsm-frontend/src/**/__tests__/`、`itsm-frontend/tests/e2e/` 或 `scripts/__tests__/` |
| 2 | **URL / 契约断言要精确** | 历史上 `expect.stringContaining('/api/v1/tickets')` 放过了后端根本不存在的 `/approval-workflows` 路由（见 `scripts/test-coverage-guard.js` 文件头）。路径断言用完整字符串或 `toHaveBeenCalledWith(url, ...)`；当前仓库仍有 67 处 `expect.stringContaining` 待逐批收紧 |
| 3 | **不要用存在性断言当唯一断言** | `toBeDefined()` / `toBeTruthy()` / `expect.any()` 单独出现时，实现返回任何垃圾都绿；必须再断言值、形状或副作用。当前前端单测有 198 处 `toBeDefined`、14 处 `toBeTruthy`，需结合上下文判断 |
| 4 | **skip 必须带原因与开关** | E2E 有 14 文件 / 48 处 skip，其中 11 处计划占位、10 处环境开关、**27 处无原因裸 `test.skip()`**。裸 skip 等于删用例；改用 `test.skip(cond, 'reason')`。逐文件分类、触发条件与治理顺序见 [`docs/testing/e2e-skip-inventory.md`](../docs/testing/e2e-skip-inventory.md)（该台账由 `e2e-skip-inventory.test.js` 自动对齐，改 spec 不改台账会变红） |
| 5 | **凭据只从环境变量取** | `ITSM_TEST_USER` / `ITSM_TEST_PASSWORD`，禁止在用例或夹具里写固定口令；`ga-gate.yml` 的 compose 开发夹具仅限该一次性栈，不得复用到生产或写入测试文档 |
| 6 | **守卫/契约类测试必须反向验证** | 注入旧实现（或改回旧值）后测试**必须变红**，否则断言是同义反复。`scripts/__tests__/ci-workflow-contract.test.js` 用同一断言函数验证「去掉退出码捕获必红」 |
| 7 | **一测一因，失败信息可定位** | Go 用例优先 `require`/`assert` + 表驱动 `t.Run`；避免 `time.Sleep` 做同步（当前 6 文件 14 处），改用 channel / 条件等待 |
| 8 | **认清守卫强弱** | `test-coverage-guard.js` 是「改了源码 ⇒ 同包存在测试文件」的**弱守卫**（`backendCandidates()` 接受同包任意 `*_test.go`），不是符号级覆盖；不要把它当覆盖充分的证明。2026-11-14 起提供 `--strict-diff` **观察模式**（只输出「diff 内没改任何候选测试」的 warning 与 JSON 字段，绝不改变退出码），由 `test-coverage-guard.yml` 写入 job summary，供后续决定是否加严 |

### 分层归属速查

| 改动对象 | 应写测试 | 必须接的 runner |
|:--|:--|:--|
| `itsm-backend/internal/services/**`、`handlers/**` | 同包 `*_test.go` | `backend-ci.yml` 的 `go test` |
| `itsm-frontend/src/lib/**` | `src/lib/<area>/__tests__/<name>.test.ts` | `frontend-ci.yml` 的 Jest |
| 跨页面业务流程 | `itsm-frontend/tests/e2e/*.spec.ts` | 若要成为门禁证据，需显式加进 `ga-gate.yml`（当前不自动执行） |
| CI workflow / 门禁脚本 | `scripts/__tests__/*.test.js` | `docs-gate.yml` 的 `Script Guard Tests`；**修改被测 workflow 时还要同步其 `paths:` 触发项** |

## 5. 缓存与产物（不属于交付内容）

`.pytest_cache/`、`__pycache__/`、`*.pyc`、`reports/`、`*.png` 失败截图均为本地产物，
已被 `.gitignore` 忽略。2026-11-14 已清理 16 个 ignored 缓存/产物，并用
`git add -u -- <path>` 移除 6 个历史上误入库的 `*.cpython-310.pyc`（原件归档于
`output/backups/itsm-scripts-tests-cleanup-20261114-0850.tar.gz`）。
同批又用 `git rm --cached` 从索引移除两份运行产物（磁盘副本保留）：
`test-coverage-guard.json`（守卫失败诊断）与
`itsm-frontend/test-results/.last-run.json`（Playwright 上次失败用例状态）。
两者早已被 `.gitignore` 覆盖，`repo-hygiene.test.js` 会阻止再次入库。
**不要用 `git add -A` 把缓存带回索引**。

## 6. 相关文档

- 测试与门禁入口索引：[`scripts/README.md`](../scripts/README.md)
- E2E 跳过用例台账：[`docs/testing/e2e-skip-inventory.md`](../docs/testing/e2e-skip-inventory.md)
- 角色级产品测试计划：`docs/testing/role-based-product-test-plan.md`
- 文档状态与事实源：[`docs/documentation-governance.md`](../docs/documentation-governance.md)
- 完整盘点报告（本地）：`output/scripts-tests-plans-audit-2026-11-14.md`
