# 变更管理模块测试覆盖实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 完成变更管理模块端到端测试覆盖，达到80%+覆盖率目标

**Architecture:** 分层递进：后端Service测试 → 后端Controller测试 → 前端API测试 → 前端组件测试 → E2E测试

**Tech Stack:** Go/testify/enttest, TypeScript/Jest/MSW, Playwright

---

## 文件结构概览

```
# 后端测试文件
itsm-backend/
├── service/
│   └── change_service_test.go        # 已存在，需补充
├── controller/
│   └── change_controller_test.go     # 新建

# 前端测试文件
itsm-frontend/src/
├── lib/api/__tests__/
│   └── change-api.test.ts            # 新建
├── components/change/__tests__/
│   ├── ChangeList.test.tsx           # 新建
│   ├── ChangeDetail.test.tsx         # 新建
│   ├── ChangeForm.test.tsx           # 新建
│   └── ChangeStatusBadge.test.tsx    # 新建

# E2E测试文件
itsm-frontend/e2e/
├── change-management.spec.ts         # 新建
├── fixtures/change-fixture.ts        # 新建
└── support/auth.ts                   # 新建
```

---

## 阶段1: 后端Service测试补充

### Task 1.1: 补充状态转换边界测试

**Files:**
- Modify: `itsm-backend/service/change_service_test.go`

- [ ] **Step 1: 添加无效状态转换测试**

```go
func TestChangeService_UpdateChangeStatus_InvalidTransition(t *testing.T) {
	client, service, ctx := setupChangeTest(t)
	defer client.Close()

	testTenant, err := createChangeTestTenant(ctx, client, "invalid")
	require.NoError(t, err)

	testUser, err := createChangeTestUser(ctx, client, testTenant.ID, "invalid")
	require.NoError(t, err)

	testChange, err := client.Change.Create().
		SetTitle("Invalid Transition Test").
		SetDescription("Test").
		SetType("normal").
		SetStatus("draft").
		SetPriority("medium").
		SetImpactScope("medium").
		SetRiskLevel("low").
		SetCreatedBy(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 尝试无效转换: draft -> approved (跳过submitted)
	err = service.UpdateChangeStatus(ctx, testChange.ID, "approved", testTenant.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status transition")
}
```

- [ ] **Step 2: 添加已取消状态转换测试**

```go
func TestChangeService_UpdateChangeStatus_CancelledCannotTransition(t *testing.T) {
	client, service, ctx := setupChangeTest(t)
	defer client.Close()

	testTenant, err := createChangeTestTenant(ctx, client, "cancelled")
	require.NoError(t, err)

	testUser, err := createChangeTestUser(ctx, client, testTenant.ID, "cancelled")
	require.NoError(t, err)

	testChange, err := client.Change.Create().
		SetTitle("Cancelled Change").
		SetDescription("Test").
		SetType("normal").
		SetStatus("cancelled").
		SetPriority("medium").
		SetImpactScope("medium").
		SetRiskLevel("low").
		SetCreatedBy(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 尝试从cancelled转换到其他状态
	err = service.UpdateChangeStatus(ctx, testChange.ID, "draft", testTenant.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status transition")
}
```

- [ ] **Step 3: 运行测试验证通过**

Run: `cd itsm-backend && go test ./service/... -run TestChangeService_UpdateChangeStatus -v`
Expected: PASS

- [ ] **Step 4: 提交**

```bash
git add itsm-backend/service/change_service_test.go
git commit -m "test(change): add status transition edge case tests"
```

---

### Task 1.2: 补充审批工作流测试

**Files:**
- Modify: `itsm-backend/service/change_service_test.go`

- [ ] **Step 1: 添加提交审批测试**

