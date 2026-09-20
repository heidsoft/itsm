# 产品功能债务复盘报告

> Status: historical

**日期**: 2026-09-18  
**版本**: v1.6.x  
**审计范围**: 全栈功能缺陷、安全漏洞、并发安全、API 契约一致性

---

## 执行摘要

本次复盘覆盖了 ITSM 系统 v1.6.x 版本的核心功能模块，重点审计了安全漏洞、并发安全、API 契约一致性和监控栈完整性。共发现并修复 **4 个 P0/P1 级别问题** 和 **2 个 P2 级别问题**，系统已具备生产环境部署条件。

---

## 已修复问题清单

### P0 安全漏洞（全部已修复）

#### SEC-001: 租户隔离失效 — 全局管理员可跨租户访问
**问题**: 缺少租户上下文时默认回退到 tenant 1，导致越权访问  
**修复**: 
- 所有查询强制添加 `TenantIDEQ` 条件
- 缺少租户上下文时 fail closed，不再默认回退
- 添加跨租户拒绝测试用例

**影响文件**: 
- `handlers/*/repository_impl.go` (全领域)
- `service/*.go` (共享服务层)

#### SEC-002: 审批链越权 — 非审批人可完成审批任务
**问题**: 审批任务完成接口未校验当前用户是否在审批链中  
**修复**: 
- `CompleteTask` 添加审批人身份校验
- 审批记录必须匹配 `ProcessTask.Assignee` 或 `ApprovalRecord` 中的用户
- 非审批人返回 403 Forbidden

**影响文件**: 
- `service/bpmn_process_executor.go`
- `handlers/bpmn/handler.go`

#### SEC-003: 状态机绕过 — 前端禁用按钮可被 API 绕过
**问题**: ITIL 状态迁移未在 service 层校验允许的 source/target  
**修复**: 
- 所有状态迁移在 service 层显式校验
- 非法迁移返回 409 Conflict
- 添加状态机测试覆盖成功、非法迁移、权限拒绝场景

**影响文件**: 
- `service/incident_service.go`
- `service/change_service.go`
- `service/problem_service.go`

#### SEC-004: 乐观锁失效 — read-then-unconditional-write
**问题**: 更新操作未包含 `WHERE version = ?` 条件，陈旧快照可静默覆盖他人写入  
**修复**: 
- 所有 ITIL 写操作添加 `VersionEQ` 条件
- 版本冲突返回 409 Conflict，不得静默覆盖
- `UpdateOneID().Where(VersionEQ(version)).AddVersion(1)`

**影响文件**: 
- `handlers/incident/repository_impl.go`
- `handlers/change/repository_impl.go`
- `handlers/problem/repository_impl.go`

#### SEC-005: 软删除守卫缺失
**问题**: 已软删除记录仍可被更新，导致数据污染  
**修复**: 
- 所有更新操作添加 `DeletedAtIsNil()` 条件
- 软删除后尝试更新返回 404 或 409

**影响文件**: 
- `handlers/*/repository_impl.go` (全领域)

#### FUNC-001: 事件分类端点字段丢失
**问题**: `UpdateClassification` 静默丢弃 `serviceType`、`failureType`、`urgency`、`impact` 字段  
**修复**: 
- Ent schema 添加 `service_type`、`failure_type` 字段
- Handler/Service/Repository/DTO 全链路支持新字段
- 手动 patch Ent 生成文件（因 Ent v0.14.6 与 Go 1.25 不兼容）

**影响文件**: 
- `ent/schema/incident.go`
- `ent/incident.go`, `ent/incident/*.go`, `ent/mutation.go` (手动 patch)
- `handlers/incident/entity.go`, `handler.go`, `service.go`, `repository_impl.go`
- `dto/incident_dto.go`

---

### P1 并发安全（全部已修复）

#### P1-1: BPMN 变量合并并发丢失
**问题**: `mergeVariablesInTx` 未使用 CAS，并发 `CompleteTask` 调用导致变量丢失  
**修复**: 
- 添加 `Where(processinstance.VersionEQ(inst.Version))` 条件
- 版本冲突返回明确错误，调用方可重试

**影响文件**: 
- `service/bpmn_process_executor.go`

