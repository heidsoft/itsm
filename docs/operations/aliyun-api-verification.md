# 阿里云接入面真实账号验证报告（2026-11-03）

本文记录 ITSM 阿里云接入面在**真实账号**上的零费用验证结论：哪些链路已被证实可用、哪些是账号侧条件、哪些是代码缺陷（已修）。

**验证前提**：账号不充值、不创建/修改/删除任何云资源、不产生费用。

> **脱敏说明**：本文所有公网/私网 IP、实例 ID、交换机/安全组 ID、资产 UUID、组件版本号均已替换为占位符。
> 原始证据（含真实标识）只留在本地 `output/` 目录，该目录被 `.gitignore` 整目录排除（理由见 `.gitignore` 中 `output/` 规则：生产验证产物可能含真实凭据），不入版本库。

---

## 1. 结论

| 链路 | 签名实现 | 验证结论 |
|---|---|---|
| 服务请求交付（ECS） | 官方 Tea SDK `ecs-20140526/v7` | ✅ 参数/权限/配额全部通过，**唯一阻塞是账号余额** |
| 云资源发现（ECS） | 同上 | ✅ 16 个只读接口全部返回真实数据 |
| 云安全中心漏洞（SAS） | **手写 RPC 签名** | ✅ 签名对真实端点成立；发现并修复 4 个缺陷 |
| 云监控告警（CMS） | **手写 RPC 签名** | ✅ 同上；错误形态被正确识别 |
| 漏洞同步端到端落库 | — | ✅ prod 真实部署验证：漏洞入库、CI 关联、毫秒时间戳、云侧 Level 严重度全部正确 |

**合起来的结论：ITSM 的阿里云接入在代码侧没有阻塞项。** 剩下的都是账号侧条件：交付链路需要充值，安全事件链路需要开通 SAS 付费版与积累真实报警数据。

> 漏洞同步已在 prod 真实部署形态下端到端验证：修复前 `state=unavailable`，修复后 `state=ready`、`items_synced=1`，
> 那条 `emg`/`asap` 漏洞正确入库并关联到 CMDB 配置项（详见 §6）。

---


## 2. 交付链路（官方 SDK）

### 2.1 零费用的关键手段

`RunInstances` 支持 `DryRun=true`：阿里云只校验必填参数、请求格式、RAM 权限、服务限制与可用资源，**不创建实例、不计费**。校验通过返回 `DryRunOperation`，业务拒绝返回真实业务错误码——两者都是有效证据。

这是「不充值也能验写 API 参数链」的唯一正路。命令内置运行期守卫：`DryRun` 未置 true 时直接拒绝向云侧发起调用。

### 2.2 实测结果

只读面 16 项全部 HTTP 200（地域/账号属性/可用区/镜像/镜像支持规格/交换机/安全组/安全组规则/规格/规格族/库存/实例/实例状态/询价/密钥对/标签），均返回真实数据与 RequestId。

写 API 校验链：

```
RunInstances(DryRun=true) → 403 InvalidAccountStatus.NotEnoughBalance  (permanent=true)
```

**这条结果证明**：凭据、签名、客户端构造、endpoint 解析全部正确；交付四要素（镜像/交换机/安全组/规格）组合合法；云侧已越过参数与权限校验，**停在余额闸门**。作为对照，故意传非法镜像时云侧确实返回 `404 InvalidImageId.NotFound`。

账号侧门槛（`DescribeAccountAttributes`）：

| 属性 | 实测值 | 判读 |
|---|---|---|
| real-name-authentication | `yes` | 实名认证已完成，不是阻塞项 |
| max-postpaid-instance-vcpu-count | 87654321 | 按量配额充足，不是阻塞项 |
| used-postpaid-instance-vcpu-count | 0 | 当前无按量占用 |
| instance-network-type | `vpc` | 支持 VPC 网络（交付要求） |

**结论：配额与实名认证都不是阻塞，唯一阻塞是余额。**

### 2.3 必须记住的云侧契约

**阿里云多个 `Describe*` 对不存在的 ID 返回 `HTTP 200 + 空列表`，不抛 err**（镜像/交换机/安全组/规格四项实测如此）。存在性校验**必须判返回条数**（`len(...)==0`），只判 `err` 或状态码会把「资源不存在」当成「查到」。

`cmd/aliyun-connection-check` 已按返回条数判定存在性。

---


## 3. 安全事件链路（手写 RPC 签名）

### 3.1 为什么这条链路此前是盲区

