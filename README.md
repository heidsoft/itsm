# ITSM 系统

基于 Go + Gin + Ent + Next.js 的IT服务管理系统

## 项目结构

```
itsm/
├── itsm-backend/          # Go 后端服务
│   ├── ent/               # Ent ORM 生成的代码
│   ├── controller/        # 控制器层
│   ├── service/           # 业务逻辑层
│   ├── middleware/        # 中间件
│   ├── config/            # 配置文件
│   └── tests/             # 测试文件
├── itsm-prototype/        # Next.js 前端应用
└── .trae/                 # Trae AI 项目规则

```

## 开发环境要求

- Go 1.22+
- Node.js 18+
- PostgreSQL 15+

## 快速开始

### 后端服务

```bash
cd itsm-backend
go mod tidy
go run main.go
```

### 前端应用

```bash
cd itsm-prototype
npm install
npm run dev
```

## Git 提交规范

请遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

- `feat:` 新功能
- `fix:` 修复bug
- `docs:` 文档更新
- `style:` 代码格式调整
- `refactor:` 代码重构
- `test:` 测试相关
- `chore:` 构建过程或辅助工具的变动

示例：

```
feat(backend): 添加工单审批流程
fix(frontend): 修复用户登录状态异常
docs: 更新API文档
```

```

## 4. 可选：pre-commit hooks

如果需要代码质量检查，可以创建：

```bash:/.githooks/pre-commit
#!/bin/sh

# Go 代码检查
echo "Running Go tests and linting..."
cd itsm-backend
go fmt ./...
go vet ./...
go test ./...

if [ $? -ne 0 ]; then
    echo "Go tests failed. Commit aborted."
    exit 1
fi

# Node.js 代码检查
echo "Running Node.js linting..."
cd ../itsm-prototype
npm run lint

if [ $? -ne 0 ]; then
    echo "Node.js linting failed. Commit aborted."
    exit 1
fi

echo "All checks passed. Proceeding with commit."
```

然后执行：

```bash
chmod +x .githooks/pre-commit
git config core.hooksPath .githooks
```

这样配置后，你的ITSM项目就有了完整的git提交文件配置，能够：

1. ✅ 忽略不必要的文件（构建产物、依赖、临时文件等）
2. ✅ 正确处理不同类型文件的行结尾
3. ✅ 提供清晰的项目文档
4. ✅ 可选的代码质量检查

建议按顺序创建这些文件，特别是.gitignore文件是必需的。
