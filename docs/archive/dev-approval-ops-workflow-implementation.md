# 开发审批运维流程实施完成报告

## 执行摘要

已成功创建三级审批工作流（开发提交 → 主管审批 → 运维操作）并完成系统注册。该流程已通过 BPMN 模板机制部署，可通过工单类型绑定使用。

## 完成的工作

### 1. BPMN 工作流定义

**文件**: `itsm-backend/service/bpmn/dev_approval_ops_flow.bpmn`

**流程节点**:
```
开始 → 开发提交 → 主管审批 → [审批结果网关]
                              ├─ 通过 → 运维操作 → 结束
                              └─ 驳回 → 开发修改 → 主管审批（循环）
```

**节点配置**:

| 节点 | 类型 | 处理人角色 | 超时 | 说明 |
|------|------|-----------|------|------|
| 开发提交 | userTask | developer | 24h | 开发人员提交请求 |
| 主管审批 | userTask | manager | 8h | 主管审批，需审批标记 |
| 运维操作 | userTask | ops_engineer | 16h | 审批通过后执行操作 |
| 开发修改 | userTask | developer | 24h | 驳回后修改重新提交 |

**流程特性**:
- ✅ 三级审批链路完整
- ✅ 驳回循环机制（可无限次修改重提）
- ✅ 审批结果网关条件表达式
- ✅ 每个节点配置超时时间
- ✅ BPMN DI 图形布局完整

### 2. 模板元数据注册

**文件**: `itsm-backend/service/bpmn_template_service.go` (第 146-151 行)

```go
case "dev_approval_ops_flow":
    info.Name = "开发审批运维流程"
    info.Category = "ticket"
    info.SubCategory = "approval"
    info.Description = "开发提交→主管审批→运维操作三级流程"
```

**作用**: 系统启动时自动发现并部署该模板，提供人类可读的元数据。

### 3. 流程绑定配置

**文件**: `itsm-backend/config/seed/default.json` (第 129 行)

```json
{"business_type": "dev_ops_request", "process_definition_key": "dev_approval_ops_flow", "is_default": false}
```

**作用**: 将业务类型 `dev_ops_request` 绑定到该流程定义，支持通过 ProcessBinding 路由机制自动关联。

### 4. 编译验证

```bash
✅ 后端编译成功，无错误
✅ BPMN XML 通过 lint 检查
✅ go:embed 自动包含新文件
```

## 使用方式

### 方式一：通过 API 创建工单类型并绑定

**API 端点**: `POST /api/v1/ticket-types`

**请求体**:
```json
{
  "code": "dev_ops_request",
  "name": "开发运维申请",
  "description": "开发环境运维操作申请，需主管审批后由运维执行",
  "icon": "CodeOutlined",
  "color": "#1890ff",
  "defaultPriority": "medium",
  "workflowDefinitionKey": "dev_approval_ops_flow",
  "approvalEnabled": true,
  "customFields": [
    {
      "id": "env_type",
      "name": "环境类型",
      "label": "目标环境",
      "type": "select",
      "required": true,
      "options": [
        {"label": "开发环境", "value": "dev"},
        {"label": "测试环境", "value": "test"},
        {"label": "预发布环境", "value": "staging"}
      ],
      "order": 1
    },
    {
      "id": "operation_type",
      "name": "操作类型",
      "label": "运维操作类型",
      "type": "select",
      "required": true,
      "options": [
        {"label": "服务重启", "value": "restart"},
        {"label": "配置变更", "value": "config_change"},
        {"label": "数据备份", "value": "backup"},
        {"label": "权限开通", "value": "permission"}
      ],
      "order": 2
    },
    {
      "id": "description",
      "name": "description",
      "label": "操作说明",
      "type": "textarea",
      "required": true,
      "placeholder": "请详细描述需要执行的运维操作",
      "order": 3
    }
  ],
  "slaEnabled": true,
  "autoAssignEnabled": false
}
```

**响应示例**:
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 10,
    "code": "dev_ops_request",
    "name": "开发运维申请",
    "workflowDefinitionKey": "dev_approval_ops_flow",
    "status": "active",
    "approvalEnabled": true,
    ...
  }
}
```

### 方式二：通过前端界面创建

1. 登录系统（管理员账号）
2. 进入 **系统管理 → 工单类型**
3. 点击 **新建工单类型**
4. 填写基本信息：
   - 代码: `dev_ops_request`
   - 名称: `开发运维申请`
   - 描述: `开发环境运维操作申请，需主管审批后由运维执行`
5. 在 **工作流配置** 下拉框选择: `开发审批运维流程`
6. 配置自定义字段（环境类型、操作类型、操作说明）
7. 启用审批和 SLA
8. 点击 **保存**

### 方式三：使用预设安装（如已配置）

**API 端点**: `POST /api/v1/ticket-type-presets/dev_ops_request/install`

```bash
curl -X POST http://localhost:8090/api/v1/ticket-type-presets/dev_ops_request/install \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json"
```

## 流程执行链路

### 1. 工单创建

```
用户创建工单 (ticket_type=dev_ops_request)
  ↓
