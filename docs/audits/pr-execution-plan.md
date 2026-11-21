# 整改 PR 执行计划

> 基准 commit：`b3bfd748604041d5082a320ed6d5c447221ac76a`
> 所有 PR 从 `HEAD` 切独立分支；主工作树有 67 个并行会话 WIP，一律不触碰。

## PR-01 · 生产安全与租户隔离

分支：`audit/production-tenant-readiness`（本审计工作树）

**目标**：生产配置检查、部署验收清单、负例测试覆盖确认。

**低风险首批（本批实施）**：
1. `.env.prod.example`：把 `METRICS_REQUIRE_AUTH=false` 改为 `true`，把关闭方式写成注释，说明仅监控 profile 使用。
2. 新增 `scripts/__tests__/production-env-guard.test.js`：断言所有随仓库发布的 `*.env*.example` 不得在**非注释行**关闭 `/metrics` 鉴权，且 compose 默认值仍为 `true`。该文件会被现有 `node --test scripts/__tests__/*.test.js` 自动执行，无需改 workflow。
3. 新增部署验收清单 `docs/operations/production-readiness-checklist.md`，覆盖 env、迁移/播种职责、RLS 模式与 readiness、备份/恢复、镜像固定、健康与资源限制。
4. CHANGELOG `[Unreleased]` 增加用户视角条目。

**明确不在本批（需审批）**：AUDIT-002 重复索引收敛（数据库结构）、AUDIT-013/014 权限判定语义与缓存广播（权限语义）、翻 RLS 默认值。

**验收**：`node --test scripts/__tests__/*.test.js` 全绿；对守卫注入 `METRICS_REQUIRE_AUTH=false` 必须变红；`git diff --check` 干净。

## PR-02 · 菜单与路由一致性

分支：`audit/menu-route-consistency`（待创建）

**最小方案（分两步，避免把权限变更混进低风险批次）**：

步骤 A（低风险，仅显示元数据）：
- `baseline.go` 把 `/service-requests`、`/my-requests` 改为 `service_request:read`，`/workflow/audit` 改为 `bpmn:read`，`/dashboard` 改为 `dashboard:read`。
- 扩展 `baseline_guard_test.go`：断言这四条码值；新增"菜单路径→`page.tsx` 存在"的前端源码扫描测试。
- 修正 `docs/product/frontend-menu-and-features.md` 的 102→101 与退役入口描述。

步骤 B（需审批，改 API 授权）：
- `router/ticket_type_routes.go` 的 GET 改为 `ticket_type:read`（而非给角色加 `ticket:read`），重生成 `rbac_precheck_gen.go`，补 `service_catalog_admin` 的 200 与无权限 403 两个 HTTP 负例。
- 迁移 `20261121_*menu*.sql` 按 `(tenant_id,path)` 修正存量租户的 `name/parent_id`，并把 `/ai/chat/list` 纳入退役 WHERE；补双租户 up/down/幂等测试。

**验收**：`go test ./pkg/menubaseline/ ./migration/ ./router/ ./handlers/rbac/`；前端 `product-language-governance.test.ts` 与契约测试；新增守卫必须能因翻转期望码而变红。

## PR-04 · Release 发布质量

分支：`audit/release-quality-gate`（待创建）

按 `release-audit.md` §3 的 G1–G7 顺序实施；先做不改产品运行时的脚本与测试（tag↔CHANGELOG、禁用语、版本一致性、workflow 契约），再做 workflow 硬门禁；保留 docs-gate 现有硬检查不降级。

## PR-05 · UI 组件一致性

分支：`audit/frontend-design-consistency`（待创建）

先修三个已红的静态门禁（AUDIT-060），再处理 AUDIT-061/062 的结构性修复；所有视觉结论必须有浏览器证据（桌面/平板/手机 × 明暗主题 × 典型角色），否则保持 `suspected`。
本工作树缺少 `node_modules` 与后端服务，运行验证需要在可执行环境中完成。

## 顺序与依赖

```
PR #114 (首装, 待合并)
   ↓ 独立
PR-01 (生产配置) ── 本批已实施
   ↓ 独立
PR-02 步骤 A (显示元数据) → 步骤 B (API 授权, 需审批)
   ↓ 独立
PR-04 (Release) / PR-05 (UI) 可并行
```

任何涉及生产数据、权限扩大、数据库结构或自动部署的动作，在进入实施前单独提请人工审批。