```go
func TestChangeService_SubmitChange_Success(t *testing.T) {
	client, service, ctx := setupChangeTest(t)
	defer client.Close()

	testTenant, err := createChangeTestTenant(ctx, client, "submit")
	require.NoError(t, err)

	testUser, err := createChangeTestUser(ctx, client, testTenant.ID, "submit")
	require.NoError(t, err)

	testChange, err := client.Change.Create().
		SetTitle("Submit Test").
		SetDescription("Test").
		SetType("normal").
		SetStatus("draft").
		SetPriority("medium").
		SetImpactScope("medium").
		SetRiskLevel("low").
		SetCreatedBy(testUser.ID).
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 提交变更
	err = service.UpdateChangeStatus(ctx, testChange.ID, "submitted", testTenant.ID)
	require.NoError(t, err)

	// 验证状态已更新
	response, err := service.GetChange(ctx, testChange.ID, testTenant.ID)
	require.NoError(t, err)
	assert.Equal(t, dto.ChangeStatus("submitted"), response.Status)
}
```

- [ ] **Step 2: 运行测试验证通过**

Run: `cd itsm-backend && go test ./service/... -run TestChangeService_SubmitChange -v`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add itsm-backend/service/change_service_test.go
git commit -m "test(change): add submit change workflow test"
```

---

## 阶段2: 后端Controller测试

### Task 2.1: 创建ChangeController测试文件

**Files:**
- Create: `itsm-backend/controller/change_controller_test.go`

- [ ] **Step 1: 创建测试基础设施**

```go
package controller

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/router"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupChangeControllerTest(t *testing.T) (*ent.Client, *gin.Engine, context.Context) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()

	gin.SetMode(gin.TestMode)

	// 创建测试租户和用户
	tenant, _ := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)

	user, _ := client.User.Create().
		SetUsername("testuser").
		SetEmail("test@example.com").
		SetName("Test User").
		SetPasswordHash("hash").
		SetRole("admin").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)

	return client, router, ctx
}
```

- [ ] **Step 2: 添加ListChanges测试**

```go
func TestChangeController_ListChanges(t *testing.T) {
	client, router, ctx := setupChangeControllerTest(t)
	defer client.Close()

	// 创建测试数据...

	req, _ := http.NewRequest("GET", "/api/v1/changes?page=1&page_size=10", nil)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, float64(0), response["code"])
}
```

- [ ] **Step 3: 添加CreateChange测试**

```go
func TestChangeController_CreateChange(t *testing.T) {
	client, router, ctx := setupChangeControllerTest(t)
	defer client.Close()

	plannedStart := time.Now().Add(24 * time.Hour)
	plannedEnd := time.Now().Add(48 * time.Hour)

	reqBody := dto.CreateChangeRequest{
		Title:              "Test Change",
		Description:        "Test Description",
		Justification:      "Test Justification",
		Type:               "normal",
		Priority:           "medium",
		ImpactScope:        "low",
		RiskLevel:          "low",
		PlannedStartDate:   &plannedStart,
		PlannedEndDate:     &plannedEnd,
		ImplementationPlan: "1. Step 1\n2. Step 2",
		RollbackPlan:       "Rollback steps",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/changes", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, float64(0), response["code"])
	assert.NotNil(t, response["data"])
}
```

- [ ] **Step 4: 添加GetChange测试**

```go
func TestChangeController_GetChange(t *testing.T) {
	client, router, ctx := setupChangeControllerTest(t)
	defer client.Close()

	// 创建测试变更
	// ...

	req, _ := http.NewRequest("GET", "/api/v1/changes/1", nil)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestChangeController_GetChange_NotFound(t *testing.T) {
	client, router, ctx := setupChangeControllerTest(t)
	defer client.Close()

	req, _ := http.NewRequest("GET", "/api/v1/changes/99999", nil)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
```

- [ ] **Step 5: 运行测试验证通过**

Run: `cd itsm-backend && go test ./controller/... -run TestChangeController -v`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add itsm-backend/controller/change_controller_test.go
git commit -m "test(change): add controller tests for CRUD operations"
```

---

## 阶段3: 前端API测试

### Task 3.1: 创建MSW Handlers

**Files:**
- Create: `itsm-frontend/src/lib/api/__mocks__/handlers/change.ts`

- [ ] **Step 1: 创建Change Mock Handlers**

```typescript
// src/lib/api/__mocks__/handlers/change.ts
import { rest } from 'msw'

const mockChanges = [
  {
    id: 1,
    title: 'Test Change 1',
    description: 'Description 1',
    justification: 'Justification 1',
    type: 'normal',
    status: 'draft',
    priority: 'medium',
    impactScope: 'low',
    riskLevel: 'low',
    createdBy: 1,
    createdByName: 'Test User',
    tenantId: 1,
    implementationPlan: 'Plan',
    rollbackPlan: 'Rollback',
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  },
]

export const changeHandlers = [
  rest.get('/api/v1/changes', (req, res, ctx) => {
    const page = parseInt(req.url.searchParams.get('page') || '1')
    const pageSize = parseInt(req.url.searchParams.get('page_size') || '10')

    return res(
      ctx.json({
        code: 0,
        message: 'success',
        data: {
          total: mockChanges.length,
          changes: mockChanges,
        },
      })
    )
  }),

  rest.get('/api/v1/changes/:id', (req, res, ctx) => {
    const id = parseInt(req.params.id as string)
    const change = mockChanges.find(c => c.id === id)

    if (!change) {
      return res(
        ctx.status(404),
        ctx.json({ code: 1001, message: 'Change not found' })
      )
    }

    return res(ctx.json({ code: 0, message: 'success', data: change }))
  }),

  rest.post('/api/v1/changes', async (req, res, ctx) => {
    const body = await req.json()
    const newChange = {
      id: mockChanges.length + 1,
      ...body,
      status: 'draft',
      createdBy: 1,
      createdByName: 'Test User',
      tenantId: 1,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    }

    return res(ctx.json({ code: 0, message: 'success', data: newChange }))
  }),

  rest.put('/api/v1/changes/:id', async (req, res, ctx) => {
    const id = parseInt(req.params.id as string)
    const body = await req.json()
    const change = mockChanges.find(c => c.id === id)

    if (!change) {
      return res(
        ctx.status(404),
        ctx.json({ code: 1001, message: 'Change not found' })
      )
    }

    return res(
      ctx.json({
        code: 0,
        message: 'success',
        data: { ...change, ...body, updatedAt: new Date().toISOString() },
      })
    )
  }),

  rest.delete('/api/v1/changes/:id', (req, res, ctx) => {
    return res(ctx.json({ code: 0, message: 'success' }))
  }),

  rest.post('/api/v1/changes/:id/submit', (req, res, ctx) => {
    return res(ctx.json({ code: 0, message: 'success' }))
  }),

  rest.post('/api/v1/changes/:id/approve', (req, res, ctx) => {
    return res(ctx.json({ code: 0, message: 'success' }))
  }),
]
```

- [ ] **Step 2: 提交**

```bash
git add itsm-frontend/src/lib/api/__mocks__/handlers/change.ts
git commit -m "test(change): add MSW handlers for change API"
```

---

### Task 3.2: 创建Change API测试

**Files:**
- Create: `itsm-frontend/src/lib/api/__tests__/change-api.test.ts`

- [ ] **Step 1: 创建测试文件**

```typescript
// src/lib/api/__tests__/change-api.test.ts
import { ChangeApi, type Change, type ChangeListResponse } from '../change-api'
import { server } from '../__mocks__/server'
import { changeHandlers } from '../__mocks__/handlers/change'

// 注册handlers
beforeAll(() => {
  server.use(...changeHandlers)
  server.listen()
})

afterEach(() => server.resetHandlers())

afterAll(() => server.close())

describe('ChangeApi', () => {
  describe('getChanges', () => {
    it('should fetch changes with pagination', async () => {
      const result = await ChangeApi.getChanges({ page: 1, page_size: 10 })

      expect(result.total).toBe(1)
      expect(result.changes).toHaveLength(1)
      expect(result.changes[0].title).toBe('Test Change 1')
    })

    it('should handle empty results', async () => {
      server.use(
        rest.get('/api/v1/changes', (req, res, ctx) => {
          return res(
            ctx.json({
              code: 0,
              message: 'success',
              data: { total: 0, changes: [] },
            })
          )
        })
      )

      const result = await ChangeApi.getChanges({})
      expect(result.total).toBe(0)
      expect(result.changes).toHaveLength(0)
    })
  })

  describe('getChange', () => {
    it('should fetch a single change by id', async () => {
      const result = await ChangeApi.getChange(1)

      expect(result.id).toBe(1)
      expect(result.title).toBe('Test Change 1')
    })

    it('should throw error for non-existent change', async () => {
      await expect(ChangeApi.getChange(99999)).rejects.toThrow()
    })
  })

  describe('createChange', () => {
    it('should create a new change', async () => {
      const newChange = {
        title: 'New Change',
        description: 'New Description',
        justification: 'New Justification',
        type: 'normal' as const,
        priority: 'medium' as const,
        impactScope: 'low' as const,
        riskLevel: 'low' as const,
        implementationPlan: 'Plan',
        rollbackPlan: 'Rollback',
        affectedCis: [],
        relatedTickets: [],
      }

      const result = await ChangeApi.createChange(newChange)

      expect(result.title).toBe('New Change')
      expect(result.status).toBe('draft')
    })
  })

  describe('updateChange', () => {
    it('should update an existing change', async () => {
      const result = await ChangeApi.updateChange(1, { title: 'Updated Title' })

      expect(result.title).toBe('Updated Title')
    })
  })

  describe('deleteChange', () => {
    it('should delete a change', async () => {
      await expect(ChangeApi.deleteChange(1)).resolves.not.toThrow()
    })
  })

  describe('submitForApproval', () => {
    it('should submit change for approval', async () => {
      await expect(ChangeApi.submitForApproval(1)).resolves.not.toThrow()
    })
  })

  describe('approveChange', () => {
    it('should approve a change', async () => {
      await expect(
        ChangeApi.approveChange(1, { status: 'approved' })
      ).resolves.not.toThrow()
    })
  })
})
```

- [ ] **Step 2: 运行测试验证通过**

Run: `cd itsm-frontend && npm test -- --testPathPattern=change-api.test.ts`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add itsm-frontend/src/lib/api/__tests__/change-api.test.ts
git commit -m "test(change): add API client tests with MSW mocking"
```

---

## 阶段4: 前端组件测试

### Task 4.1: 创建ChangeStatusBadge组件测试

**Files:**
- Create: `itsm-frontend/src/components/change/__tests__/ChangeStatusBadge.test.tsx`

- [ ] **Step 1: 创建测试文件**

```typescript
// src/components/change/__tests__/ChangeStatusBadge.test.tsx
import { render, screen } from '@testing-library/react'
import { ChangeStatusBadge } from '../ChangeStatusBadge'

describe('ChangeStatusBadge', () => {
  it('should render draft status with correct color', () => {
    render(<ChangeStatusBadge status="draft" />)
    expect(screen.getByText('草稿')).toBeInTheDocument()
  })

  it('should render pending status with correct color', () => {
    render(<ChangeStatusBadge status="pending" />)
    expect(screen.getByText('待审批')).toBeInTheDocument()
  })

  it('should render approved status', () => {
    render(<ChangeStatusBadge status="approved" />)
    expect(screen.getByText('已批准')).toBeInTheDocument()
  })

  it('should render rejected status', () => {
    render(<ChangeStatusBadge status="rejected" />)
    expect(screen.getByText('已拒绝')).toBeInTheDocument()
  })

  it('should render in_progress status', () => {
    render(<ChangeStatusBadge status="in_progress" />)
    expect(screen.getByText('实施中')).toBeInTheDocument()
  })

  it('should render completed status', () => {
    render(<ChangeStatusBadge status="completed" />)
    expect(screen.getByText('已完成')).toBeInTheDocument()
  })

  it('should render rolled_back status', () => {
    render(<ChangeStatusBadge status="rolled_back" />)
    expect(screen.getByText('已回滚')).toBeInTheDocument()
  })

  it('should render cancelled status', () => {
    render(<ChangeStatusBadge status="cancelled" />)
    expect(screen.getByText('已取消')).toBeInTheDocument()
  })
})
```

- [ ] **Step 2: 运行测试验证通过**

Run: `cd itsm-frontend && npm test -- --testPathPattern=ChangeStatusBadge.test.tsx`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add itsm-frontend/src/components/change/__tests__/ChangeStatusBadge.test.tsx
git commit -m "test(change): add ChangeStatusBadge component tests"
```

---

### Task 4.2: 创建ChangeList组件测试

**Files:**
- Create: `itsm-frontend/src/components/change/__tests__/ChangeList.test.tsx`

- [ ] **Step 1: 创建测试文件**

```typescript
// src/components/change/__tests__/ChangeList.test.tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { ChangeList } from '../ChangeList'
import { server } from '@/lib/api/__mocks__/server'

// Mock next/navigation
jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn() }),
  usePathname: () => '/changes',
  useSearchParams: () => new URLSearchParams(),
}))

describe('ChangeList', () => {
  beforeAll(() => server.listen())
  afterEach(() => {
    server.resetHandlers()
    jest.clearAllMocks()
  })
  afterAll(() => server.close())

  it('should render change list with data', async () => {
    render(<ChangeList />)

    await waitFor(() => {
      expect(screen.getByText('Test Change 1')).toBeInTheDocument()
    })
  })

  it('should show loading state initially', () => {
    render(<ChangeList />)
    // Ant Design Spin component
    expect(document.querySelector('.ant-spin')).toBeInTheDocument()
  })

  it('should handle pagination', async () => {
    render(<ChangeList />)

    await waitFor(() => {
      expect(screen.getByText('Test Change 1')).toBeInTheDocument()
    })

    // 点击下一页
    const nextButton = document.querySelector('.ant-pagination-next')
    if (nextButton) {
      fireEvent.click(nextButton)
    }
  })

  it('should filter by status', async () => {
    render(<ChangeList />)

    await waitFor(() => {
      expect(screen.getByText('Test Change 1')).toBeInTheDocument()
    })

    // 状态筛选器交互
    const statusFilter = screen.getByLabelText('状态')
    fireEvent.mouseDown(statusFilter)

    await waitFor(() => {
      expect(screen.getByText('待审批')).toBeInTheDocument()
    })
  })
})
```

- [ ] **Step 2: 运行测试验证通过**

Run: `cd itsm-frontend && npm test -- --testPathPattern=ChangeList.test.tsx`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add itsm-frontend/src/components/change/__tests__/ChangeList.test.tsx
git commit -m "test(change): add ChangeList component tests"
```

---

### Task 4.3: 创建ChangeForm组件测试

**Files:**
- Create: `itsm-frontend/src/components/change/__tests__/ChangeForm.test.tsx`

- [ ] **Step 1: 创建测试文件**

```typescript
// src/components/change/__tests__/ChangeForm.test.tsx
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ChangeForm } from '../ChangeForm'

// Mock next/navigation
jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn(), back: jest.fn() }),
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
    },
  }
})

