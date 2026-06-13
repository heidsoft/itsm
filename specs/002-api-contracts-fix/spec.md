# Feature Specification: API Contracts Fix

**Feature Branch**: `002-api-contracts-fix`

**Created**: 2026-06-13

**Status**: Draft

**Input**: User description: "统一分页响应格式，补充 sla_policies 种子数据，修复 assets 接口分页"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 统一分页响应格式 (Priority: P1)

前端开发者在调用任意 API 列表接口时，使用一致的分页数据结构，无需针对每个接口写不同的解析逻辑。

**Why this priority**: 分页是最常用的 API 模式，格式不一致导致前端代码重复且易错。

**Independent Test**: 可以通过 curl 测试所有列表接口，对比响应字段是否完全一致。

**Acceptance Scenarios**:

1. **Given** 调用 `/api/v1/tickets?page=1&page_size=10`, **When** 返回列表, **Then** 响应包含 `page`, `pageSize`, `total`, `totalPages` 四个字段
2. **Given** 调用 `/api/v1/incidents?page=2&page_size=5`, **When** 返回列表, **Then** 响应格式与 tickets 完全一致（无冗余字段）
3. **Given** 调用 `/api/v1/problems`, **When** 返回列表, **Then** 响应包含 `totalPages`
4. **Given** 调用 `/api/v1/changes`, **When** 返回列表, **Then** 使用 `pageSize` 而非 `size`

---

### User Story 2 - 补充 SLA Policies 种子数据 (Priority: P2)

运维团队在配置 SLA 策略时，系统已有预设的 SLA 政策模板，无需从零配置。

**Why this priority**: SLA 是 ITSM 核心功能，空表导致 SLA 监控无法关联工单。

**Independent Test**: 可以通过查询 `/api/v1/sla-definitions` 和 `/api/v1/admin/sla-policies` 验证预设数据存在。

**Acceptance Scenarios**:

1. **Given** 系统初始状态, **When** seeder 运行时, **Then** `sla_policies` 表包含至少 3 条预设策略（P1/P2/P3 响应时间）
2. **Given** 新建工单, **When** 关联 SLA 策略, **Then** 可从下拉列表选择预设策略
3. **Given** SLA 策略存在, **When** 工单超时, **Then** 触发 SLA 告警记录

---

### User Story 3 - 修复 Assets 接口分页 (Priority: P1)

前端调用资产列表接口时，支持标准分页参数，响应数据量可控。

**Why this priority**: 资产数据量大时无分页会导致性能问题和内存溢出。

**Independent Test**: 可以通过 `curl /api/v1/assets?page=1&page_size=10` 验证响应包含分页字段。

**Acceptance Scenarios**:

1. **Given** 调用 `/api/v1/assets?page=1&page_size=10`, **When** 返回资产列表, **Then** 响应包含 `page`, `pageSize`, `total`, `totalPages` 四个字段
2. **Given** 传入非法分页参数, **When** 调用接口, **Then** 返回参数校验错误（400 或对应 code）

---

### Edge Cases

- 分页参数 page < 1 时：前端/后端均应做 min=1 校验
- pageSize > 100 时：应限制最大值为 100
- 空列表时：total=0, totalPages=0, 列表为空数组
- incidents 接口同时返回 incidents[] 和 items[] 冗余字段问题

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 所有列表 API 响应必须包含统一的分页字段：`page`（当前页）、`pageSize`（每页条数）、`total`（总条数）、`totalPages`（总页数）
- **FR-002**: API 响应中不得包含冗余的列表字段（如 incidents 同时返回 `incidents[]` 和 `items[]`）
- **FR-003**: `/api/v1/assets` 必须支持 `page` 和 `page_size` 查询参数
- **FR-004**: `sla_policies` 表必须包含预设种子数据，至少覆盖 P1/P2/P3 三个优先级
- **FR-005**: 分页参数必须做参数校验（page ≥ 1, pageSize ≤ 100）
- **FR-006**: `/api/v1/changes` 响应必须使用 `pageSize` 字段名（而非 `size`）

### Key Entities

- **SLA Policy**: id, name, description, priority, response_time_minutes, resolution_time_minutes, is_active, tenant_id
- **Ticket**: id, title, status, priority, sla_policy_id（关联 SLA）
- **PaginatedListResponse**: page, pageSize, total, totalPages

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 前端可使用统一的分页组件，无需针对每个 API 做特殊处理
- **SC-002**: 所有列表接口响应结构一致，字段差异为零
- **SC-003**: seeder 运行后 sla_policies 表至少 3 条记录
- **SC-004**: `/api/v1/assets` 支持标准分页参数，端到端验证通过

## Assumptions

- 现有 API 返回的数据内容正确，仅需修复响应格式
- seeder 已有完整的 SLA definitions（6条），仅需补充 policies
- 前端使用 Ant Design Table 组件，依赖 page/pageSize/total 字段
