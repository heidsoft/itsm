# Agent 身份实体：设计先行

> 结论先行：**本阶段只做设计，不实施**。把 agent 从「请求级 context 值」
> 提升为 First-Class 主体，会连带改动五处已收口的口径。这五处必须一次性
> 对齐，否则会出现「审计里的 agent 与鉴权里的 agent 不是同一个东西」。
>
> 对应《Agent 控制面差距报告》原计划 P2。日期 2026-11-21，基线 `HEAD=ac01a2c8b`。

---

## 1. 现状：agent 是什么，不是什么

### 1.1 它是一个请求级身份标记

```go
// internal/authz/policy.go
type principalContextKey struct{}

// WithPrincipal 标记本次操作的主体类型。
func WithPrincipal(ctx context.Context, principal string) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

const (
	PrincipalHuman   = "human"
	PrincipalAgent   = "agent"
	PrincipalService = "service"
)
```

`agent` 是**三个字面量之一**，通过 `authz.WithPrincipal` 塞进 `ctx`，由
`ChatStream` / Skill 执行路径在调用工具前显式标记。设计注释已经写明了
它的边界：

> 刻意不放在 JWT claim 里：同一位登录用户既可能自己点按钮、也可能让 AI
> 代做，主体类型描述的是**这一次动作的发起者**，不是账号的类型。

这个设计是对的，且是必要的——它让「同一个人」的两种行为走两条不同的
鉴权链路（human 走 RBAC，agent 额外走 Gate 2.5 零信任策略）。

### 1.2 它不是一个实体

**agent 没有 ID，没有名字，没有版本，没有所属租户，没有独立的凭据。**
`tool_invocations` 表里能查到的只有：

| 列 | 记录了什么 | 缺失了什么 |
|:--|:--|:--|
| `user_id` | **发起这次调用的人的 ID** | agent 自己的 ID |
| `role_snapshot` | **发起时的载体角色**（人的角色） | agent 自己的角色/能力集 |
| `principal`（隐含在 `status`/`approval_state` 上） | 「这次是 agent 发起的」 | 是**哪一个** agent |
| `run_id` | 逻辑轮次 | 轮次属于哪个 agent |
| `policy_version` | 判定输入指纹 | 判定时 agent 的配置状态 |

关键推论：**当前无法回答「是哪个 agent 做的」。** 只能回答「是一次 agent
发起的调用」。当系统里有多个 Agent（对话助手、调查编排器、SOP 生成器、
未来的 ActionFlow 执行者）时，审计无法区分它们。

### 1.3 这个缺失已经产生具体后果

1. **审计不可归因**：出问题时无法定位「调查编排器这条路径的 agent 是否
   一直在越权尝试」。所有 agent 的拒绝记录混在一起。
2. **无法按 agent 撤销授权**：发现某个 agent 行为异常时，只能撤销
   `permissions.description` 里 `allowedPrincipals` 的 `agent` 项——
   而这会同时废掉所有 agent 的能力。
3. **无法表达 agent 的差异化能力**：调查编排器只需要读工具，SOP 生成器
   需要写工具。当前两者共享同一个「agent 有/无某权限码」判定。
4. **`minApprovals` 无法按 agent 差异化**：双人审批现在按权限码配置，
   不能表达「这个 agent 的写操作要 2 人，那个只要 1 人」。

## 2. 为什么不能简单地「加一个 agent_id 列」

如果把 `agent_id` 加到 `tool_invocations`，会立刻遇到五个未对齐的口径：

### 口径 1：Gate 2 角色 RBAC 用的是「人的角色」

```go
// handlers/ai/service.go —— evaluateToolCall 内部
middleware.HasResourcePermission(ctx, s.entClient, role, resource, action, tenantID)
```

`role` 来自 `EngineerUser.Role`（人的角色），复核时来自
`inv.RoleSnapshot`（人的角色快照）。**Gate 2 判的是「这个人的角色有没有
这个权限」**，不是「这个 agent 有没有」。

如果引入 agent 实体，必须回答：Gate 2 应该按 agent 的能力集判、还是
仍按人的角色判？两者都可能：
- 按人的角色判 → agent 不能超出操作者的权限（**当前行为**，安全但受限）；
- 按 agent 能力集判 → agent 有自己的能力，与操作者解耦（灵活但需要
  新的「agent 能力」授权模型和一套新的审批资格判定）。

这是**产品决策**，不是技术实现。两者都合理，但不能含糊。

### 口径 2：Gate 2.5 主体策略只区分「agent 这个类」

```go
// internal/authz/policy.go —— PermissionPolicy
type PermissionPolicy struct {
	AllowedPrincipals []string // ["human"] / ["human", "agent"] / ...
	MinApprovals      int
	...
}
```