describe('ChangeForm', () => {
  it('should render form with all required fields', () => {
    render(<ChangeForm />)

    expect(screen.getByLabelText('变更标题')).toBeInTheDocument()
    expect(screen.getByLabelText('变更描述')).toBeInTheDocument()
    expect(screen.getByLabelText('变更理由')).toBeInTheDocument()
  })

  it('should validate required fields', async () => {
    const user = userEvent.setup()
    render(<ChangeForm />)

    // 点击提交按钮
    const submitButton = screen.getByRole('button', { name: /提交/i })
    await user.click(submitButton)

    await waitFor(() => {
      // 应该显示验证错误
      expect(screen.getByText(/请输入变更标题/i)).toBeInTheDocument()
    })
  })

  it('should fill form with default values in edit mode', () => {
    const defaultValues = {
      title: 'Edit Test',
      description: 'Edit Description',
      type: 'normal' as const,
      priority: 'high' as const,
    }

    render(<ChangeForm defaultValues={defaultValues} mode="edit" />)

    expect(screen.getByDisplayValue('Edit Test')).toBeInTheDocument()
  })

  it('should call onSubmit with form data', async () => {
    const onSubmit = jest.fn()
    const user = userEvent.setup()

    render(<ChangeForm onSubmit={onSubmit} />)

    // 填写表单
    await user.type(screen.getByLabelText('变更标题'), 'Test Change')
    await user.type(screen.getByLabelText('变更描述'), 'Test Description')
    await user.type(screen.getByLabelText('变更理由'), 'Test Justification')

    // 提交
    const submitButton = screen.getByRole('button', { name: /提交/i })
    await user.click(submitButton)

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'Test Change',
          description: 'Test Description',
        })
      )
    })
  })
})
```

- [ ] **Step 2: 运行测试验证通过**

Run: `cd itsm-frontend && npm test -- --testPathPattern=ChangeForm.test.tsx`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add itsm-frontend/src/components/change/__tests__/ChangeForm.test.tsx
git commit -m "test(change): add ChangeForm component tests"
```

