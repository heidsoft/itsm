# BPMN 流程抄送任务处理器

## 模块概述

`cc_handler.go` 是 BPMN 工作流引擎中的抄送（Carbon Copy）任务处理器，负责在工作流执行过程中自动添加工单抄送人并发送通知。

## 核心功能

### 1. 抄送人解析

支持多种抄送人来源：

| 类型 | 参数 | 说明 |
|------|------|------|
| `user` | `ccUserIds` | 直接指定用户ID列表 |
| `group` | `ccGroupIds` | 从用户组获取所有成员 |
| `role` | `ccRoleIds` | 从角色获取所有成员 |
| `variable` | `ccVariable` | 从流程变量动态获取 |

### 2. 变量类型支持

抄送人变量支持多种数据类型：

```go
// 数组类型
ccUsers := []interface{}{1, 2, 3}

// 逗号分隔字符串
ccUsers := "1,2,3"

// 单个整数
ccUsers := 1
```

### 3. 通知渠道

支持多渠道通知：

| 渠道 | 说明 |
|------|------|
| `in_app` | 应用内通知（默认） |
| `email` | 邮件通知 |
| `sms` | 短信通知 |
| `feishu` | 飞书通知 |
| `dingtalk` | 钉钉通知 |
| `wecom` | 企业微信通知 |
| `webhook` | Webhook 回调 |

## BPMN 配置示例

在 BPMN 流程中配置抄送服务任务：

```xml
<bpmn:serviceTask id="Task_CCMembers" name="抄送相关人员">
  <bpmn:extensionElements>
    <camunda:inputOutput>
      <camunda:inputParameter name="ccType">group</camunda:inputParameter>
      <camunda:inputParameter name="ccGroupIds">${targetGroupId}</camunda:inputParameter>
      <camunda:inputParameter name="ccNotify">true</camunda:inputParameter>
      <camunda:inputParameter name="notifyChannels">in_app,email</camunda:inputParameter>
    </camunda:inputOutput>
  </bpmn:extensionElements>
</bpmn:serviceTask>
```

## 输入参数

| 参数名 | 类型 | 必填 | 说明 |
|-------|------|------|------|
| `ticket_id` | int | ✅ | 工单ID |
| `ccType` | string | ✅ | 抄送类型：user/group/role/variable |
| `ccUserIds` | string | 否 | 用户ID列表（逗号分隔） |
| `ccGroupIds` | string | 否 | 用户组ID列表 |
| `ccRoleIds` | string | 否 | 角色ID列表 |
| `ccVariable` | string | 否 | 动态变量名 |
| `ccNotify` | bool | 否 | 是否发送通知（默认true） |
| `notifyChannels` | string | 否 | 通知渠道（逗号分隔） |
| `addedBy` | int | 否 | 添加人用户ID |
| `tenant_id` | int | ✅ | 租户ID |

## 输出变量

| 变量名 | 类型 | 说明 |
|-------|------|------|
| `added_cc_users` | []int | 成功添加的抄送人ID列表 |

## 使用示例

### 1. 在服务中注册处理器

```go
ccHandler := bpmn.NewCCTaskHandler(client, logger)
processEngine.RegisterServiceTaskHandler(ccHandler)
```

### 2. 触发抄送任务

```go
// 通过流程变量触发
variables := map[string]interface{}{
    "ticket_id":      123,
    "ccType":         "user",
    "ccUserIds":      "5,6,7",
    "ccNotify":       true,
    "notifyChannels": "in_app",
    "addedBy":        1,
    "tenant_id":      1,
}

result, err := processEngine.ExecuteServiceTask(ctx, task, variables)
```

## 注意事项

1. **幂等性**：重复添加同一抄送人不会创建重复记录
2. **权限隔离**：查询用户组成员时自动按租户隔离
3. **日志记录**：所有操作都有详细的日志记录，便于审计
4. **错误处理**：部分抄送人添加失败不影响其他抄送人

## 相关文件

- `service/bpmn_types.go` - BPMN 任务处理器接口定义
- `service/bpmn_process_engine.go` - 流程引擎核心
- `ent/schema/ticketcc.go` - 抄送记录数据模型
