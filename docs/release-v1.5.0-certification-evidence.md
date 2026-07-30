# v1.5.0 发布认证证据归档（2026-07-30）

本文档归档 `docs/initialization-release-certification.md` 中 P7/P8/P9 阻断项的关闭证据，以及生产模式部署验证结果。

## 一、自动化回归（全部通过）

| 验证项 | 命令 | 结果 |
|---|---|---|
| 后端全量测试 | `cd itsm-backend && go test ./... -count=1` | ✅ 34 个包全部 ok，0 FAIL（含 sql_store 修复后复跑） |
| 前端类型检查 | `npm run type-check` | ✅ 通过 |
| 前端生产构建 | `npm run build` | ✅ 通过 |
| 前端单元测试 | `npm run test:unit` | ✅ 189 suites / 3300 passed / 13 skipped / 0 failed |

## 二、P7：Manifest 版本 + Checksum + 权限声明全覆盖

实现（`itsm-backend/connector/`、`service/skill_manifest.go`、`service/skill_registry.go`）：

- `Manifest.ComputeChecksum()`：对 name|version|provider|type|capabilities|required_permissions|min_itsm_ver 做确定性 SHA-256，前缀 `sha256:`。
- `Manifest.ValidateForRegistration()`：name/version/requiredPermissions 缺一即拒绝注册（fail-closed，`Registry.Register` panic）。
- 5 个官方连接器（console/dingtalk/feishu/webhook/wecom）全部声明 `IsOfficial + RequiredPermissions`，注册时自动计算 checksum。
- SkillManifest 采用相同 checksum 与 fail-closed 校验约定，保护未来注册的 AI 技能。
- 静态门禁测试：`connector/manifest_gate_test.go`（覆盖率断言 + checksum 确定性 + 不完整 manifest 拒绝），已纳入 `go test ./...`。

生产端到端证据：`GET /api/v1/connectors`（带 JWT）返回每个官方连接器的
`"isOfficial": true`、`"requiredPermissions": [...]`、`"checksum": "sha256:..."`（camelCase DTO）。

## 三、P8：PostgreSQL RLS / 初始化 / 备份恢复演练证据

环境：`docker-compose.prod.yml` 生产栈，PostgreSQL 17.10（`itsm-postgres-prod`，库 `itsm_prod`）。

### 3.1 迁移与 RLS enforce

- `schema_migrations`：007/008/009/010 全部应用。
- RLS：17 张租户表 `relrowsecurity = true`，17 条 `tenant_isolation_*` 策略，
  策略表达式 `tenant_id = get_current_tenant_id()`。

### 3.2 初始化账本（fail-closed 已实证）

- 首次启动 fail-closed 拦截默认 JWT_SECRET（`DEFAULT_JWT_SECRET` fatal），轮换为强随机值后放行 —— 护栏有效。
- 首次初始化暴露并修复 42P08 参数类型歧义（`internal/initialization/sql_store.go`，
  复用参数补 `::text` 显式类型；已在真实 PG 上以 `PREPARE` 验证并复跑单测）。
- `itsm-init-prod` exit 0，六个组件账本全部 `succeeded v1.0.0`：
  cmdb-core / extension-core / identity-rbac / itil-core / sla-core / workflow-core。

### 3.3 备份恢复演练（真实 schema，非 toy 表）

```
pg_dump -Fc itsm_prod → pg_restore → itsm_dr_drill
RESTORE_EXIT=0
source_tables=146 → restored_tables=146（全量一致）
restored_policies=17（RLS 策略随备份完整恢复）
restored_migrations=4
```

### 3.4 CI 长期证据

`.github/workflows/pg-disaster-recovery.yml` 持续运行 backup-drill / rls-validation /
fencing-fault-injection / pg-upgrade-verification / ci-evidence-summary 五个作业；
fencing stale-writer 由 `TestFencingTokenPreventsStaleWriter`、`TestFencingTokenCrashRecovery` 覆盖。

已知限制（不阻断本次发布，列入 v1.5 跟踪）：跨两个正式 PG 大版本的 `pg_upgrade` 实机演练、
大规模租户滚动升级压测尚未在生产等价数据量下执行。

## 四、P9：前端全量测试 + 关键 E2E 认证归档

- 单元测试：189 suites / 3300 passed / 13 skipped / 0 failed（Jest，2026-07-30）。
- 类型检查与生产构建：通过（见第一节）。
- 生产模式浏览器 E2E 冒烟（对 `http://localhost:3000` 生产容器执行）：
  - 登录页渲染 → admin 登录成功 → 跳转 `/dashboard`；
  - 仪表盘 KPI/图表正常渲染；工单列表页渲染 11 条数据、分页正常；
  - 控制台无 error/warn 级消息；
  - 截图归档：`test-results/smoke-dashboard.png`、`test-results/smoke-module.png`。
- 非阻断遗留：工单列表个别状态/类型字段展示英文原值（i18n 映射遗漏），列入后续优化。

## 五、生产模式部署验证

```
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

- 7 服务全部 Up：postgres/redis/minio healthy，itsm-init exit 0，
  itsm-backend-prod healthy（`GET /api/v1/health` → `{"status":"ok"}`），frontend healthy。
- 认证 fail-closed：错误密码 → `{"code":2001,"message":"invalid credentials"}`。
- 版本对齐：CHANGELOG 切版 `[1.5.0] - 2026-07-30`；`itsm-frontend/package.json` → `1.5.0`。