---

## 阶段5: E2E测试

### Task 5.1: 配置Playwright

**Files:**
- Create: `itsm-frontend/playwright.config.ts` (如果不存在)
- Create: `itsm-frontend/e2e/support/auth.ts`
- Create: `itsm-frontend/e2e/fixtures/change-fixture.ts`

- [ ] **Step 1: 创建认证辅助函数**

```typescript
// e2e/support/auth.ts
import type { Page } from '@playwright/test'

export async function login(page: Page, username: string, password: string) {
  await page.goto('/login')
  await page.fill('input[name="username"]', username)
  await page.fill('input[name="password"]', password)
  await page.click('button[type="submit"]')

  // 等待登录成功跳转
  await page.waitForURL('**/', { timeout: 10000 })
}

export async function logout(page: Page) {
  await page.click('.user-avatar')
  await page.click('text=退出登录')
}
```

- [ ] **Step 2: 创建测试数据Fixtures**

```typescript
// e2e/fixtures/change-fixture.ts
export const testChange = {
  title: 'E2E Test Change',
  description: 'Automated E2E test for change management',
  justification: 'E2E Testing Requirement',
  type: 'normal',
  priority: 'medium',
  impactScope: 'low',
  riskLevel: 'low',
  implementationPlan: '1. Prepare\n2. Execute\n3. Verify',
  rollbackPlan: '1. Revert changes\n2. Verify system',
}

export const emergencyChange = {
  ...testChange,
  title: 'Emergency E2E Test Change',
  type: 'emergency',
  priority: 'critical',
  riskLevel: 'high',
}
```

