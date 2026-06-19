# 30分钟搭建一套完整的ITSM系统

## Docker一键部署，从零到生产可用的完整指南

---

很多人对"企业级系统"有个误解：觉得部署起来一定很复杂，需要专业运维团队，需要几天甚至几周时间。

这是个误区。

借助Docker容器化和现代化的架构设计，ITSM系统可以做到**5分钟启动，30分钟完成基础配置投入使用**。

本文将手把手带你完成整个过程。无论你是想先体验一下产品功能，还是想在生产环境正式部署，都能找到对应的指引。

---

## 第一部分：部署方案选型

### 1.1 方案对比

ITSM提供三种部署方案：

| 方案 | 适用场景 | 复杂度 | 性能 | 维护成本 |
|:---|:---|:---:|:---:|:---:|
| **Docker一键部署** | 快速体验、开发测试 | 极简 | 中 | 低 |
| **Docker Compose** | 中小规模生产 | 简单 | 中高 | 中 |
| **Kubernetes** | 大规模、高可用 | 复杂 | 高 | 高 |

对于大多数场景，**Docker Compose部署**是最佳选择。它兼顾了部署的简便性和运行时的性能，足以支撑中小企业500人规模的IT团队使用。

### 1.2 硬件要求

Docker Compose部署的硬件要求：

| 资源 | 最低配置 | 推荐配置 |
|:---|:---|:---|
| CPU | 2核 | 4核 |
| 内存 | 4GB | 8GB |
| 磁盘 | 20GB | 50GB |
| 操作系统 | Ubuntu 20.04+ / CentOS 8+ / macOS 12+ | Ubuntu 22.04 LTS |

建议使用SSD硬盘，能显著提升数据库性能。

### 1.3 网络要求

服务器需要能访问以下地址：

- Docker Hub（拉取镜像）
- GitHub（代码获取）
- OpenAI/Anthropic API（如果启用AI功能，需要访问海外API）

如果服务器在私有网络环境，需要提前准备Docker镜像仓库和内网AI模型部署方案。

---

## 第二部分：快速体验（5分钟）

### 2.1 安装Docker

首先检查Docker是否已安装：

```bash
docker --version
docker-compose --version
```

如果没有安装，按以下步骤安装：

**Ubuntu系统**：

```bash
# 安装依赖
sudo apt-get update
sudo apt-get install -y apt-transport-https ca-certificates curl software-properties-common

# 添加Docker官方GPG密钥
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

# 添加Docker仓库
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# 安装Docker
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# 启动Docker
sudo systemctl start docker
sudo systemctl enable docker

# 添加当前用户到docker组（免sudo）
sudo usermod -aG docker $USER
```

**macOS系统**：

下载Docker Desktop for Mac：https://www.docker.com/products/docker-desktop/

安装后启动Docker Desktop即可。

### 2.2 一键启动

克隆项目代码：

```bash
git clone https://github.com/heidsoft/itsm.git
cd itsm
```

使用Docker Compose一键启动所有服务：

```bash
make dev-up
```

这个命令会：

1. 拉取必要的Docker镜像
2. 创建Docker网络
3. 启动PostgreSQL数据库
4. 启动Redis缓存
5. 启动后端服务
6. 启动前端服务

启动过程大约需要2-3分钟（取决于网络速度）。首次启动会拉取镜像，后续启动会快很多。

### 2.3 访问系统

启动成功后，在浏览器中打开：

- **前端地址**：http://localhost:3000
- **后端API地址**：http://localhost:8080
- **API文档地址**：http://localhost:8080/swagger

使用默认账号登录：

- **用户名**：admin
- **密码**：admin123

登录后即可看到ITSM的管理界面。

---

## 第三部分：基础配置（15分钟）

首次登录后，需要完成一些基础配置才能正式使用。

### 3.1 创建部门结构

在"系统管理 → 组织架构"中创建部门结构。

典型的IT团队部门划分：

```
IT部
├── 基础架构组
│   ├── 服务器运维
│   ├── 网络安全
│   └── 桌面支持
├── 应用开发组
│   ├── 后端开发
│   └── 前端开发
└── 项目管理组
```