即使引用了 agent 实体，`AllowedPrincipals` 仍是**类级别**白名单。
要按具体 agent 授权，需要把策略从「主体类型」扩到「主体实例」——
这意味着 `policy_version` 的指纹输入变化，**所有历史 `policy_version`
值将不再可比**。而「相同指纹 ⟺ 相同判据」是本项目审计可回放的核心等式。

**这不是不能做，而是必须承认它是一次破坏性的口径升级**，需要版本号区分
「旧指纹算法」与「新指纹算法」。当前 `PolicyVersion()` 没有算法版本前导，
引入后必须加（否则历史指纹会被误判为「判据变了」）。

### 口径 3：审批资格判定按「人」

`DecideToolInvocation` 校验的是「审批人这个人的角色 + 不得是发起人」。
发起人取自 `user_id`。若 agent 成为独立主体，「不得自批」的语义要重新定义：
- agent 发起的调用，谁能批？（当前：任何有 `ai:approve` 的人，除了发起人）
- agent 自己能不能批自己？**绝对不能**（那就等于取消审批）。但代码里
  现在没有这条显式拒绝——因为 agent 没有身份，无法被当作审批人候选。
  引入实体后必须显式拒绝，否则一旦 agent 有了主体身份，它可能被当成
  合法审批人。

### 口径 4：运行轮次归属

`run_id` 是「跨请求的逻辑轮次」，由 `EnsureRunID` 在 `ChatStream` /
`InvestigateTicket` 入口注入。它**不属于任何 agent**——它是请求链路的属性。

若引入 agent 实体，`run_id` 与 `agent_id` 是两个正交维度（一个 agent 可以
跨多个 run，一个 run 内可能有多个 agent 协作）。**不能把它们合并**，
但需要一张关联表或至少在 `tool_invocations` 上同时落两列，否则
「这一轮是哪个 agent 主导的」仍不可答。

### 口径 5：审计字段的单一序列化函数

本批已经把 `tool_invocation` 的所有审计字段收敛进 `toolInvocationView`
（`handlers/ai/handler.go`）。新增 `agent_id`/`agent_name` 必须同时进这个
函数，否则会重演「详情页缺、列表页有」的字段级漂移——那比彻底缺更危险，
因为复核单次调用时看的正是详情页。

## 3. 建议的设计形态（若将来实施）

### 3.1 实体定义

```go
// ent/schema/agent.go（尚未创建，仅为设计）
type Agent struct {
	ent.Schema
}

func (Agent) Fields() []ent.Field {
	return []ent.Field{
		field.Int("id"),
		field.Int("tenant_id").Comment("agent 必须归属租户：跨租户的 agent 无法被治理"),
		field.String("name").Comment("稳定标识，用于审计归因；如 investigator / sop-writer"),
		field.String("display_name"),
		field.String("kind").Comment("chat | investigator | sop | actionflow —— 决定默认能力集"),
		field.Bool("active").Default(true),
		field.Time("created_at").Default(time.Now),
	}
}

func (Agent) Indexes() []ent.Index {
	return []ent.Index{
		// 必须用 Indexes() 而不是 Index()：写 Index() 会被 entc 静态忽略
		//（本仓已有 3 处此坑）。
		index.Fields("tenant_id", "name").Unique(),
	}
}
```

**`name` 而非自增 ID 作为审计标识**：审计里出现 `agent_id=7` 无法被人读懂，
而 `agent=investigator` 可以。ID 仍保留为主键，但审计优先展示 name。

**决策：不为每个 Agent 建独立的凭据/密钥。** 理由是凭据会引入第二套认证
链路，而当前 agent 的调用总是「在某个人的请求上下文里」发生的——
它是**代表某个人**在动，不是独立行动者。给 agent 独立凭据会让「agent
以谁的权限行动」变成另一个需要治理的问题，而当前设计（agent 继承操作者
角色 + 额外的 Gate 2.5 零信任）已经把风险控制住了。

### 3.2 与 `tool_invocations` 的关联

```go
// 新增两列（可为空，兼容存量行）
field.Int("agent_id").Optional().Nillable()
field.String("agent_name").Default("") // 审计用冗余，避免 join
```

`agent_name` 冗余存储而非 join：审计读面必须能在 agent 被重命名或删除后
仍还原当时的名称。这与 `role_snapshot` 的理由相同（快照而非外键）。

**存量行保持空值，不回填。** 理由：存量行确实不知道是哪个 agent（当时
没有这个概念），回填成 `investigator` 是**伪造审计事实**。

### 3.3 判定链的接入点（唯一一处）

