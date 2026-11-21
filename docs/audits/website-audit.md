# 官网与商业化体验审计

> 基准 commit：`b3bfd748604041d5082a320ed6d5c447221ac76a`
> 证据来源：仓库文件 + 本次实际 DNS 解析与 webfetch；未修改任何线上内容。

## 1. 域名核实

| 域名 | 仓库声明 | 实际解析/加载 |
|:---|:---|:---|
| `cloudmesh.top` | `CNAME:1`、`website/CNAME:1`、`README.md:605`、`COMMUNITY.md:13`、`itsm-frontend/src/lib/brand.ts:23`、`website/index.html:299,409` | ✅ 解析到 `39.97.54.216`（与 `deploy/nginx/SECURITY-HARDENING.md:3` 一致），webfetch 可获取完整页面 |
| `cloudmes.top`（少 h） | 全仓零匹配 | ❌ `dig` 无 A 记录、curl 无法解析、webfetch transport error（NXDOMAIN/未配置） |

结论：规范域名是 `cloudmesh.top`；`cloudmes.top` 不是有效域名，也不是仓库笔误（仓库中从未出现）。

其他实际服务：`https://itsm.cloudmesh.top/login?demo=1` 提供真实应用演示壳；`https://cloudmesh.top/tickets/` 是标注"静态界面演示，不连接真实后端"的页面。

## 2. 发现项

```yaml
id: AUDIT-050
priority: P1
category: website/trust
status: confirmed
title: /trust.html 实为另一产品内容，指向 openclaw
evidence: 页面出现 git clone github.com/openclaw/openclaw、npm uninstall -g openclaw、support@openclaw.ai、ClawOps 公开安装包暂未提供；并给出"100% 代码开源 / 0 恶意代码"等绝对值
impact: 企业信任页面把访客导向无关包与外部邮箱，直接损害可信度
fix: 重写为 CloudMesh / heidsoft/itsm 的真实仓库、README 快速开始、SECURITY-HARDENING.md 事实、heidsoft@qq.com + GitHub Issues；删除绝对值；链接 /privacy/
```

```yaml
id: AUDIT-051
priority: P1
category: website/claims
status: confirmed
title: 英文页出现无来源的安装量数字与能力承诺
evidence: https://cloudmesh.top/index.en.html 卡片显示 12,580+/8,960+/15,680+/11,230+ Installs，仓库零匹配；同页称 7x24 技术支持、蓝绿/灰度发布
impact: 与中文页"都是能查证的事实"和 products/itsm.html"不含客户证言与效果承诺"自相矛盾，构成虚构规模
fix: 删除无来源数字或补来源+测量日期；英文页与中文页对齐
```

```yaml
id: AUDIT-052
priority: P1
category: website/source
status: confirmed
title: 版本控制的 website/ 与线上站点严重不一致，演示按钮自链接
evidence: website/index.html:2-6 标题为"ITSM - 企业级 IT 服务管理系统"、无 meta description/OG/JSON-LD；:299 演示 CTA href=https://cloudmesh.top（自己）；:255-257 移动端导航整体隐藏。线上是多页 CloudMesh 站点，含 title/meta/OG/canonical/hreflang + 3 组 JSON-LD
impact: 按仓库部署会发布劣质且功能错误的页面；真源不明
fix: 明确线上站点源码归属；若保留 website/ 则修正标题/meta/演示链接（指向 itsm.cloudmesh.top/login?demo=1）
```

```yaml
id: AUDIT-053
priority: P1
category: website/legal
status: confirmed
title: 隐私政策入口缺失、条款页 404、免费试用与付费 PoC 文案冲突
evidence: /privacy/ 存在但仅 /about/ 与 /index.en.html 链接；中文首页 footer 无隐私入口；/privacy.html、/terms.html、/terms/ 均 404；/contact/ 写"30 天免费试用，无需信用卡"，/pricing.html 写"付费 PoC 1.5–3万/项目"
fix: 全局 footer 增加隐私政策（及条款，如有）；统一试用与 PoC 口径；修复 /about/ 的 /#pricing 链接
```

```yaml
id: AUDIT-054
priority: P2
category: website/brand
status: confirmed
title: 品牌名在站点、应用与代码间碎片化
evidence: 站点 CloudMesh / CloudMesh 云网 / CMCloudMesh；/about/ 称"我们打造了 OpenClaw 和 ITSM 两款核心产品"；演示应用标题 AI-Native ITSM；brand.ts:16-24 规定 NAME 为 AI-Native ITSM/智能服务管理且禁止裸用 ITSM 作产品名
fix: 确定唯一产品名与母品牌，统一 JSON-LD/about/trust/product/演示应用/brand.ts
```

```yaml
id: AUDIT-055
priority: P2
category: website/claims
status: verify
title: 博客标题"30 家工厂的库存教训"可能被读作客户规模
evidence: 首页博客摘要标题含"30 家工厂"；仓库零匹配；products/itsm.html 明示不含客户证言
fix: 核实文章是否给出可查证来源，否则改为不带数字的示例表述
```

## 3. 正例（无虚构商业成果）

中文首页声明"都是能查证的事实，不是包装出来的数字"；产品页声明"不含客户证言与效果承诺"；`/open-source/` 的 84 Stars / 24 Forks 标注了 2026-11-14 测量日期且"以仓库实时状态为准"；`/pricing.html` 的价格标注"以 SOW 为准"；演示账号降级为只读 `audit_admin`（`SECURITY-HARDENING.md:8-13`）。
除 AUDIT-051/055 外，未发现虚构客户案例、客户数量、部署规模或商业化成果。
