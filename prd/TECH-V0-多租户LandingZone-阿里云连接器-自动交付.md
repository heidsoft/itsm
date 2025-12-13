# TECH-V0：多租户Landing Zone（每租户VPC）+ 阿里云连接器 + 自动交付（可控执行）

## 1. 总体架构（V0）

### 1.1 形态
- SaaS：公有云部署，多租户。
- 多云策略：前端可展示多云选项；V0仅实现 **阿里云（Aliyun）Provider** 的自动交付执行。

### 1.2 关键组件
- **Core ITSM API**：ServiceCatalog / ServiceRequest / Approval / Audit / Notification
- **AI服务**：Copilot建议（推荐/补全/摘要/风险提示）；Agent执行（工具调用 + 审批门禁）
- **Integration/Provisioning**：
  - Provider抽象：`CloudProvider`（V0实现 Aliyun）
  - 执行编排：`ProvisioningOrchestrator`（幂等、回滚、验证、回写）
- **数据层**：PostgreSQL（主数据）+ Redis（锁/限流/缓存）+ Object Storage（附件/日志归档）
- **异步任务**：队列（建议引入，至少用于：执行、验证、回收、通知）

---

## 2. 多租户隔离模型（V0）

### 2.1 应用层隔离
- 每条业务数据必须带 `tenant_id`（或 `tenant_code`），并在API层通过 TenantMiddleware 强制注入过滤条件。
- RBAC权限校验包含租户维度；审计日志同样带租户维度。

### 2.2 云侧隔离（Landing Zone）
**每租户一套隔离VPC（Landing Zone）**，作为所有云资源交付的默认落点。

---

## 3. Landing Zone 初始化（租户生命周期）

### 3.1 目标
为租户提供一致、可控、可审计的云落点，并使“安全审批”可聚焦业务与风险点，而不是基础网络隔离。

### 3.2 初始化内容（建议最小集）
- VPC（按Region；V0可先限定单Region或按租户配置）
- 交换机：
  - 私有子网（必选）
  - 公有子网（可选；公网入口仍需安全门禁）
- 安全组基线（必选）：
  - 入站默认拒绝
  - 公网仅允许 443（且必须源IP白名单）
  - 运维端口 22/3389：不对公网开放，必须走堡垒机
- 标签/命名规范（必选）：
  - `itsm:tenant_code`
  - `itsm:managed=true`
  - `itsm:landing_zone=true`

### 3.3 初始化触发方式
- 方式A（推荐）：**租户创建后异步初始化**，完成后将Landing Zone信息写回租户配置。
- 方式B：租户管理员手动绑定（用于已有VPC的客户，V1可做）。

---

## 4. 阿里云执行身份与凭证（安全关键）

### 4.1 执行身份
- 推荐：每租户绑定一个 **RAM角色（Execution Role）**，平台通过 **STS临时凭证** 执行。
- 禁止：长期AK硬编码在服务端配置。

### 4.2 权限最小化（V0）
执行角色权限需严格白名单化，限制：
- 仅允许指定Region
- 仅允许在租户VPC/子网内创建资源
- 仅允许规格白名单（2C4G/4C8G/8C16G）相关API调用
- 仅允许创建/绑定带 `itsm:managed=true` 标签的资源
- IAM权限：仅允许从模板创建/绑定（V0）

---

## 5. Provider抽象与连接器接口

### 5.1 Provider接口（示意）
- `ValidateRequest(ctx, req) -> validationResult`
- `Provision(ctx, req, plan) -> provisionResult`
- `Verify(ctx, provisionResult) -> verifyResult`
- `Rollback(ctx, provisionResult) -> rollbackResult`
- `Reclaim(ctx, resources) -> reclaimResult`（到期回收）

### 5.2 V0支持资源
- ECS（VM）
- RDS
- OSS
- VPC（按需：允许创建附属资源或仅使用Landing Zone，V0建议“仅使用Landing Zone”，避免VPC碎片化）
- RAM（IAM）：用户/角色/策略绑定（模板化）

