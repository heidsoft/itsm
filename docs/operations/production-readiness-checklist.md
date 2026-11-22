# 生产部署验收清单

> 适用于 `docker-compose.prod.yml` 私有化部署；由实施人与复核人共同签署。
> 依据：`docs/audits/deployment-audit.md`（基准 commit `b3bfd748604041d5082a320ed6d5c447221ac76a`）。

## 1. 配置与密钥

- [ ] 已从 `.env.prod.example` 复制为 `.env.prod`，且未把示例文件直接用作生产配置。
- [ ] `JWT_SECRET` 已显式设置且长度 ≥ 32，不是占位值（启动失败闸门会拒绝弱值）。
- [ ] `ADMIN_PASSWORD` 已显式设置，非弱口令（弱口令闸门会拒绝）。
- [ ] `METRICS_REQUIRE_AUTH=true`；若确需关闭，须有隔离网络证明与部署记录。
- [ ] 所有 Connector / 云凭证通过 secret 管理注入，不在 compose 或仓库文件中明文出现。
- [ ] 应用镜像版本通过 `${VERSION}` 固定；基础设施镜像已固定次要版本或 digest。

## 2. 初始化与迁移职责

- [ ] `itsm-init` 一次性任务退出码 0；backend / worker 的 `ITSM_AUTO_MIGRATE=false`、`ITSM_AUTO_SEED=false`。
- [ ] 迁移账本中每条磁盘迁移都有非空 checksum（历史收养条目除外）。
- [ ] 初始化完成后 `mcp:read` 权限码存在，`agent` / `audit_admin` 有授权。
- [ ] 重复执行一次 bootstrap 不产生新租户、新用户、新菜单或重复授权。
- [ ] 不手工编辑 `schema_migrations.checksum`；迁移内容变更一律走新的版本化迁移。

## 3. RLS 与租户隔离

- [ ] 记录 `RLS_MODE` 实际值；当前生产默认为 `off`，属于已记录的风险接受，不得擅自改为 `enforce`。
- [ ] 若启用 `shadow` / `enforce`，先运行 readiness 探测：连接角色不是 superuser/BYPASSRLS，且策略已覆盖目标表。
- [ ] 跨租户负例测试已执行（用户/工单/CI/知识/附件/通知/审计/变更等真实 Router 入口）。
- [ ] 应用更新后重新跑租户隔离回归，而不是只依赖数据库层。

## 4. 备份与恢复

- [ ] 已安装宿主定时备份（`scripts/prod-backup.sh` + plist/cron），`backups/` 至少有一份可用备份。
- [ ] 已执行一次 `prod-restore-drill.sh` 且退出码 0；恢复后行数与关键业务数据校验通过。
- [ ] 升级前已手动备份，并记录版本与校验值。

## 5. 健康、资源与可观测

- [ ] 全部服务 healthy，`/api/v1/health` 与 readyz 正常。
- [ ] 各服务有 CPU/内存限制与日志轮转。
- [ ] 日志中无密码、JWT、API key、连接器密钥与完整工单正文。
- [ ] 监控 profile 默认不启用；启用时 Prometheus 使用服务账号 JWT 抓取 `/metrics`。

## 6. 发布与回滚

- [ ] 待发布版本在 CHANGELOG 有对应段落，包含用户问题、兼容性、已知限制与验证范围。
- [ ] 发布前记录当前镜像 digest / 版本与数据库备份点。
- [ ] 回滚路径已书面确认：镜像回退 + 数据库备份点；涉及迁移的回滚需先确认账本 checksum 对齐方案。
- [ ] 未经人工审批，不执行破坏性迁移、权限扩大或自动部署到生产。

## 7. 签署

| 角色 | 姓名 | 日期 | 结论 |
|:---|:---|:---|:---|
| 实施人 | | | |
| 复核人 | | | |
