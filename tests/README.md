# Tests 目录说明

> Status: current（口径纠正于 2026-11-14，基于 HEAD `0de5fd68b`）
> 权威层级遵循 [`docs/documentation-governance.md`](../docs/documentation-governance.md)：
> 计划与历史报告不证明功能已实现，以当前源码、迁移、运行时与**当前 CI 结果**为准。

## 1. 先读这一段：本目录不在当前 CI 链路上

仓库当前被 CI 实际执行的测试是：

| 层 | 位置 | Runner | CI |
|:--|:--|:--|:--|
| 后端单元 / 集成 | `itsm-backend/**/*_test.go` | `go test ./...` | `backend-ci.yml`、`make test-backend` |
| 后端 RLS 专项 | `itsm-backend/database/rls/`、`integration/` | `go test -tags integration_rls` | `rls-golden-path.yml` |
| 前端单元 | `itsm-frontend/src/**/*.test.ts(x)` | `jest` | `frontend-ci.yml`、`make test-frontend` |
| 前端 E2E | `itsm-frontend/tests/e2e/` | `playwright test`（`testDir: './tests/e2e'`，相对前端目录） | `frontend-ci.yml`、`make test-e2e` |
| 门禁脚本自测 | `scripts/__tests__/` | `node --test` | 见 `scripts/README.md` §2.6 |

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

## 5. 缓存与产物（不属于交付内容）

`.pytest_cache/`、`__pycache__/`、`*.pyc`、`reports/`、`*.png` 失败截图均为本地产物，
已被 `.gitignore` 忽略。2026-11-14 已清理 16 个 ignored 缓存/产物，并用
`git add -u -- <path>` 移除 6 个历史上误入库的 `*.cpython-310.pyc`（原件归档于
`output/backups/itsm-scripts-tests-cleanup-20261114-0850.tar.gz`）。
**不要用 `git add -A` 把缓存带回索引**。

## 6. 相关文档

- 测试与门禁入口索引：[`scripts/README.md`](../scripts/README.md)
- 角色级产品测试计划：`docs/testing/role-based-product-test-plan.md`
- 文档状态与事实源：[`docs/documentation-governance.md`](../docs/documentation-governance.md)
- 完整盘点报告（本地）：`output/scripts-tests-plans-audit-2026-11-14.md`
