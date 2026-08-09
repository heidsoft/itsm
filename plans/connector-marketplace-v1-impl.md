# v1.1 连接器市场实现计划

## 目标

根据 ROADMAP，v1.1 需要完成连接器市场 v1，包括：
- Feishu (IM + Approval)
- DingTalk (IM + Work Notice)
- WeCom (IM)
- Webhook
- 通过 `/api/v1/connectors/lifecycle` 管理生命周期

## 当前状态分析

### 已完成
- ✅ 连接器框架 (`connector/connector.go`)
- ✅ 连接器注册表 (`connector/registry.go`)
- ✅ 连接器管理器 (`connector/manager.go`)
- ✅ 飞书连接器基础实现 (535行)
- ✅ 钉钉连接器骨架 (81行)
- ✅ 企微连接器骨架 (80行)
- ✅ Webhook 连接器骨架
- ✅ 市场服务基础框架 (`service/marketplace/`)
- ✅ 市场数据模型 (`ent/schema/marketplace_item.go`)

### 待完成
- ❌ 钉钉连接器完整实现 (消息发送、卡片、工作通知)
- ❌ 企微连接器完整实现 (消息发送)
- ❌ Webhook 连接器完整实现
- ❌ 连接器生命周期 API (`/api/v1/connectors/lifecycle`)
- ❌ 连接器健康检查 API
- ❌ 连接器配置验证
- ❌ 市场安装/卸载与连接器注册联动

## 实现计划

### 阶段 1: 完善钉钉连接器 (1-2天)

**目标**: 实现钉钉工作通知和群机器人消息发送

**任务**:
1. 完善 `dingtalk/client.go`
   - 实现 access_token 获取和缓存
   - 实现工作通知发送 API
   - 实现群机器人消息发送 API
   - 支持 text/markdown/actionCard 消息类型

2. 完善 `dingtalk/message.go`
   - 统一消息结构转换
   - 支持 @人 功能

3. 添加测试
   - 单元测试覆盖核心功能
   - Mock API 响应

### 阶段 2: 完善企微连接器 (1天)

**目标**: 实现企微消息发送

**任务**:
1. 完善 `wecom/client.go`
   - 实现 access_token 获取和缓存
   - 实现应用消息发送 API
   - 实现群机器人消息发送 API

2. 完善 `wecom/message.go`
   - 统一消息结构转换

3. 添加测试

### 阶段 3: 完善 Webhook 连接器 (1天)

**目标**: 实现通用 HTTP Webhook 出站

**任务**:
1. 完善 `webhook/webhook.go`
   - 支持自定义 HTTP 方法 (GET/POST/PUT)
   - 支持自定义 Headers
   - 支持 JSON/Form/Text 请求体
   - 支持签名验证 (HMAC-SHA256)

2. 添加测试

### 阶段 4: 连接器生命周期 API (2天)

**目标**: 实现连接器生命周期管理 API

**任务**:
1. 创建 `controller/connector_lifecycle_controller.go`
   - `POST /api/v1/connectors/:name/provision` - 配置并启用连接器
   - `POST /api/v1/connectors/:name/revoke` - 停用连接器
   - `GET /api/v1/connectors/:name/health` - 健康检查
   - `GET /api/v1/connectors` - 列出已注册连接器
   - `GET /api/v1/connectors/:name/config` - 获取配置（敏感信息脱敏）

2. 创建 DTO
   - `dto/connector_lifecycle_dto.go`

3. 添加路由注册

4. 添加测试

### 阶段 5: 市场安装/卸载联动 (1天)

**目标**: 市场安装/卸载时自动配置/移除连接器

**任务**:
1. 修改 `service/marketplace/service.go`
   - InstallItem 时调用 Manager.Provision
   - UninstallItem 时调用 Manager.Revoke

2. 添加连接器配置验证
   - 根据 Manifest 验证必填配置项

3. 添加测试

### 阶段 6: 集成测试和文档 (1天)

**目标**: 端到端测试和文档

**任务**:
1. 编写集成测试
   - 测试连接器注册、配置、启用、禁用流程
   - 测试市场安装、卸载流程

2. 更新 API 文档

3. 更新 README

## 文件清单

### 新增文件
- `itsm-backend/controller/connector_lifecycle_controller.go`
- `itsm-backend/dto/connector_lifecycle_dto.go`
- `itsm-backend/service/connector_lifecycle_service.go`
- `itsm-backend/connector/builtin/dingtalk/client_test.go`
- `itsm-backend/connector/builtin/wecom/client_test.go`
- `itsm-backend/connector/builtin/webhook/webhook_test.go`

### 修改文件
- `itsm-backend/connector/builtin/dingtalk/client.go`
- `itsm-backend/connector/builtin/dingtalk/message.go`
- `itsm-backend/connector/builtin/wecom/client.go`
- `itsm-backend/connector/builtin/wecom/message.go`
- `itsm-backend/connector/builtin/webhook/webhook.go`
- `itsm-backend/service/marketplace/service.go`
- `itsm-backend/router/router.go` (添加新路由)

## 验收标准

1. **功能验收**
   - [ ] 飞书连接器可发送消息和卡片
   - [ ] 钉钉连接器可发送工作通知和群消息
   - [ ] 企微连接器可发送应用消息
   - [ ] Webhook 连接器可发送 HTTP 请求
   - [ ] 连接器可通过 API 配置和管理
   - [ ] 市场安装/卸载自动配置连接器

2. **质量验收**
   - [ ] 单元测试覆盖率 ≥ 40%
   - [ ] 集成测试覆盖核心流程
   - [ ] 无 lint 错误
   - [ ] 敏感信息不暴露到 API 响应

3. **文档验收**
   - [ ] API 文档更新
   - [ ] 连接器配置说明
   - [ ] README 更新

## 时间估算

| 阶段 | 任务 | 估算时间 |
|------|------|----------|
| 1 | 完善钉钉连接器 | 1-2天 |
| 2 | 完善企微连接器 | 1天 |
| 3 | 完善 Webhook 连接器 | 1天 |
| 4 | 连接器生命周期 API | 2天 |
| 5 | 市场安装/卸载联动 | 1天 |
| 6 | 集成测试和文档 | 1天 |
| **总计** | | **7-8天** |

## 风险和依赖

### 风险
1. 第三方 API 变更 - 需要关注飞书/钉钉/企微 API 更新
2. 测试环境 - 需要配置测试账号和应用

### 依赖
1. 飞书开放平台测试应用
2. 钉钉开放平台测试应用
3. 企微开放平台测试应用

## 后续迭代

v1.5 计划：
- 连接器市场 UI
- 连接器版本管理
- 连接器权限隔离
- 连接器使用统计
