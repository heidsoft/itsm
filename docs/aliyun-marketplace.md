# 阿里云镜像市场（及云市场）上架材料

> 配套：`deploy/docker-compose.saas.yml` + `deploy/Caddyfile` + `deploy/.env.saas.example` + `docs/deploy-saas.md`。
> 版本：`v1.6.58`（本地 HEAD，**非 GA**）。协议：**Apache-2.0**。
> 本文是上架 **Runbook + 物料模板**；实际推 ACR / 提交审核需用户阿里云账号与凭证（见 §6 边界）。

---

## 1. 上架形态选择

| 形态 | 交付物 | 适用 | 门槛 |
|---|---|---|---|
| **镜像市场 — 自定义镜像** | 预装 Docker + ITSM 的 ECS 镜像，用户一键开通 | 单实例开箱即用，最贴合 SaaS 试用 | 需镜像市场服务商资质 + 审核 |
| **云市场 — 容器镜像** | ACR 企业版/个人版镜像，用户拉取后自部署 | 已有 K8s/容器平台用户 | 较低，需 ACR 仓库 |
| **计算巢 / 服务市场** | 编排模板（ROS），自动拉起整套 | 企业多组件交付 | 较高 |

本材料以**「自定义镜像（镜像市场）」为主路径**，附 ACR 推送流（自定义镜像的镜像也来自同一套构建）。

## 2. 商品文案模板

> 口径红线：不得写「替代 ServiceNow」「正式商用 GA」「企业级替代方案」等；
> 统一表述为「开源 ITSM 平台，对标企业级 ITSM 能力」「供试用与私有化部署」。

**商品名称**：ITSM 开源 IT 服务管理平台（单实例多租户版）
**副标题**：事件 / 问题 / 变更 / 配置(CMDB) / 知识库 / SLA / 工作流(BPMN) / 邮件工单闭环
**一句话简介**：基于 Go + Next.js 的开源 ITSM 平台，提供开箱即用的 IT 服务管理核心能力，支持单实例多租户部署，适用于中小企业 IT 运维与服务平台化试点。

**核心功能（6–10 条）**：
1. 事件管理（Incident）：受理、分派、SLA、升级、关闭全流程。
2. 问题管理（Problem）：根因跟踪与已知错误库。
3. 变更管理（Change）：变更申请、审批链、风险评级、日历视图。
4. 配置管理（CMDB）：CI 建模、关系拓扑、云资源发现（需启用适配器）。
5. 知识库（Knowledge）：文章、可信 RAG 检索、AI 辅助分诊。
6. 服务级别（SLA）：目标定义、 breaches 预警、度量。
7. 工作流引擎（BPMN）：可视化流程编排与任务调度。
8. 邮件工单闭环（Email-to-Ticket）：IMAP 接入→自动建单（演示/试用模式含脱敏种子）。
9. 多租户与鉴权：RBAC、租户上下文、连接器凭据加密。
10. 可观测：Prometheus/Grafana 可选监控套件。

**适用场景**：中小企业 IT 服务台、运维中台试点、DevOps/ITIL 落地教学、私有化 ITSM 替代评估。
**不适用/免责**：本镜像为社区开源版本试用，**非高可用架构**，未宣称 GA；生产级多副本、跨 AZ、备份演练需另行规划。

**规格建议**：最低 2c4G（试用），推荐 4c8G；系统盘 40G 起，建议 100G；依赖公网 80/443 与出网 443。

## 3. ACR 镜像推送流

```bash
# 0) 准备（用户提供 ACR 命名空间与凭证）
REGION=cn-hangzhou
NAMESPACE=heidsoft
REGISTRY=$REGION.aliyuncs.com/$NAMESPACE
VERSION=v1.6.58

# 1) 登录 ACR
docker login $REGION.aliyuncs.com -u <ACR_USER> -p <ACR_PASSWORD>

# 2) 本地构建三镜像（仓库根目录）
VERSION=$VERSION make build-prod   # 产出 itsm-backend / itsm-ai-service / itsm-frontend :$VERSION

# 3) 打 ACR tag 并推送
for img in itsm-backend itsm-ai-service itsm-frontend; do
  docker tag $img:$VERSION $REGISTRY/$img:$VERSION
  docker push $REGISTRY/$img:$VERSION
done

# 4) 在 deploy/.env.saas 设 REGISTRY=$REGISTRY/ 后，SaaS compose 即从 ACR 拉取
```

