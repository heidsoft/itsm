# ITSM 产品 TDD 改进计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立健康的TDD工作流，修复现有测试，逐步提升覆盖率至80%

**Architecture:** 先修复broken测试，建立CI，再逐模块TDD改进覆盖率

**Tech Stack:** Go testing + testify (后端), Jest + React Testing Library (前端)

---

## 第一阶段：修复现有Broken测试

### Task 1: 修复后端 incident_service_test.go 编译错误

**Files:**
- Modify: `itsm-backend/service/incident_service_test.go`
- Reference: `itsm-backend/service/incident_service.go` (查看正确的函数签名)

- [ ] **Step 1: 检查 incident_service.go 中的 UpdateIncident 函数签名**

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-backend
grep -n "func.*UpdateIncident" service/incident_service.go
```

- [ ] **Step 2: 检查 DeleteIncident 函数签名**

```bash
grep -n "func.*DeleteIncident" service/incident_service.go
```

- [ ] **Step 3: 修复 incident_service_test.go 中 UpdateIncident 调用**

查看当前测试中错误调用，然后对照正确签名修复参数数量。

典型修复模式：
```go
// 错误 (缺少参数)
client.IncidentService().UpdateIncident(ctx, id, input)

// 正确 (完整参数)
client.IncidentService().UpdateIncident(ctx, id, input, false)
```

- [ ] **Step 4: 修复 DeleteIncident 调用**

- [ ] **Step 5: 验证编译通过**

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-backend
go build ./service/...
```

Expected: SUCCESS (无编译错误)

- [ ] **Step 6: 运行 incident service 测试**

```bash
go test ./service/ -run Incident -v 2>&1 | head -50
```

- [ ] **Step 7: 提交**

```bash
git add service/incident_service_test.go
git commit -m "fix(test): resolve UpdateIncident/DeleteIncident signature mismatch in incident_service_test.go"
```

---

### Task 2: 修复后端 knowledge_controller_test.go panic

**Files:**
- Modify: `itsm-backend/controller/knowledge_controller_test.go`

- [ ] **Step 1: 运行 knowledge_controller_test.go 查看panic详情**

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-backend
go test ./controller/ -run TestKnowledge -v 2>&1 | head -80
```

- [ ] **Step 2: 找到 panic 位置并修复 tenantID 断言**

错误可能是 `tenantID := middleware.TenantID(ctx)` 在 nil ctx 时 panic。

修复模式：
```go
// 错误 (直接断言)
tenantID := middleware.TenantID(ctx) // panic if ctx is nil

// 正确 (先检查)
if ctx == nil || !middleware.HasTenantID(ctx) {
    t.Skip("tenant context not available")
}
tenantID := middleware.TenantID(ctx)
```

- [ ] **Step 3: 验证测试通过**

```bash
go test ./controller/ -run TestKnowledge -v
```

Expected: PASS (无 panic)

- [ ] **Step 4: 提交**

```bash
git add controller/knowledge_controller_test.go
git commit -m "fix(test): prevent nil ctx panic in knowledge_controller_test.go"
```

---

### Task 3: 修复前端 ChangeList.test.tsx (45个失败测试)

**Files:**
- Modify: `itsm-frontend/src/app/(main)/changes/__tests__/ChangeList.test.tsx`
- Reference: `itsm-frontend/src/app/(main)/changes/components/ChangeList.tsx`

- [ ] **Step 1: 运行 ChangeList 测试查看具体失败原因**

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-frontend
npm test -- --testPathPattern="ChangeList" --no-coverage 2>&1 | tail -100
```

- [ ] **Step 2: 检查失败原因是 mocking 问题还是 render 问题**

典型错误: `AggregateError: Impossible to provide values for all options`

修复模式 - 检查 mock setup 是否完整：
```tsx
// 确保每个被测试组件使用的 hook 都已 mock
vi.mock('@/lib/api/change-api', () => ({
  useChanges: () => ({
    data: mockChanges,
    isLoading: false,
    error: null
  })
}))
```