部门结构用于工单的归属和分配。

### 3.2 导入用户

在"系统管理 → 用户管理"中创建用户账号。

支持以下导入方式：

**手动创建**：适合少量用户，直接在界面填写信息。

**批量导入**：准备CSV文件，一次性导入大量用户。CSV格式：

```csv
username,email,department,role,phone
zhang_san,zhangsan@example.com,基础架构组,engineer,13800138001
li_si,lisi@example.com,应用开发组,engineer,13800138002
wang_wu,wangwu@example.com,项目管理组,manager,13800138003
```

**LDAP同步**：如果企业有LDAP/AD，可以配置自动同步。用户体系与公司现有目录服务打通。

### 3.3 配置角色权限

系统预置了几个常用角色：

| 角色 | 权限范围 |
|:---|:---|
| admin | 系统全部功能 |
| manager | 团队管理、工单审批 |
| engineer | 工单处理、知识库编辑 |
| user | 提交工单、查看服务目录 |

如果预置角色不满足需求，可以在"系统管理 → 角色权限"中创建自定义角色，精细控制每个功能模块的访问权限。

### 3.4 配置通知渠道

在"系统管理 → 通知设置"中配置通知渠道。

ITSM支持多种通知方式：

**邮件通知**：配置SMTP服务器，发送工单状态变更邮件。

```yaml
email:
  smtp_host: smtp.example.com
  smtp_port: 465
  smtp_user: notifications@example.com
  smtp_password: xxxxxxxx
  from_address: itsm@example.com
```

**WebHook**：当工单状态变更时，向指定URL发送POST请求。可以对接企业微信、钉钉、飞书等办公软件。

```json
{
  "event": "ticket_created",
  "ticket_id": "T-2024-001",
  "title": "无法访问OA系统",
  "priority": "high",
  "created_by": "zhangsan"
}
```

**短信通知**：紧急工单可以触发短信通知，需要配置短信网关（阿里云、腾讯云等）。

---

## 第四部分：工单流程配置（10分钟）

### 4.1 配置工单类型

在"工单管理 → 工单类型"中配置工单类型。

系统支持多种工单类型：

| 类型 | 适用场景 | 默认流程 |
|:---|:---|:---|
| 事件 | 系统故障、异常情况 | 事件处理流程 |
| 请求 | 服务申请、权限开通 | 请求审批流程 |
| 问题 | 复杂问题、根因分析 | 问题分析流程 |
| 变更 | 系统变更、配置调整 | 变更审批流程 |

每个工单类型可以设置：

- 默认优先级
- 关联的服务目录项
- 自动分配规则
- 升级策略

### 4.2 配置SLA策略

SLA（Service Level Agreement）定义了工单的处理时限。

在"工单管理 → SLA策略"中创建SLA：

```yaml
名称: P1紧急响应
适用条件: priority = 'critical'
响应时限: 30分钟
处理时限: 4小时
升级规则: |
  如果30分钟内未响应
  升级给 二线技术负责人

名称: P2高优处理
适用条件: priority = 'high'
响应时限: 2小时
处理时限: 8小时
升级规则: |
  如果2小时内未响应
  升级给 运维主管
```

系统会自动监控SLA合规状态。当工单即将超时时，会发送预警通知。

### 4.3 配置工作流

在"工作流 → 流程设计"中设计工单处理流程。

对于刚上手的用户，建议先用默认流程。默认流程包括：

```
提交工单 → 客服受理 → 一线处理
    ↓(无法解决)
二线处理 → 问题解决 → 关闭工单
    ↓(需要变更)
变更申请 → 变更审批 → 实施变更 → 验证 → 关闭
```

流程设计器支持可视化拖拽，可以根据企业实际需求调整审批节点、处理人、超时时间等。

---

## 第五部分：服务目录配置

### 5.1 创建服务目录

在"服务目录 → 目录管理"中创建服务项。

一个典型的服务目录结构：