> 镜像 tag、API `/api/v1/version`、发布版本三者必须一致，避免「镜像已新、接口报旧」的漂移。

## 4. ECS 自定义镜像制作方法

```bash
# 1) 用一台「制作机」ECS（同地域）跑通 SaaS 部署（见 docs/deploy-saas.md §3–§7）
#    确认 https://$SAAS_DOMAIN/health 与 /api/v1/readyz 正常。

# 2) 固化编排文件到镜像内（便于用户登录后一键起停）：
#    将 deploy/ 目录与一份已填好非敏感项的 .env.saas 放入 /opt/itsm/，
#    并写入启动说明 README。注意：含密钥的 .env.saas 不要打进公共镜像——
#    用 .env.saas.example 占位，用户首次启动前自行填密钥（或经控制台参数注入）。

# 3) 在 ECS 控制台「实例 → 创建自定义镜像」，命名如 itsm-saas-v1.6.58。
#    镜像市场要求的检测项通常由平台脚本验证：
#      - 系统内可 docker --version
#      - /opt/itsm 下存在 docker-compose 文件
#      - 首次启动脚本可拉起服务

# 4) 提交镜像市场：控制台「镜像市场 → 接入镜像」，填 §2 文案，关联自定义镜像，
#    走服务商资质审核 → 上架。
```

**首次启动脚本建议**（用户开机自启）：
```bash
#!/usr/bin/env bash
# /opt/itsm/boot.sh —— 用户首次开机执行：填好 .env.saas 后运行
cd /opt/itsm
[ -f .env.saas ] || cp .env.saas.example .env.saas
echo "请先编辑 /opt/itsm/.env.saas 填入 DB/JWT/ADMIN 等密钥，然后重新运行本脚本"
grep -q "__CHANGE_ME" .env.saas && { echo "检测到未填密钥，已退出"; exit 1; }
docker compose --env-file .env.saas -f deploy/docker-compose.saas.yml up -d
```

## 5. 合规与口径红线（务必遵守）

- **协议**：Apache-2.0，镜像/文档须随附 LICENSE 与 NOTICE；不得闭源再分发或移除版权。
- **版本**：固定为 `v1.6.58` 本地 HEAD，**不升版、不宣称 GA**；文案用「试用/开源版本」。
- **表述**：禁止出现「替代 ServiceNow」「官方替代方案」「企业级替代」等；统一为「对标企业级 ITSM 能力」「开源 ITSM 平台」。
- **数据**：演示种子为脱敏占位数据（示例科技有限公司 / SUP-2026-001），不含任何真实凭据；公钥/密钥仅经环境变量注入，不得固化进镜像。
- **安全**：制作机与最终镜像均需 `yum/apt update` 打补丁；SSH 仅限密钥；80/443 入站放通、22 限制运维 IP。

## 6. 执行边界（本环境能力所限）

- ✅ 本会话已交付：SaaS 部署包、Caddyfile、env 模板、本上架材料与文案模板、ACR 推送/ECS 镜像制作 Runbook。
- ⏸️ 需用户提供阿里云凭证后方可执行：① `docker push` 到 ACR；② 在 ECS 控制台制作/提交自定义镜像；③ 提交镜像市场审核。
- 这些操作涉及用户云账号费用与资质，不在本会话自动执行范围；待用户授权并提供凭证（或委托其在控制台操作）后继续。

## 7. 后续（上架后）

- 维护 CHANGELOG 与版本号联动；新版本需重新制作镜像并走审核更新。
- 收集试用反馈，回流到 P1（空态/回执/演示租户）与 P2（营收叙事/计量）路线图（见 `plans/product-improvement-email-to-ticket-2026-11-15.md`）。
- 不代用户签署任何 SLA 或合规承诺；对外仅声明开源试用形态。
