# Code Review 流程规范

> 适用于 1-2 人小团队的轻量级 Code Review 流程

## 1. 流程概述

### 1.1 核心原则

| 原则 | 说明 |
|------|------|
| **小步提交** | 每次 PR 控制在 400 行以内，超过需拆分 |
| **快速反馈** | PR 必须在 24 小时内响应 |
| **对事不对人** | 评论聚焦代码，而非开发者 |
| **自动化优先** | 机器能检查的绝不人工检查 |

### 1.2 流程图

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  开发分支   │───▶│  提交 PR    │───▶│  CI 检查   │───▶│  代码审查  │
│  (feature) │    │             │    │ (自动通过) │    │  (人工)    │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                                                              │
                           ┌───────────────────────────────────┘
                           ▼
                    ┌─────────────┐    ┌─────────────┐
                    │  需要修改   │───▶│  重新提交   │
                    │  (有评论)   │    │  (回到CI)   │
                    └─────────────┘    └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │   合并     │
                    │  (approved)│
                    └─────────────┘
```

## 2. PR 模板

### 2.1 模板文件位置

```
.github/
└── PULL_REQUEST_TEMPLATE.md
```

### 2.2 模板内容

```markdown
## 概述
<!-- 简短描述这次改动解决了什么问题 -->

## 改动类型
- [ ] 新功能 (New Feature)
- [ ] Bug 修复 (Bug Fix)
- [ ] 重构 (Refactor)
- [ ] 文档 (Documentation)
- [ ] 性能优化 (Performance)
- [ ] 测试 (Test)

## 改动范围
<!-- 影响的模块/服务 -->

## 自检清单
<!-- 在提交前确保已完成 -->
- [ ] 代码符合 golangci-lint / ESLint 规范
- [ ] 新功能有对应的单元测试
- [ ] 改动已更新相关文档
- [ ] 无敏感信息泄露（API Key、密码等）

## 依赖变更
<!-- 是否有新增依赖 -->
- 新增依赖: [列表]
- 移除依赖: [列表]

## 相关 Issue
<!-- 关联的任务/Issue -->
Closes #

## 审查重点（可选）
<!-- 特别希望审查者关注的地方 -->
```

## 3. 检查清单

### 3.1 代码质量（必须检查）

| # | 检查项 | 严重度 |
|---|--------|--------|
| 1 | 无 `fmt.Println` / `console.log`（生产代码） | 🔴 P0 |
| 2 | 无 `panic()`（非灾难性场景） | 🔴 P0 |
| 3 | 错误已正确处理，无 `_ = err` 忽略 | 🔴 P0 |
| 4 | 无硬编码的敏感信息 | 🔴 P0 |
| 5 | 复杂逻辑有注释说明 | 🟡 P2 |
| 6 | 变量/函数命名清晰 | 🟡 P2 |

### 3.2 架构设计（重点检查）

| # | 检查项 | 严重度 |
|---|--------|--------|
| 1 | Controller 只做参数校验和响应转换 | 🟠 P1 |
| 2 | 业务逻辑在 Service 层 | 🟠 P1 |
| 3 | 无直接访问 DB 的查询（在 Repository 层） | 🟠 P1 |
| 4 | DTO 与 Ent 模型正确隔离 | 🟠 P1 |
| 5 | 多租户隔离正确（tenant_id 贯穿） | 🔴 P0 |

### 3.3 性能安全（必须检查）

| # | 检查项 | 严重度 |
|---|--------|--------|
| 1 | 无 N+1 查询问题 | 🟠 P1 |
| 2 | 大列表查询有分页 | 🟠 P1 |
| 3 | 敏感操作有日志记录 | 🟠 P1 |
| 4 | 权限校验在 API 层 | 🔴 P0 |
| 5 | 输入校验使用 binding/validation | 🟠 P1 |

### 3.4 测试覆盖（根据改动类型）

| 改动类型 | 最低要求 |
|----------|----------|
| 新功能 | 必须有单元测试 |
| Bug 修复 | 必须有回归测试 |
| 重构 | 保持现有测试通过 |
| 性能优化 | 添加性能基准测试 |

## 4. 审批规则

### 4.1 审批权限矩阵

| 改动类型 | 审查人 | 需要批准 |
|----------|--------|----------|
| 自己代码 | 另一位开发者 | ✅ 必须 |
| 紧急 Bug 修复 | 另一位开发者 | ✅ 必须 |
| 文档/配置 | 可跳过 | ❌ 可选 |
| 已有测试修改 | 另一位开发者 | ✅ 必须 |

### 4.2 审批条件

**通过条件（满足任一即可）：**
- 1 位审查者 approve + CI 全部通过

**必须修改条件（满足任一即需修改）：**
- 审查者提出 P0/P1 级别问题
- CI 检查失败
- 缺少必要的测试

### 4.3 审批时限

| 场景 | 响应时限 | 处理方式 |
|------|----------|----------|
| 普通 PR | 24 小时 | 未响应可催办 |
| 紧急修复 | 4 小时 | 立即通知 |
| 重构/大改动 | 48 小时 | 可拆分审查 |

## 5. 自动化集成

### 5.1 GitHub Actions 工作流

```yaml
# .github/workflows/ci.yml
name: CI

