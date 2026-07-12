# API 契约检查工具

## 概述

本工具用于静态检查前后端 API 路径的一致性，防止前端调用了后端未注册的 API 路径导致生产环境错误。

## 功能特性

- **前后端路径匹配检查**：对比前端 API 调用与后端路由注册
- **路径规范化处理**：支持动态参数路径（如 `/tickets/:id`）的匹配
- **命名规范检查**：检测 snake_case 与 camelCase 混用问题
- **CI/CD 集成**：可作为 GitHub Actions 工作流自动执行

## 使用方法

### 命令行使用

```bash
# 基本检查
node scripts/check-api-paths.js

# 详细输出
node scripts/check-api-paths.js --verbose

# JSON 格式输出（用于CI/CD）
node scripts/check-api-paths.js --json
```

### GitHub Actions 自动检查

工作流文件：`.github/workflows/api-contract-check.yml`

在以下情况下自动触发：
- 向 `main` 或 `develop` 分支推送时
- 修改 `itsm-frontend/src/lib/**`、`itsm-backend/router/**` 等关键文件时
- 手动触发

## 检查规则

### 1. 路径匹配规则

| 前端路径 | 后端路径 | 匹配结果 |
|---------|---------|---------|
| `/api/v1/tickets/123` | `/api/v1/tickets/:id` | ✅ 匹配 |
| `/api/v1/tickets/abc-123` | `/api/v1/tickets/:id` | ✅ 匹配 |
| `/api/v1/unknown/123` | (无) | ❌ 缺失 |

### 2. 命名规范规则

- ✅ 使用 camelCase：`ticketNumber`, `assigneeId`
- ⚠️ 警告使用 snake_case：`ticket_number`, `assignee_id`

## 退出码

| 退出码 | 含义 |
|-------|------|
| 0 | 所有路径匹配或仅有警告 |
| 1 | 检测到缺失的路径 |

## 配置项

在 `scripts/check-api-paths.js` 中修改：

```javascript
// 后端路由文件路径
const BACKEND_ROUTER_PATH = path.join(__dirname, '../itsm-backend/router/router.go');

// 前端API目录
const FRONTEND_API_DIR = path.join(__dirname, '../itsm-frontend/src/lib');
const FRONTEND_API_SUBDIRS = ['api', 'services'];
```

## 示例输出

```
🔍 ITSM API Path Static Check

Parsing backend routes...
  Found 150 backend routes

Extracting frontend API paths...
  Found 200 frontend API paths

Checking paths...

📊 Results:
  Matched: 195
  Missing: 5

⚠️  Some frontend paths are not registered in the backend!
    Either add the backend route or fix the frontend path.
```

## 相关文件

- `scripts/check-api-paths.js` - 检查脚本主文件
- `.github/workflows/api-contract-check.yml` - CI/CD 工作流
- `itsm-backend/router/router.go` - 后端路由定义
- `itsm-frontend/src/lib/api/` - 前端 API 调用