交付与发现走官方 Tea SDK，而 SAS/CMS 走的是**手写 RPC 签名**（HMAC-SHA1 v1.0）。此前只有 mock 单测——签名串、端点、版本、参数名、错误形态都没有对真实端点验证过。

### 3.2 实测结果

19 次只读调用（产品路径 5 + 覆盖缺口 3 + 开通资产面 7 + 负向 4）：

- **15 个只读动作全部通过**：`DescribeVulList`（cve/sys/cms/sca/app/emg）、`DescribeAlertLogList`、`DescribeVersionConfig`、`DescribeAllGroups`、`DescribeCloudCenterInstances`、`DescribeCheckWarningSummary`、`DescribeWarningMachines`、`DescribeVulNumStatistics`、`DescribeMetricList`；
- **4 个负向用例按预期报错**，且一律是 **HTTP 400 + 响应体业务码**（`IllegalParam` / `MissingType` / `ParameterInvalid`）——安全侧错误不会藏进 200 响应体，生产 `parseRPCError` 路径已覆盖，**无需改动生产**。

这与 ECS 侧「Describe* 返回 200 + 空列表」正好相反，两面都已核实。

### 3.3 账号侧事实

`DescribeVersionConfig` 实测：`IsPaidUser=false`、`IsSasOpening=false`、`TrialVersion=2`、`AuthVersionName=免费版`。

即**云安全中心是免费版且未开通付费能力**，漏洞扫描没有运行——`cve`/`sys`/`cms`/`sca`/`app` 五类漏洞全为 0 是「**没扫**」，不是「没有」。唯一一条真实漏洞来自 SCA（软件成分分析）通道，与扫描是否运行无关。

**为什么没有据此把同步判成失败**：免费版下 `IsSasOpening=false` 的语义并不等于「漏洞采集完全不可用」（SCA 通道确实在工作），据此判失败会误报。正确做法是如实记录，让使用者知道「0 漏洞」在这套账号上的含义。

---


## 4. 验证打出的缺陷（已全部修复）

### 4.1 交付链路

**① `ClassifySDKError(nil)` 空指针 panic**
`describeAliyunError` 直接调 `err.Error()`，传 `nil` 触发 SIGSEGV。只读校验里「故意传非法参数却返回成功」是合法观测结果，分类函数必须 fail-safe。

**② 错误文案承诺脱敏但实际未脱敏**
注释写「避免把请求体（可能含密码）写进任务错误和审计」，实现却只做空白折叠 + 截断。云侧若在 message 里回显 `Password=...` / `AccessKeySecret=...`，会原样进审计日志。现前置敏感值脱敏（覆盖 `password`/`access_key_secret`/`token`/`api_key`/`private_key`/`credential`/`authorization`，兼容 `key=value` 与 JSON 两种形态），且**先脱敏后截断**（先截断会把敏感值切成半截绕过匹配）；`code=`/`requestId=`/`message=` 等对账字段不误伤。

**③ 地域配错被判可重试**
`InvalidOperation.NotSupportedEndpoint`（HTTP 400「The specified endpoint can't operate this region」）不在永久错误清单里，同一份错配置会被重投 5 次。现补入两个确切前缀，但不整段收 `InvalidOperation.*`——该命名空间下还有「实例状态冲突、资源正在变更」这类重试有意义的临时拒绝。

### 4.2 安全事件链路

**① 时间戳单位错误：毫秒被当秒解析**
云安全中心官方文档明确 `FirstTs`/`LastTs`/`ModifyTs`/`RepairTs`「单位为毫秒」（响应示例 `1620404763000`，真实账号实测 `1781413840000` 同为毫秒）。代码此前按**秒**解析，真实数据会被算成**公元 58655 年**——漏洞的「首次发现/最后发现」落在几万年之后，排序、水位线比较与「多久没复现」判断全部失真。

现按毫秒解析并保留秒级兜底（分界 `1e11`：毫秒时间戳大于它才落在 1973 年之后，秒时间戳小于它才落在 5138 年之前，两侧不可能重叠，不会误伤任何真实值）。

> 附带修正：既有测试夹具本身就是秒级并断言秒级解析——**测试把 bug 一起实现了**，所以单测全绿也发现不了这个问题。夹具与断言一并改为毫秒。

