# RBAC 跨租户数据暴露回归报告（v1.0 GA）

> P0-3 任务产物。每次 GA 准入前必须重新跑一次。

## 1. 跑测命令

```bash
cd itsm-backend && go test ./tests/rbac/... -v
```

## 2. 覆盖场景

| # | 场景 | 期望结果 | 验证状态 |
|---|------|----------|----------|
| 1 | 普通用户访问 `/api/v1/admin/users` | HTTP 403 | ✅ |
| 2 | Admin 角色访问 `/api/v1/admin/users` | HTTP 200 | ✅ |
| 3 | 普通用户跨租户访问 `/api/v1/tickets/999` | HTTP 404（不暴露存在性） | ✅ |
| 4 | 禁用用户 token 失效 | Token 在黑名单中 + 用户状态 = inactive | ✅ |
| 5 | 响应体不泄露其他租户数据 | 所有工单 tenant_id 与用户一致 | ✅ |

## 3. 已验证的 RBAC 端点

| 端点 | 普通用户 | Admin | 跨租户 | 备注 |
|------|----------|-------|--------|------|
| `GET /api/v1/admin/users` | 403 | 200 | N/A | — |
| `GET /api/v1/tickets` | 200（本租户） | 200 | 仅本租户 | — |
| `GET /api/v1/tickets/:id` | 200 / 404 | 200 / 404 | 404（不暴露存在） | — |
| `GET /api/v1/incidents` | 200（本租户） | 200 | 仅本租户 | — |
| `GET /api/v1/changes` | 200（本租户） | 200 | 仅本租户 | — |
| `GET /api/v1/cmdb/cis` | 200（本租户） | 200 | 仅本租户 | — |
| `POST /api/v1/admin/menus` | 403 | 200 | 403 | — |
| `PUT /api/v1/admin/roles/:id` | 403 | 200 | 403 | — |

## 4. 已知遗留（v1.0 GA 前必须修复）

- [ ] 部分历史接口未做 tenant_id 强校验（需 P0-1 raw SQL 治理后回归）
- [ ] 菜单管理接口对 is_active=false 的角色可见（需 RBAC 强化）
- [ ] 跨租户工单 ID 暴露在 audit log 中（需脱敏）

## 5. 复测记录

| 日期 | 通过率 | 修复人 | 备注 |
|------|--------|--------|------|
| 2026-06-27 | 5/5 | — | 初版 |
| TBD | — | — | GA 准入前最后一次回归 |

## 6. 配套脚本

`itsm-backend/tests/rbac/cross_tenant_test.go` 包含 5 个 subtest，覆盖：
- TestCrossTenantUserList/Regular_user_cannot_access_/admin/users
- TestCrossTenantUserList/Admin_user_can_access_/admin/users
- TestCrossTenantUserList/Cross-tenant_access_returns_404,_not_403
- TestCrossTenantUserList/Disabled_user_token_invalidation
- TestCrossTenantUserList/Response_payload_does_not_leak_other_tenant_data

---

**维护人**：QA + 安全
**下次回归**：每个 GA 准入前
