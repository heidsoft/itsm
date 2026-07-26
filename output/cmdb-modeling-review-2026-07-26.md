# CMDB 建模专项评审（对比 BlueKing / ServiceNow）

- **日期**：2026-07-26
- **范围**：`itsm-backend` CMDB 数据模型（`ent/schema`）+ `itsm-frontend` CMDB 建模 UI
- **核心问题**：是否落入「一类资源一张表」反模式？模型驱动程度如何？企业级可扩展性短板在哪？
- **证据可信度**：✅ 本仓库结论已读源码并通过回归测试核实；⚠️ 竞品物理存储实现仅采用官方公开资料可证实的部分，不把推测当作设计依据。

---

## 0. 复核与整改状态（2026-07-26）

本次复核确认：当前是通用 CI 主表 + 类型元数据 + 通用关系边，不是手写的“一类资源一张业务表”。已完成：

- ✅ `ci_type` 增加租户内类型名称唯一约束、`parent_type_id` 自引用层级和父子边。
- ✅ 服务层校验父类型必须属于同一租户，拒绝自继承和循环继承。
- ✅ 属性查询按“根类型 → 子类型”合并；子类型同名属性覆盖父类型属性。
- ✅ `ci_attribute_definition` 落库补齐 `description / enum_values / reference_type / display_order / group_name / placeholder / help_text / is_searchable / is_system`，修复 DTO 声明能力与持久层实际能力不一致。
- ✅ 属性定义增加 `(tenant_id, ci_type_id, name)` 唯一约束，并按显示顺序查询。
- ✅ `configuration_item.serial_number` 从全局唯一修正为租户内唯一。
- ✅ 自动迁移前增加重复类型名、重复属性名、孤儿属性定义和租户内重复序列号预检，并显式移除旧的全局序列号唯一索引；发现脏数据时失败关闭。
- ✅ 前端建模页支持选择继承类型和显式解除继承。
- ✅ CI 创建/编辑页复用父到子的有效属性模板合并，并完整分页加载类型树；旧 CMDB 写入链路按根到子确定性覆盖并校验父类型属性定义。
- ✅ 历史模板含编辑器不支持的字段时禁止覆盖保存，避免静默降级和元数据丢失。
- ✅ 删除类型前检查关联 CI、子类型和属性定义，不再把外键错误伪装成内部错误。
- ✅ 增加继承合并、父类型必填校验、跨租户父类型拒绝、循环继承拒绝的回归测试。

尚未完成且不得误报为已解决：

- ⏳ JSON 动态属性仍未形成数据库可查询的类型化索引；`is_searchable` 目前只是模型元数据。
- ⏳ 云资源/资产仍需通过统一 CI 身份与来源映射治理，当前双实体同步风险尚存。
- ⏳ 关系合法组合、方向和基数约束尚未模型化。
- ⏳ 父子同租户与无环约束目前由服务层保护；仍需版本化 PostgreSQL 迁移提供数据库最终约束/并发保护。

## 1. 结论先行（回答你的核心疑问）

**好消息：核心 CMDB 没有落入「一类资源一张表」的反模式。** 它走的是**模型驱动（model-driven）的通用 CI 模型**——单一 `configuration_item` 主表 + `ci_type`（类型定义）+ `ci_attribute_definition`（属性元数据）+ `ci_relationship`（关系边表）。新增一类资源**不需要新建数据库表**，只需在 `ci_type` + `ci_attribute_definition` 插数据。方向上与 BlueKing / ServiceNow 一致。

**但有三个会随企业规模放大、且直接决定「CMDB 是否真得好用」的架构短板**，恰好对应你说的「建模非常重要」：

1. 🔴 **动态属性用 JSON 大字段存储 → 数据库层无法按属性查询/索引/约束**（源码自证：service 第 1324 行「ent 不支持 JSON key/value 谓词，此处跳过，由调用方在内存中过滤」）。这是「避免分表」走到另一个极端：**查询能力丧失**。
2. ✅ **`ci_type` 原先没有继承层级**：本次已加入 `parent_type_id`、租户边界、循环检测和继承属性合并。
3. 🟠 **云资源 / 资产是独立孤岛表，未纳入 `ci_type` 体系**：拓扑与影响分析（依赖 `ci_relationship`）天然覆盖不到云资源，违背 CMDB「唯一可信源」承诺。

---

## 2. 已核实的数据模型（✅ 源码）

