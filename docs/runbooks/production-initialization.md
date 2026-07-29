# 生产数据初始化运行手册

## 发布前提

- 使用 PostgreSQL 17，并已完成可恢复备份和恢复抽检。
- 发布制品、迁移文件和初始化 manifest 来自同一 release version。
- 显式提供生产环境变量文件；不得使用仓库默认凭据。
- 普通 Web 容器必须设置 `ITSM_AUTO_MIGRATE=false`、`ITSM_AUTO_SEED=false`。

## 标准发布

1. 停止新变更，记录数据库版本、应用版本和租户模板版本。
2. 校验备份、可用磁盘、数据库 DDL/DML 权限和连接数余量。
3. 运行一次性 `itsm-init` Job。该 Job 是唯一允许启用
   `ITSM_BOOTSTRAP_ONLY=true` 的生产进程。
4. 查看初始化 run 和 component attempt，确认六个组件均为 `succeeded`：
   `identity-rbac`、`itil-core`、`workflow-core`、`sla-core`、`cmdb-core`、
   `extension-core`。
5. 启动 Web 实例。只有 `GET /api/v1/readyz` 返回 200 才允许接入流量。
6. 对 private、saas 或 saas_msp 的目标租户执行开通验证，并保留 run ID。

Docker Compose 必须显式传入环境文件：

```bash
docker compose -f docker-compose.prod.yml --env-file .env.prod config
docker compose -f docker-compose.prod.yml --env-file .env.prod up itsm-init
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

## 失败与重试

- 不得手工修改账本为成功。
- 先通过 run ID 定位失败组件和错误；修复根因后使用初始化 CLI 的 `retry`。
- 组件写入和组件内验证在同一事务内执行；失败组件不会提交部分数据。
- lease 未过期时禁止强制接管。executor 崩溃后，等待 lease 到期并确认原进程已停止，
  再由新 executor 重试。
- checksum 不匹配表示发布制品或 manifest 被修改，必须停止发布并重新生成制品。
- 单租户失败只隔离该租户；平台组件失败必须保持全局 Not Ready。

## 回滚与恢复

- 数据库迁移遵循 expand/contract；优先回滚应用镜像，不反向删除运行数据。
- 初始化模板采用 forward-fix。已经被流程实例引用的定义不得物理删除。
- 若结构迁移不可兼容，恢复到发布前备份，在隔离环境验证后再恢复服务。
- 恢复完成后先运行 migration verify，再执行初始化 `verify`；不得直接绕过 readiness。

## 发布证据

归档以下内容：release version、migration 状态与 checksum、初始化 run ID、六组件版本与
checksum、三种部署模式测试结果、R0 零写入结果、备份/恢复演练记录、已知风险与签字人。
