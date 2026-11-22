# Skill InputSchema 执行前校验

## 问题

`dto.SkillManifest.InputSchema` 是 manifest 的字段即声明：注册时接受、入库时持久化
（`skill_definitions.input_schema`）、发现面（`ListEntries` / 详情）原样暴露。但
`handler.Invoke` 从 `Validate` 到 `InvokeWithMetrics` 的整条执行链从不读它。后果是
技能作者写下 `required: ["query"]`、`limit: {type: "integer"}`，调用方漏传或传错类型
照样执行——声明是装饰，只有写技能的人以为它有用。

这与已修复的 `RequiredPermissions`、connector `required_permissions`、skill `owners`
属同一范式：**声明若只被列举/校验消费而执行入口不消费，就可以被绕过**。

## 消费点

`handlers/skill/handler.go` 的 `Invoke`，放在 `GetEntryScoped` 成功之后、身份字段注入
之前：

1. `entry.Manifest.InputSchema` 已在 scoped 查找时取到，无需加列、无需迁移。
2. 校验用**原始入参**：身份字段 `tenantId/userId/role` 由认证上下文在本校验之后
   无条件注入。若顺序颠倒，调用方不传 `tenantId` 却声明了 `required:["tenantId"]`
   的 schema 会被注入值掩盖而假通过；额外的身份字段也可能触发未预期的约束。
3. 失败走 `dto.ErrSkillValidation` → `failInvoke` 映射 400，与 `Skill.Validate`
   的既有错误口径一致；同时落 `ok=false` + `errorClass=validation` 失败底账，
   审计不因拒绝而缺席。

## 支持的 JSON Schema 子集

不引入完整 JSON Schema 依赖（go.mod 无此类库，加依赖的成本远大于本缺口），实现
覆盖最常见的四个关键字：

| 关键字 | 语义 | 边界 |
|:--|:--|:--|
| `type` | object/array/string/number/integer/boolean/null，支持字符串数组 union | 未知 type 不静默通过 |
| `required` | object 顶层必填字段名 | 字段存在但值为 null 按缺失处理 |
| `properties` | 逐字段递归校验，可嵌套 | 非必填字段缺席即跳过 |
| `enum` | 取值枚举，JSON 往返归一化数字类型后比较 | 合法枚举数组外的取值拒绝 |

`schema == nil` / 空 object → 放行。多数内置技能 `BuildManifest` 未填 InputSchema，
放行保持向后兼容；`additionalProperties` 等高级关键字暂不实现，遇到时按「声明了
但无法识别」处理：**本实现忽略未列出的关键字，不因此拒绝**——这是与小子集共存的
务实边界，不把它伪装成完整 JSON Schema 实现。

`schema` 非空但**无法解析为 JSON object**（标量、坏 RawMessage）→ fail-closed：
声明存在却读不懂时静默放行会让校验形同虚设，与策略信封「读不出即拒绝」同口径。

## 为什么不是第二个判定点

InputSchema 校验的是**调用方入参形状**，与 Gate 2（角色 RBAC）、Gate 2.5（主体
策略）、市场交付权益、租户归属是正交维度：前者回答「参数写对了吗」，后者回答
「这个身份能做这件事吗」。它不参与执行授权，也不能放行任何工具调用——所有授权
判定仍集中在 `evaluateToolCall`。把它放执行前只是让声明生效，不是新增一条授权路径。

## 回归锁

新增 `handlers/skill/input_schema_test.go`，全部打真实路由并断言技能执行计数
（不只看状态码）：

- `required` 缺失 → 400 且 `calls=0`
- `type` 不符 → 400 且 `calls=0`
- `enum` 不符 → 400 且 `calls=0`
- 合法入参 → 200 且 `calls=1`（防止改成一律拒绝）
- 空 schema → 200 且 `calls=1`（向后兼容）
- **顺序锁**：`required:["tenantId"]` 而调用方不传 → 400 且 `calls=0`。若校验被移到
  身份注入之后，注入的 `tenantId` 会让它假通过，这一例专门锁住顺序。

## 反向验证

临时把 `handler.Invoke` 里的 `validateSkillInput(...)` 分支替换成恒不触发的空判断：
四例拒绝用例全部转红，实际返回 `200`（非法入参被放行），正例与空 schema 例仍绿。
恢复后 skill 全包 `go test ./handlers/skill/ -count=1` 与 `go vet` 通过。

## 边界与遗留

- 未做：`additionalProperties` / `minLength` / `pattern` / `minimum` 等完整 JSON Schema
  关键字；如将来有技能真正依赖这些，应引入成熟库并把本文件降级为兼容层，而不是
  在最小实现上层层加码。
- 未做：`OutputSchema` 执行后校验（属另一缺口，且输出 schema 的失败语义与输入不同）。
- 零 schema 变更、零迁移、零新增权限码。
