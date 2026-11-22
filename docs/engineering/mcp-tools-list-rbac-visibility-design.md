# MCP tools/list RBAC 可见性

MCP 生产入口为 router 注册的 MCP Handler → Service.Handle →
handleToolsList → ToolCatalog.ListToolsForTenant → ProjectTools →
EntitlementChecker → VisibilityChecker。bootstrap.BuildMCPHandler 是生产与
端到端测试共用的装配入口，生产 app 注入绑定 Ent client 的适配器。

AI GET /agent/tools 与 MCP 列举共用 middleware.HasResourcePermission，
最终委托 AuthorizeResourceForRole；MCP 不复制角色规则。检查使用认证 Caller
的角色与租户，以及工具定义的 Resource/Action。投影保留这两个内部字段，
但 JSON 不输出它们。顺序固定为投影、交付权益、RBAC；任一拒绝均隐藏名称及 schema。

列举过滤只减少发现面，不授予执行权限，不构成第二个执行判定点。
tools/call 和 ai.Service.ExecuteTool 保持原有执行链，Gate 2/2.5、
审批与审计仍负责执行授权。知道工具名称也不能绕过执行授权。

service 主体以 UserID=0、Role=audit_admin 列举。该角色具有 ticket:read、
cmdb:read 等全租户只读权限及 mcp:read，没有写权限，所以读工具仍可发现，
写工具被隐藏；可见不代表主体获准执行。

Visibility 未装配时返回非 nil 空工具列表，防止名称/schema 泄露。
这与 AI 缺少 Ent client 时整体 AuthFailed 的 fail-closed 原则一致，
但保留 MCP 列表响应协议。生产装配缺少 VisibilityChecker 立即报错；
适配器缺少 client 也拒绝。权益检查行为保持原样。

回归覆盖真实协议 dispatcher 的 human 允许/拒绝、缺件、service 只读、
权益叠加及执行链独立性；既有 HTTP e2e 继续通过共用装配入口验证路由。
