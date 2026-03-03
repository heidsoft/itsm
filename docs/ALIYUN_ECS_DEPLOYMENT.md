# OpenClaw 阿里云 ECS 部署方案

**部署模式**: 一人一实例  
**基础设施**: 阿里云 ECS  
**状态**: ✅ 已完成

---

## 🏗️ 架构设计

### 部署架构

```
┌─────────────────────────────────────────┐
│         OpenClaw 部署管理系统            │
└─────────────────┬───────────────────────┘
                  │
        ┌─────────┴─────────┐
        │                   │
┌───────▼───────┐   ┌───────▼───────┐
│  用户 A - ECS  │   │  用户 B - ECS  │
│  独立实例      │   │  独立实例      │
│  独立域名      │   │  独立域名      │
│  独立账号      │   │  独立账号      │
└───────────────┘   └───────────────┘
```

### 资源配置

**专业版配置** (¥9,800/年):
- ECS: 2 核 4GB
- 带宽：3Mbps
- 系统盘：40GB SSD
- 独立域名：xxx.openclaw.cn
- 独立账号：用户名 + 密码

**企业版配置** (定制):
- ECS: 4 核 8GB 或更高
- 带宽：5Mbps 或更高
- 系统盘：100GB SSD
- 自定义域名：自定义
- 独立账号 + SSO

---

## 📦 ECS 实例管理

### 实例规格

| 套餐 | CPU | 内存 | 带宽 | 磁盘 | 价格 |
|------|-----|------|------|------|------|
| 社区版 | 1 核 | 1GB | 1Mbps | 20GB | ¥0 |
| 专业版 | 2 核 | 4GB | 3Mbps | 40GB | ¥9,800/年 |
| 企业版 | 4 核 | 8GB | 5Mbps | 100GB | 定制 |

### 实例命名规范

```
格式：openclaw-{user_id}-{env}

示例:
- openclaw-u001-prod  (用户 001 生产环境)
- openclaw-u002-test  (用户 002 测试环境)
```

---

## 🌐 域名管理

### 域名分配

**专业版**:
- 子域名：{username}.openclaw.cn
- 示例：zhangsan.openclaw.cn
- SSL 证书：自动申请（Let's Encrypt）

**企业版**:
- 自定义域名：www.company.com
- SSL 证书：自动申请或自带

### DNS 配置

```
A 记录：
{username}.openclaw.cn → {ECS 公网 IP}

CNAME 记录：
www.company.com → {username}.openclaw.cn
```

---

## 👤 账号管理

### 账号创建

**专业版**:
- 用户名：用户邮箱或自定义
- 密码：自动生成（16 位随机）
- 角色：Admin

**企业版**:
- 用户名：自定义
- 密码：自定义
- 角色：Admin + 多用户支持
- SSO：支持企业微信/钉钉集成

### 账号信息

```json
{
  "user_id": "u001",
  "username": "zhangsan",
  "email": "zhangsan@company.com",
  "password": "随机生成密码",
  "instance_id": "openclaw-u001-prod",
  "domain": "zhangsan.openclaw.cn",
  "created_at": "2026-03-01 12:00:00",
  "status": "active"
}
```

---

## 🔧 自动化部署流程

### 步骤 1: 用户下单

```
用户选择套餐
  ↓
填写配置信息
  ↓
支付订单
  ↓
创建部署任务
```

### 步骤 2: 自动创建 ECS

```bash
# 调用阿里云 API 创建 ECS
aliyun ecs CreateInstance \
  --InstanceType ecs.n4.small \
  --ImageId ubuntu_20_04_x64_20G_alibase_20210120.vhd \
  --SecurityGroupId sg-xxx \
  --InstanceName openclaw-u001-prod
```

### 步骤 3: 配置环境

```bash
# SSH 连接 ECS
ssh root@{ECS_IP}

# 安装 Docker
curl -fsSL https://get.docker.com | bash

# 拉取 OpenClaw 镜像
docker pull openclaw/openclaw:latest

# 启动容器
docker run -d \
  --name openclaw \
  -p 3000:3000 \
  -e USER_ID=u001 \
  -e DOMAIN=zhangsan.openclaw.cn \
  openclaw/openclaw:latest
```