process_resolver.go 解析流程
  ├─ 查找 ticket_type.workflow_definition_key = "dev_approval_ops_flow"
  ├─ 或查找 ProcessBinding(business_type="dev_ops_request")
  └─ 返回流程定义
  ↓
bpmn_engine.CreateProcessInstance()
  ↓
流程实例启动，进入第一个节点：开发提交
```

### 2. 任务流转

```
[开发提交] 节点
  ├─ 分配给: developer 角色
  ├─ 超时: 24小时
  └─ 完成后 → 流转到 [主管审批]

[主管审批] 节点
  ├─ 分配给: manager 角色
  ├─ 超时: 8小时
  ├─ 审批通过 (approved=true) → 流转到 [运维操作]
  └─ 审批驳回 (approved=false) → 流转到 [开发修改]

[开发修改] 节点
  ├─ 分配给: developer 角色
  ├─ 超时: 24小时
  └─ 完成后 → 流转到 [主管审批]（重新审批）

[运维操作] 节点
  ├─ 分配给: ops_engineer 角色
  ├─ 超时: 16小时
  └─ 完成后 → 流转到 [结束]
```

### 3. 任务分配机制

**角色解析** (参考 `enterprise-approver-resolution-strategies.md`):

| 角色标识 | 解析策略 | 说明 |
|---------|---------|------|
| developer | 请求人所属部门的开发角色 | 或工单创建者 |
| manager | 请求人的直属上级 | 通过 user_departments 表查询 |
| ops_engineer | 运维部门指定角色 | 或 SLA 配置的运维组 |

## 验证清单

### 后端验证

```bash
# 1. 编译检查
cd itsm-backend && go build -o /tmp/itsm-backend-test main.go
✅ 编译成功

# 2. 启动服务（自动部署模板）
./itsm-backend

# 3. 检查模板部署
curl -H "Authorization: Bearer <token>" \
  http://localhost:8090/api/v1/workflow/definitions | \
  jq '.[] | select(.key == "dev_approval_ops_flow")'

# 预期输出:
{
  "id": "dev_approval_ops_flow-v1",
  "key": "dev_approval_ops_flow",
  "name": "开发审批运维流程",
  "category": "ticket",
  "subCategory": "approval",
  "version": "1.0.0",
  ...
}

# 4. 检查流程绑定
curl -H "Authorization: Bearer <token>" \
  http://localhost:8090/api/v1/workflow/bindings | \
  jq '.[] | select(.businessType == "dev_ops_request")'

# 预期输出:
{
  "businessType": "dev_ops_request",
  "processDefinitionKey": "dev_approval_ops_flow",
  "isDefault": false,
  "isActive": true
}
```

### 前端验证

```bash
# 1. 类型检查
cd itsm-frontend && npm run type-check
✅ 无错误

# 2. 启动开发服务器
npm run dev

# 3. 浏览器访问 http://localhost:3000
# 4. 进入 系统管理 → 工单类型
# 5. 点击 新建工单类型
# 6. 在"工作流配置"下拉框应看到: 开发审批运维流程
```

### 端到端验证

```bash
# 1. 创建工单类型
curl -X POST http://localhost:8090/api/v1/ticket-types \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "dev_ops_request",
    "name": "开发运维申请",
    "workflowDefinitionKey": "dev_approval_ops_flow",
    "approvalEnabled": true,
    "customFields": []
  }'

# 2. 创建工单
curl -X POST http://localhost:8090/api/v1/tickets \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "申请重启开发环境数据库服务",
    "ticketTypeCode": "dev_ops_request",
    "priority": "medium",
    "description": "开发环境数据库服务异常，需要重启"
  }'

# 3. 检查流程实例
curl -H "Authorization: Bearer <token>" \
  http://localhost:8090/api/v1/workflow/instances?businessId=<ticket_id> | \
  jq '.items[0] | {processDefinitionKey, currentStatus, activeTasks}'

# 预期: processDefinitionKey="dev_approval_ops_flow", 当前节点="开发提交"

# 4. 完成开发提交任务
curl -X POST http://localhost:8090/api/v1/workflow/tasks/<task_id>/complete \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"variables": {"submitted": true}}'

# 5. 检查下一节点
# 预期: 当前节点="主管审批"

# 6. 审批通过
curl -X POST http://localhost:8090/api/v1/workflow/tasks/<task_id>/complete \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"variables": {"approved": true}}'

# 7. 检查下一节点
# 预期: 当前节点="运维操作"

# 8. 完成运维操作
curl -X POST http://localhost:8090/api/v1/workflow/tasks/<task_id>/complete \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"variables": {"executed": true}}'

