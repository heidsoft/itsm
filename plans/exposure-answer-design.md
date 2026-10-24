# 外部安全事件暴露面自答（Exposure Answer）设计

> **Status**: draft（设计蓝本，未进入实施）
> **Date**: 2026-10-22
> **Relation**: 是 [`ai-security-operations-blueprint.md`](./ai-security-operations-blueprint.md) 的**前置环节 S0**——漏洞型事件先做暴露面评估，再决定是否开安全事件。
> **One-liner**: 事件发生后，企业不缺信息源，缺一个能回答「**我们受影响吗、证据是什么、该谁做、现在到哪了**」的地方。
> **下游**: [security-operations-feature-design.md](./security-operations-feature-design.md)（F1–F10 可实施功能规格，本文对应 F1+F2）。

---

## 0. 一句话结论

**「到处找信息」的根因不是信息不够，是三样东西缺失：**

1. **缺对齐键** —— 公告写 `PHP 8.2.x < 8.2.35`，CMDB 写 `web-prod-03 / php-fpm 8.2.34`，这两句话没有共同主键，匹配不上。
2. **缺冲突裁决** —— 同一天，某国 CERT 说"未观察到在野利用"，某安全厂商说"Patchstack 与 Unit 42 已报在野利用"。企业拿到两个相反结论，没有人知道该按哪个行动。
3. **缺本企业视图** —— 十一份公告都在讲"世界范围内有多少台受影响"，没有一份讲"**你的**哪 7 台、哪个业务、谁负责、什么时候必须打完"。

所以这个能力不是"再做一个威胁情报聚合器"，而是三件具体的事：**归一 → 对齐 → 裁决**。

---

## 1. 缘起：一次真实事件的 48 小时

以 **PHP2Shell**（CVE-2026-109618 / CVE-2026-109619）为标本。事实锚点：

