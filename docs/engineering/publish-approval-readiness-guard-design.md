# 发布期审批组就绪守卫设计（CloudMesh Product Hardening V3.0 · 第 2 轮）

## 1. 本轮范围与验收标准

本轮只做一件可独立验收的事：**流程定义发布/激活时，若某个显式声明的候选审批组解析不出任何审批人，阻断该次发布并返回可定位的中文诊断**。

验收标准：

1. 空审批组（组存在但零成员，且无同名角色可回退）被 `candidateGroups` 引用 → 发布返回 409，文案含节点 ID、组名、原因。
2. 同名角色可回退且该角色有活跃用户 → 放行（与运行时 `ExpandGroupsToUsers` 口径一致）。
3. 组存在且有成员 → 放行。
4. 跨租户隔离：A 租户的空组不阻断 B 租户同名组有成员的流程发布。
5. 坏 XML → 失败关闭，拒绝发布（不因解析失败而放行）。
6. 非 HTTP 路径（直接调用服务层 `PublishProcessDefinition` / `SetProcessDefinitionActive`）同样被阻断——守卫在服务端，不在 handler。
7. 反向验证：中和守卫后，上述阻断用例必须变红。

刻意不做（避免假装修好）：

- 不自动创建用户、不自动授予管理员角色、不自动通过审批来「填」空审批组。
- 不阻断 `assignee` / `candidateUsers` 已显式指定审批人的节点；不阻断 `approvalLevel > 0` 的变量驱动审批节点（审批人在运行时由 `approval_plan` 决定，发布期无法静态判定）。
- 不检查未显式声明 `candidateGroups` 的节点（其分配语义可能来自脚本/动态变量，静态判定会误伤）。
- 不覆盖审批链 `chain[].role = "group:<name>"` 的发布期诊断（审批链无独立「发布」入口，留作后续轮次）。`handlers/group` 侧对审批链引用的**变更期**阻断已由第 1 轮 PR #147 覆盖。
- 不修改运行时 `ExpandGroupsToUsers` 的既有语义（该解析器对「组存在但成员全被禁用」仍会解析出用户 ID，属既有运行时口径，本轮不引入与其不一致的诊断判定；该点作为遗留单列）。

## 2. 根因

第 1 轮（PR #147）堵住了「改名/删除被引用审批组 → 审批人静默清零」，但只覆盖**变更期被动阻断**。存量流程与另一条路径仍有缺口：

- 组成员可被单独移除（不触发组级改名/删除守卫），此时流程引用仍有效，但 `ExpandGroupsToUsers` 展开结果为空；
- 流程照常发布、任务照常创建，候选审批人恒空，任务永远停在待审批，且无任何报错或诊断——这是审批绕过/业务死锁，用户侧表现为「流程发得出去、任务没人能批」。

第 1 轮的 `findGroupReferences` 是「组名 → 引用位置」的正向查询，无法回答「这个待发布流程的审批人是否为空」这一反向问题，因此本轮新增反向诊断，而非复用它。

## 3. 运行时单一真源（判定口径）

`service/bpmn/bpmn_group_resolver.go` 的 `ExpandGroupsToUsers` 是运行时唯一审批人解析入口，诊断必须逐条对齐，否则会误报（把可审批流程拦下）或漏报（放行实际空审批人流程）：

| 场景 | 运行时行为 | 诊断判定 |
|:--|:--|:--|
| `candidateGroups` 命中本租户同名组，组有成员 | 取 `Edges.Members` 展开 | 放行 |
| 组存在但零成员 | 不回退角色（`found[name]=true` 直接跳过角色分支），返回空 | 阻断 |
| 组不存在，但本租户存在同名 `role.code`，且有活跃用户 | 角色回退：M2M 边 ∪ 主角色枚举，`user.ActiveEQ(true)` | 放行 |
| 组不存在，且无同名角色 / 角色无活跃用户 | 容忍返回空（运行时不报错） | 阻断 |

补充口径：

