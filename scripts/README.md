# Scripts 目录索引

> Status: current · 最后梳理：2026-11-14（基于 HEAD `0de5fd68b`）
> 本文只做入口索引与接线状态说明；各脚本的参数以文件头 Usage 为准。
> 完整盘点与判定依据见本地报告 `output/scripts-tests-plans-audit-2026-11-14.md`（`output/` 不入库）。

## 1. 推荐入口（优先用这些，不要直接拼子脚本）

| 目的 | 命令 | 说明 |
|:--|:--|:--|
| 文档 / 产品口径门禁 | `make docs-gate` 或 `bash scripts/docs-gate/run-all.sh --strict` | C.1–C.8 全套；CI `docs-gate.yml:125,128` 调用。长任务请后台跑 |
| 静态契约守卫 | `bash scripts/static-gates/run-all.sh` | 8 个守卫统一入口；CI `backend-ci.yml:203,207,211` 也单独调其中 3 个 |
| 前端 P0 守卫（CI 必过） | `bash scripts/check-frontend-p0-guards.sh` | 见 `README-p0-guards.md` |
| 前端 P1 守卫（存量跟踪） | `bash scripts/check-frontend-p1-guards.sh` | 同上 |
| 工程契约 | `make check-contracts` → `node scripts/check-engineering-contracts.js` | `api-contract-check.yml` |
| API 路径一致性 | `node scripts/check-api-paths.js --json` | 见 `README-check-api-paths.md` |
| 空卷首装门禁 | `make fresh-install-gate` → `bash scripts/fresh-install-gate.sh` | 产物写 `reports/fresh-install-gate/`，CI 失败时上传 artifact |
| 产品功能黑盒 | `python3 scripts/acceptance/product-function-smoke.py --help` | v1.6.58 实测 81 PASS / 3 WARN / 0 FAIL |
| 部署后验收 | `bash scripts/prod-post-deploy-verify.sh --help` | 24 项验收 |
| 校验全部 shell 语法 | `make verify-scripts` | 遍历 `scripts/*.sh` 与 `scripts/docs-gate/*.sh` 做 `bash -n` |
| 门禁脚本自测 | `node --test scripts/__tests__/*.test.js` | CI `docs-gate.yml` 的 `Script Guard Tests` job 整目录执行 |

## 2. 分组

### 2.1 文档门禁 `docs-gate/`（必留）

`run-all.sh` 为唯一编排入口，下辖 `check-broken-links.sh`、`check-doc-sync.sh`、
`check-duplicate-roadmap.sh`、`check-hardcoded-passwords.sh`、`check-layering-convergence.sh`、
`check-product-drift.sh`、`check-release-claims.sh`、`check-scope-creep.sh`、`find-frontend-orphans.py`。

同目录的 `*.txt` 是**棘轮基线/豁免账本**，不是脚本：
`product-surface-baseline.txt`、`product-scope-baseline.txt`、`service-prod-baseline.txt`、
`handler-service-imports-baseline.txt`、`product-drift-waivers.txt`、`retired-services.txt`、
`scope-shell-whitelist.txt`。
改 `internal/bootstrap/app.go` 或新增前端页时，须**同 commit** 按 C.6.4 规程调基线并写 CHANGELOG。

### 2.2 静态契约守卫 `static-gates/`（必留）

`run-all.sh` + `check-bare-json.sh`、`check-context-bg.sh`、`check-http-status-mapping.sh`、
`check-list-envelope.sh`、`check-pagination-shape.sh`、`check-raw-fetch.sh`、
`check-shell-var-boundary.sh`、`check-tx-constraint-swallow.sh`。

注意 `static-gates/check-list-envelope.sh`（查后端 handler）与
`check-list-envelope-contract.js`（查前端 TSX）**检查对象不同，不是重复实现**，两者都保留。

### 2.3 CI 契约工具（必留）

| 脚本 | CI 调用处 |
|:--|:--|
| `check-engineering-contracts.js` | `api-contract-check.yml:80` |
| `check-api-paths.js` | `api-contract-check.yml:105` |
| `generate-acl-manifest.js` | `acl-gate.yml` |
| `test-coverage-guard.js` | `test-coverage-guard.yml:117,125` |
| `test-trend.js` | `backend-ci.yml:321` |
| `generate-star-history.mjs` | `star-history.yml:29` |
| `check-go-toolchain-consistency.sh` / `check-workflow-action-pinning.sh` | `backend-ci.yml:143,149` |

### 2.4 产品验收 / 发布 / 部署（必留）

`acceptance/product-function-smoke.py`、`acceptance/workflow-cross-domain-acceptance.sh`、
`prod-post-deploy-verify.sh`、`fresh-install-gate.sh`、`e2e-isolated-stack.sh`、
`rebuild_and_verify.sh`、`release-evidence.sh`、`release.sh`、`prepare-release.sh`、
`preflight.sh`、`smoke-test.sh`、`coverage-report.sh`、`coverage-summarize.js`、
`check-deploy-version.sh`、`build-images.sh`、`build-frontend-image.sh`、
`deploy-dev.sh`、`deploy-prod.sh`、`start-dev.sh`、`stop-dev.sh`。