- 2026-10-20 漏洞细节**提前泄露**，PHP 团队被迫提前公布发布计划（[大河财立方转财联社](https://app.dahecube.com/nweb/pc/article.html?artid=290611)）
- 2026-10-21 13:30 UTC 发布修复：**8.5.12 / 8.4.27 / 8.3.36 / 8.2.35**（[CERT-EU 2026-018](https://cert.europa.eu/publications/security-advisories/2026-018)）
- 两个漏洞均为 **CVSS v4.0 = 9.2**，未认证远程代码执行：`CVE-2026-109618`（OOM 内存破坏，GHSA-w8hw-w9fx-j79w）、`CVE-2026-109619`（stale global state，GHSA-6wmf-4rv5-p3rp）
- 受影响 SAPI：**PHP-FPM / Apache mod_php / php-cgi / FrankenPHP / LiteSpeed**；PHP 团队判定 **mod_php 风险最高**——未认证单请求可在应用代码执行前触发
- 8.1 及更早分支已 EOL，无官方更新；PHP 团队为 7.4/8.0/8.1 提供最小源码回移植
- 发现者含 Wiz、Calif.io 研究者，**发现过程有 OpenAI 参与**（[BtCIRT 20261021012](https://btcirt.bt/critical-remote-code-execution-in-php-cve-2026-109618-and-cve-2026-109619-20261021012)）

### 1.1 信息散落在 11 个地方

| # | 来源 | 它独家有什么 | 企业为什么必须看 |
|:--|:---|:---|:---|
| 1 | PHP 官方 archive + 2 个 GHSA | 权威受影响版本区间、回移植 | 唯一权威 |
| 2 | CERT-EU 2026-018 | 技术机理、`memory_limit=-1` 缓解、SAPI 清单 | 缓解手段来源 |
| 3 | 加拿大 Cyber Centre AL26-026 | 检测建议（日志/进程排查） | 事后排查清单 |
| 4 | 不丹 BtCIRT 20261021012 | **反直觉警告：不要用 RLIMIT_AS/RLIMIT_DATA 缓解**、`session.upload_progress.enabled=0`、IOC 具体到日志特征 | 用错缓解=加重风险 |
| 5 | 中国 CNVD（CNVD-2026-41674/41675） | 国内编号体系、国内 CMS 生态影响面 | 合规上报口径 |
| 6 | 工信部 NVDB / 各地网信办 | 国内处置要求、临时下线建议 | 监管对齐 |
| 7 | 腾讯云 / 阿里云 / 奇安信 / 360 通告 | 云上资产视角、免费体检入口 | 云上客户动作 |
| 8 | Unit 42（Cortex Xpanse） | 2260 万台互联网暴露实例的量级估算 | 判断"这波有多大" |
| 9 | NCSC-NL GitHub 仓库 | **第三方产品/设备内嵌 PHP 清单** | 嵌入式资产唯一线索 |
| 10 | 各国政府实际动作（荷兰鹿特丹港、Overheid.nl、KNVB、政党网站、医院；丹麦 Kombit） | 同行实际处置强度 | 校准自身响应级别 |
| 11 | 媒体（扬子晚报等） | 传播与舆情 | 领导问询应对 |

### 1.2 四处口径冲突（这才是真痛点）

| 冲突点 | 甲方口径 | 乙方口径 | 影响 |
|:---|:---|:---|:---|
| **是否已在野利用** | 加拿大 Cyber Centre：PoC 可用，**"目前未发现在野利用"** | HawkEye：**Patchstack 与 Unit 42 已报告利用活动**（10-21）；CERT-EU：PoC 流传，"提高已被利用的可能性" | 决定是"常规变更"还是"紧急变更 + 主动狩猎" |
| **暴露规模** | Unit 42：**2260 万台**（互联网暴露实例，扫描估算） | 360：**逾百万台**已确认暴露 | 决定资源投入与对外表态（两个口径不可混用） |
| **缓解手段** | CERT-EU：`memory_limit=-1` | BtCIRT：**禁用 RLIMIT_AS/RLIMIT_DATA——反而会触发漏洞路径** | 用错=把风险放大 |
| **受影响面** | CNVD：列出 WordPress/Drupal/Joomla/Magento/Laravel/Symfony 等 CMS 生态 | PHP 官方 + NCSC-NL：重点在**内嵌 PHP 的第三方设备与控制面板** | 决定扫描范围是否只扫"我们的 PHP 服务器" |

> 注意第 3 条：一份公告说"这样缓解"，另一份说"那样做会加重"。这类知识如果只从一家云厂商通告里读，**很可能读不到**。这正是需要"多来源观测并存 + 冲突显式呈现"的产品理由，而不是靠某一家"权威源"。

### 1.3 而企业内部的真实困境

```
安全部：我们有多少台跑 PHP？
运维部：几百台吧，得查。
安全部：版本呢？
运维部：有些是 yum 装的，有些是容器里的，有些是厂商设备自带的。
安全部：哪些暴露在公网？
运维部：这个要找网络组。
业务部：我们受影响吗？客户在问。
领导：到底要不要像荷兰那样先关站？
```

这段对话每天在几十家企业重复。**没有人缺"信息"，所有人缺"同一个答案"。**

---

## 2. 根因诊断

| 表层症状 | 真实根因 | 产品对策 |
|:---|:---|:---|
| 到处找公告 | 同一事件 5 种编号（CVE / GHSA / CNVD / CERT 自定义 / 厂商 ID），无归一 | `threat_advisory` 单一实体 + `advisory_source_observation` 多源观测 |
| 找到了也不敢信 | 权威源之间直接矛盾，且矛盾往往藏在正文细节里 | **冲突不覆盖、只记录**，差异显式上屏，裁决权留给人 |
| 不知道自己有没有 | CMDB 只记"主机"，不记"主机上装了什么软件、什么版本、什么部署形态" | 新增 `software_instance`（软件实例）= 对齐键 |
| 评估完还是不知道谁做 | 结论停在"命中 N 台"，没有 owner、没有 deadline | `exposure_assessment` 直接带 `recommended_action` + `action_deadline` + CI owner |
| 各部门重复问同一件事 | 没有一份可共享的"当前结论" | 六问事件页（§3.4） |
| 处置学到了下次又忘 | 事件关闭即知识蒸发 | 结论回写知识库（复用评审状态机） |

---

## 3. 设计

### 3.1 五张表

```
threat_advisory                    外部事件（归一后，本租户视角）
  advisory_key        CVE-2026-109619（首选 CVE，缺失回落 GHSA→CNVD→vendor id）
  aliases             []string（GHSA-w8hw-w9fx-j79w / CNVD-2026-41674 / ngCERT-2026-100019 …）
  code_name           "php2shell"（便于人肉沟通，媒体传播时通常只有代号）
  title / summary
  severity            critical|high|medium|low
  cvss_score / cvss_version     9.2 / v4.0  ← 版本必须记，v3 与 v4 不可混比
  exploit_status      none | poc_public | in_the_wild | weaponized   ← 带来源与时间
  exploit_status_source   "Unit 42 / Patchstack（经 HawkEye 转述）"
  affected_products   JSON [{vendor, product, purl, version_range, sapi_scope, eol}]
  fixed_versions      JSON [{branch:"8.2", fixed:"8.2.35"}]
  mitigations         JSON [{kind, instruction, caveat}]   ← caveat 承载"不要用 RLIMIT_AS"
  status              monitoring | active | resolved
  tenant_id           可空（系统级共享情报 vs 租户自建）

advisory_source_observation        多来源观测（冲突留证）
  advisory_id / source_code / source_url / source_id
  trust_tier          official_vendor | national_cert | cloud_vendor | research | media
  published_at
  claims              JSON（该来源声称的字段子集）
  conflicts_with      JSON（与本实体当前采纳值的差异，自动计算）
  raw_excerpt         原文关键段落（不可变，供事后追溯"我们当时依据什么决策"）

software_instance                  软件实例 = 对齐键（P0 真正的难点）
  ci_id               挂在哪个 CI
  vendor / product_name
  purl                pkg:deb/debian/php@8.2.34 / pkg:generic/php@8.2.34
  version / version_normalized
  branch              8.2（用于分支级区间匹配）
  deployment_sapi     fpm | mod_php | cgi | frankenphp | litespeed | cli | unknown
  is_embedded         bool（嵌在第三方产品/设备/控制面板内）
  embedded_in_ci_id
  exposure_scope      internet | intranet | offline
  source              cmdb_manual | discovery | sbom | host_agent
  confidence          confirmed | inferred | stale
  last_seen_at

exposure_assessment                一条「事件 × 资产」结论
  advisory_id / ci_id / software_instance_id
  verdict             confirmed | likely | not_affected | unknown    ← 四态，unknown 不是失败
  match_method        exact_version | range | product_only | sapi_scope | manual
  evidence            JSON（命中区间、SAPI 依据、版本原始串、置信度）
  recommended_action  patch | mitigate | isolate | migrate | verify_only | no_action
  action_deadline
  assessed_by         system | human
  decided_by / decided_at / decision_note     （unknown → 人工裁决留痕）

advisory_exposure_rollup           事件级汇总（供页面与上报）
  advisory_id / tenant_id
  total_cis / confirmed / likely / not_affected / unknown
  blind_ratio         unknown / total（必须上屏的负面指标）
  first_answer_at     首次产出结论时间 → 算「自答时延」
```

### 3.2 四态结论（不要三态）

| 结论 | 含义 | 后续动作 |
|:---|:---|:---|
| `confirmed` | 版本落在受影响区间 + SAPI 命中 + 非 EOL 豁免 | 打补丁，走标准/紧急变更 |
| `likely` | 部分证据成立（版本未知但产品命中；或 SAPI 作用域只含 mod_php 而资产是 FPM） | 人工确认 → 升级为 confirmed 或 not_affected |
| `not_affected` | 版本已在修复版之上，或 SAPI 明确不在作用域 | 记录依据（以便复核） |
| **`unknown`** | **CMDB 里没有软件实例记录 / 版本过旧无法判定** | **单列为「盲区」，指派核查——这是最有价值的一栏** |

🔴 **`unknown` 必须独立成桶，不许并入 `not_affected`。** 把"我不知道"和"我确认没影响"混在一起，是漏洞管理最常见的自欺形式，也是领导层最容易被误导的地方。

### 3.3 匹配引擎的三个硬细节

**① 版本区间必须按分支语义比较，不能字符串比较**

`8.2.x < 8.2.35`。字符串比较会把 `8.2.4` 判为 `>` `8.2.35`（因为 `4 > 3`），直接漏掉最该报的资产。实现要求：
- 解析为 `(major, minor, patch)` 三元组 + 保留原始串
- 区间匹配先按 `branch` 命中（8.2 / 8.3 / 8.4 / 8.5），再做 patch 比较
- EOL 分支（≤8.1）单独走 `migrate` 建议，不进 `patch` 路径

**② SAPI 作用域会改变结论等级**

同是 `php 8.2.34`：
- `deployment_sapi = mod_php` → `confirmed`（PHP 团队判定风险最高，未认证请求可在应用代码前触发）
- `deployment_sapi = fpm` → `confirmed`（CVE-2026-109619 确认路径）
- `deployment_sapi = unknown` → **降级为 `likely`**，因为无法排除最高风险部署形态

这条要求资产台账必须采 `deployment_sapi`——一个绝大多数 CMDB 都不采的字段，也正是这个能力的技术门槛所在。

**③ 嵌入式资产不能靠版本排除**

`is_embedded = true`（设备自带 PHP、控制面板自带 PHP）时，**厂商自有版本号与上游 PHP 版本无对应关系**，必须查厂商公告。策略：
- 嵌入式资产一律先置 `likely` 或 `unknown`，不置 `not_affected`
- 对接 NCSC-NL 那类"内嵌 PHP 产品清单"作为线索库
- 设备类走 vendor 台账（已有 `handlers/vendor` 域可复用）

### 3.4 六问事件页（"不用到处找"的字面实现）

一个 URL，安全 / 运维 / 业务 / 领导共用同一份答案，固定六块：

| # | 问题 | 数据来源 |
|:--|:---|:---|
| 1 | 这是什么 | `advisory.summary` + `code_name`（人话版，可给业务方看） |
| 2 | 我们受影响吗 | rollup 四态分桶 + 涉及业务系统（经 CMDB 关系展开） |
| 3 | 证据是什么 | `evidence`（版本串、SAPI、匹配方法、置信度）逐条可点开 |
| 4 | 该怎么办 | `recommended_action` + `mitigations[].caveat`（把"不要用 RLIMIT_AS"这类反直觉警告顶到最前） |
| 5 | 谁负责 | CI owner + 值班人 + 审批链 |
| 6 | 现在到哪了 | confirmed 中已处置/待处置计数、deadline 倒计时、`unknown` 待核查数 |

**设计约束**：第 2 与第 6 块必须带时间戳与数据新鲜度（`last_seen_at` 最旧值），否则这份答案会在 3 天后变成新的错误信息源。

### 3.5 路由

```
POST   /api/v1/security/advisories                     advisory:write   入库（AI 抽取草稿 + 人工确认）
GET    /api/v1/security/advisories                     advisory:read
GET    /api/v1/security/advisories/:key                advisory:read
PATCH  /api/v1/security/advisories/:key                advisory:write
POST   /api/v1/security/advisories/:key/match          advisory:write   触发匹配（异步 job，幂等）
GET    /api/v1/security/advisories/:key/exposure       advisory:read    四态分桶 + 盲区清单
POST   /api/v1/security/exposure/:id/decide            advisory:write   人工裁决 unknown/likely 留痕
POST   /api/v1/security/advisories/:key/spawn-incident security_incident:write  一键开安全事件（接 S1）
POST   /api/v1/security/software-instances/import      security_asset:write   SBOM / 主机代理批量导入
GET    /api/v1/security/software-instances              security_asset:read
```

**工程约束**（沿用本项目既有教训）
- 匹配 job 必须**幂等**：同一 `advisory_key + 资产快照` 重复触发不得产生重复 assessment（用唯一键 + 原生 SQL upsert 语义，注意 ent 未开 `sql/upsert` 时不能指望 `ON CONFLICT`）
- 匹配不得在事务里吞约束错误做幂等（PG 会 25P02 整笔连坐）
- 写路由必须进 `ResourceActionMap` 并同批跑 `cmd/authz-gen` + 5 道守卫
- `unknown` 计数查询走单一 owner，page size 有上界（历史踩过"条件式分页丢 LIMIT 导致整表读"）

---

## 4. AI 在这里做什么、不做什么

| 用途 | 做 | 不做 |
|:---|:---|:---|
| **公告结构化** | 从 11 份异构文本抽取 `affected_products` / `version_range` / `mitigations` / `exploit_status`，产出**草稿** | 直接入库生效 |
| **冲突标注** | 比对多源 `claims`，指出"A 说未利用、B 说已利用" | **裁决谁对** |
| **影响面叙述** | 把图遍历结果写成业务方能读的一段话 | 决定 `verdict` |
| **匹配建议** | 对 `unknown` 桶给出"建议核查哪台、查什么命令"（如 `php -v`、`apachectl -M \| grep php`） | 自动改 `verdict` |

🔴 **硬约束**：外部公告文本、厂商通告、媒体稿全部是**不可信输入**（攻击者可投毒公告内容诱导 AI 抽取错误版本区间，制造漏报或误报）。因此 AI 抽取结果必须 `reviewed_by` 人工确认后才 `status=active`；未确认的草稿不得参与匹配。

---

## 5. 与 S1 安全事件域的衔接

```
外部事件（threat_advisory）
   ├─ 暴露面匹配 → exposure_assessment（四态）
   │     ├─ confirmed 且 deadline 内无法完成 → 自动开 security_incident（SEC-*）
   │     ├─ likely     → 人工裁决任务（派给 CI owner）
   │     └─ unknown    → 盲区核查任务（派给资产管理员）
   └─ 结论 + 处置过程 → 回写知识库（评审后发布，供下次同类事件秒查）
```

这形成真正的闭环：**情报 → 自有资产 → 决策 → 处置 → 知识**。上一轮蓝图补了"决策→处置→审计"，这一份补的是最前面那一段。

---

## 6. 度量（这个能力自己的 KPI）

| 指标 | 定义 | 为什么是它 |
|:---|:---|:---|
| **自答时延** | 公告首次出现 → 本企业产出四态结论的时间 | 直接对应"到处找"的痛，比 MTTD 更贴切 |
| **盲区占比** | `unknown / total_cis` | 唯一能逼出"我们不知道自己有什么"的指标 |
| **重复问询次数** | 同一 advisory 被多少部门/人以多少渠道重复问 | 量化"信息散落"的组织成本 |
| **误报率** | `confirmed` 中被人工翻案为 `not_affected` 的比例 | 防匹配引擎靠撒谎换取"看起来覆盖全" |
| **知识复用率** | 同类事件（同 product/tactic）二次发生时，手册直接命中的比例 | 证明沉淀真的有用 |

🔴 **只报"覆盖率"是危险的**：把 100% 资产标成 `unknown` 也能算出漂亮的覆盖率。四态必须同时出现，缺一不可。

---

## 7. 不做什么

| 不做 | 理由 | 替代 |
|:---|:---|:---|
| 自建漏洞扫描器 | 资产发现是另一个产品 | 接扫描器/主机代理/SBOM 导入 |
| 自建情报 feed 采集集群 | 合规与出境风险，且红海 | connector 接 NVD/CNVD/厂商/CERT；客户自带 feeds |
| 自动打补丁 | 无授权边界的自动化 = 事故 | 只出 `recommended_action`，执行走变更/处置门禁 |
| AI 裁决冲突 | 投毒面 + 责任不可归属 | 显式呈现冲突，人裁决并留痕 |
| 面向外部的情报门户 | 定位是企业内部运营 | 六问页仅内部可见，按 clearance 分级 |

---

## 8. 验收用例（拿 PHP2Shell 当夹具，可直接落测）

| # | 用例 | 期望 |
|:--|:---|:---|
| 1 | 资产 `php 8.2.34 / fpm / internet` + advisory 区间 `8.2.x < 8.2.35` | `confirmed`，`match_method=range`，evidence 含原始版本串 |
| 2 | 资产 `php 8.2.4 / fpm` | `confirmed`（**字符串比较会误判为不受影响，此例专门锁死该 bug**） |
| 3 | 资产 `php 8.2.36 / fpm` | `not_affected` |
| 4 | 资产 `php 8.2.34 / sapi=unknown` | `likely` 而非 `confirmed`（无法排除 mod_php） |
| 5 | 资产 `php 8.3.10 / is_embedded=true` | `likely`/`unknown`，**不得**因"厂商版本号看起来更高"判 `not_affected` |
| 6 | CI 存在但无 `software_instance` 记录 | `unknown`，且进入盲区清单，**不计入** `not_affected` |
| 7 | 同一 advisory 重复触发 match | assessment 数量不增（幂等） |
| 8 | 同一 advisory 挂 3 个来源，其中 2 个在 `exploit_status` 上冲突 | 冲突入库、页面显式呈现，`exploit_status` 保留来源标注 |
| 9 | AI 抽取草稿未 `reviewed_by` | 不参与匹配（`status != active`） |
| 10 | 公告正文注入"受影响版本为 0.0.0"的恶意指令 | 抽取结果被打回人工确认，**不得**静默改变区间 |

---

## 9. 落地判断

- **依赖已有能力**：CMDB（CIType/CI/关系/影响分析）、vendor 域、知识库评审、SLA、审批链、authz 平面——**无一项需要新建基础设施**。
- **真新增成本在两处**：`software_instance` 的数据采集（SBOM/代理/人工，属数据工程而非代码），以及 `deployment_sapi` 这个字段的组织推行。
- **建议顺序**：`software_instance` 先做（哪怕只有手工导入）→ 再做 advisory 归一 → 最后做匹配。**先有对齐键，情报才有意义**；反过来做，接十个 feed 也只会得到"我们可能有影响"这种回答。
- **与 S1 的关系**：S0 可以不依赖 S1 独立上线（它输出的是"评估结论 + 待办"，不是事件）。但 S1 上线后闭环才完整。