# 9. 检查流程状态
# 预期: currentStatus="completed"
```

## 技术细节

### BPMN XML 结构

```xml
<bpmn:process id="dev_approval_ops_flow" isExecutable="true">
  <!-- 开始事件 -->
  <bpmn:startEvent id="StartEvent_1" />
  
  <!-- 开发提交 -->
  <bpmn:userTask id="Activity_DevSubmit" name="开发提交">
    <bpmn:extensionElements>
      <bpmn:metaData name="assignee_type">developer</bpmn:metaData>
      <bpmn:metaData name="timeout_hours">24</bpmn:metaData>
    </bpmn:extensionElements>
  </bpmn:userTask>
  
  <!-- 主管审批 -->
  <bpmn:userTask id="Activity_ManagerApproval" name="主管审批">
    <bpmn:extensionElements>
      <bpmn:metaData name="assignee_type">manager</bpmn:metaData>
      <bpmn:metaData name="approval_required">true</bpmn:metaData>
    </bpmn:extensionElements>
  </bpmn:userTask>
  
  <!-- 审批结果网关 -->
  <bpmn:exclusiveGateway id="Gateway_ApprovalResult" />
  
  <!-- 运维操作 -->
  <bpmn:userTask id="Activity_OpsExecute" name="运维操作">
    <bpmn:extensionElements>
      <bpmn:metaData name="assignee_type">ops_engineer</bpmn:metaData>
    </bpmn:extensionElements>
  </bpmn:userTask>
  
  <!-- 连线与条件 -->
  <bpmn:sequenceFlow sourceRef="Gateway_ApprovalResult" targetRef="Activity_OpsExecute">
    <bpmn:conditionExpression>${variables['approved'] == true}</bpmn:conditionExpression>
  </bpmn:sequenceFlow>
</bpmn:process>
```

### 流程解析优先级

`process_resolver.go` 3-tier resolution:

1. **显式绑定**: `ticket_type.workflow_definition_key = "dev_approval_ops_flow"` (最高优先级)
2. **ProcessBinding 路由**: `ProcessBinding(business_type="dev_ops_request")` → `dev_approval_ops_flow`
3. **默认流程**: `ticket_general_flow` (兜底)

### 自动部署机制

```
服务启动
  ↓
bpmn_template_service.LoadAndDeployTemplates()
  ↓
go:embed 读取 bpmn/*.bpmn (包含 dev_approval_ops_flow.bpmn)
  ↓
listTemplates() 遍历并设置元数据
  ↓
对每个模板:
  ├─ isTemplateDeployed(key, tenantID) 检查是否已部署
  ├─ 如未部署: deployTemplate() 创建 ProcessDefinition
  └─ 记录部署日志
```

## 与现有系统的集成

### 1. 与审批链的关系

- **审批链** (`approval_chains` 表): 定义静态审批人序列
- **BPMN 工作流**: 定义动态任务流转和网关条件
- **本流程**: 使用 BPMN 工作流，审批通过网关条件表达式控制

### 2. 与 SLA 的关系

- 工单类型启用 SLA 后，每个节点可配置独立的 SLA 策略
- 节点超时 (timeout_hours) 与 SLA 响应时间独立
- 建议为"主管审批"节点配置高优先级 SLA (8h 超时)

### 3. 与通知的关系

- 每个节点完成时自动触发通知（如已配置 NotificationConfig）
- 审批驳回时通知开发人员
- 审批通过时通知运维团队

## 后续优化建议

### 短期优化

1. **添加审批意见字段**: 在主管审批节点添加 `approval_comment` 变量
2. **添加抄送机制**: 审批通过/驳回时抄送给运维主管
3. **添加自动升级**: 超时未审批自动升级到更高层级

### 中期优化

1. **添加并行网关**: 支持多主管并行审批
2. **添加子流程**: 运维操作拆分为多个子任务
3. **添加定时器事件**: 定期提醒未处理的任务

### 长期优化

1. **添加 AI 辅助**: 自动分类操作类型和风险等级
2. **添加知识库集成**: 相似操作自动推荐解决方案
3. **添加自动化执行**: 低风险操作自动执行（如服务重启）

## 参考文档

- **BPMN 表达式语法**: `BPMN Expression Syntax Convention` (memory)
- **审批架构设计**: `P0-1 BPMN-Approval Integration Strategy` (memory)
- **流程绑定机制**: `process_resolver.go:3-tier resolution`
- **工单类型 API**: `dto/ticket_type_dto.go:CreateTicketTypeRequest`

## 总结

✅ **已完成**:
- BPMN 工作流定义（3 节点 + 1 网关 + 驳回循环）
- 模板元数据注册
- 流程绑定配置
- 后端编译验证

✅ **可使用**:
- 通过 API 创建工单类型并绑定
- 通过前端界面创建工单类型
- 流程自动部署和解析

✅ **符合规范**:
- 遵循 BPMN 2.0 标准
- 符合项目 camelCase 命名规范
- 符合企业级审批架构设计
- 通过编译和 lint 检查

**下一步**: 使用上述 API 或界面创建工单类型，开始端到端测试。
