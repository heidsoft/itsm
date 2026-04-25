# ITSM 测试覆盖提升设计文档

**日期**: 2026-04-25
**状态**: 待批准
**范围**: 变更管理模块完整测试覆盖（后端 → 前端API → 组件 → E2E）

---

## 1. 背景与目标

### 1.1 当前状态

| 模块 | 后端测试 | 前端测试 | 覆盖率 |
|------|---------|---------|--------|
| AI功能 | ❌ 无 | ❌ 无 | 0% |
| 工作流引擎 | ❌ 无 | ❌ 无 | 0% |
| 变更管理 | ❌ 无 | ❌ 无 | 0% |
| 系统整体 | ⚠️ 部分 | ⚠️ 部分 | 35% |

### 1.2 目标

- **系统整体测试覆盖率**: 35% → 80%
- **本次迭代目标**: 完成变更管理模块端到端测试覆盖
- **方法论**: 严格TDD（RED-GREEN-REFACTOR）

---

## 2. 实施策略

### 2.1 分层递进方案

```
阶段1: 后端测试
   ↓ Controller + Service 测试通过
阶段2: 前端API测试
   ↓ API客户端测试通过
阶段3: 前端组件测试
   ↓ 组件测试通过
阶段4: E2E测试
   ↓ 完整流程验证
```

### 2.2 TDD工作流

每个测试用例遵循标准TDD循环：

```
RED → 编写失败的测试
GREEN → 编写最小实现使测试通过
REFACTOR → 重构代码，保持测试通过
```

---

## 3. 后端测试架构

### 3.1 文件结构

```
itsm-backend/
├── controller/
│   └── change_controller_test.go    # HTTP handler测试
├── service/
│   └── change_service_test.go       # 业务逻辑测试
└── ent/schema/
    └── change.go                    # 已存在的schema
```

### 3.2 测试覆盖范围

#### Controller测试 (`change_controller_test.go`)

| 测试用例 | 描述 |
|---------|------|
| `TestListChanges` | 列表查询，分页，过滤 |
| `TestGetChange` | 获取详情，404处理 |
| `TestCreateChange` | 创建变更，验证错误处理 |
| `TestUpdateChange` | 更新变更，并发冲突处理 |
| `TestDeleteChange` | 删除变更，软删除验证 |
| `TestSubmitChange` | 提交审批状态变更 |
| `TestApproveChange` | 审批通过操作 |
| `TestRejectChange` | 审批拒绝操作 |
| `TestImplementChange` | 实施变更操作 |
| `TestCompleteChange` | 完成变更操作 |

#### Service测试 (`change_service_test.go`)

| 测试用例 | 描述 |
|---------|------|
| `TestCreateChange_Validation` | 创建验证逻辑 |
| `TestCreateChange_RiskAssessment` | 风险评估计算 |
| `TestUpdateChange_Concurrency` | 并发更新处理 |
| `TestChangeStatus_Transition` | 状态机流转验证 |
| `TestChangeApproval_Workflow` | 审批工作流 |
| `TestChangeImpact_Analysis` | 影响分析 |
| `TestChangeRollback_Plan` | 回滚计划验证 |

### 3.3 Mock策略

使用 `enttest` 创建内存SQLite数据库：

```go
func setupTest(t *testing.T) *ent.Client {
    client := enttest.Open(t, dialect.SQLite,
        "file:ent?mode=memory&cache=shared&_fk=1")
    return client
}

func teardownTest(client *ent.Client) {
    client.Close()
}
```

### 3.4 测试模板

```go
func TestCreateChange(t *testing.T) {
    client := setupTest(t)
    defer teardownTest(client)

    svc := service.NewChangeService(client)

    req := &dto.CreateChangeRequest{
        Title:       "Test Change",
        Description: "Test Description",
        ChangeType:  "standard",
        RiskLevel:   "medium",
    }

    change, err := svc.CreateChange(context.Background(), req)

    assert.NoError(t, err)
    assert.NotNil(t, change)
    assert.Equal(t, "Test Change", change.Title)
}
```