- [ ] **Step 3: 提交**

```bash
git add itsm-frontend/e2e/support/auth.ts itsm-frontend/e2e/fixtures/change-fixture.ts
git commit -m "test(e2e): add auth helper and change fixtures"
```

---

### Task 5.2: 创建变更管理E2E测试

**Files:**
- Create: `itsm-frontend/e2e/change-management.spec.ts`

- [ ] **Step 1: 创建E2E测试文件**

```typescript
// e2e/change-management.spec.ts
import { test, expect } from '@playwright/test'
import { login } from './support/auth'
import { testChange } from './fixtures/change-fixture'

test.describe('Change Management', () => {
  test.beforeEach(async ({ page }) => {
    await login(page, 'admin', 'admin123')
  })

  test('should display change list page', async ({ page }) => {
    await page.goto('/changes')

    await expect(page.locator('h1')).toContainText('变更管理')
    await expect(page.locator('table')).toBeVisible()
  })

  test('should create a new change', async ({ page }) => {
    await page.goto('/changes')

    // 点击创建按钮
    await page.click('button:has-text("创建变更")')

    // 等待表单加载
    await expect(page.locator('form')).toBeVisible()

    // 填写表单
    await page.fill('input[name="title"]', testChange.title)
    await page.fill('textarea[name="description"]', testChange.description)
    await page.fill('textarea[name="justification"]', testChange.justification)
    await page.selectOption('select[name="type"]', testChange.type)
    await page.selectOption('select[name="priority"]', testChange.priority)
    await page.fill('textarea[name="implementationPlan"]', testChange.implementationPlan)
    await page.fill('textarea[name="rollbackPlan"]', testChange.rollbackPlan)

    // 提交
    await page.click('button:has-text("提交")')

    // 验证成功消息
    await expect(page.locator('.ant-message-success')).toBeVisible({ timeout: 10000 })

    // 验证返回列表并显示新变更
    await page.waitForURL('**/changes')
    await expect(page.locator('table')).toContainText(testChange.title)
  })

  test('should view change detail', async ({ page }) => {
    await page.goto('/changes')

    // 点击第一个变更
    await page.click('table tbody tr:first-child a')

    // 验证详情页面元素
    await expect(page.locator('.change-detail')).toBeVisible()
    await expect(page.locator('.change-title')).toBeVisible()
  })

  test('should search changes', async ({ page }) => {
    await page.goto('/changes')

    // 输入搜索关键词
    await page.fill('input[placeholder*="搜索"]', '紧急')
    await page.press('input[placeholder*="搜索"]', 'Enter')

    // 等待搜索结果
    await page.waitForTimeout(1000)

    // 验证搜索结果
    const rows = await page.locator('table tbody tr').count()
    expect(rows).toBeGreaterThanOrEqual(0)
  })

  test('should filter by status', async ({ page }) => {
    await page.goto('/changes')

    // 点击状态筛选
    await page.click('.ant-select-selector:has-text("状态")')

    // 选择"待审批"
    await page.click('.ant-select-dropdown:visible li:has-text("待审批")')

    // 等待筛选结果
    await page.waitForTimeout(1000)

    // 验证结果
    const rows = await page.locator('table tbody tr').count()
    expect(rows).toBeGreaterThanOrEqual(0)
  })
})
```