```
服务目录
├── IT支持
│   ├── 账号申请
│   │   ├── 邮箱账号开通
│   │   ├── OA账号开通
│   │   └── 权限申请
│   ├── 设备问题
│   │   ├── 电脑故障申报
│   │   ├── 打印机问题
│   │   └── 网络故障
│   └── 软件支持
│       ├── 软件安装申请
│       └── 许可证申请
├── 系统运维
│   ├── 服务器监控
│   ├── 数据库维护
│   └── 安全事件上报
└── 变更申请
    ├── 普通变更
    └── 紧急变更
```

每个服务项可以配置：

- 审批流程
- 处理团队（SLG）
- 服务时限（SLA）
- 关联知识库分类

### 5.2 启用自助门户

在"服务目录 → 门户设置"中启用自助服务门户。

自助门户面向终端用户，用户可以：

- 自行浏览服务目录
- 在线提交服务申请
- 追踪工单处理进度
- 评价服务质量
- 搜索知识库

门户支持白名单配置，只允许特定用户访问。

---

## 第六部分：AI功能启用（可选）

### 6.1 配置AI服务

如果需要启用AI功能（智能分类、摘要生成、RAG知识库），需要配置AI服务提供商。

在"系统管理 → AI设置"中配置：

```yaml
AI配置:
  # 选择AI提供商
  provider: openai  # openai | claude | ollama | dashscope

  # OpenAI配置
  openai:
    api_key: sk-xxxxxxxxxxxx
    model: gpt-4o-mini  # gpt-4o | gpt-4o-mini | gpt-4-turbo
    base_url: https://api.openai.com/v1

  # Claude配置
  claude:
    api_key: sk-ant-xxxxxxxxxxxx
    model: claude-3-5-haiku-latest

  # 本地Ollama配置（无需API Key，数据完全本地）
  ollama:
    base_url: http://localhost:11434
    model: qwen2.5:7b
```

### 6.2 私有化AI部署

对于数据敏感的企业，可以使用Ollama进行私有化部署。

Ollama允许在本地运行大语言模型，数据完全不外传。

```bash
# 安装Ollama
curl -fsSL https://ollama.com/install.sh | sh

# 下载模型
ollama pull qwen2.5:7b

# 启动服务
ollama serve
```

在AI设置中选择Ollama提供方，配置好模型名称即可。

### 6.3 知识库RAG配置

启用RAG知识库需要额外配置向量数据库。

```yaml
RAG配置:
  # 向量数据库类型
  vector_db: chroma  # chroma | milvus | qdrant

  # Chroma配置（默认，内置）
  chroma:
    persist_dir: ./data/chroma

  # Embedding模型
  embedding:
    provider: openai  # openai | local
    model: text-embedding-3-small
    local_model: m3e-base  # 如果使用本地模型
```

---

## 第七部分：生产环境部署

### 7.1 生产环境检查清单

部署到生产环境前，确认以下事项：

**服务器检查**：

- [ ] 硬件配置满足要求（4核CPU、8GB内存、50GB SSD）
- [ ] 操作系统是推荐的版本（Ubuntu 22.04 LTS）
- [ ] 服务器有公网IP或内网可达
- [ ] 域名已解析到服务器

**数据库检查**：

- [ ] PostgreSQL版本 >= 14
- [ ] 数据库字符集是UTF8
- [ ] 有定期备份策略
- [ ] 连接数配置足够（建议100+）

**安全检查**：

- [ ] 防火墙只开放必要端口（80/443）
- [ ] 数据库只允许内网访问
- [ ] 开启HTTPS（使用Let's Encrypt证书）
- [ ] 修改默认管理员密码
- [ ] 配置登录失败锁定策略

**反向代理配置**（推荐使用Nginx）：

```nginx
server {
    listen 80;
    server_name itsm.example.com;

    # 重定向到HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name itsm.example.com;

    # SSL证书
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    # 前端代理
    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # 后端API代理
    location /api {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # Swagger文档
    location /swagger {
        proxy_pass http://localhost:8080/swagger;
    }
}
```