---

## 4. 前端API层测试

### 4.1 文件结构

```
itsm-frontend/src/lib/api/__tests__/
└── change-api.test.ts              # API客户端测试
```

### 4.2 测试覆盖范围

| 测试用例 | 描述 |
|---------|------|
| `getChanges` | 列表查询，分页参数 |
| `getChange` | 获取详情 |
| `createChange` | 创建变更 |
| `updateChange` | 更新变更 |
| `deleteChange` | 删除变更 |
| `submitChange` | 提交审批 |
| `approveChange` | 审批通过 |
| `rejectChange` | 审批拒绝 |
| `implementChange` | 实施变更 |
| `completeChange` | 完成变更 |

### 4.3 Mock策略

使用 MSW (Mock Service Worker)：

```typescript
// src/lib/api/__mocks__/handlers/change.ts
import { rest } from 'msw'

export const changeHandlers = [
  rest.get('/api/v1/changes', (req, res, ctx) => {
    return res(ctx.json({
      code: 0,
      message: 'success',
      data: {
        list: mockChangeList,
        total: 100,
        page: 1,
        pageSize: 10,
      },
    }))
  }),

  rest.post('/api/v1/changes', (req, res, ctx) => {
    return res(ctx.json({
      code: 0,
      message: 'success',
      data: { id: 1, ...req.body },
    }))
  }),

  // ...其他handlers
]
```

### 4.4 测试模板

```typescript
import { ChangeApi } from '../change-api'
import { server } from '__mocks__/server'

describe('ChangeApi', () => {
  beforeAll(() => server.listen())
  afterEach(() => server.resetHandlers())
  afterAll(() => server.close())

  describe('getChanges', () => {
    it('should fetch changes with pagination', async () => {
      const result = await ChangeApi.getChanges({ page: 1, pageSize: 10 })

      expect(result.code).toBe(0)
      expect(result.data.list).toHaveLength(10)
      expect(result.data.total).toBe(100)
    })

    it('should handle errors gracefully', async () => {
      server.use(
        rest.get('/api/v1/changes', (req, res, ctx) => {
          return res(ctx.status(500), ctx.json({ code: 5001, message: 'Internal Error' }))
        })
      )

      await expect(ChangeApi.getChanges({})).rejects.toThrow()
    })
  })
})
```

---

## 5. 前端组件测试

### 5.1 文件结构

```
itsm-frontend/src/components/change/__tests__/
├── ChangeList.test.tsx              # 列表组件测试
├── ChangeDetail.test.tsx            # 详情组件测试
├── ChangeForm.test.tsx              # 表单组件测试
└── ChangeStatusBadge.test.tsx       # 状态徽章测试
```

### 5.2 测试覆盖范围

#### ChangeList.test.tsx

| 测试用例 | 描述 |
|---------|------|
| 渲染变更列表 | 验证表格正确渲染 |
| 分页交互 | 点击分页器触发数据加载 |
| 搜索过滤 | 输入搜索条件过滤列表 |
| 状态筛选 | 按状态筛选变更 |
| 空状态显示 | 无数据时显示空状态 |
| 加载状态 | 数据加载中显示骨架屏 |

#### ChangeDetail.test.tsx

| 测试用例 | 描述 |
|---------|------|
| 渲染变更详情 | 验证详情信息正确显示 |
| 状态流转按钮 | 根据状态显示对应操作按钮 |
| 审批历史时间线 | 显示审批历史记录 |
| 关联CI展示 | 显示关联的配置项 |
| 风险评估展示 | 显示风险评估信息 |
| 返回按钮 | 点击返回列表页 |

#### ChangeForm.test.tsx