**② 漏洞类型覆盖缺口：账号唯一一条漏洞进不来**
真实账号唯一一条漏洞走 `emg`（应急）通道：`emg:SCA:AVD-2026-1860246`（OpenClaw 多漏洞合集）、`Necessity=asap`、CVSS 8.8、`InstanceId` 可关联 CMDB 配置项。而 `vulnSyncTypes` 原为 `{cve,sys,cms}`，这条漏洞因此永远进不了 ITSM；前端标签表本来就有 `emg`「紧急漏洞」——**缺口只在后端同步列表**。

现默认集合改为 `{cve,sys,cms,emg}`，并支持环境变量 `ITSM_CLOUD_SIGNAL_VULN_TYPES`（逗号分隔）收窄/扩展，配置为空或解析不出合法值时回落默认值——不能因为一个变量拼错就同步不到任何漏洞。`app`（应用漏洞/web 扫描器）仍不纳入：它要求资产上装 web 扫描器，没有扫描器时恒为空。

**③ 缺同步完整性闸门：漏同步会被收口成「全部已修复」**
收口逻辑「本轮未出现的漏洞 → 标记已修复」没有校验本轮是否真的拉全了。一旦某类型因权限变化、类型被过滤、分页中断等原因一条都没拉到，收口就会把该类型下**真实存在的漏洞全部误标为「已修复」**——安全上比漏报更危险（用户看到「已修复」就不会再管）。

现取云侧权威计数（`DescribeVulNumStatistics`）在收口前交叉校验：云侧该类型计数 > 0 而本轮拉到 0 条时，记入 `MismatchedTypes` 并**跳过该类型收口**，同时打结构化 Warn；拿不到计数时只记日志、不阻断同步（交叉校验是安全网，不是同步的前置条件）。

---


## 5. 验证资产

### 5.1 命令

| 命令 | 用途 | 费用 |
|---|---|---|
| `go run ./cmd/aliyun-sdk-api-verify` | 交付/发现链路：16 只读 + `RunInstances(DryRun=true)` + 6 个错误分类用例 | 零 |
| `go run ./cmd/aliyun-security-api-verify` | 安全事件链路：19 次只读调用（产品路径/覆盖缺口/开通资产面/负向） | 零 |
| `go run ./cmd/aliyun-connection-check` | 交付就绪度：4 类参数存在性 | 零 |

三个命令的凭据都只经 `env://ITSM_ALIYUN` 从进程环境读取，AK/SK 不进命令行、不进输出、不写日志。

### 5.2 只读边界

**交付侧**：源码扫描守卫——非 `Describe*` 的 SDK 调用只允许 `RunInstances`，且**逐处**判定必须走 DryRun（不能靠「源码里别处有 DryRun」蒙混）；必须调用生产的 `BuildDryRunRequest`（匹配调用形式而非裸标识符，否则注释也能满足断言）；出现 `CreateInstance`/`DeleteInstance`/`ReleaseInstance`/`RunCommand` 等写方法直接失败。守卫带三重反向验证：注入写 API、绕开 DryRun、把 `DryRun` 置 false，守卫必须各自变红。

**安全侧**：**运行时白名单**——`service/cloud/aliyun/probe.go` 的 `readOnlySecurityActions`/`readOnlyMonitorActions` 只收 `Describe*` 动作；不在册直接返回 `ErrNotReadOnlyAction`，**连请求都不发**。守卫测试用「不存在的凭据」调用写动作：若白名单失效，错误会是签名/鉴权失败而非白名单错误，由此证明拦截发生在发请求之前。

### 5.3 复用生产代码路径

验证命令不是「另一份实现」：

| 环节 | 复用的生产代码 |
|---|---|
| 凭据解析 | `service/cloud.ResolveAliyunCredential` |
| SDK 客户端构造 | `service/cloud/aliyun.NewECSClientForProbe` + `ProbeRuntimeOptions` |
| DryRun 请求 | `alicloud.BuildDryRunRequest`（内部即交付用的 `parseECSLaunchParams` + `buildRunInstancesRequest`） |
| ECS 错误分类 | `alicloud.ClassifySDKError`（内部即交付失败写审计那套逻辑） |
| RPC 签名 | `rpcClient.callRaw`（与生产 `Call` 同一份签名/编码/端点/超时实现） |

`TestBuildDryRunRequestReusesProductionParams` 逐字段比对 DryRun 请求与真实交付请求（除 `DryRun` 外必须完全一致），防止「验证跑通的参数」与「真正交付的参数」漂移。

---


## 6. 端到端落库验证（2026-11-03 晚，prod 真实部署）