```go
// evaluateToolCall 的签名扩展（设计，非实现）
func (s *Service) evaluateToolCall(
	ctx context.Context,
	tenantID, userID int,
	role, name string,
	principal string,
	agentID int,      // 新增：0 表示非 agent 主体
) (*toolDecision, error)
```

Gate 2.5 的策略查询从 `(resource, action, principal)` 变为
`(resource, action, principal, agentID)`。**只在策略查询这一层扩展**，
不在 Gate 2（人的角色 RBAC）扩展——除非产品决定采用「按 agent 能力集判」。

### 3.4 策略信封的版本升级（必须）

```json
{
  "v": 2,                        // 从 1 升到 2
  "policy": {
    "allowedPrincipals": ["human", "agent"],
    "allowedAgents": ["investigator"],   // 新增：实例级白名单，为空=不限制
    "minApprovals": 2,
    ...
  }
}
```

**`v=1` 必须继续可读**（`DecodeDefinition` 允许双版本），否则所有存量配置
会在升级瞬间 fail-closed 全拒。但 `v=1` 的行在 Gate 2.5 时**无法按 agent
实例过滤**——就是「所有 agent 都可以」。这必须被明确记录为已知降级，
并让控制面聚合读面能报出「有多少权限点仍是 v=1」（否则运维不知道
有多少策略还没升级）。

**`PolicyVersion()` 必须加算法版本前导**：

```go
const policyVersionScheme = "v2:"  // 新算法
// 旧指纹（无前导）继续被读取，但不与新指纹比较
```

否则历史指纹与新指纹会被误判为「判据已变」，`policyComparison` 的
`identical` 会大面积退化成 `changed`，让审计回放失去意义。

## 4. 实施顺序（若批准）

必须按此顺序，每步独立可验证：

1. **schema + 迁移**（`agents` 表 + `tool_invocations` 两列，均可空/可空值）。
   验证：存量行不受影响，`go test ./handlers/ai/` 全绿。
2. **`PolicyVersion()` 加前导 + 双算法测试**。验证：旧指纹读取不变、
   新指纹可比较、混合比较返回 `unknown` 而非 `changed`。
3. **`evaluateToolCall` 扩展 `agentID`**，Gate 2.5 查询带上它。
   验证：同一工具、同一 agent 类、不同 agent 实例得到不同结论（若策略配了
   `allowedAgents`）；未配时行为与当前完全一致（**这条最重要**，
   否则是破坏性变更）。
4. **`recordToolAudit` / `toolInvocationView` 落 `agent_id` + `agent_name`**。
   验证：详情页与列表页都透出（防字段级漂移）。
5. **审批拒绝「agent 自批」**。验证：构造 agent 身份调用 approve 端点，必须被拒。
6. **控制面聚合读面报出「v=1 策略点数」**。验证：升级进度可见。
7. **前端展示 agent 归因**（审批页的待办卡片显示「由 investigator 发起」）。

## 5. 本阶段不实施的理由（诚实版）

1. **它不是当前最短路径上的阻塞项。** 控制面聚合读面（本批已完成）让
   「策略改了生效没有」可答；`run_id`（前批）让「这一轮做了什么」可答。
   剩下的「是哪个 agent 做的」在多 Agent 协作真正铺开之前，收益有限。
2. **它是一次破坏性口径升级，不是增量能力。** `policy_version` 的指纹算法
   变化会影响所有历史审计记录的可比性。这种改动应该与「审计体系本身需要
   升级」一起做，而不是夹在功能开发里。
3. **它需要产品决策（Gate 2 按谁判）**，而该决策会影响安全模型的根本假设。
   在没有明确决策前实施，等于用实现代码替产品拍板。

## 6. 与 ActionFlow 的关系

若将来实施 ActionFlow（见 `actionflow-engine-assessment.md`），它是
**最需要 agent 实体的场景**——一次 flow 由某个 agent 发起、跨多个步骤、
需要按 agent 追溯。两者的依赖方向是：**ActionFlow 依赖 Agent 实体，
Agent 实体不依赖 ActionFlow**。因此若 ActionFlow 被批准，Agent 实体
必须先落地。

这个依赖关系是本阶段判断的另一依据：ActionFlow 暂缓，则 Agent 实体
也可以等；反之则不行。

## 7. 明确记录的不确定项

- `agent.kind` 是否应成为权限判定的输入（而非仅用于展示/默认能力集）？
  若成为输入，则策略需要按 kind 而非实例配置，会产生第三种白名单维度。
- agent 的**生命周期**（创建/停用）是否需要审批？停用一个 agent 意味着
  撤销它所有能力，是否需要与权限变更同级的审计与审批？
- 多 agent 协作时（一个 flow 内多个 agent），`tool_invocations` 上的
  `agent_id` 记「发起者」还是「主导者」？两者可能不同。