| 测试用例 | 描述 |
|---------|------|
| 表单验证 | 必填字段验证提示 |
| 日期选择器 | 计划开始/结束日期选择 |
| 风险等级选择 | 风险等级下拉选择 |
| CI关联选择 | 关联配置项选择器 |
| 提交操作 | 提交表单调用API |
| 取消操作 | 取消返回上一页 |
| 编辑模式 | 编辑时回显数据 |

#### ChangeStatusBadge.test.tsx

| 测试用例 | 描述 |
|---------|------|
| 状态颜色 | 各状态对应正确颜色 |
| 状态图标 | 各状态对应正确图标 |
| 状态文本 | 显示正确的状态文本 |

### 5.3 Mock策略

```typescript
// Mock next/navigation
jest.mock('next/navigation', () => ({
  useRouter: () => ({
    push: jest.fn(),
    back: jest.fn(),
  }),
  usePathname: () => '/changes',
  useSearchParams: () => new URLSearchParams(),
}))

// Mock antd message
jest.mock('antd', () => {
  const actual = jest.requireActual('antd')
  return {
    ...actual,
    message: {
      success: jest.fn(),
      error: jest.fn(),
      warning: jest.fn(),
      info: jest.fn(),
    },
  }
})
```

### 5.4 测试模板

```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ChangeList } from '../ChangeList'
import { server } from '__mocks__/server'

describe('ChangeList', () => {
  beforeAll(() => server.listen())
  afterEach(() => {
    server.resetHandlers()
    jest.clearAllMocks()
  })
  afterAll(() => server.close())

  it('should render change list', async () => {
    render(<ChangeList />)

    await waitFor(() => {
      expect(screen.getByText('Test Change 1')).toBeInTheDocument()
    })
  })

  it('should filter by status', async () => {
    render(<ChangeList />)

    const statusFilter = screen.getByLabelText('状态')
    fireEvent.mouseDown(statusFilter)

    await waitFor(() => {
      expect(screen.getByText('待审批')).toBeInTheDocument()
    })
  })

  it('should show empty state when no data', async () => {
    server.use(
      rest.get('/api/v1/changes', (req, res, ctx) => {
        return res(ctx.json({ code: 0, data: { list: [], total: 0 } }))
      })
    )

    render(<ChangeList />)

    await waitFor(() => {
      expect(screen.getByText('暂无变更记录')).toBeInTheDocument()
    })
  })
})
```

---

## 6. E2E测试

### 6.1 文件结构

```
itsm-frontend/e2e/
├── change-management.spec.ts        # E2E流程测试
├── fixtures/
│   └── change-fixture.ts            # 测试数据
└── support/
    ├── auth.ts                      # 登录辅助函数
    └── helpers.ts                   # 通用辅助函数
```

### 6.2 测试场景

| 场景 | 步骤 | 验证点 |
|------|------|--------|
| 创建变更流程 | 登录 → 列表页 → 创建 → 填写 → 提交 | 创建成功提示，列表显示新记录 |
| 审批流程 | 登录 → 待审批 → 审批操作 → 确认 | 状态变更为已审批/已拒绝 |
| 编辑变更 | 登录 → 详情页 → 编辑 → 修改 → 保存 | 更新成功，详情显示新数据 |
| 搜索过滤 | 登录 → 输入搜索 → 验证结果 | 列表仅显示匹配结果 |
| 删除变更 | 登录 → 详情页 → 删除 → 确认 | 删除成功，列表不再显示 |

### 6.3 测试工具配置

```typescript
// playwright.config.ts
import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
  },
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
  },
})
```

### 6.4 测试模板