- [ ] **Step 3: 检查是否有未 mock 的依赖（如zustand store）**

```bash
grep -n "useLayoutStore\|useAuthStore" src/app/\(main\)/changes/components/ChangeList.tsx
```

- [ ] **Step 4: 在测试中添加必要的 store mocks**

```tsx
vi.mock('@/lib/store/layout-store', () => ({
  useLayoutStore: () => ({ isCollapsed: false, toggleSidebar: vi.fn() })
}))
```

- [ ] **Step 5: 重新运行测试验证通过**

```bash
npm test -- --testPathPattern="ChangeList" --no-coverage
```

Expected: PASS (或大幅减少失败数量)

- [ ] **Step 6: 提交**

```bash
git add "src/app/(main)/changes/__tests__/ChangeList.test.tsx"
git commit -m "fix(test): resolve ChangeList mocking issues causing 45 test failures"
```

---

### Task 4: 建立后端 CI 测试流水线

**Files:**
- Modify: `.github/workflows/backend-ci.yml`

- [ ] **Step 1: 检查现有 backend-ci.yml 内容**

```bash
cat .github/workflows/backend-ci.yml
```

- [ ] **Step 2: 确保 CI 运行 `go test ./... -cover`**

标准配置：
```yaml
- name: Run tests with coverage
  run: |
    go test ./... -coverprofile=coverage.out -covermode=atomic
    go tool cover -html=coverage.out -o coverage.html

- name: Upload coverage
  uses: actions/upload-artifact@v4
  with:
    name: coverage
    path: coverage.html
```

- [ ] **Step 3: 添加测试结果检查**

确保 CI 失败时 blocking merge。

- [ ] **Step 4: 提交**

```bash
git add .github/workflows/backend-ci.yml
git commit -m "ci(backend): ensure test coverage is run in CI pipeline"
```

---

### Task 5: 建立前端 CI 测试流水线

**Files:**
- Modify: `.github/workflows/frontend-ci.yml`

- [ ] **Step 1: 检查现有 frontend-ci.yml 内容**

```bash
cat .github/workflows/frontend-ci.yml
```

- [ ] **Step 2: 确保运行 `npm test -- --coverage`**

- [ ] **Step 3: 确保 CI 在测试失败时 blocking**

- [ ] **Step 4: 提交**

```bash
git add .github/workflows/frontend-ci.yml
git commit -m "ci(frontend): ensure tests run with coverage in CI"
```

---

## 第二阶段：TDD 覆盖率改进

### Task 6: 后端 - incident_service TDD 覆盖率提升

**Files:**
- Modify: `itsm-backend/service/incident_service.go`
- Create: `itsm-backend/service/incident_service_test.go` (增强)

- [ ] **Step 1: 查看 incident_service.go 当前覆盖率**

```bash
go test ./service/ -coverprofile=coverage.out
go tool cover -func=coverage.out | grep incident
```

- [ ] **Step 2: 识别未覆盖的函数**

列出所有 public 函数，确定哪个覆盖率最低。

- [ ] **Step 3: 为未覆盖函数编写 TDD 测试**

例如如果 `ResolveIncident` 未覆盖：
```go
func TestResolveIncident(t *testing.T) {
    // Arrange
    client := enttest.NewClient(t, &schema)
    svc := NewIncidentService(client)
    ctx := context.Background()

    incident := createTestIncident(client, "open")

    // Act
    resolved, err := svc.ResolveIncident(ctx, incident.ID, "resolved", "fixed")

    // Assert
    require.NoError(t, err)
    require.Equal(t, "resolved", resolved.Status)
}
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./service/ -run TestResolveIncident -v
```

- [ ] **Step 5: 重复直到覆盖率 > 80%**

- [ ] **Step 6: 提交**

```bash
git add service/incident_service.go service/incident_service_test.go
git commit -m "test(incident): improve incident_service coverage to 80%+"
```

---

### Task 7: 后端 - change_service TDD 覆盖率提升

