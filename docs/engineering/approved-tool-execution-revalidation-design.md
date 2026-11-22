# 已批准 AI 工具执行前重判

## 问题与生产调用链

审批通过只代表审批时允许，不是永久授权。原执行器直接执行工具，使审批后角色撤权、主体策略改变或工具移除无法阻止副作用。

生产链路：operational_commands 的 ai.tool.execute → ToolExecutor.HandleCommand → ExecuteApprovedInvocation → ApprovedToolGuard.RevalidateApprovedTool → handlers/ai.Service.evaluateToolCall → ToolRegistry.Execute。bootstrap 在 AI Service 完成 Ent client 和权限目录装配后注入 guard。执行器只消费判定，不复制 Gate 2 RBAC 或 Gate 2.5 主体策略。

## 零迁移边界

不新增列，不修改 Ent schema、生成代码或迁移。使用已持久化的 tenant_id、user_id、role_snapshot、policy_version 重判。审批后的 approval_state 已统一为 approved，不能恢复原主体；不从审批人 context 或 approval_state 推断主体。

policy_version 记录当时的判定输入指纹：空串表示当时无主体策略参与，对应 human；任何非空值（包括 unknown）对应 agent。unknown 表示无法读取策略，必须 fail closed。此规则是当前已批准调用的恢复契约，不新增 service 主体恢复机制。

先显式选择 principal，调用唯一判定源 evaluateToolCall；仅 d.Allowed 且当前指纹与落库指纹完全一致、当前指纹不是 unknown 时放行。工具缺失、Resource/Action 漏配、角色撤权、策略改变、读取错误或 guard 未注入均拒绝。判定使用现有权限目录及其缓存失效协议，不引入另一套策略读取机制。

## 执行与审计

先按 invocation ID + command tenant 加载并条件领取；领取成功后、解析 arguments 前重判。拒绝通过 finalize 在同一次租户限定 UPDATE 写入 status=failed、error、permission_check=denied 和 permission_reason；不调用工具。guard 错误使用稳定安全描述，避免原始错误写入执行审计。

已完成调用仍幂等返回；跨租户调用不能领取。保留现有 command 重试与显式重放机制。本修复不将策略判断与远端工具副作用做成原子事务，也不扩展 dry_run 语义或审批人数模型。

## 验证

internal/services/tool_executor_test.go 经真实 Worker handler、真实 AI Service 和数据库权限目录重判，以 counter spy 观察副作用边界。覆盖角色授权后撤销、策略指纹变更、unknown、正常审批且仅执行一次、跨租户、nil guard、guard 返回错误和工具移除。原有空指纹 human 调用测试保留。反向验证临时移除执行器的重判块，拒绝用例应在副作用计数 expected=0 / actual=1 上失败，然后还原实现。

本批实际反向验证：移除执行器重判块后，revoked_role、changed_policy、unknown_policy、unknown_unreadable、nil_guard、guard_error、missing_tool 七项均失败在 execution side effects 断言，expected: 0 / actual: 1（go test 退出 1）；正常执行与跨租户用例未失败。随后已恢复重判块，并在本会话独立重做一次反向验证得到同样七项红，还原后全包复测通过。

本批实际验证：go test ./internal/services/ -run ToolExecutor -count=1 通过（最终 0.955s）；go test ./handlers/ai/ -count=1 通过（4.594s）；go vet ./internal/services/、./handlers/ai/、./internal/bootstrap/ 均通过；gofmt 与 git diff --check 通过。反向测试退出码 1，六项拒绝场景均在 execution side effects 断言报 expected: 0 / actual: 1。make docs-gate 仍有既有 C.6 两项规模基线失败（shared_services_go_files、bootstrap_app_lines），本批不调整基线；未运行全量构建，未改 internal/layering。
