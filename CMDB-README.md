# ITSM CMDB 系统

企业级配置管理数据库(CMDB)系统，支持多云环境、网络设备、Kubernetes等资源的自动发现和管理。

## 🚀 快速开始

### 系统要求

- Docker 20.0+
- Docker Compose 2.0+
- 8GB+ 内存
- 20GB+ 磁盘空间

### 一键启动

```bash
# 克隆项目
git clone <repository-url>
cd itsm

# 启动系统
./start-cmdb.sh start
```

### 访问地址

- **前端界面**: http://localhost
- **API文档**: http://localhost/api/docs
- **监控面板**: http://localhost:3001 (admin/admin)
- **指标监控**: http://localhost:9090

## 📋 功能特性

### 核心功能

- ✅ **配置项管理**: 支持多种CI类型的创建、更新、删除
- ✅ **关系管理**: CI之间的依赖关系映射和可视化
- ✅ **自动发现**: 网络扫描、云平台API、Kubernetes集成
- ✅ **服务映射**: 业务服务到基础设施的完整映射
- ✅ **影响分析**: 变更影响范围分析和风险评估
- ✅ **多租户**: 支持多租户数据隔离

### 发现能力

- 🌐 **网络发现**: SNMP、SSH、WMI协议支持
- ☁️ **云平台**: AWS、阿里云、腾讯云、火山云
- 🐳 **容器平台**: Kubernetes、Docker
- 📊 **监控集成**: Zabbix、Prometheus、Nagios

### 可视化

- 📈 **实时仪表板**: CI统计、健康状态、发现状态
- 🗺️ **服务地图**: D3.js驱动的交互式拓扑图
- 📊 **影响分析**: 变更影响可视化
- 📱 **响应式设计**: 支持桌面和移动设备

## 🏗️ 系统架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   前端界面       │    │   API网关       │    │   CMDB服务      │
│   (Next.js)     │◄──►│   (Nginx)       │◄──►│   (Go)          │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │
                       ┌─────────────────┐             │
                       │   发现引擎       │◄────────────┘
                       │   (多适配器)     │
                       └─────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        │                       │                       │
┌─────────────┐        ┌─────────────┐        ┌─────────────┐
│  网络发现    │        │  云平台发现  │        │  K8s发现     │
│  (SNMP/SSH) │        │  (API集成)  │        │  (API集成)   │
└─────────────┘        └─────────────┘        └─────────────┘
```

## 📦 部署架构

### 服务组件

| 服务 | 端口 | 描述 |
|------|------|------|
| Nginx | 80/443 | 反向代理和负载均衡 |
| Frontend | 3000 | Next.js前端应用 |
| CMDB Backend | 8080 | Go后端API服务 |
| PostgreSQL | 5432 | 主数据库 |
| Redis | 6379 | 缓存和会话存储 |
| Prometheus | 9090 | 指标收集 |
| Grafana | 3001 | 监控可视化 |

### 数据存储

```
volumes/
├── postgres_data/     # PostgreSQL数据
├── redis_data/        # Redis数据
├── prometheus_data/   # Prometheus指标数据
└── grafana_data/      # Grafana配置数据
```

## 🔧 配置说明

### 环境变量

```bash
# 数据库配置
DATABASE_URL=postgres://postgres:password@localhost:5432/itsm_cmdb?sslmode=disable

# Redis配置
REDIS_URL=redis://localhost:6379

# 服务配置
PORT=8080
GIN_MODE=release

# 前端配置
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### 发现源配置

#### 网络发现

```json
{
  "ip_ranges": ["192.168.1.0/24", "10.0.0.0/16"],
  "snmp_community": "public",
  "snmp_version": "2c",
  "snmp_timeout": 5,
  "parallel_limit": 50
}
```

#### 云平台发现

```json
{
  "provider": "aliyun",
  "region": "cn-hangzhou",
  "access_key": "your-access-key",
  "secret_key": "your-secret-key"
}
```