- CSV 分割统一为「逗号分隔 + 逐项 `TrimSpace` + 丢弃空项」，与运行时 `splitCSV` 一致；`candidateGroups` 采用**整 token 相等**匹配，子串不误命中（`approvers-l` 不等于 `approvers-l1`）。
- 查询严格限定当前租户（`group.TenantID(tenantID)`、`role.TenantID(tenantID)`、`user.TenantID(tenantID)`）。
- 角色回退的活跃用户判定复刻 `resolveRoleUsers`：`user.Or(HasRolesWith(role.ID), RoleEQ(user.Role(code)))` + `user.ActiveEQ(true)`。
- 任一流程定义 XML 解析失败 → 返回 error，调用方拒绝发布（失败关闭）。

## 4. 落点、变更面与分层约束

- **生产代码**：`internal/services/bpmn_process_definition_service.go`（已有文件）。发布逻辑本身在此，服务持有 `client`，能覆盖所有调用路径（HTTP handler、内部服务调用、后续新增入口）。
  - L.1 棘轮锁 `internal/services/` **顶层生产文件**文件名（基线 125，新增即 FAIL），因此**不新增生产文件**；新增 `*_test.go` 不被 L.1 阻断。
  - L.3 棘轮要求 `handlers/<domain>/` 生产文件对 `itsm-backend/service` / `internal/services` 的依赖恒为 0。本方案不新增任何 handler → shared 依赖，不触碰该棘轮。
  - 与并行的审批 SOD worktree 隔离：该会话改的是 `internal/services/bpmn_task_service.go` / `ticket_workflow_service.go`，本轮只改 `bpmn_process_definition_service.go`，文件名不重叠。
- **XML 类型**：复用纯类型包 `internal/bpmncontracts.BPMNDefinitions`（只依赖 `encoding/xml`），与运行时同一份 tag 定义，不另立结构。`internal/bpmncontracts` 与 `internal/services` 无循环依赖。
- **注入点**：
  - `PublishProcessDefinition`：`validateActivatableBPMNXML` 与 `validateCandidateDefinition` 之后、开事务之前（拒绝时零写入）。
  - `SetProcessDefinitionActive`：`active=true` 分支内，`validateActivatableBPMNXML` 之后、写库之前。
- **错误语义**：`common.NewAppError(common.ErrCodeConflict, "...", http.StatusConflict, nil)`，与同文件删除流程定义时的 409 口径一致。`common.RespondError` 会透传 `AppError.Message`，第 1 轮已修 `handlers/group` 的诊断吞文案问题，发布路径本就使用 `RespondError`，无需改前端。

## 5. 诊断文案示例

```
发布前审批人就绪校验失败：节点 [Activity_Approve] 候选组 [approvers-l1] 解析不到任何审批人
（组存在但无成员，且无同名角色可回退）；请为该组添加成员，或改用有成员的角色/组后再发布。
```

多问题用 `；` 连接并去重，保证一次发布能看到全部待修节点。

## 6. 兼容性与回滚

- 零 schema / 零迁移 / 零新权限码 / 零新路由；纯服务端校验，回滚 = 回退该提交即可，静态数据不受影响。
- 内置模板候选组 `managers`、`it_director` 走角色回退分支；种子用户为活跃用户时放行，不会阻断内置流程发布。
- 现存已发布流程不被动扫描（本轮只守发布/激活入口），避免一次性阻断历史数据；存量空审批人的发现和修复属后续轮次。

## 7. 证据与反证要求

- 正向用例覆盖第 1 节 1–6 条，使用内存 SQLite（`file:...mode=memory&cache=shared&_fk=1`）真实 ent 查询，禁止手抄判定逻辑副本。
- 反证：把守卫调用注掉后，阻断类用例必须红；把「组不存在→角色回退」分支注掉后，角色回退放行用例必须红。两轮反证红点层级不同，证明用例能区分两条路径。
- 全量后端 `go test ./...`、`internal/layering`、`go vet`、gofumpt、docs-gate；前端 `tsc --noEmit`（本轮无前端改动，作为回归基线）。
