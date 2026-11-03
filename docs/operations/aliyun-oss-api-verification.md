# 阿里云 OSS SDK 只读验证报告（2026-11-03）

**验证前提**：账号不充值、不创建/修改/删除任何 Bucket 与对象、不产生费用。
**验证对象**：阿里云对象存储 OSS 的官方 SDK（`github.com/aliyun/aliyun-oss-go-sdk v3.0.2+incompatible`）。
**背景**：能力矩阵把「第二资源类型」列为服务请求交付链路晋升 GA 的条件之一；本报告为选择 OSS 作为第二资源类型提供实测依据。

**结论**：**OSS SDK 在真实账号上完全可用——凭据、OSS 签名、桶级读权限全部验证通过；同时实测出 3 条与 ECS 截然不同的技术契约，其中「对象级操作必须按 Bucket 所在地域路由」会直接决定交付适配器的接口形态。**

> **脱敏说明**：本文所有 Bucket 名、Endpoint 后缀标识均已替换为占位符。原始证据（含真实 Bucket 名）只留在本地 `output/`，该目录被 `.gitignore` 整目录排除，不入版本库。

---

## 1. 结论

| 项 | 结果 |
|---|---|
| SDK 可获取性与编译 | ✅ `v3.0.2+incompatible` 与主项目 `go 1.26.10` 兼容，**未升级任何既有依赖** |
| 凭据复用 | ✅ **同一套 AK/SK 直接可用**，交付链路的 `env://ITSM_ALIYUN` 凭据模型无需改动 |
| OSS 签名 | ✅ `ListBuckets` 成功即证明 `Authorization: OSS AK:sig` 签名成立 |
| 服务级权限 | ✅ `oss:ListBuckets` 可用 |
| 桶级读权限与数据面签名 | ✅ 按 Bucket 地域路由后 `ListObjects` 成功 |
| 账号现存 Bucket | 1 个（`oss-pai-…-cn-shanghai`，location `oss-cn-shanghai`，Standard，2025-08-04 创建，空桶） |
| 成本模型差异 | ✅ **OSS 按存储量与请求数计费，不需要预付费余额** |

---

## 2. 实测明细

### 2.1 只读面

| API | 结果 | 耗时 | 验证的能力 |
|---|---|---|---|
| 客户端构造 | ✅ | — | 凭据格式合法 |
| `ListBuckets` | ✅ 1 个 Bucket | 269ms | OSS 签名 + `oss:ListBuckets` 权限 |
| `ListObjects`（按 Bucket 地域） | ✅ 0 个对象 | 186ms | 桶级读权限 + 数据面签名 |

### 2.2 负向用例（错误形态）

| 用例 | 云侧响应 | 结论 |
|---|---|---|
| 不存在的 Bucket | **404 `NoSuchBucket`** | 资源不存在 |
| 错凭据 | **403 `InvalidAccessKeyId`** | 凭据无效 |
| 错地域的 `ListBuckets` | **成功** | 服务级列举**与地域无关** |

---

## 3. 三条必须记住的技术契约（与 ECS 完全不同）

### 3.1 OSS 用自己的签名，不能复用 ECS 的 RPC 签名

| | ECS / OpenAPI | OSS |
|---|---|---|
| 形态 | RPC 风格（表单参数 + `Signature` 字段） | REST（`Authorization: OSS <AK>:<sig>` 头） |
| 算法 | HMAC-SHA1，签「方法 + 规范化查询串」 | HMAC-SHA1，签「方法 + Content-MD5 + Content-Type + Date + 规范化 OSS 头 + **CanonicalizedResource**」 |
| 资源定位 | 参数里的 `RegionId` 等 | **CanonicalizedResource 含 Bucket 名与对象名**，还要按 OSS 规则做路径编码 |

**影响**：项目现有的 `service/cloud/signing` 原语是 RPC 口径，**不能**用于 OSS；必须引入官方 SDK 或另写一套 OSS 签名。这是第二资源类型落地时**必须新写**的部分。

### 3.2 对象级操作必须按 Bucket 所在地域路由（最容易踩的一条）

用账号级地域访问别的地域的 Bucket，会得到：

```
403 AccessDenied
"The bucket you are attempting to access must be addressed using the specified endpoint.
 Please send all future requests to this endpoint."
Endpoint=oss-cn-shanghai.aliyuncs.com
```

关键点：**错误码是 `AccessDenied`，不是 `PermanentRedirect`**——错误码本身不提示"要换地域"，**正确的 endpoint 只出现在响应体的 `Endpoint` 字段里**，必须自己解析才能自动纠正。

而**服务级 `ListBuckets` 与地域无关**：任意地域 endpoint 都能列出全部 Bucket。

**影响**：现有 `cloud_accounts.region_whitelist` 是**账号级地域**模型（ECS 用），OSS 则是**资源级地域**（每个 Bucket 有自己的 `location`）。交付适配器不能沿用账号级地域：

- 创建 Bucket 时地域是入参；
- 访问已有 Bucket 时必须先取它的 `location` 再路由 endpoint；
- 交付参数模型需要从「regionId」扩展为「bucket 名 + 由 location 推导的 endpoint」。

### 3.3 错误码空间独立，且错误结构不同

| | ECS / OpenAPI | OSS |
|---|---|---|
| 载体 | JSON 响应体的 `Code` 字段 | XML `<Error><Code>…</Code></Error>` |
| 典型码 | `InvalidAccountStatus.*`、`QuotaExceeded.*`、`InvalidImageId.*`、`Forbidden.*` | `NoSuchBucket`、`BucketAlreadyExists`、`AccessDenied`、`InvalidAccessKeyId`、`InvalidBucketName`、`TooManyRequests` |