#### Kubernetes发现

```json
{
  "kubeconfig_path": "/path/to/kubeconfig",
  "namespaces": ["default", "kube-system"],
  "resource_types": ["pods", "services", "deployments"]
}
```

## 🛠️ 开发指南

### 本地开发

```bash
# 后端开发
cd itsm-backend
go mod tidy
go run cmd/cmdb/main.go

# 前端开发
cd itsm-frontend
npm install
npm run dev
```

### 数据库迁移

```bash
# 生成迁移文件
go run -mod=mod entgo.io/ent/cmd/ent generate ./ent/schema

# 运行迁移
go run cmd/cmdb/main.go migrate
```

### API文档

API文档使用Swagger生成，启动服务后访问：
- Swagger UI: http://localhost:8080/swagger/index.html
- OpenAPI JSON: http://localhost:8080/swagger/doc.json

## 📊 监控和运维

### 健康检查

```bash
# 服务健康检查
curl http://localhost/health

# 数据库连接检查
curl http://localhost/api/v1/health/db

# 发现引擎状态
curl http://localhost/api/v1/cmdb/discovery/status
```

### 日志查看

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f cmdb-backend

# 查看实时日志
tail -f logs/cmdb.log
```

### 性能监控

- **Grafana仪表板**: 预配置的CMDB监控面板
- **Prometheus指标**: 自定义业务指标收集
- **应用性能**: 响应时间、错误率、吞吐量

## 🔒 安全配置

### 认证授权

- JWT Token认证
- RBAC权限控制
- API访问限制
- 数据加密传输

### 网络安全

```bash
# 启用HTTPS
./start-cmdb.sh start --ssl

# 配置防火墙
ufw allow 80/tcp
ufw allow 443/tcp
ufw deny 8080/tcp  # 隐藏后端端口
```

## 🚨 故障排除

### 常见问题

1. **服务启动失败**
   ```bash
   # 检查端口占用
   netstat -tlnp | grep :8080
   
   # 检查Docker状态
   docker-compose ps
   ```

2. **数据库连接失败**
   ```bash
   # 检查数据库状态
   docker-compose exec postgres pg_isready
   
   # 重置数据库
   ./start-cmdb.sh clean
   ```

3. **发现任务失败**
   ```bash
   # 查看发现日志
   docker-compose logs -f cmdb-backend | grep discovery
   
   # 检查网络连通性
   docker-compose exec cmdb-backend ping target-host
   ```

### 性能优化

1. **数据库优化**
   - 定期执行VACUUM
   - 优化查询索引
   - 配置连接池

2. **缓存优化**
   - Redis缓存配置
   - 查询结果缓存
   - 静态资源缓存

## 📚 API参考

### CI管理

```bash
# 创建CI
POST /api/v1/cmdb/cis
{
  "name": "web-server-01",
  "sys_class_name": "cmdb_ci_server",
  "environment": "production"
}

# 查询CI
GET /api/v1/cmdb/cis?class_name=cmdb_ci_server&limit=20

# 搜索CI
GET /api/v1/cmdb/cis/search?q=web-server

# 获取关系
GET /api/v1/cmdb/cis/{sys_id}/relationships
```

### 发现管理

```bash
# 创建发现源
POST /api/v1/cmdb/discovery/sources
{
  "name": "网络扫描",
  "source_type": "network_scan",
  "discovery_config": {...}
}

# 运行发现
POST /api/v1/cmdb/discovery/sources/{source_id}/run
```

## 🤝 贡献指南

1. Fork项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建Pull Request

## 📄 许可证

本项目采用MIT许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 📞 支持

- 📧 邮箱: support@example.com
- 💬 讨论: [GitHub Discussions](https://github.com/your-repo/discussions)
- 🐛 问题: [GitHub Issues](https://github.com/your-repo/issues)

---

**ITSM CMDB** - 让IT资产管理更简单、更智能！