### 2.1 通用 CI 主表 `configuration_item`（`configurationitem.go`）
```
ci_type_id        Int   外键 → ci_type（通用建模核心，所有类型共用此表）
ci_type           String 冗余类型名（反规范化，与 ci_type_id 重复）
status / environment / criticality / source   通用枚举列（Optional）
asset_tag / serial_number / model / vendor / location   常用冗余列（均为 Optional）
attributes        JSON(map[string]interface{})   ★ 动态属性值（非固定列、非 EAV 行表）
cloud_provider / cloud_account_id / cloud_region / cloud_zone /
cloud_resource_id / cloud_resource_type / cloud_metadata / cloud_tags /
cloud_metrics / cloud_sync_time / cloud_sync_status / cloud_resource_ref_id   ★ 一大块云冗余列
indexes: cloud_provider / cloud_account_id / cloud_region / cloud_resource_id
edges: cloud_resource_ref → CloudResource（单向镜像）
```
**判定**：(b) 通用/抽象模型 + JSON 动态属性 + 稀疏常用列。**无任何 per-type 表**（专门 grep `Server/Switch/Database/Host/NetworkDevice` 仅命中 `Application` 一个非 CI 业务表）。

### 2.2 类型与属性元数据（模型驱动核心）
- `ci_type`（`citype.go`）：现包含 `parent_type_id`，支持同租户树状继承；类型名称在租户内唯一。
- `ci_attribute_definition`（`ciattributedefinition.go`）：现包含排序、分组、枚举、引用、输入提示、帮助文本和可检索标记；属性名称在同租户同类型内唯一。

### 2.3 关系 `ci_relationship`（`ci_relationship.go`）
```
relationship_type  String（depends_on/hosts/runs_on/connects_to/impacts/… 12 种，Go 常量）
source_ci_id / target_ci_id   Int  唯一复合索引防重
strength / impact_level       Enum(critical/high/medium/low)
is_discovered / is_active / metadata(JSON)
```
另有 `relationship_type` 表存关系类型元数据（name/directional/reverse_name/description）。**标准通用边表，无需 per-type 关系表。** ✅

### 2.4 云资源 / 资产孤岛（⚠️ 偏离模型驱动）
`cloud_account` / `cloud_service` / `cloud_resource` / `asset` 是**四张独立表**，不继承 `ci_type`：
- 自动发现时，`cloud_discovery_service.go` 同时 upsert 一条 `CloudResource` 与一条 `ConfigurationItem`，把云信息写进 CI 的 `cloud_*` 冗余列（`SetCloudResourceRefID(...)`）。
- 即：云资源是**独立实体**，仅通过 `cloud_resource_ref_id` 单向镜像回 CI，**未建模为某一种 `ci_type`**。前端 `/cmdb/cloud-*` 三个独立页面对应此数据模型。

### 2.5 服务层（`service/`）
`configuration_item_service.go`（45KB）**全程只操作单一 `ent.ConfigurationItem` 客户端**，无 `case "server"` 之类的类型分支；`CreateCI` 把 `req.Attributes` 整体写入 JSON。`CITypeService` / `CIAttributeDefinitionService` 同理。云资源走独立 `CloudDiscoveryService`。

---

## 3. 与 BlueKing / ServiceNow 的对比（你的参照系）

| 维度 | 本系统现状 | BlueKing CMDB | ServiceNow CMDB | 差距 |
|---|---|---|---|---|
| 类型建模 | `ci_type` 已支持树状继承、同名覆盖与循环检测 | 对象继承能力需按版本核验 | 类层级（cmdb_ci→cmdb_ci_hardware→…） | 🟡 尚缺基类 CI 查询与变更审计 |
| 属性存储 | **JSON 大字段**，当前服务未实现 DB 属性谓词 | 官方开源实现需进一步按版本核验，本文不再断言“每模型自动独立建表” | 官方文档确认支持类/表继承，并可采用 table-per-class 或 table-per-hierarchy | 🔴 属性检索能力不足 |
| 属性约束 | `required/unique` 仅应用层，`type` 仅声明 | 列级类型 + 索引 + 唯一约束 | 字典约束 + 索引 | 🟠 无 DB 约束 |
| 属性布局 | 后端元数据已补齐，前端独立属性编辑器待完善 | 支持能力需按版本核验 | 支持 Sections、变量、引用 | 🟡 UI 待完善 |
| 关系 | 通用边表 + 类型表（好） | 关联约束（合法组合校验） | CI Relation + 方向/基数 | 🟡 缺约束 |
| 云/资产 | **孤岛表**，未并入 CI | 云资源即 CI（cmdb_ci_cloud_*） | 云资源即 CI 类 | 🟠 割裂 |
| 唯一性/查重 | 仅 CI 级，属性级靠应用层 | 属性级唯一索引 | 字典唯一性 | 🟠 |

**关键澄清（很重要）**：应当避免的是“每增加一种 CI 就由开发者手写 Schema、迁移、Repository 和类型分支”的业务分表反模式，而不是把所有多表或继承表设计一概视为反模式。ServiceNow 官方资料确认其支持表继承，子类继承父字段，并存在 table-per-class 与 table-per-hierarchy 两种扩展模型。BlueKing 的具体物理建表策略必须按所对标版本的源码和迁移代码确认；缺少同等证据时不能写成既定事实。