备份链路 `backup.sh` / `restore.sh` / `prod-backup.sh` / `prod-restore-drill.sh` /
`com.itsm.prod-backup.plist.template` 由 `pg-disaster-recovery.yml` 与 Makefile 调用。

### 2.5 手动运维 / 修复入口（保留，通常无文档引用）

这些脚本不写进 CI 是正常的，**不要因为「零引用」就删**：

| 脚本 | 用途 |
|:--|:--|
| `build-optimized.sh` | 带 `--no-cache` / `--parallel` / `--push` 的手动构建 |
| `verify-dev-cold-start.sh` | dev 冷启动端到端验证（compose + init + SQL 断言） |
| `verify-codeowners.py` | CODEOWNERS 格式/覆盖率审计，支持 `--check` / `--audit` |
| `verify-acl-coverage.go` | ACL 覆盖校验（见 `docs/architecture/adr-004-...md`） |
| `fix_swag.py` | 迭代修复 swagger 未定义类型引用 |
| `prod-dead-letter-check.sh` | 死信队列检查 |
| `init-labels.sh` | GitHub label 初始化（见 `docs/label-schema.md`） |
| `itsm_api_runner.py` | 通用 API runner，支持 `--interactive` |
| `itsm-test-utils.sh` | 被 `docs-gate/check-hardcoded-passwords.sh` 复用 |
| `lib/common.sh`、`fixtures/fresh-gate-negative.override.yml` | 公共库与门禁夹具 |

数据夹具：`test-data-init.sql`、`seed_hospital_kb_cmdb_sla.sql`。

### 2.6 门禁脚本自测 `__tests__/`（必留，已全量接线）

5 个 `node --test` 文件，本地 `node --test scripts/__tests__/*.test.js` 共 37 用例。
2026-11-14 起 CI `docs-gate.yml` 的 `Script Guard Tests` job **整目录执行**这 5 个文件；
此前只跑 `docs-gate.test.js`，另外 3 个无 runner。
`api-contract-check.yml:77` 仍单独跑 `check-engineering-contracts.test.js` 作为该 workflow 专属门槛（重复执行无害）。

`docs-gate.yml` 的 `pull_request` / `push` 触发路径同时补齐了 `scripts/__tests__/**` 与
被测对象（`build-images.sh`、`deploy-prod.sh`、`test-trend.js`、`generate-acl-manifest.js`、
`docker-compose.prod.yml`），否则改了被测脚本不触发测试，接线仍是假绿。

`build-start-scripts.test.js` 的两个断言同批改为「锁语义不锁字面」：
不再硬编码 `up -d postgres redis minio` 与 `image: itsm-backend:${VERSION:-latest}`，
改为断言每个 compose 启动阶段都有 fail-fast `return 1`、以及所有 `itsm-backend` 服务
共用同一个强制 `VERSION:?` 镜像表达式。反向验证（把一处 `return 1` 改成 `return 0`）确实变红。

### 2.7 历史 / 待退役（勿新增依赖）

| 组 | 说明 |
|:--|:--|
| ~~`local-codex-*.sh`（4 个）~~ | **2026-11-14 已退役**：机器绑定的旧自动化，硬编码本机绝对路径、`/usr/local/bin/codex`、CC Switch app 与未安装的 LaunchAgent；配套 `orchestrate-codex-worker.sh` 已在 `074c5cf52` 归档移除。原件归档于 `output/backups/itsm-scripts-tests-cleanup-20261114-0850.tar.gz`。现由 WorkBuddy / QCode 会话流程取代 |
| `stage4_runner.py`、`stage5_runner.py`、`regression_p1p2.py`、`test_permissions_and_menus.sh` | 10-01 批次配套 runner，唯一引用方是历史计划 `plans/product-feature-completeness-plan-2026-10-01.md` |

## 3. 缓存与产物

`acceptance/__pycache__/` 是 Python 缓存（已被 `.gitignore` 忽略），不属于交付内容，可随时清理。
`scripts/` 下的 `*.pyc` 不入库；若发现被 tracked，用 `git rm --cached` 移除并确认 `.gitignore` 覆盖。

## 4. 维护约定

1. 新增守卫脚本必须同时给出统一入口（优先加进对应 `run-all.sh`），并在本文登记 CI 接线状态。
2. 新增门禁自测必须接进 CI；「写了测试但没人跑」视为假绿。
3. 基线 txt 的修改遵循 C.6.4 棘轮：只降不升，上调须逐处比对、零命中即失败、反向验证改回旧值必红。
4. 长构建与全量门禁在资源紧张时一律后台执行，避免前台 SIGTERM 137。