前面各节验证的是 **API 面**。这一节验证的是**完整生产链路**：真实镜像 → worker 定时调度 → 云 API → 落库 → CI 关联。

### 7.1 部署形态

| 项 | 值 |
|---|---|
| 镜像 | `itsm-backend:v1.6.42`（重建自 HEAD，Created 20:32） |
| 构建方式 | 干净 worktree + HEAD 构建，**不含并行会话的未提交 WIP** |
| 容器 | backend/worker 重建（`Created` 更新），7 容器全 healthy |
| init 退出码 | 0（迁移与基线校验通过） |
| readyz / health | 200 / 200 |
| 登录冒烟 | `code=0` |
| 云凭据透传 | backend 与 worker **各 2 个变量**（此前 backend 缺失） |

> **踩到的坑**：容器自 2026-11-01 起只 restart 过、从未重建，而云凭据透传是当天才加进 compose 的——`docker restart` **不会**更新容器环境变量，所以 backend 一直拿不到凭据。改 compose 后必须**重建容器**（`up -d` 检测到配置变化会 recreate），`restart` 不够。

### 7.2 漏洞同步：修复前 → 修复后

| 阶段 | `state` | `items_synced` | 错误 |
|---|---|---|---|
| 修复前（重建后未含形态兼容） | `unavailable` | 0 | `describe_vulnerabilities_failed`：解码失败 |
| **修复后** | **`ready`** | **1** | 无 |

落库内容（已脱敏）：

| 字段 | 值 |
|---|---|
| vuln_type | `emg` |
| severity | `high`（来自云侧 `Level`，**不是** unknown） |
| status | `open` |
| finding_key | `8218810662` |
| name | `emg:SCA:AVD-2026-1860246` |
| necessity | `asap` |
| instance_id | `i-…`（可关联 CMDB） |
| region | `cn-beijing` |
| cvss_score | 空（emg 类型无 cvss_score，只有 Level——预期） |
| first_seen_at / last_seen_at | **2026-06-14 05:10:40+00**（毫秒解析正确；按秒解析会是公元 58655 年） |
| CI 关联 | **已关联**（`ci_id` 非空） |

**这一条同时验证了三项修复**：漏洞类型覆盖（emg 进来了）、毫秒时间戳（不是几万年之后）、云侧 Level 严重度（不是 unknown）。CI 关联生效说明漏洞能落到具体资产上，而不只是躺在列表里。

### 7.3 告警同步

`state=ready`、`items_synced=0`、无错误——账号确实没有报警历史（`DescribeAlertLogList` 恒空），链路本身通畅。**告警自动建事件仍未验证**（无数据可走通）。

### 7.4 这轮验证打出的第四个缺陷

`ExtendContentJson` 在不同漏洞类型下形态不同：**cve/sys/cms 是 JSON 字符串，emg 是 JSON 对象**。结构体把该字段定义成 `string`，解到 emg 记录时直接报

```
json: cannot unmarshal object into Go struct field
SecurityVulnerability.VulRecords.ExtendContentJson of type string
```

——**整个类型的漏洞一条都进不来**，同步从「拉不到」变成「报错」。这是把 emg 纳入同步类型后才暴露的形态差异。

修法：新增 `flexJSONObject` 兼容两种形态，`SecurityVulnerability` 自定义 `UnmarshalJSON` 归一成 JSON 字符串（字段类型保持 `string`，外部调用点零改动）；严重度优先用云侧 `Level`，无 Level 才回退 CVSS 推导；单条脏数据不拖垮整轮（丢一条扩展内容好过整类丢光）。

> **教训**：验证工具只解析了 HTTP 状态与条数，**没有走产品的 JSON 解码路径**，所以第一轮验证没发现这个形态差异。API 面的"200 + 1 条"不等于"产品能解码这一条"。这类缺陷只有把真实响应喂给产品代码才会暴露——所以守卫测试用的是**真实账号返回的原样响应**作夹具。

---


## 7. 仍未验证的部分