```typescript
import { test, expect } from '@playwright/test'
import { login } from '../support/auth'
import { testChange } from '../fixtures/change-fixture'

test.describe('Change Management', () => {
  test.beforeEach(async ({ page }) => {
    await login(page, 'admin', 'admin123')
  })

  test('should create a new change', async ({ page }) => {
    await page.goto('/changes')

    await page.click('button:has-text("创建变更")')

    await page.fill('input[name="title"]', testChange.title)
    await page.fill('textarea[name="description"]', testChange.description)
    await page.selectOption('select[name="changeType"]', testChange.changeType)
    await page.selectOption('select[name="riskLevel"]', testChange.riskLevel)

    await page.click('button:has-text("提交")')

    await expect(page.locator('.ant-message-success')).toBeVisible()
    await expect(page.locator('table')).toContainText(testChange.title)
  })

  test('should approve a change', async ({ page }) => {
    await page.goto('/changes?status=pending_approval')

    await page.click('tr:has-text("待审批") >> a')
    await page.click('button:has-text("审批通过")')
    await page.click('button:has-text("确认")')

    await expect(page.locator('.ant-message-success')).toBeVisible()
    await expect(page.locator('.change-status')).toContainText('已审批')
  })

  test('should search changes', async ({ page }) => {
    await page.goto('/changes')

    await page.fill('input[placeholder="搜索"]', '紧急')
    await page.press('input[placeholder="搜索"]', 'Enter')

    await expect(page.locator('table tbody tr')).toHaveCount(3)
  })
})
```

---

## 7. 实施计划

### 7.1 时间表

| 阶段 | 任务 | 预估时间 | 产出 |
|------|------|---------|------|
| 阶段1 | 后端测试 | 1-2天 | Service + Controller测试通过 |
| 阶段2 | 前端API测试 | 0.5-1天 | API客户端测试通过 |
| 阶段3 | 前端组件测试 | 1-2天 | 组件测试通过 |
| 阶段4 | E2E测试 | 1天 | 5个核心流程测试通过 |
| **总计** | | **4-6天** | 变更管理模块80%+覆盖率 |

### 7.2 验收标准

| 层级 | 目标 | 验证命令 |
|------|------|---------|
| 后端 Service | ≥85% | `go test -cover ./service/...` |
| 后端 Controller | ≥80% | `go test -cover ./controller/...` |
| 前端 API | ≥90% | `npm test -- --coverage --testPathPattern=change-api` |
| 前端 组件 | ≥75% | `npm test -- --coverage --testPathPattern=change` |
| E2E | 5个场景 | `npx playwright test e2e/change-management.spec.ts` |

### 7.3 质量门禁

每次提交必须满足：

- [ ] 所有新测试通过
- [ ] 无跳过的测试用例
- [ ] 覆盖率不低于上一次提交
- [ ] 无 TypeScript 编译错误
- [ ] 无 ESLint 错误

---

## 8. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| MSW与实际API不一致 | 测试假阳性 | 使用OpenAPI规范生成Mock |
| 组件依赖复杂 | Mock困难 | 抽取可测试的纯组件 |
| E2E测试不稳定 | CI失败 | 增加重试机制，优化等待策略 |
| 后端API未完成 | 测试无法编写 | 先完成API，再写测试 |

---

## 9. 后续迭代

完成变更管理模块后，按相同模式推进：

1. **工作流引擎** - 中等复杂度
2. **AI功能** - 高复杂度（需要Mock LLM API）

最终目标：系统整体测试覆盖率达到80%。

---

## 附录

### A. 测试命令速查

```bash
# 后端测试
cd itsm-backend
go test ./service/... -cover
go test ./controller/... -cover
go test ./... -coverprofile=coverage.out

# 前端测试
cd itsm-frontend
npm test
npm test -- --coverage
npm test -- --testPathPattern=change

# E2E测试
npx playwright test
npx playwright test --ui
npx playwright test e2e/change-management.spec.ts
```

### B. 参考资料

- [Go Testing Guidelines](https://go.dev/doc/tutorial/add-a-test)
- [React Testing Library](https://testing-library.com/docs/react-testing-library/intro/)
- [MSW Documentation](https://mswjs.io/docs/)
- [Playwright Documentation](https://playwright.dev/docs/intro)