---

## 4. 三大短板的后果（为什么「好用的 CMDB 不容易」）

### 4.1 🔴 JSON 属性不可查（最致命）
- 源码自证：`configuration_item_service.go:1324`「自定义属性过滤（ent 不支持 JSON key/value 谓词，此处跳过，由调用方在内存中过滤）」。
- 后果：
  - 无法在 SQL 层做「CPU>8 的服务器」「到期日在本月内的证书」这类属性过滤 → 全量拉回内存过滤，**百万级 CI 必超时/爆内存**。
  - 无法对 `unique` 属性建立数据库唯一约束 → 重复 IP、重复序列号只能应用层兜底，并发下仍可能脏写。
  - 无法按属性建索引 → 列表检索、拓扑展开慢。
  - 类型安全在 DB 层缺失 → JSON 里塞错类型（字符串当数字）难发现。
- 这正是企业级 CMDB 的命门：蓝鲸/ServiceNow 之所以能撑住，靠的就是**属性类型化 + 可索引**。

### 4.2 🟠 无模型继承
- `citype.go` 无 `parent_type_id`。后果：
  - 「物理服务器 / 虚拟机 / 容器」想共享 IP、OS、CPU 等公共属性，必须每个类型各写一遍 → 属性漂移、含义不一致。
  - 改一个公共属性要改 N 个类型。
  - 无法做「按基类查询」（如「所有计算设备」）。

### 4.3 🟠 云/资产孤岛
- 云资源不入 `ci_type` → `/cmdb/topology`、影响分析、根因定位天然缺云侧节点；同一台云主机在 `cloud_resource` 和 `configuration_item` 各存一份，靠冗余列同步，**双写不一致风险**。
- 资产同理。与「CMDB 是唯一可信源」理念冲突。

### 4.4 🟠 属性定义信息不足（影响建模 UI 好用程度）
- `ci_attribute_definition` 缺 `order/group/options/placeholder/help_text` → 前端建模表单只能靠 `attribute_schema`(JSON Schema 文本) 自行约定，且属性展示顺序、分组、枚举下拉、录入提示都无 DB 级支撑，体验弱、易出错。

---

## 5. 修复路线图（按性价比）

### P0（先止血：让 CMDB 能撑住企业规模）
1. **属性可查询化**——先做 ADR/压测再选型，不直接引入无治理的双写：
   - PostgreSQL 单一部署基线：优先评估 JSONB + GIN/表达式索引，并对“热属性”维护受控索引目录。
   - 需要跨数据库或强类型范围查询：评估类型化属性索引表 `ci_attribute_value`。该表只能是 CI JSON 的检索投影，必须具备同事务写入、存量回填、重建、校验和漂移监控，不能成为第二事实源。
   - 不采用运行时为每个 CI 类型自动建物理表；这会扩大迁移、锁表、权限和租户运维面。
2. **在 DB 层补属性级约束**：`required/type/enum/reference` 在服务写入入口统一校验；唯一性由规范化值和数据库唯一键保证。仅做“先查再写”的应用层唯一校验存在并发竞态，不算完成。

### P1（建模能力对齐 BlueKing/ServiceNow）
3. **`ci_type` 加 `parent_type_id`，支持模型继承**：✅ 后端层级、租户/循环校验、属性合并和建模页父类型选择已完成。后续仍需补“按基类查询全部后代 CI”与模型变更审计。
4. **`ci_attribute_definition` 增加布局和输入元数据**：✅ Schema/DTO/Mapper/排序已完成；前端独立属性定义编辑器和拖拽排序仍待接入。
5. **云资源 / 资产并入 `ci_type` 体系**（作为一类 CI，或加 `is_cloud` 标记 + 保留 `cloud_resource` 作为同步明细），让拓扑/影响分析覆盖云侧，消除双写。

### P2（打磨）
6. 关系加「合法组合约束」（A 类型能否关联 B 类型、方向/基数），对齐 BlueKing 关联约束。
7. `ci_type` 的冗余 `ci_type` 字符串列与 `ci_type_id` 二选一，消除不一致。
8. 前端 `cmdb/ci-types` / `admin/cmdb-types` 建模向导：类型继承选择、属性分组预览、关系约束配置的可视化。

---

## 6. 一句话总结
**方向对（模型驱动、非 per-type 表），但落在了「单表 + JSON」这一极，导致属性不可查询——这是企业级 CMDB 的硬伤。** 补上「类型化属性表 + 索引 + 约束」「模型继承」「云/资产并入 CI 体系」三件事，就能从「能建模」走向「好用、可扩展」，真正接近 BlueKing / ServiceNow 的成熟度。