**影响**：交付层的永久错误分类要**为 OSS 单独一套码表**，不能复用 ECS 的 `permanentAliyunCodePrefixes`。

---

## 4. 第二资源类型落地的前置条件清单

以 OSS 作为服务请求交付链路的第二资源类型，需要满足下面这些（按依赖顺序）：

| # | 前置条件 | 说明 | 可否复用现有实现 |
|---|---|---|---|
| 1 | **OSS 签名与客户端** | 引入官方 SDK 或自写 OSS 签名 | ❌ 必须新写（RPC 签名不通用） |
| 2 | **地域模型** | 从账号级 `region_whitelist` 扩展到 Bucket 级 `location` 路由 | ❌ 必须扩展 |
| 3 | **错误分类** | OSS 独立码表 + 识别「endpoint 不对」型 AccessDenied 并按响应体 `Endpoint` 重定向 | ❌ 必须新写 |
| 4 | **凭据链路** | `env://ITSM_ALIYUN` → AK/SK | ✅ **完全复用**（实测同一套凭据直接可用） |
| 5 | **幂等语义** | `CreateBucket` 天然幂等（Bucket 名唯一），重复创建返回 `BucketAlreadyExists`，可直接映射为「已交付」 | ✅ 比 ECS 的 `ClientToken` 更简单 |
| 6 | **目录项资源类型** | `provisioningtask.resource_type` 注释已列 `oss`，但目录项/交付模板/校验链路尚未支持 | ⚠️ 需扩展 |
| 7 | **释放/回滚** | `DeleteBucket`（需先清空对象）；对象级删除要单独设计 | ❌ 必须新写 |
| 8 | **成本模型** | OSS 按存储量 + 请求数计费，**不需要预付费余额** | ✅ 天然绕开「余额不足」这个 GA 阻塞 |

**一条值得单独强调的收益**：ECS 交付链路的 GA 阻塞是「账号余额不足」，而 **OSS 不受余额门槛约束**——若把 OSS 作为第二资源类型落地，交付链路将首次拥有一条「不依赖充值即可完成验收」的资源类型路径。

---

## 5. 验证资产

### 5.1 命令

`itsm-backend/cmd/aliyun-oss-api-verify/`

```bash
export ITSM_ALIYUN_ACCESS_KEY_ID=... ITSM_ALIYUN_ACCESS_KEY_SECRET=...
go run ./cmd/aliyun-oss-api-verify -region cn-hangzhou -json-out /tmp/oss-verify.json
```

- 只做只读调用，不写任何资源；
- 凭据只经 `env://ITSM_ALIYUN` 从进程环境读取，AK/SK 不进命令行、不进输出、不写日志；
- 输出人类可读表格 + 结构化 JSON（每项含 HTTP 状态、错误码、RequestId、条数、耗时、结论归类）。

### 5.2 只读边界：源码扫描守卫

OSS SDK 的写方法有几十个（Bucket 生命周期、对象上传、策略、加密、CORS、分片上传……），一旦误用会**真的建桶/删桶/改数据**，而且不产生费用提示——比 ECS 那种「建实例会报错」的风险更高。因此守卫用**源码扫描**而不是「人工核对调用清单」：

- 显式列出 60+ 个写方法名（`CreateBucket`/`DeleteBucket`/`PutObject`/`SetBucket*`/`UploadPart`…），出现即失败；
- 另有一道**动词维度兜底**：源码里不得出现以 `Create`/`Delete`/`Put`/`Set`/`Upload`/`Copy`/`Abort` 开头的 OSS 调用；
- 再加一道**白名单维度**：所有 SDK 调用点（接收者限定为代码里持有的客户端/桶变量与 `oss` 包）逐一判定必须落在只读白名单内（`ListBuckets`/`ListObjects`/`Bucket`/`New`/`MaxKeys`）；
- **扫描器零命中即失败**——防止将来调用写法一变，守卫静默变绿；
- 反向清单测试覆盖 `writeMethodsForbidden` 的每一项，防止清单被自己删掉。

### 5.3 错误分类守卫

`classifyOSSCode` 把 OSS 码表映射成可行动结论（`not-found` / `bad-credential` / `access-denied` / `already-exists` / `invalid-parameter` / `throttled` / `cloud-error`），并用测试锁住：

- 误判方向两边都危险——把限流判成永久错误会让一次抖动变成「交付永久失败」；把「已存在」判成「不存在」会让幂等判断失效；
- 未识别的码归 `cloud-error`，**不**归「永久失败」。

---

## 6. 仍未验证的部分

| 未验证项 | 原因 | 何时能验 |
|---|---|---|
| `CreateBucket` / `DeleteBucket` 等写路径 | 本轮严格只读；写操作会真的产生 Bucket 与费用 | 需要产品决策后单独做一次带清理的验收 |
| 桶策略 / 加密 / 生命周期配置 | 属写操作 | 同上 |
| 对象上传/下载与传输加速 | 属写操作，且会产生存储费用 | 同上 |
| OSS 交付适配器 | 尚未开发 | 按 §4 的前置条件清单推进 |
| OSS 资源在 CMDB 的投影形态 | OSS 资源不是「实例」，与现有 CI 模型（主机/Pod）不同 | 需先定 CI 类型 |

**未验证 ≠ 未通过**。本轮能证明的是：SDK 可用、凭据可复用、签名成立、服务级与桶级只读权限具备、错误形态已摸清、地域路由契约已验证——这些正是第二资源类型落地前必须先确定的部分。
