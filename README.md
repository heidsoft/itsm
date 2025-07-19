# ITSM System

一个完整的 IT 服务管理系统，包含前端（Next.js）和后端（Go）组件。

## 项目结构

```
itsm/
├── itsm-backend/          # Go 后端服务
├── itsm-prototype/        # Next.js 前端应用
├── Makefile              # 构建和运行脚本
└── README.md            # 项目文档
```

## 最近修复的问题

### 后端 (itsm-backend)

1. **编译冲突修复**
   - 添加了构建标签来解决多个 main 函数冲突
   - 迁移脚本现在使用 `//go:build migrate` 标签
   - 用户创建脚本使用 `//go:build create_user` 标签
   - 主程序使用 `//go:build !migrate && !create_user` 标签

2. **依赖管理**
   - 运行 `go mod tidy` 更新依赖
   - 确保所有依赖版本兼容

### 前端 (itsm-prototype)

1. **ESLint 错误修复**
   - 创建了智能修复脚本 `scripts/smart-fix-lint.js`
   - 自动检测和修复未使用的导入
   - 修复了组件未定义的问题
   - 优化了类型定义

2. **代码质量改进**
   - 修复了 ErrorBoundary 组件的类型问题
   - 改进了 ESLint 配置
   - 减少了 `any` 类型的使用

## 快速开始

### 使用 Makefile（推荐）

```bash
# 安装依赖并构建项目
make setup

# 运行开发环境（同时启动前后端）
make dev

# 仅运行后端
make run-backend

# 仅运行前端
make run-frontend

# 修复代码问题
make lint-fix

# 清理构建文件
make clean
```

### 手动运行

#### 后端

```bash
cd itsm-backend
go mod tidy
go build -o itsm-backend .
./itsm-backend
```

#### 前端

```bash
cd itsm-prototype
npm install
npm run dev
```

## 开发工具

### 构建标签使用

后端项目使用构建标签来区分不同的可执行文件：

```bash
# 构建主程序
go build -o itsm-backend .

# 构建迁移工具
go build -tags migrate -o migrate-tool .

# 构建用户创建工具
go build -tags create_user -o create-user .
```

### 代码质量

前端项目包含自动化的代码质量工具：

- ESLint 配置优化
- 智能导入修复脚本
- TypeScript 类型检查

## 配置

### 后端配置

配置文件位于 `itsm-backend/config.yaml`：

```yaml
database:
  host: localhost
  port: 5432
  user: dev
  password: "123456!@#$%^"
  dbname: itsm
  sslmode: disable

server:
  port: 8080
  mode: debug

jwt:
  secret: "itsm-dev-secret-key-2024"
  expire_time: 900
  refresh_expire_time: 604800
```

### 前端配置

前端 API 配置在 `itsm-prototype/src/app/lib/api-config.ts` 中：

```typescript
export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
```

## 开发指南

### 添加新功能

1. 在后端添加新的控制器和服务
2. 在前端添加对应的 API 调用
3. 创建相应的 UI 组件
4. 更新路由配置

### 代码规范

- 使用 TypeScript 进行类型检查
- 遵循 ESLint 规则
- 使用 Prettier 格式化代码
- 编写单元测试

## 故障排除

### 常见问题

1. **编译错误**

   ```bash
   make clean
   make setup
   ```

2. **依赖问题**

   ```bash
   make install-deps
   ```

3. **代码质量问题**

   ```bash
   make lint-fix
   ```

### 日志查看

- 后端日志：查看控制台输出
- 前端日志：浏览器开发者工具

## 贡献

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 许可证

MIT License
