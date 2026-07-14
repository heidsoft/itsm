# ITSM 项目测试最佳实践

基于 ITSM 项目实际测试经验总结的 E2E 测试最佳实践。

## 1. 测试文件结构

### 推荐的测试文件组织

```
tests/
├── e2e/
│   ├── auth-utils.ts          # 认证辅助函数（统一登录管理）
│   ├── login.spec.ts          # 登录测试
│   ├── navigation.spec.ts     # 导航测试
│   ├── module-access.spec.ts  # 权限保护测试
│   ├── tickets.spec.ts        # 工单模块
│   ├── incidents.spec.ts       # 事件模块
│   ├── problems.spec.ts       # 问题模块
│   ├── changes.spec.ts        # 变更模块
│   ├── cmdb.spec.ts          # CMDB模块
│   ├── knowledge.spec.ts      # 知识库模块
│   └── ...
```

## 2. 认证管理最佳实践

### 2.1 创建统一的认证工具

```typescript
// tests/e2e/auth-utils.ts
import { test as base, Page } from '@playwright/test';

/**
 * 统一的登录辅助函数
 * 解决多个测试文件重复登录逻辑的问题
 */
export async function loginAsAdmin(page: Page) {
  await page.goto('/login');
  await page.waitForLoadState('domcontentloaded');

  // 使用 Ant Design 的通用选择器
  await page.waitForSelector('.ant-input, input.ant-input', { timeout: 15000 });

  const inputs = page.locator('input.ant-input');
  await inputs.nth(0).fill('admin');
  await inputs.nth(1).fill('admin123');

  await page.click('button[type="submit"]');
  await page.waitForURL(/\/(dashboard|tickets|incidents)/, { timeout: 20000 });
}
```

### 2.2 分离认证测试和非认证测试

```typescript
// 公开页面测试 - 无需认证
test.describe('公开页面', () => {
  test('should show login page', async ({ page }) => {
    await page.goto('/login');
    await expect(page.locator('form')).toBeVisible();
  });
});

// 需要认证的测试 - 使用 beforeEach 登录
test.describe('受保护页面', () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test('should display dashboard', async ({ page }) => {
    await page.goto('/dashboard');
    // ...
  });
});
```

## 3. 选择器最佳实践

### 3.1 Ant Design 组件选择器

```typescript
// ❌ 不推荐 - 容易失效
await page.fill('input[name="username"]', 'admin');
await page.fill('input[name="password"]', 'admin123');

// ✅ 推荐 - 使用 Ant Design 类名
await page.waitForSelector('.ant-input, input.ant-input', { timeout: 15000 });
const inputs = page.locator('input.ant-input');
await inputs.nth(0).fill('admin');
await inputs.nth(1).fill('admin123');
```

### 3.2 菜单导航选择器

```typescript
// ❌ 不推荐 - href 可能不存在
const ticketsLink = page.locator('a[href="/tickets"]');

// ✅ 推荐 - 直接使用 URL 导航
await page.goto('/tickets');
await page.waitForURL(/\/tickets/);

// 或使用 Menu 组件类名
const ticketsMenuItem = page.locator('.ant-menu-item').filter({ hasText: /工单/i });
```

### 3.3 避免 Strict Mode Violations

```typescript
// ❌ 不推荐 - 可能匹配多个元素
await expect(page.locator('h1, h2')).toBeVisible();

// ✅ 推荐 - 明确指定
await expect(page.locator('h1, h2').first()).toBeVisible();
```

## 4. 并行测试问题

### 4.1 登录状态冲突

并行测试时，多个 worker 共享登录状态会导致冲突：

```typescript
// ❌ 并行测试时不稳定
test.beforeEach(async ({ page }) => {
  await loginAsAdmin(page);  // 多个 worker 可能冲突
});
```

### 4.2 解决方案

1. **单线程运行关键测试**
   ```bash
   npx playwright test --workers=1
   ```

2. **使用 storageState**
   ```typescript
   // 创建预认证状态
   test.use({
     storageState: './auth.json',  // 需要先创建
   });
   ```

3. **简化测试逻辑** - 先测试无需认证的功能
   ```typescript
   test.describe('无需认证', () => {
     test('should show login', async ({ page }) => {
       await page.goto('/login');
       // ...
     });
   });
   ```

## 5. 路由权限保护测试

### 5.1 测试中间件保护

```typescript
const protectedRoutes = [
  '/releases',
  '/licenses',
  '/assets',
  '/workflow',
  // ...
];

test.describe('Route Protection', () => {
  protectedRoutes.forEach((route) => {
    test(`should redirect ${route} to login`, async ({ page }) => {
      await page.goto(route);
      await expect(page).toHaveURL(/\/login/, { timeout: 15000 });
    });
  });
});
```

### 5.2 添加缺失路由到中间件

```typescript
// src/middleware.ts
const protectedRoutes = [
  '/dashboard',
  '/tickets',
  '/releases',      // 确保添加
  '/licenses',      // 确保添加
  '/assets',        // 确保添加
  '/workflow',
  // ...
];
```

## 6. TypeScript 类型修复

### 6.1 常见 API 类型问题

```typescript
// ❌ 错误 - response.data 类型 unknown
const response = await httpClient.get('/api/data');
return response.data;

// ✅ 正确 - 使用泛型
return httpClient.get<DataType>('/api/data');
```

### 6.2 Dayjs 类型问题

```typescript
// ❌ 错误 - string 类型与 Ant Design DatePicker 不兼容
const [dateRange, setDateRange] = useState<[string, string]>([...]);

// ✅ 正确 - 使用 Dayjs 类型
import dayjs, { Dayjs } from 'dayjs';
const [dateRange, setDateRange] = useState<[Dayjs, Dayjs]>([...]);
```

## 7. 测试调试技巧

### 7.1 查看失败截图

```typescript
// Playwright 自动保存失败截图到 test-results/
```

### 7.2 调试模式

```bash
# UI 模式
npx playwright test --ui

# 调试模式
npx playwright test --debug

# 单文件测试
npx playwright test tests/e2e/login.spec.ts --project=chromium
```

### 7.3 查看日志

```bash
# 显示浏览器控制台日志
npx playwright test --reporter=list
```

## 8. 测试运行策略

### 8.1 本地开发

```bash
# 快速验证 - 单线程
npx playwright test --workers=1

# 完整测试 - 全部通过后
npx playwright test
```

### 8.2 CI/CD

```yaml
# .github/workflows/test.yml
- name: E2E Tests
  run: npx playwright test --reporter=html
- name: Upload Report
  uses: actions/upload-artifact@v3
  with:
    name: playwright-report
    path: playwright-report/
```

## 9. 常见问题排查

| 问题 | 原因 | 解决方案 |
|-----|------|---------|
| 登录超时 | 选择器不匹配 | 使用 `.ant-input` 选择器 |
| 并行测试失败 | 登录状态冲突 | 使用 `--workers=1` |
| 页面加载慢 | 网络请求慢 | 增加 timeout |
| 权限测试失败 | 路由未添加到中间件 | 更新 middleware.ts |
| TypeScript 错误 | API 类型不匹配 | 使用泛型指定类型 |

## 10. 测试覆盖率目标

| 测试类型 | 目标覆盖率 |
|---------|----------|
| 单元测试 | 70%+ |
| 集成测试 | 50%+ |
| E2E 测试 | 核心流程全覆盖 |

---

**经验总结**: 测试的本质是保证产品质量，E2E 测试应该关注核心用户流程，而不是追求覆盖率。优先测试：
1. 登录/认证流程
2. 核心业务功能
3. 权限控制
4. 数据 CRUD 操作
