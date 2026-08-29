# 五域双分层端点对账清单（2026-08-29）

> 对应 [蓝图](../../plans/production-readiness-industry-blueprint.md) P1-1.1。
> 结论:双分层重叠比预期轻——多数域已完成或接近完成迁移,真正问题集中在 **incident(未迁移)** 与 **knowledge(死代码未清理)**。

## 对账结果

| 域 | 旧层（controller/）现状 | 新层（handlers/）现状 | 判定 |
|---|---|---|---|
| **incident** | 承载全部生产路由:`/incidents` 30+ 端点（`router/router.go:944-1007`） | 实现约六成（775 行 handler + 1200 行测试,含 CRUD/escalate/root-cause/impact/comments）,但**零路由**;`app.go:765` 仅编译占位 | **未迁移**,旧层独担 |
| **cmdb** | 仅剩弃用别名 `/configuration-items/*`（`router/cmdb_routes.go` 注明"弃用,勿新增端点"）+ `/incidents/configuration-items` | 主体 21 端点:`/cmdb/*`、`/cloud/*` | **基本完成**,余弃用别名 |
| **knowledge** | `knowledge_controller.go` 等**零路由接线**（死代码） | `/knowledge/articles` + `/knowledge-articles` 全在新层 | **已完成**,旧层可删 |
| **problem** | `/problem-investigation/*`（调查子域）+ `/problems/:id/relationships` | `/problems` CRUD 与生命周期 | **互补非重叠**,调查为独立子功能 |
| **sla** | 仅 `SLATemplateController`（模板） | `/sla` 全套（含旧路径兼容 `:1262`） | **基本完成**,模板留旧层 |

## 新层 incident 缺口（若迁移）

`handlers/incident` 缺少旧层已有的生命周期动作:

- `POST /:id/acknowledge`、`/:id/resolve`、`/:id/close`、`/:id/reopen`
- `POST /:id/assign`、`/:id/major-incident`、`/:id/convert-to-problem`
- `PUT /:id/sla/pause`、`/:id/sla/resume`
- 事件创建 `POST /events`、告警/指标已有读端点

## 建议动作（待决策后执行）

1. **knowledge**:删除 `controller/knowledge_controller.go`、`knowledge_integration_controller.go` 死代码（需先确认无其他引用）。
2. **incident**:二选一——
   - a) 补齐 9 个生命周期端点后一次切路由（符合 `domain-ownership.md` 目标,工作量中等）;
   - b) 显式延后到 v1.7:删除 `handlers/incident` 编译占位（保留或归档实现）,更新 `domain-ownership.md` 记录实际状态。
3. **cmdb/sla**:为弃用别名与模板设定移除里程碑,纳入 1.3 的 CI 冻结范围。