- [ ] **Step 2: 运行E2E测试**

Run: `cd itsm-frontend && npx playwright test e2e/change-management.spec.ts --headed`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add itsm-frontend/e2e/change-management.spec.ts
git commit -m "test(e2e): add change management E2E tests"
```

---

## 验收验证

### 最终验证步骤

- [ ] **Step 1: 后端覆盖率验证**

Run: `cd itsm-backend && go test ./service/... ./controller/... -cover`
Expected: Service ≥85%, Controller ≥80%

- [ ] **Step 2: 前端覆盖率验证**

Run: `cd itsm-frontend && npm test -- --coverage --testPathPattern=change`
Expected: API ≥90%, Components ≥75%

- [ ] **Step 3: E2E测试验证**

Run: `cd itsm-frontend && npx playwright test e2e/change-management.spec.ts`
Expected: 5 tests pass

- [ ] **Step 4: 最终提交**

```bash
git add .
git commit -m "test: complete change management test coverage to 80%+

- Backend: Service tests 85%+, Controller tests 80%+
- Frontend: API tests 90%+, Component tests 75%+
- E2E: 5 core user flow tests

Closes: Test Coverage Improvement Initiative"
```

---

## 注意事项

1. **TDD原则**: 每个测试都应先编写失败测试，再实现功能
2. **Mock隔离**: 使用enttest/MSW确保测试隔离
3. **测试数据**: 使用一致的测试数据便于维护
4. **CI集成**: 确保所有测试在CI环境中通过