| 未验证项 | 原因 | 何时能验 |
|---|---|---|
| 一次真实创建并释放 ECS 的闭环 | 需账号余额（按量 ECS 要求余额 ≥ 100 元） | 充值后执行一轮完整交付验收 |
| 创建后的 SLA 首响/解决、BPMN 收口、CMDB 投影的实际数据 | 依赖上一条 | 同上 |
| 漏洞同步端到端落库（`security_findings` 写入、CI 关联、事件自动建单） | 需要在部署环境跑一轮同步 | 在部署环境触发一次同步即可 |
| 告警自动建事件 | 账号无报警历史（`DescribeAlertLogList` 恒空） | 有真实报警后重跑 |
| 基线检查告警 | 账号 0 条基线风险项 | 开通并跑基线检查后 |
| SAS 付费版能力（专业版漏洞库、SCA 组件、容器资产） | 账号为免费版 | 开通付费版后 |
| 交付中途失败的自动回滚/补偿 | 未实现（现为幂等键对账 + 人工裁决兜底） | 补偿动作开发后 |
| 多资源类型（OSS 等） | 代码尚未接入 | 第二资源类型开发后 |

**未验证 ≠ 未通过。** 本轮能证明的是：签名、参数、权限、配额、实名认证、地域、资产组合全部正确，代码侧没有阻塞；剩下的都是账号侧条件或尚未开发的功能。

---


## 8. 复现与判据

```bash
cd itsm-backend
export PATH="/usr/local/go/bin:/opt/homebrew/bin:$PATH"

# 单元与守卫
go test ./infrastructure/cloud/alicloud/ ./service/cloud/aliyun/ \
        ./handlers/cloudsignal/ ./cmd/aliyun-sdk-api-verify/ ./cmd/aliyun-security-api-verify/ -count=1

# 真实账号零费用验证
export ITSM_ALIYUN_ACCESS_KEY_ID=... ITSM_ALIYUN_ACCESS_KEY_SECRET=...
go run ./cmd/aliyun-sdk-api-verify -region cn-beijing
go run ./cmd/aliyun-security-api-verify
```

判据（不要只看退出码）：

- 交付侧：16 个只读项必须全部 `items > 0`（`DescribeKeyPairs`/`DescribeTags` 允许为 0）；`RunInstances(DryRun=true)` 必须返回业务码且 `permanent` 符合预期；4 个非法参数用例必须是 `http=200 + items=0`（不是报错）；
- 安全侧：`product-path` 5/5、`activation` 7/7、`coverage-gap` 3/3 必须全通过；`negative` 4 项必须**失败**且带业务码；`DescribeVulNumStatistics` 的各类型计数必须与 `DescribeVulList` 的条数一致。

---


## 9. 修复后的回归覆盖

| 缺陷 | 守卫测试 |
|---|---|
| `ClassifySDKError(nil)` panic | `TestClassifySDKErrorNilIsFailSafe` |
| 错误文案未脱敏 | `TestClassifySDKErrorNeverLeaksRequestBody`、`TestSanitizeMessageKeepsReconciliationFields`、`TestSanitizeMessageCoversJSONShape` |
| 地域配错被判可重试 | `TestIsPermanentAliyunCode`（新增两个永久码 + 两个 `InvalidOperation.*` 可重试用例） |
| DryRun 参数与生产漂移 | `TestBuildDryRunRequestReusesProductionParams`、`TestBuildDryRunRequestValidatesLikeExecute` |
| 时间戳单位 | `TestTimestampsAreMilliseconds`、`TestTimestampsFallBackToSeconds`、`TestZeroTimestampIsUnknown` |
| 漏洞类型覆盖缺口 | `TestDefaultVulnSyncTypesCoverRealAccountFindings`、`TestResolveVulnSyncTypesHonoursEnv` |
| 漏同步被误判成已修复 | `TestVulSyncSkipsResolvedCollectionWhenCloudCountMismatch`、`TestVulSyncCollectsWhenCountsMatch`、`TestVulSyncSurvivesStatisticsFailure` |
| 只读边界被绕过 | `TestProbeRejectsWriteActions`、`TestVerifyCommandOnlyUsesAllowlistedActions`、`TestVerifyCommandNeverCallsWriteActions` |
| 真实数据解析回归 | `TestRealAccountEMGFindingParsesCorrectly`、`TestRealAccountVulStatisticsParsesCorrectly` |
| 扩展内容形态兼容（对象/字符串） | `TestSecurityVulnerabilityDecodesObjectShapedExtendContent`、`TestSecurityVulnerabilityDecodesStringShapedExtendContent` |
| 严重度优先级（云侧 Level 优先） | `TestSeverityPrefersCloudLevel` |
| 单条脏数据不拖垮整轮 | `TestFlexJSONObjectToleratesMalformedStringContent` |
| 记录 JSON 往返（raw_payload 落库） | `TestSecurityVulnerabilityRoundTripsThroughMarshal` |