on:
  pull_request:
    branches: [main, develop]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest
      - name: Run Go vet
        run: go vet ./...

  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - name: Run tests with coverage
        run: |
          go test -coverprofile=coverage.out ./...
          go tool cover -func=coverage.out
      - name: Upload coverage
        uses: actions/upload-artifact@v4
        with:
          name: coverage
          path: coverage.out

  frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup Node
        uses: actions/setup-node@v4
        with:
          node-version: '20'
      - name: Install dependencies
        run: npm ci
      - name: Run ESLint
        run: npm run lint
      - name: Run type check
        run: npm run type-check

  build:
    runs-on: ubuntu-latest
    needs: [lint, test, frontend]
    steps:
      - uses: actions/checkout@v4
      - name: Build Docker image
        run: docker build -t itsm:${{ github.sha }} .
```

### 5.2 覆盖率门禁

| 模块 | 最低覆盖率 | 目标覆盖率 |
|------|------------|------------|
| service 层 | 60% | 80% |
| controller 层 | 40% | 60% |
| 新增代码 | 70% | 85% |

**覆盖率下降处理：**
- 覆盖率下降 > 10%：阻止合并
- 覆盖率下降 5-10%：需要说明理由
- 覆盖率下降 < 5%：可合并

### 5.3 PR Size 检查

```yaml
# 在 CI 中添加
- name: Check PR size
  run: |
    # 计算改动的行数（排除空行和注释）
    CHANGES=$(git diff --stat --stat-width=200 | tail -1)
    echo "Changes: $CHANGES"
    # PR 超过 400 行需要拆分
```

## 6. 常见问题处理

### 6.1 审查者不在场

**解决方案：**
1. 紧急修复可先合并后审查
2. 使用 GitHub 的 "Require review from Code Owners" 功能
3. 每日站会同步代码审查状态

### 6.2 代码风格分歧

**解决方案：**
1. 优先遵循现有代码风格
2. 有争议时参考官方 style guide
3. 记录决策到团队规范文档

### 6.3 大改动拆分

**拆分原则：**
1. 按功能模块拆分
2. 按依赖关系排序（先底层后上层）
3. 每个 PR 可独立测试和合并

**拆分示例：**
```
❌ 一次大 PR: "重写用户模块"
  - 修改 User Entity
  - 修改 User Service
  - 修改 User Controller
  - 新增测试

✅ 拆分为:
  PR #1: "User Entity 添加字段" (50行)
  PR #2: "User Service 业务逻辑重构" (150行)
  PR #3: "User Controller API 调整" (80行)
  PR #4: "User 模块测试补全" (120行)
```

## 7. 实施检查表

- [ ] 创建 `.github/PULL_REQUEST_TEMPLATE.md`
- [ ] 配置 GitHub Actions CI 流程
- [ ] 配置 golangci-lint 在 CI 中运行
- [ ] 配置前端 ESLint + type-check 在 CI 中运行
- [ ] 设置覆盖率门禁（可选）
- [ ] 团队成员熟悉检查清单
- [ ] 建立每日/每周代码审查习惯

## 8. 附录

### 8.1 golangci-lint 配置文件

详见项目根目录 `.golangci.yml`

### 8.2 ESLint 配置

详见前端目录 `eslint.config.mjs`

### 8.3 相关文档

- [团队技术提升指导](./team-tech-improvement-guide.md)
- [Go 代码规范](https://go.dev/wiki/CodeReviewComments)
- [Google Go Style Guide](https://google.github.io/styleguide/go/)