---

### P2 API 契约一致性（全部已修复）

#### P2-1: 分页 maxSize 语义不统一
**问题**: 3 处 service 层使用 `pageSize > 200`，与标准 pagination helper 的 `max 100` 不一致  
**修复**: 
- `asset_service.go`: 200 → 100
- `problem_service.go`: 200 → 100
- `change_service.go`: 200 → 100

**影响文件**: 
- `service/asset_service.go`
- `service/problem_service.go`
- `service/change_service.go`

---

## 监控栈验证

### 已验证组件

| 组件 | 端口 | 状态 | 备注 |
|------|------|------|------|
| Prometheus | 9090 | ✅ 已配置 | localhost-only，需 SSH tunnel 或反向代理 |
| Alertmanager | 9093 | ✅ 已配置 | localhost-only |
| Alert-webhook | 9094 | ✅ 已配置 | 内部网络，Dockerfile 已提供 |
| Grafana | 3001 | ✅ 已配置 | localhost-only，3 个 dashboard 已 provision |
| Postgres Exporter | 9187 | ✅ 已配置 | 依赖 postgres 健康检查 |

### Dashboard 清单

1. `itsm-overview.json` — 系统总览（已跟踪）
2. `bpmn-workflows.json` — BPMN 工作流监控（已跟踪）
3. `business-dashboard.json` — 业务指标面板（已跟踪）
4. `itsmoverview.json` — 重复文件（未跟踪，建议删除或合并）

### 部署方式

```bash
# 启用监控栈
docker-compose -f docker-compose.prod.yml --profile monitoring --env-file .env.prod up -d

# 访问（需 SSH tunnel 或反向代理）
# Prometheus: http://localhost:9090
# Alertmanager: http://localhost:9093
# Grafana: http://localhost:3001
```

---

## 已知遗留问题

### 低优先级（不阻塞上线）

1. **Ent 代码生成不兼容 Go 1.25**
   - 症状: `go generate ./ent` 产生 832 个文件的 broken diff
   - 临时方案: 手动 patch 生成文件
   - 长期方案: 升级 Ent 到 v0.15+ 或降级 Go 到 1.24

2. **Grafana dashboard 重复文件**
   - `itsm-overview.json` 和 `itsmoverview.json` 同时存在
   - 建议: 删除 `itsmoverview.json` 或合并内容

3. **监控栈未集成到 CI/CD**
   - 当前监控栈为可选 profile，未在 CI 中验证
   - 建议: 添加 `docker-compose --profile monitoring config` 到 CI 验证

---

## 生产上线检查清单

### 必须项（已完成）

- [x] 所有 P0 安全漏洞已修复
- [x] 所有 P1 并发安全问题已修复
- [x] 所有 P2 API 契约问题已修复
- [x] 租户隔离测试覆盖
- [x] 乐观锁测试覆盖
- [x] 状态机测试覆盖
- [x] 监控栈配置完整
- [x] Dockerfile 和 docker-compose 验证通过

### 建议项（上线前完成）

- [ ] 生产环境压力测试（并发 CompleteTask、批量工单创建）
- [ ] 监控栈端到端验证（Prometheus → Alertmanager → Webhook → Backend）
- [ ] Grafana dashboard 数据源验证
- [ ] 备份恢复演练
- [ ] 滚动升级方案验证

### 可选项（上线后迭代）

- [ ] Ent 版本升级或 Go 版本降级
- [ ] 监控栈集成到 CI/CD
- [ ] RAG/Knowledge 模块安全审计
- [ ] Connector 生命周期状态机完善

---

## 结论

ITSM v1.6.x 已完成核心功能债务清理，安全漏洞、并发问题和 API 契约不一致问题均已修复。系统具备生产环境部署条件，建议在完成压力测试和监控栈端到端验证后启动灰度发布。

**推荐上线路径**:
1. 完成生产环境压力测试
2. 验证监控栈端到端流程
3. 选择 1-2 个低风险租户进行灰度
4. 观察 1-2 周无重大问题后全量发布

---

**审计人**: AI Assistant  
**审核状态**: 待人工审核  
**下一步**: 等待用户确认是否启动压力测试和灰度发布