### 7.2 数据备份策略

生产环境必须配置数据备份。

**数据库备份**：

```bash
# 每天凌晨2点执行全量备份
0 2 * * * pg_dump -U postgres -d itsm > /backup/itsm_$(date +\%Y\%m\%d).sql

# 保留30天备份
find /backup -name "itsm_*.sql" -mtime +30 -delete
```

**文件备份**：

```bash
# 备份上传文件
0 3 * * * tar -czf /backup/uploads_$(date +\%Y\%m\%d).tar.gz /app/uploads

# 备份配置
0 3 * * * tar -czf /backup/config_$(date +\%Y\%m\%d).tar.gz /app/config
```

### 7.3 监控告警

推荐使用Prometheus + Grafana监控ITSM系统。

关键监控指标：

- 服务可用性（前端、后端API）
- 数据库连接数
- 响应时间分布
- 错误率
- CPU/内存/磁盘使用率

告警阈值建议：

| 指标 | 警告值 | 严重值 |
|:---|:---:|:---:|
| 服务不可用 | - | 持续1分钟 |
| API响应时间 | > 2秒 | > 5秒 |
| 错误率 | > 1% | > 5% |
| CPU使用率 | > 70% | > 90% |
| 内存使用率 | > 75% | > 90% |
| 磁盘使用率 | > 70% | > 85% |

---

## 第八部分：常见问题排查

### 8.1 服务启动失败

**PostgreSQL连接失败**：

```bash
# 检查容器状态
docker ps -a | grep postgres

# 查看日志
docker logs itsm-postgres

# 常见问题：数据目录权限
chown -R 999:999 /data/postgres
```

**后端启动失败**：

```bash
# 查看后端日志
docker logs itsm-backend

# 常见问题：数据库迁移
docker exec -it itsm-backend sh -c "go run -tags migrate main.go"
```

### 8.2 前端访问白屏

检查浏览器控制台错误：

- 如果是网络错误，检查nginx代理配置
- 如果是API请求404，检查后端服务是否正常运行
- 如果是ES Module错误，可能是缓存问题，清除浏览器缓存或重启前端容器

```bash
# 重启前端容器
docker-compose restart frontend
```

### 8.3 性能问题

**数据库慢查询**：

```sql
-- 查看慢查询
SELECT * FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 10;
```

**增加数据库连接池**：

```yaml
# docker-compose.yml
services:
  backend:
    environment:
      DB_MAX_OPEN_CONNS: 50
      DB_MAX_IDLE_CONNS: 10
```

---

## 总结：快速开始

ITSM系统的Docker部署非常简单：

| 步骤 | 操作 | 预计时间 |
|:---|:---|:---:|
| 1 | 安装Docker | 5分钟 |
| 2 | 克隆代码 + 一键启动 | 5分钟 |
| 3 | 访问系统登录 | 即时 |
| 4 | 配置部门和用户 | 10分钟 |
| 5 | 配置工单类型和SLA | 5分钟 |
| 6 | 配置服务目录 | 10分钟 |
| 7 | （可选）启用AI功能 | 5分钟 |

**全程约30-40分钟**，一套完整的ITSM系统就可以投入使用。

现在就去体验吧！

👉 **GitHub地址**：https://github.com/heidsoft/itsm

👉 **在线文档**：https://docs.cloudmesh.top/itsm

👉 **社区讨论**：https://github.com/heidsoft/itsm/discussions

**本系列全部文章**：

- 第1篇：《为什么你的IT团队每天像在救火？》
- 第2篇：《从混乱到有序：AI驱动的智能化工单分类实战》
- 第3篇：《变更管理不再是噩梦：BPMN工作流设计器详解》
- 第4篇：《让知识流动起来：RAG知识库搭建指南》
- 第5篇：《多租户架构：MSP服务商的高效运营之道》
- 第6篇：《30分钟搭建一套完整的ITSM系统》

如果部署过程中遇到任何问题，欢迎在GitHub Issues中提问，或者在评论区留言交流。祝部署顺利！
