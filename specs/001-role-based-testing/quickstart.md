# Quickstart: 角色驱动的全产品测试

**Feature**: 001-role-based-testing
**Audience**: 开发 / QA / 运维
**Estimated time**: 本地首跑约 20 分钟，CI 单跑约 8 分钟。

---

## 0. 前置

- 已 clone 仓库，处于 `001-role-based-testing` 分支。
- 本地具备：`go 1.22+`, `node 20+`, `docker`, `psql`, `jq`, `curl`, `bash`。
- 端口空闲：`3000`（前端）、`8090`（后端）、`5432`（PG）、`6379`（Redis）。

---

## 1. 一键起依赖

```bash
docker compose -f docker-compose.dev.yml up -d postgres redis
```

期望：`docker ps` 中两个容器 `Up`。

---

## 2. 启后端

```bash
cd itsm-backend
DB_PASSWORD=itsm_password_2026 \
JWT_SECRET=dev-secret \
ADMIN_PASSWORD=admin123 \
go run main.go &
echo $! > /tmp/itsm-backend.pid
```

健康检查：

```bash
curl -fsS http://localhost:8090/api/v1/health
curl -fsS http://localhost:8090/api/v1/readiness/ga | jq '.data.modules | length'
# 期望 12
```

---

## 3. 自动化基线

```bash
# 后端
cd itsm-backend && go test ./... -count=1

# 前端
cd ../itsm-frontend
npm ci
npm run type-check
npm run lint:check
npm run test:unit
npm run build
```

全过 → 进入冒烟。

---

## 4. API 冒烟（≥25 端点）

```bash
bash docs/scripts/smoke-api.sh
echo "exit=$?"
```

期望 `exit=0`。失败行形如：

```
[FAIL] tickets-list GET /api/v1/tickets -> 401 (expect 200)
```

修复后重跑直到 `All endpoints OK`。

---

## 5. 启前端 dev server

```bash
cd itsm-frontend
NEXT_PUBLIC_API_URL=http://localhost:8090 \
npx next dev -p 3000 &
echo $! > /tmp/itsm-frontend.pid

# 等待启动
until curl -fsS http://localhost:3000 >/dev/null; do sleep 1; done
```

---

## 6. 角色 E2E

```bash
cd itsm-frontend

# 跑全部 7 角色
npx playwright test tests/e2e/roles --reporter=line

# 单角色（独立可交付，FR Independent Test）
npx playwright test tests/e2e/roles/end-user.spec.ts
npx playwright test tests/e2e/roles/engineer.spec.ts
npx playwright test tests/e2e/roles/super-admin.spec.ts
```

P1 套件 100% 通过；P2 套件 ≥90%。

---

## 7. 跨角色 FLOW

```bash
npx playwright test tests/e2e/flows --reporter=line
```

`FLOW-1..10` 期望全过；`FLOW-9` 多租户隔离零跨租户泄露。

---

## 8. 收尾

```bash
kill $(cat /tmp/itsm-backend.pid) $(cat /tmp/itsm-frontend.pid) 2>/dev/null
docker compose -f docker-compose.dev.yml down
```

---

## CI 简版

`.github/workflows/ga-readiness.yml` 串行执行步骤 2→3→4→5→6→7。失败时上传 `playwright-report/` 与 `/tmp/itsm-*.log` 作为 artifact。

---

## GA Readiness 验收

完成上述 8 步并满足 spec §SC：

- SC-101 P1 100% / SC-102 P2 ≥90% / SC-103 FLOW 100%
- SC-201 后端测试 100% / SC-202 前端测试 100% / SC-204 ≥25 端点冒烟
- SC-401 OWASP A01-A10 通过 / SC-402 readiness 12/12 / SC-403 P0=0

→ 进入 RC → GA。