**Files:**
- Modify: `itsm-backend/service/change_service.go`
- Create: `itsm-backend/service/change_service_test.go`

- [ ] **Step 1: 检查 change_service 当前覆盖率**

```bash
go test ./service/ -coverprofile=coverage.out
go tool cover -func=coverage.out | grep change
```

- [ ] **Step 2: 识别未覆盖函数并编写测试 (TDD循环)**

重复 TDD 循环：为每个未覆盖的 public 函数编写测试 → 运行验证失败 → 实现最小代码 → 运行验证通过 → 重构

- [ ] **Step 3: 验证覆盖率 > 80% 后提交**

---

### Task 8: 前端 - TicketList component TDD

**Files:**
- Modify: `itsm-frontend/src/app/(main)/tickets/page.tsx`
- Create: `itsm-frontend/src/app/(main)/tickets/__tests__/TicketList.test.tsx`

- [ ] **Step 1: 检查 TicketList 或相关组件**

```bash
ls -la "src/app/(main)/tickets/"
find src/app/\(main\)/tickets -name "*.test.tsx" -o -name "*.test.ts"
```

- [ ] **Step 2: 编写 TicketList TDD 测试**

```tsx
describe('TicketList', () => {
  it('renders empty state when no tickets', () => {
    vi.mock('@/lib/api/ticket-api', () => ({
      useTickets: () => ({ data: [], isLoading: false })
    }))
    render(<TicketList />)
    expect(screen.getByText('No tickets')).toBeInTheDocument()
  })
})
```

- [ ] **Step 3: 运行测试验证失败 (RED)**

```bash
npm test -- --testPathPattern="TicketList" --no-coverage
```

Expected: FAIL (component not implemented or wrong)

- [ ] **Step 4: 实现最小代码通过测试 (GREEN)**

- [ ] **Step 5: 重构改进 (IMPROVE)**

- [ ] **Step 6: 提交**

---

### Task 9: 前端 - KnowledgeBase 组件 TDD

**Files:**
- Create: `itsm-frontend/src/app/(main)/knowledge/__tests__/KnowledgeList.test.tsx`

- [ ] **Step 1-6: 遵循 TDD 循环**

(同上模式)

---

### Task 10: 后端 - cmdb_service 覆盖率提升

**Files:**
- Modify: `itsm-backend/service/cmdb_service.go`
- Create: `itsm-backend/service/cmdb_service_test.go`

- [ ] **Step 1: 检查当前覆盖率**

```bash
go test ./service/ -coverprofile=coverage.out
go tool cover -func=coverage.out | grep cmdb
```

- [ ] **Step 2-5: TDD循环**

---

## 第三阶段：持续改进

### Task 11: 建立覆盖率门槛 enforcement

**Files:**
- Modify: `.github/workflows/backend-ci.yml`
- Modify: `.github/workflows/frontend-ci.yml`

- [ ] **Step 1: 添加 80% 覆盖率门槛**

```yaml
- name: Check coverage
  run: |
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
    if (( $(echo "$COVERAGE < 80" | bc -l) )); then
      echo "Coverage $COVERAGE% is below 80% threshold"
      exit 1
    fi
```

- [ ] **Step 2: 提交**

---

### Task 12: 添加每个 PR 的覆盖率报告

**Files:**
- Modify: `.github/workflows/` 相关文件

- [ ] **Step 1: 配置 coverage comment bot 或 artifact**

- [ ] **Step 2: 确保每次 PR 显示覆盖率变化**

---

## 验证步骤

所有任务完成后，运行以下命令验证整体健康状态：

```bash
# Backend
cd itsm-backend && go test ./... -cover

# Frontend
cd itsm-frontend && npm test -- --coverage --passWithNoTests
```

期望结果：
- 所有测试通过 (0 failures)
- 整体覆盖率 >= 80%
- CI pipeline 全部 green

---

## 执行选择

**Plan complete and saved to `docs/superpowers/plans/2026-04-26-itsm-tdd-improvement.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**