---

## 6. 执行编排（Provisioning Orchestrator）

### 6.1 幂等策略（必须）
目标：同一请求重复触发不产生重复资源。
- 全局唯一：`service_request_id`（或 `request_id`）作为幂等键
- 执行记录表：`tool_invocations` / `provisioning_runs`（建议新增或复用现有ToolInvocation）
- 云侧幂等：
  - 资源创建统一打标签：`itsm:request_id=<id>`
  - 创建前先按标签/名称查询是否已存在

### 6.2 回滚策略（必须）
执行失败时：
- 优先自动回滚已创建资源（按创建顺序逆序删除/解绑）
- 回滚失败：
  - 标记请求为 `DeliveryFailed`
  - 创建“人工接管任务”
  - 发通知给履约工程师/租户管理员

### 6.3 验证策略（必须）
创建完成后进入 Verification：
- 资源状态可用（ECS Running、RDS Available等）
- 安全组规则符合策略：
  - 公网入站仅443
  - 源IP白名单已生效
  - 无22/3389公网入站
- IAM策略已绑定且带到期信息（至少记录在系统侧）

### 6.4 回写策略
回写到 ServiceRequest：
- 云资源ID（ECS实例ID、RDS实例ID、Bucket、VPC/SG等）
- 执行日志摘要（长日志落对象存储）
- 校验结果与证据
- 标签信息（tenant/request_id/cost_center/data_classification/expire_at）
并同步到：
- CMDB（V0可先写“最小CI”：资源ID/类型/Region/所属租户/关联请求）
- 审计日志（人/AI/工具执行全链路）

---

## 7. 策略护栏（Policy Guardrails）

### 7.1 规格白名单
- ECS/RDS 仅允许：2C4G / 4C8G / 8C16G

### 7.2 公网策略
- 允许公网IP，但：
  - 入站仅允许 443
  - 必须配置源IP白名单（CIDR列表）
  - 运维端口必须走阿里云堡垒机（不开放公网SSH/RDP）

### 7.3 数据分级与合规
根据数据分级触发强制控制项：
- 敏感/受监管：强制加密/审计/留存策略勾选（产品层门禁）

---

## 8. 到期回收（V0必须）

### 8.1 标签与数据
创建资源时必须写入标签：
- `itsm:tenant_code`
- `itsm:request_id`
- `itsm:expire_at`（ISO时间或时间戳）
- `itsm:cost_center`
- `itsm:data_classification`
- `itsm:owner`

### 8.2 回收作业
定时任务扫描：
- 到期资源 → 自动回收（释放EIP/删除实例/撤销权限等）
- 回收失败：
  - 生成接管任务
  - 通知相关人
  - 记录审计与失败原因

---

## 9. 堡垒机（阿里云）接入（V0策略）

### 9.1 目标
保证运维访问有统一入口与审计，并避免公网暴露运维端口。

### 9.2 V0实现建议
- V0以“策略与流程强制”为主：
  - 安全组禁止对公网开放22/3389
  - 请求单展示“运维访问必须通过阿里云堡垒机”的控制项
- 若租户已开通阿里云堡垒机：
  - 在租户配置中录入堡垒机实例/接入方式（最小字段）
  - 在交付回写中提供“运维访问指引”

---

## 10. 可观测与审计（企业级必备）

### 10.1 Trace/Log
- 每次执行生成 `run_id`，贯穿：审批→执行→验证→回滚→回收
- 执行日志分级：摘要入库，详细日志落对象存储并可追溯

### 10.2 审计字段（必须）
- who：操作者（人/系统/AI），租户，角色
- what：执行的工具、参数摘要、资源变更
- when：时间线
- result：成功/失败/回滚/回收

---

## 11. 安全与风险控制（V0上线门槛）
- 执行身份最小权限 + STS临时凭证
- 公网仅443 + 源IP白名单强制 + 运维端口不暴露
- 幂等/回滚/验证/审计/回收必须可证明（演示用例+自动化测试后续补齐）


