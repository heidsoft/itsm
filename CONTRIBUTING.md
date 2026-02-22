# Contributing to ITSM

感谢您对 ITSM 项目的关注！我们欢迎并感谢您的贡献。

## 目录

- [行为准则](#行为准则)
- [如何贡献](#如何贡献)
- [开发环境设置](#开发环境设置)
- [代码风格指南](#代码风格指南)
- [提交信息规范](#提交信息规范)
- [Pull Request 流程](#pull-request-流程)
- [测试要求](#测试要求)

## 行为准则

请阅读并遵守我们的 [行为准则](CODE_OF_CONDUCT.md)，确保社区保持友好和包容。

## 如何贡献

您可以通过以下方式贡献：

1. **报告 Bug** - 使用 GitHub Issues 报告问题
2. **提出新功能** - 在 GitHub Discussions 中讨论
3. **提交代码** - 通过 Pull Request
4. **完善文档** - 帮助改进文档
5. **审查代码** - 参与代码审查

## 开发环境设置

### 前置要求

- Go 1.21+
- Node.js 18+
- PostgreSQL 14+
- Redis 7+

### 快速开始

```bash
# 克隆仓库
git clone https://github.com/your-org/itsm.git
cd itsm

# 设置前端
cd itsm-frontend
npm install
npm run dev

# 设置后端
cd itsm-backend
go mod download
go run main.go
```

### 环境变量

参考 `.env.example` 文件配置环境变量：

```bash
# 复制示例配置
cp .env.example .env

# 编辑配置文件
vim .env
```

## 代码风格指南

### Go (后端)

- 遵循 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 使用 `gofmt` 和 `goimports` 格式化代码
- 变量命名使用 camelCase
- 公共函数必须有文档注释
- 错误处理始终返回 error（不要忽略）

```go
// 正确示例
func CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
    if err := req.Validate(); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    // ...
}

// 错误示例
func CreateUser(ctx context.Context, req *CreateUserRequest) *User {
    // 不应该忽略错误
}
```

### TypeScript/React (前端)

- 遵循 ESLint 配置
- 使用 TypeScript 严格模式
- 组件使用函数式组件和 hooks
- 使用 Tailwind CSS 进行样式设计

```tsx
// 正确示例
interface UserCardProps {
  user: User;
  onEdit?: (user: User) => void;
}

export const UserCard: React.FC<UserCardProps> = ({ user, onEdit }) => {
  return <div>{user.name}</div>;
};
```

## 提交信息规范

使用 [Conventional Commits](https://www.conventionalcommits.org/) 格式：

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### 类型 (Type)

- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更改
- `style`: 代码格式（不影响功能）
- `refactor`: 代码重构
- `perf`: 性能优化
- `test`: 测试相关
- `chore`: 构建/工具链变更

### 示例

```bash
# 功能
git commit -m "feat(ticket): add ticket priority field"

# 修复
git commit -m "fix(auth): resolve token expiration issue"

# 文档
git commit -m "docs: update API documentation"

# 重构
git commit -m "refactor(controller): extract common logic"
```

## Pull Request 流程

### 1. 创建分支

```bash
git checkout -b feature/my-new-feature
# 或
git checkout -b fix/issue-description
```

### 2. 开发

- 在本地进行开发
- 编写测试覆盖新代码
- 保持提交原子性

### 3. 提交

```bash
git add .
git commit -m "feat: add new feature"
```

### 4. 推送并创建 PR

```bash
git push origin feature/my-new-feature
```

### 5. PR 描述模板

```markdown
## 描述
简要描述这个 PR 解决的问题

## 变更类型
- [ ] Bug 修复
- [ ] 新功能
- [ ] 破坏性变更
- [ ] 文档更新

## 测试
- [ ] 单元测试通过
- [ ] 集成测试通过
- [ ] 手动测试通过

## 截图（如适用）
```

### 审查标准

- 代码符合项目规范
- 有适当的测试覆盖
- 文档已更新（如需要）
- 无安全漏洞

## 测试要求

### 后端测试

```bash
# 运行所有测试
go test ./...

# 运行测试并显示覆盖率
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# 运行特定包的测试
go test ./middleware/...
go test ./service/...
go test ./controller/...
```

### 前端测试

```bash
# 运行所有测试
npm test

# 运行测试并生成覆盖率报告
npm run test:coverage

# 运行特定文件的测试
npm test -- --testPathPattern=ticket
```

### 测试规范

1. **单元测试**：每个公共函数都应有测试
2. **集成测试**：关键业务流程应有集成测试
3. **覆盖率**：核心业务逻辑覆盖率应达到 70%+
4. **Mock 依赖**：使用 mock 隔离外部依赖

### 测试命名

```go
// 正确
func TestCreateUser_Success
func TestCreateUser_InvalidEmail
func TestCreateUser_DuplicateEmail

// 错误
func TestCreateUser
func Test1
func TestUserCreationFunction
```

## 许可证

通过贡献代码，您同意您的贡献将按 MIT 许可证授权。