### 步骤 4: 配置域名

```bash
# 配置 Nginx
cat > /etc/nginx/conf.d/openclaw.conf << 'NGINX'
server {
    listen 80;
    server_name zhangsan.openclaw.cn;
    
    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
NGINX

# 申请 SSL 证书
certbot --nginx -d zhangsan.openclaw.cn
```

### 步骤 5: 创建账号

```bash
# 调用 OpenClaw API 创建账号
curl -X POST https://zhangsan.openclaw.cn/api/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "zhangsan",
    "email": "zhangsan@company.com",
    "password": "随机密码",
    "role": "admin"
  }'
```

### 步骤 6: 发送通知

```
部署完成通知:
- 访问地址：https://zhangsan.openclaw.cn
- 用户名：zhangsan
- 密码：随机密码（邮件发送）
- 管理后台：https://zhangsan.openclaw.cn/dashboard
```

---

## 📊 管理后台功能

### 实例管理

**功能**:
- ✅ 查看 ECS 列表
- ✅ 启动/停止/重启
- ✅ 配置升级
- ✅ 数据备份
- ✅ 实例监控

**监控指标**:
- CPU 使用率
- 内存使用率
- 磁盘使用率
- 网络流量
- 在线状态

### 域名管理

**功能**:
- ✅ 查看域名列表
- ✅ 绑定自定义域名
- ✅ SSL 证书管理
- ✅ DNS 配置
- ✅ 域名续费提醒

### 账号管理

**功能**:
- ✅ 查看账号列表
- ✅ 重置密码
- ✅ 账号禁用/启用
- ✅ 权限管理
- ✅ 登录日志

---

## 💰 计费模式

### 专业版（¥9,800/年）

**包含**:
- ECS: 2 核 4GB 3Mbps 40GB
- 域名：*.openclaw.cn 子域名
- SSL 证书：免费
- 备份：每日自动备份
- 监控：基础监控
- 支持：专业支持

### 企业版（定制）

**包含**:
- ECS: 4 核 8GB 或更高
- 域名：自定义域名
- SSL 证书：免费或自带
- 备份：每小时备份
- 监控：高级监控
- 支持：专属支持 + SLA

---

## 🔒 安全保障

### 网络安全
- ✅ 安全组配置
- ✅ 防火墙规则
- ✅ DDoS 防护
- ✅ SSL 加密

### 数据安全
- ✅ 数据加密存储
- ✅ 每日自动备份
- ✅ 异地灾备
- ✅ 数据隔离

### 账号安全
- ✅ 强密码策略
- ✅ 双因素认证（企业版）
- ✅ 登录日志
- ✅ 异常登录提醒

---

## 📈 监控告警

### 监控指标

| 指标 | 告警阈值 | 通知方式 |
|------|---------|---------|
| CPU 使用率 | >80% | 邮件 + 短信 |
| 内存使用率 | >90% | 邮件 + 短信 |
| 磁盘使用率 | >85% | 邮件 |
| 服务宕机 | 立即 | 短信 + 电话 |

### 告警通知

**专业版**:
- 邮件通知
- 短信通知

**企业版**:
- 邮件通知
- 短信通知
- 电话通知
- 企业微信/钉钉

---

## 🌐 访问地址

**部署服务**: https://cloudmesh.top/deploy/  
**管理后台**: https://cloudmesh.top/deploy/dashboard/  
**用户实例**: https://{username}.openclaw.cn

---

## ✅ 实施清单

### 基础设施
- [x] 阿里云账号配置
- [x] ECS 实例模板
- [x] 安全组配置
- [x] 镜像仓库

### 自动化
- [x] 部署脚本
- [x] 域名自动配置
- [x] SSL 自动申请
- [x] 账号自动创建

### 管理系统
- [x] 实例管理页面
- [x] 域名管理页面
- [x] 账号管理页面
- [x] 监控告警页面

### 文档
- [x] 部署文档
- [x] 用户手册
- [x] API 文档
- [x] FAQ

---

**OpenClaw 阿里云 ECS 部署方案已完成！** 🚀

**维护者**: 运维团队  
**部署时间**: 2026-03-01  
**下次审查**: 2026-04-01
