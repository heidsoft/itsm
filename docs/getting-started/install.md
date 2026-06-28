# 安装

> 适用版本：ITSM v1.0+
> 阅读时间：约 5 分钟
> 难度：⭐⭐（需要 Docker 基础）

## 1. 环境要求

| 组件 | 最低版本 | 推荐版本 | 备注 |
|:---|:---|:---|:---|
| **Docker** | 24.0 | 27.0+ | 包含 Compose v2 |
| **Docker Compose** | v2.20 | v2.27+ | — |
| **磁盘** | 10 GB | 30 GB SSD | 包含 PostgreSQL 数据 |
| **内存** | 4 GB | 8 GB+ | 跑 LLM 建议 16 GB |
| **CPU** | 2 cores | 4 cores+ | RAG embedding 吃 CPU |

操作系统：Linux / macOS / Windows（WSL2）

## 2. 克隆仓库

```bash
git clone https://github.com/itsm/itsm.git
cd itsm
```

## 3. 配置环境变量

```bash
cp .env.example .env
```

最小配置（必须修改）：

```dotenv
# 数据库密码（生产环境务必使用强密码）
DB_PASSWORD=ChangeMeToStrongPassword123!

# JWT 密钥（生产环境务必使用 64+ 字符随机串）
JWT_SECRET=$(openssl rand -hex 32)

# 初始管理员密码（生产环境务必修改）
ADMIN_PASSWORD=ChangeMe123!

# 日志级别
LOG_LEVEL=info  # debug | info | warn | error
```

完整配置项说明见 `.env.example` 注释。

## 4. 启动服务（开发模式）

```bash
docker compose -f docker-compose.dev.yml up -d
```

首次启动会：

1. 拉取镜像（约 5-10 分钟，取决于网速）
2. 初始化 PostgreSQL schema（[Ent migrations](itsm-backend/ent/migrate)）
3. 播种基础数据（角色、权限、菜单）
4. 创建默认管理员账户 `admin / admin123`

查看启动进度：

```bash
docker compose -f docker-compose.dev.yml logs -f itsm-backend
```

看到 `Server started on :8090` 即后端就绪。

## 5. 验证

```bash
# 健康检查
curl http://localhost:8090/api/v1/health
# 期望返回：{"code":0,"message":"success","data":{"status":"ok",...}}

# 前端访问
open http://localhost:3000  # macOS
xdg-open http://localhost:3000  # Linux
start http://localhost:3000  # Windows
```

默认登录：`admin / admin123`

## 6. 生产部署

⚠️ **生产环境必须使用独立的 `.env.prod` 文件**：

```bash
# 1. 生成强密码和密钥
openssl rand -hex 32 > JWT_SECRET.txt
openssl rand -hex 16 > DB_PASSWORD.txt
openssl rand -hex 16 > REDIS_PASSWORD.txt

# 2. 创建 .env.prod（参考 .env.prod.example）
cp .env.prod.example .env.prod
# 编辑 .env.prod，填入上述密钥

# 3. 启动（必须显式传入 --env-file）
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d
```

详见 [运维 - 部署](../operations/deployment.md)。

## 7. 卸载

```bash
# 停止并删除容器（保留数据卷）
docker compose -f docker-compose.dev.yml down

# 停止并删除容器 + 数据卷（⚠️ 会清空所有数据）
docker compose -f docker-compose.dev.yml down -v
```

## 下一步

- [本地开发](local-dev.md)：如何修改代码、运行测试
- [第一个工单](first-ticket.md)：走通端到端流程
- [架构总览](../architecture/overview.md)：理解模块划分
