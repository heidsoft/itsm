# ADR-001: 采用模块化单体作为起始架构

## Status
Proposed

## Context

ITSM系统当前是一个包含80+服务和80+控制器的Go单体应用。随着功能增加和团队扩展，面临以下挑战：

1. **部署不灵活**: 难以独立扩展某个模块（如AI推理需要更多GPU）
2. **数据库耦合**: 所有模块共享同一PostgreSQL，大租户场景性能瓶颈
3. **AI能力紧耦合**: Python AI服务与Go后端紧耦合，无法独立扩展
4. **测试困难**: 代码量大，全量测试耗时长

## Decision

采用"模块化单体 + 事件驱动中台"的渐进式演进架构：

### 1. 模块化单体结构

保持Go后端为单一部署单元，但按业务领域划分子包：

```
itsm-backend/
├── domain/
│   ├── ticket/          # 工单领域
│   │   ├── model/
│   │   ├── service/
│   │   └── repository/
│   ├── incident/        # 事件领域
│   ├── problem/         # 问题领域
│   ├── change/          # 变更领域
│   └── common/          # 公共领域
├── infrastructure/
│   ├── event/           # 事件总线实现
│   ├── cache/           # 缓存层
│   └── storage/         # 存储层
└── api/                # API层（Controller）
```

### 2. 事件驱动中台

引入Kafka实现模块间解耦：

- **工单事件**: created/updated/assigned/resolved/closed
- **审批事件**: submitted/approved/rejected
- **SLA事件**: violation_warning/violated/resolved
- **通知事件**: to_email/to_sms/to_站内信

### 3. AI服务独立化

将Python AI服务独立部署，通过HTTP/gRPC与Go后端通信：

- RAG知识检索服务
- LLM网关服务
- 向量嵌入服务
- 智能分类/摘要服务

### 4. 多租户强化

- 租户上下文贯穿全链路
- 行级数据隔离
- 租户配置独立化（主题、SLA规则、审批流程）

## Consequences

### 正面

- ✅ 保留现有开发/调试体验，无需大幅重构
- ✅ 支持独立扩展AI服务（GPU资源按需）
- ✅ 事件驱动降低模块耦合，便于后续拆分
- ✅ 多租户能力为SaaS化奠定基础

### 负面

- ⚠️ 需要引入Kafka等消息基础设施
- ⚠️ 事件一致性需要额外处理（幂等、补偿）
- ⚠️ 团队扩大后需评估是否进一步拆分为微服务

### 中性

- 📋 模块分包结构调整需要一定工作量
- 📋 需要制定事件schema规范

## 实施计划

### Phase 1: 基础设施（2周）
- [ ] 搭建Kafka集群/开发环境
- [ ] 定义事件schema规范
- [ ] 实现事件总线基础设施

### Phase 2: 模块改造（4周）
- [ ] 按领域重组织代码包结构
- [ ] 将关键业务逻辑改为事件驱动
- [ ] 完善多租户中间件

### Phase 3: AI服务化（2周）
- [ ] AI服务Docker化
- [ ] 定义Go-Python通信协议
- [ ] 实现优雅降级

## 相关文档

- [ITSM产品模块架构设计](../prd/ITSM产品模块架构设计.md)
- [事件驱动设计](./event-driven-design.md)
- [多租户架构设计](./multi-tenant-design.md)
