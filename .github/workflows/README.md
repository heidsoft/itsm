# GitHub Actions Workflows 索引

> 本目录存放所有 GitHub Actions workflow 配置。本 README 记录 workflow 设计原则、覆盖率门槛
> 策略、与 CI 修复历史。

## 目录

| Workflow | 文件 | 触发时机 | 主要职责 |
|----------|------|---------|---------|
| backend-ci | [`backend-ci.yml`](./backend-ci.yml) | push / PR | 后端 lint (gofumpt + staticcheck) + test + build |
| frontend-ci | [`frontend-ci.yml`](./frontend-ci.yml) | push / PR | 前端 type-check + build + standalone 验证 |
| ga-gate | [`ga-gate.yml`](./ga-gate.yml) | push to main / PR | GA 准入门禁（G1~G4 + Aggregate） |
| security | [`security.yml`](./security.yml) | push / PR / schedule | npm audit + Gitleaks + CodeQL |
| release | [`release.yml`](./release.yml) | tag push | 打包 zip + 推 GHCR 镜像 |

---

## GA Gate 准入（G1~G4）

`ga-gate.yml` 是 main 分支的最终守门员，4 项全过才允许合入：

| Gate | 名称 | 当前阈值 | 说明 |
|------|------|---------|------|
| G1 | Backend Test | coverage **≥ 1%** (v1.0 floor) | `go test` 全绿，1% 防退化，70% 仅 warning |
| G2 | Frontend Build | 必须 build 成功 | type-check + build + standalone 输出 |
| G3 | Compose Health | `/api/v1/health` 返回 200 | Docker Compose 全栈启动 |
| G4 | E2E Smoke | 11 个核心 API + smoke-test.sh 5 阶段 | 登录 + 列表 + DB + 前端 |

---

## 覆盖率门槛策略（三段式）

任何新项目的 CI 覆盖率门槛都要分阶段，**不能一次性设 70%**：

| 阶段 | 阈值 | 行为 | 适用 |
|------|------|------|------|
| v1.0 GA | **1%** | 防退化 floor，`< 1%` fail | 单测基本为 0 的存量项目 |
| v1.1 | **40%** | `::error::` 阻塞 PR | 关键路径已有单测 |
| v2.0 | **70%** | `::error::` 阻塞 PR | 团队形成测试习惯 |

### 当前阶段：v1.0 GA（阈值 1%）

`ga-gate.yml` 中 `Run go test` 步骤的关键代码：

```bash
TESTABLE_PKGS=$(go list ./... 2>/dev/null \
  | grep -vE '^itsm-backend/migrations$|^itsm-backend/migrations/hook$|^itsm-backend/migrations/enttest$|^itsm-backend/migrations/predicate$')
go test $TESTABLE_PKGS -coverprofile=coverage.out -covermode=set
if [ -f coverage.out ]; then
  coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
  echo "Backend coverage: ${coverage}%"
  # v1.0 GA floor: 1% — prevents regression, tracks real number.
  if (( $(echo "$coverage < 1" | bc -l) )); then
    echo "::error::Coverage ${coverage}% regressed below 1% v1.0 floor"
    exit 1
  fi
  if (( $(echo "$coverage < 70" | bc -l) )); then
    echo "::warning::Coverage ${coverage}% is below the long-term 70% target (v1.0 GA floor = 1%). See TODO(v1.1) in this workflow."
  fi
fi
```

### 阈值上调 checklist（v1.0 → v1.1）

- [ ] service/* 包单测覆盖率 ≥ 60%
- [ ] controller/* 包单测覆盖率 ≥ 60%
- [ ] 把 `< 1` 改为 `< 40`
- [ ] PR 试跑几周，确认无 critical regression
- [ ] 更新本 README 对应阶段的阈值

---

## ent cold-cache 假阳性处理

`ent generate` 同时产出两种结构：

```
itsm-backend/migrations/passwordresettoken.go          # package migrations
itsm-backend/migrations/passwordresettoken/            # package passwordresettoken
```

Go loader 在 cold-cache 下扫描 `./...` 时会误判。

**所有 workflow 必须遵循的 warmup 模式**：

```bash
# warmup：预热 + 容错（`|| true` 是关键）
go build ./migrations/... 2>&1 | tail -3 || true
go build ./... 2>&1 | tail -5 || true
go vet ./... 2>&1 | tail -5 || true
```

**所有 workflow 必须遵循的 go test 模式**：

```bash
# 排除无 _test.go 的 ent 自动生成包
TESTABLE_PKGS=$(go list ./... 2>/dev/null \
  | grep -vE '^itsm-backend/migrations$|^itsm-backend/migrations/hook$|^itsm-backend/migrations/enttest$|^itsm-backend/migrations/predicate$')
go test $TESTABLE_PKGS -coverprofile=coverage.out -covermode=set
# 注意：真正的 `go test` 步骤不能加 `|| true`
```

---

## CI 失败排查 SOP

> **本节是 P0 必读**。v1.0 GA 修复时我们曾因不遵守本 SOP 误判 3 个 commit。

任何 CI 失败必须按以下顺序排查：

```
1. 拉整段日志到本地：
   gh api /repos/<owner>/<repo>/actions/jobs/<jobId>/logs > /tmp/x.log

2. 看时间线：每个 step 有 startedAt / completedAt，把日志按时间切段

3. 在失败 step 内部找 ::::error:::、Process completed with exit code N、FAIL\sook\stotal

4. 先确认失败 step 的 continue-on-error、::warning::、|| true 是不是被忽略
   - warmup 步骤的 ##[error] 通常已 || true 容错，不会让 job 失败
   - ::warning:: 是非阻塞的
   - ::error:: + exit 1 才是真阻塞

5. 最后才去看代码
```

完整复盘见 [`docs/ci/postmortem-v1.0-GA.md`](../../docs/ci/postmortem-v1.0-GA.md)。

---

## 历史变更

| 日期 | 变更 |
|------|------|
| 2026-06-27 | v1.0 GA CI 全绿：5 个 commit（54e8de8c, 8c9e92c7, b9eb6dfb, a55b2522）+ 本 README |
| 2026-06-27 | 创建本 README，记录三段式覆盖率门槛策略 |