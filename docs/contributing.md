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
- [CI 失败排查 SOP](#ci-失败排查-sop)

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
- **CI 全绿**：所有 workflow 必须 success，包括 ga-gate 4 项 gate（详见 [.github/workflows/README.md](.github/workflows/README.md)）

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
3. **覆盖率（分阶段策略）**：
   - 当前 **v1.0 GA** 阶段：覆盖率 ≥ 1% 防退化 floor（实测 2%），70% 仅作 `::warning::`；
   - **v1.1** 目标：核心业务逻辑覆盖率应达到 40%+；
   - **v2.0** 目标：核心业务逻辑覆盖率应达到 70%+。
   - 详见 [.github/workflows/README.md](.github/workflows/README.md) 中的"覆盖率门槛策略"章节。
4. **Mock 依赖**：使用 mock 隔离外部依赖

### CI 覆盖率门槛上调 checklist（v1.0 → v1.1）

- [ ] service/* 包单测覆盖率 ≥ 60%
- [ ] controller/* 包单测覆盖率 ≥ 60%
- [ ] 修改 `.github/workflows/ga-gate.yml` 把 `< 1` 改为 `< 40`
- [ ] PR 试跑几周，确认无 critical regression
- [ ] 更新 [.github/workflows/README.md](.github/workflows/README.md) 对应阶段的阈值

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

## CI 失败排查 SOP

> **必读**。v1.0 GA 修复时我们曾因不遵守本 SOP 误判 3 个 commit。
> 完整复盘见 [docs/ci/postmortem-v1.0-GA.md](docs/ci/postmortem-v1.0-GA.md)。

当 CI 出现红色叉时，**不要**直接看错误信息就开始改代码。先按以下步骤定位真根因：

### Step 1：拉整段日志到本地

```bash
gh api /repos/heidsoft/itsm/actions/jobs/<jobId>/logs > /tmp/x.log
```

`<jobId>` 从 `gh run view <runId> --json jobs` 取得。

### Step 2：按时间线切分日志

每个 step 有 `startedAt` / `completedAt`，按时间切分。**注意 step 之间的边界**，不要把 step A 的日志当 step B 的。

### Step 3：在失败 step 内找退出信号

在失败的 step 内部找：

```
##[error]<message>
Process completed with exit code N.
FAIL    <package>
```

### Step 4：先确认 step 是否已容错

任何 `##[error]` 都必须先确认它来自哪个 step、那个 step 的 `continue-on-error` / `|| true` / `::warning::` 是否开启：

| 模式 | 行为 |
|------|------|
| warmup 步骤的 `##[error]` | 通常已 `\|\| true` 容错，**不会让 job 失败** |
| `::warning::` | 非阻塞，仅在 PR summary 显示 |
| `::error::` + `exit 1` | **真阻塞**，job 一定失败 |

**反例**（v1.0 GA 修复时踩过）：日志里出现 3 条 `##[error]migrations/client.go:59:2: ...`，但它们都在 **Warm up Go build cache** 步骤里且已 `|| true` 容错。我们却连续 3 个 commit 误以为是 Run go test 的真实编译错误。真正的失败证据是：

```
21:35:08 Backend coverage: 2.0%
21:35:08 ##[error]Coverage 2.0% < 70% threshold
```

### Step 5：最后才去看代码

按 SOP 走完前 4 步，定位到 step + 具体命令后再看代码。

### ent cold-cache 假阳性

如果失败信息涉及 `migrations/<schema>` 等 ent 自动生成包，可能遇到 cold-cache loader 误判：

```
migrations/client.go:59:2: module itsm-backend@latest found, but does not contain package migrations/<schema>
```

**处理**：

1. 确认 Warm up 步骤已用 `|| true` 容错（见 [.github/workflows/README.md](.github/workflows/README.md)）；
2. 确认 `Run go test` 步骤用 `go list | grep -vE` 排除了 ent 包；
3. **不要**修改 `migrations/client.go` 或 migrations 包本身——它们是 ent 自动生成的。

## 许可证

通过贡献代码，您同意您的贡献将按 Apache License 2.0 授权。
