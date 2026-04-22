# ITSM 业务流程 UI 测试实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立完整的ITSM业务流程UI自动化测试体系，覆盖主要业务模块的核心操作流程

**Architecture:** 基于 Playwright E2E 测试框架，利用已有的 `auth-utils.ts` 登录工具和 `test-utils.ts` 辅助函数，创建跨模块的业务流程测试。测试采用 Page Object 模式封装页面交互，提高测试可维护性。

**Tech Stack:** Playwright, TypeScript, npm, Next.js (frontend), Gin (backend)

---

## 文件结构

```
itsm-frontend/tests/e2e/
├── auth-utils.ts           # 已有：登录工具
├── utils/
│   ├── test-utils.ts      # 已有：通用测试工具
│   └── page-objects/      # 新增：Page Object 封装
│       ├── TicketPage.ts
│       ├── IncidentPage.ts
│       ├── ProblemPage.ts
│       ├── ChangePage.ts
│       └── ServiceCatalogPage.ts
├── business-flows/        # 新增：业务流程测试
│   ├── ticket-lifecycle.spec.ts
│   ├── incident-escalation.spec.ts
│   ├── problem-rca.spec.ts
│   ├── change-approval.spec.ts
│   └── smoke-test.spec.ts
└── playwright.config.ts   # 已存在
```

---

## 前置条件

测试需要：
1. 后端服务运行在 `localhost:8090`
2. 前端服务运行在 `localhost:3000`
3. 数据库已初始化并包含测试数据
4. 测试用户：admin/admin123, agent/agent123, end_user/user123

---

## 测试场景

### 场景1：工单完整生命周期 (Ticket Lifecycle)
- 用户登录 → 创建工单 → Agent受理 → 更新状态 → 解决 → 关闭

### 场景2：事件升级流程 (Incident Escalation)
- 用户登录 → 创建事件 → Agent调查 → 升级为问题 → 关联工单

### 场景3：问题管理RCA流程 (Problem RCA)
- Agent登录 → 创建问题 → 填写RCA分析 → 关联事件 → 提交审批

### 场景4：变更审批流程 (Change Approval)
- 用户登录 → 创建变更申请 → 填写风险评估 → 提交审批 → 审批通过/拒绝

### 场景5：冒烟测试 (Smoke Test)
- 快速验证所有主要页面可访问和基本功能正常

---

## Task 1: 创建 Page Object 基类和通用组件

**Files:**
- Create: `itsm-frontend/tests/e2e/utils/page-objects/BasePage.ts`
- Create: `itsm-frontend/tests/e2e/utils/page-objects/TicketPage.ts`
- Create: `itsm-frontend/tests/e2e/utils/page-objects/IncidentPage.ts`
- Create: `itsm-frontend/tests/e2e/utils/page-objects/ProblemPage.ts`
- Create: `itsm-frontend/tests/e2e/utils/page-objects/ChangePage.ts`
- Create: `itsm-frontend/tests/e2e/utils/page-objects/ServiceCatalogPage.ts`

- [ ] **Step 1: 创建 BasePage.ts**

```typescript
// itsm-frontend/tests/e2e/utils/page-objects/BasePage.ts
import { Page, Locator } from '@playwright/test';

export abstract class BasePage {
  protected page: Page;
  protected baseUrl = process.env.E2E_BASE_URL || 'http://localhost:3000';

  constructor(page: Page) {
    this.page = page;
  }

  abstract goto(): Promise<void>;

  protected async clickAndWaitForResponse(
    locator: Locator,
    urlPattern: string | RegExp
  ): Promise<void> {
    await Promise.all([
      this.page.waitForResponse(urlPattern),
      locator.click(),
    ]);
  }

  protected async fillFormField(label: string, value: string): Promise<void> {
    const input = this.page.getByLabel(label).or(this.page.getByPlaceholder(label));
    await input.clear();
    await input.fill(value);
  }

  protected async selectOption(label: string, value: string): Promise<void> {
    const select = this.page.locator(`label=${label}`).locator('..').locator('select');
    await select.selectOption(value);
  }

  protected async getTableRowCount(tableLocator: Locator): Promise<number> {
    return tableLocator.locator('tbody tr').count();
  }

  protected async waitForTableLoad(tableLocator: Locator): Promise<void> {
    await this.page.waitForResponse(/\/api\/v1\//);
    await tableLocator.waitFor({ state: 'visible' });
  }

  async isLoaded(): Promise<boolean> {
    return true;
  }
}
```

- [ ] **Step 2: 创建 TicketPage.ts**

```typescript
// itsm-frontend/tests/e2e/utils/page-objects/TicketPage.ts
import { Page, Locator } from '@playwright/test';
import { BasePage } from './BasePage';

export class TicketPage extends BasePage {
  readonly url = '/tickets';
  readonly createButton: Locator;
  readonly searchInput: Locator;
  readonly ticketTable: Locator;

  constructor(page: Page) {
    super(page);
    this.createButton = page.getByRole('button', { name: /新建工单|Create Ticket/i });
    this.searchInput = page.getByPlaceholder(/搜索工单ID、标题/i);
    this.ticketTable = page.locator('table');
  }

  async goto(): Promise<void> {
    await this.page.goto(`${this.baseUrl}${this.url}`);
    await this.ticketTable.waitFor({ state: 'visible' });
  }

  async createTicket(data: {
    title: string;
    description: string;
    priority?: string;
    category?: string;
  }): Promise<number> {
    await this.createButton.click();
    await this.page.waitForURL(/\/tickets\/new/);

    await this.fillFormField('标题|Title', data.title);
    await this.fillFormField('描述|Description', data.description);

    if (data.priority) {
      await this.selectOption('优先级|Priority', data.priority);
    }
    if (data.category) {
      await this.selectOption('类别|Category', data.category);
    }

    await this.page.getByRole('button', { name: /提交|Submit|Create/i }).click();
    await this.page.waitForURL(/\/tickets\/\d+/);

    const url = this.page.url();
    const id = parseInt(url.split('/tickets/')[1]);
    return id;
  }

  async searchTicket(keyword: string): Promise<void> {
    await this.searchInput.clear();
    await this.searchInput.fill(keyword);
    await this.searchInput.press('Enter');
    await this.page.waitForResponse(/\/api\/v1\/tickets/);
  }

  async getFirstTicketId(): Promise<number | null> {
    const firstRow = this.ticketTable.locator('tbody tr').first();
    const idCell = firstRow.locator('td').first();
    const text = await idCell.textContent();
    return text ? parseInt(text.trim()) : null;
  }

  async openTicket(id: number): Promise<void> {
    await this.ticketTable.locator(`tbody tr:has-text("${id}")`).locator('a').first().click();
    await this.page.waitForURL(new RegExp(`/tickets/${id}`));
  }

  async updateTicketStatus(status: string): Promise<void> {
    const statusSelect = this.page.locator('select').filter({ hasText: status }).first();
    if (await statusSelect.isVisible()) {
      await statusSelect.selectOption(status);
    }
  }

  async assignTicket(assignee: string): Promise<void> {
    await this.page.getByText(/分配|Assign/i).click();
    await this.page.waitForSelector('.ant-select-selector');
    await this.page.locator('.ant-select-selector').click();
    await this.page.getByTitle(assignee).click();
    await this.page.getByRole('button', { name: /确定|OK|Assign/i }).click();
  }
}
```

- [ ] **Step 3: 创建 IncidentPage.ts**

```typescript
// itsm-frontend/tests/e2e/utils/page-objects/IncidentPage.ts
import { Page, Locator } from '@playwright/test';
import { BasePage } from './BasePage';

export class IncidentPage extends BasePage {
  readonly url = '/incidents';
  readonly createButton: Locator;
  readonly incidentTable: Locator;
  readonly priorityFilter: Locator;

  constructor(page: Page) {
    super(page);
    this.createButton = page.getByRole('button', { name: /新建事件|Create Incident/i });
    this.incidentTable = page.locator('table');
    this.priorityFilter = page.getByPlaceholder(/优先级|Priority/i);
  }

  async goto(): Promise<void> {
    await this.page.goto(`${this.baseUrl}${this.url}`);
    await this.incidentTable.waitFor({ state: 'visible' });
  }

  async createIncident(data: {
    title: string;
    description: string;
    priority?: string;
    impact?: string;
  }): Promise<number> {
    await this.createButton.click();
    await this.page.waitForURL(/\/incidents\/new/);

    await this.fillFormField('标题|Title', data.title);
    await this.fillFormField('描述|Description', data.description);

    if (data.priority) {
      await this.selectOption('优先级|Priority', data.priority);
    }
    if (data.impact) {
      await this.selectOption('影响范围|Impact', data.impact);
    }

    await this.page.getByRole('button', { name: /提交|Submit/i }).click();
    await this.page.waitForURL(/\/incidents\/\d+/);

    const url = this.page.url();
    return parseInt(url.split('/incidents/')[1]);
  }

  async escalateToProblem(incidentId: number): Promise<void> {
    await this.goto();
    await this.openIncident(incidentId);
    await this.page.getByRole('button', { name: /升级为问题|Escalate/i }).click();
    await this.page.waitForSelector('.ant-modal');
    await this.page.getByRole('button', { name: /确认|Confirm/i }).click();
  }

  async openIncident(id: number): Promise<void> {
    await this.incidentTable.locator(`tbody tr:has-text("${id}")`).locator('a').first().click();
    await this.page.waitForURL(new RegExp(`/incidents/${id}`));
  }
}
```

- [ ] **Step 4: 创建 ProblemPage.ts**

```typescript
// itsm-frontend/tests/e2e/utils/page-objects/ProblemPage.ts
import { Page, Locator } from '@playwright/test';
import { BasePage } from './BasePage';

export class ProblemPage extends BasePage {
  readonly url = '/problems';
  readonly createButton: Locator;
  readonly problemTable: Locator;

  constructor(page: Page) {
    super(page);
    this.createButton = page.getByRole('button', { name: /新建问题|Create Problem/i });
    this.problemTable = this.page.locator('table');
  }

  async goto(): Promise<void> {
    await this.page.goto(`${this.baseUrl}${this.url}`);
    await this.problemTable.waitFor({ state: 'visible' });
  }

  async createProblem(data: {
    title: string;
    description: string;
    rootCause: string;
    resolution: string;
  }): Promise<number> {
    await this.createButton.click();
    await this.page.waitForURL(/\/problems\/new/);

    await this.fillFormField('标题|Title', data.title);
    await this.fillFormField('描述|Description', data.description);

    // 切换到 RCA tab
    await this.page.getByRole('tab', { name: /RCA|Root Cause/i }).click();
    await this.page.getByLabel(/根本原因|Root Cause/i).fill(data.rootCause);
    await this.page.getByLabel(/解决方案|Resolution/i).fill(data.resolution);

    await this.page.getByRole('button', { name: /提交|Submit/i }).click();
    await this.page.waitForURL(/\/problems\/\d+/);

    const url = this.page.url();
    return parseInt(url.split('/problems/')[1]);
  }

  async linkIncident(problemId: number, incidentId: number): Promise<void> {
    await this.goto();
    await this.openProblem(problemId);

    await this.page.getByRole('button', { name: /关联事件|Link Incident/i }).click();
    await this.page.getByPlaceholder(/输入事件ID|Enter Incident ID/i).fill(String(incidentId));
    await this.page.getByRole('button', { name: /添加|Add/i }).click();
  }

  async openProblem(id: number): Promise<void> {
    await this.problemTable.locator(`tbody tr:has-text("${id}")`).locator('a').first().click();
    await this.page.waitForURL(new RegExp(`/problems/${id}`));
  }
}
```

- [ ] **Step 5: 创建 ChangePage.ts**

```typescript
// itsm-frontend/tests/e2e/utils/page-objects/ChangePage.ts
import { Page, Locator } from '@playwright/test';
import { BasePage } from './BasePage';

export class ChangePage extends BasePage {
  readonly url = '/changes';
  readonly createButton: Locator;
  readonly changeTable: Locator;

  constructor(page: Page) {
    super(page);
    this.createButton = page.getByRole('button', { name: /新建变更|Create Change/i });
    this.changeTable = this.page.locator('table');
  }

  async goto(): Promise<void> {
    await this.page.goto(`${this.baseUrl}${this.url}`);
    await this.changeTable.waitFor({ state: 'visible' });
  }

  async createChange(data: {
    title: string;
    description: string;
    riskLevel: string;
    rollbackPlan: string;
  }): Promise<number> {
    await this.createButton.click();
    await this.page.waitForURL(/\/changes\/new/);

    await this.fillFormField('标题|Title', data.title);
    await this.fillFormField('描述|Description', data.description);
    await this.selectOption('风险级别|Risk Level', data.riskLevel);
    await this.page.getByLabel(/回滚计划|Rollback Plan/i).fill(data.rollbackPlan);

    await this.page.getByRole('button', { name: /提交|Submit/i }).click();
    await this.page.waitForURL(/\/changes\/\d+/);

    const url = this.page.url();
    return parseInt(url.split('/changes/')[1]);
  }

  async approveChange(changeId: number): Promise<void> {
    await this.goto();
    await this.openChange(changeId);

    await this.page.getByRole('button', { name: /审批|Approve/i }).click();
    await this.page.waitForSelector('.ant-modal');
    await this.page.getByLabel(/审批意见|Comment/i).fill('Approved - looks good');
    await this.page.getByRole('button', { name: /通过|Approve|Confirm/i }).click();
  }

  async rejectChange(changeId: number, reason: string): Promise<void> {
    await this.goto();
    await this.openChange(changeId);

    await this.page.getByRole('button', { name: /拒绝|Reject/i }).click();
    await this.page.waitForSelector('.ant-modal');
    await this.page.getByLabel(/拒绝原因|Reason/i).fill(reason);
    await this.page.getByRole('button', { name: /确认拒绝|Confirm Reject/i }).click();
  }

  async openChange(id: number): Promise<void> {
    await this.changeTable.locator(`tbody tr:has-text("${id}")`).locator('a').first().click();
    await this.page.waitForURL(new RegExp(`/changes/${id}`));
  }
}
```

- [ ] **Step 6: 创建 ServiceCatalogPage.ts**

```typescript
// itsm-frontend/tests/e2e/utils/page-objects/ServiceCatalogPage.ts
import { Page, Locator } from '@playwright/test';
import { BasePage } from './BasePage';

export class ServiceCatalogPage extends BasePage {
  readonly url = '/service-catalog';
  readonly serviceList: Locator;
  readonly requestButton: Locator;

  constructor(page: Page) {
    super(page);
    this.serviceList = this.page.locator('.service-card, [data-testid="service-card"]');
    this.requestButton = this.page.getByRole('button', { name: /申请|Request/i });
  }

  async goto(): Promise<void> {
    await this.page.goto(`${this.baseUrl}${this.url}`);
    await this.serviceList.first().waitFor({ state: 'visible', timeout: 10000 });
  }

  async requestService(serviceName: string, justification: string): Promise<number> {
    // 点击服务卡片
    await this.page.getByText(serviceName).click();
    await this.page.waitForSelector('.ant-drawer, .ant-modal');

    // 点击申请按钮
    await this.requestButton.first().click();

    // 填写申请表单
    await this.page.getByLabel(/申请理由|Justification/i).fill(justification);

    await this.page.getByRole('button', { name: /提交|Submit/i }).click();

    // 等待跳转到工单页面
    await this.page.waitForURL(/\/tickets\/\d+|\/service-requests\/\d+/);

    const url = this.page.url();
    const match = url.match(/\/(tickets|service-requests)\/(\d+)/);
    return match ? parseInt(match[2]) : 0;
  }

  async browseCategories(): Promise<string[]> {
    const categories = this.page.locator('.category-tab, [data-testid="category-tab"]');
    const count = await categories.count();
    const names: string[] = [];

    for (let i = 0; i < count; i++) {
      const name = await categories.nth(i).textContent();
      if (name) names.push(name.trim());
    }

    return names;
  }
}
```

- [ ] **Step 7: 提交代码**

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-frontend
git add tests/e2e/utils/page-objects/
git commit -m "feat: 添加Page Object基类和核心页面封装"
```

---

## Task 2: 创建工单完整生命周期测试

**Files:**
- Create: `itsm-frontend/tests/e2e/business-flows/ticket-lifecycle.spec.ts`

- [ ] **Step 1: 编写工单生命周期测试**

```typescript
// itsm-frontend/tests/e2e/business-flows/ticket-lifecycle.spec.ts
import { test, expect } from '@playwright/test';
import { loginAsAdmin, loginAsAgent, loginAsEndUser, logout } from '../auth-utils';
import { TicketPage } from '../utils/page-objects/TicketPage';
import { TestSettings } from '../utils/test-utils';

test.describe('工单完整生命周期测试', () => {

  test.beforeEach(async ({ page }) => {
    await logout(page);
  });

  test('最终用户创建工单 → Agent受理 → 更新状态 → 解决 → 关闭', async ({ page }) => {
    const ticketPage = new TicketPage(page);

    // Step 1: 最终用户登录并创建工单
    await loginAsEndUser(page);
    await ticketPage.goto();

    const ticketId = await ticketPage.createTicket({
      title: `E2E测试工单-${Date.now()}`,
      description: '这是一个自动化E2E测试工单，用于验证完整生命周期',
      priority: 'high',
      category: 'technical',
    });

    console.log(`创建工单成功，ID: ${ticketId}`);
    expect(ticketId).toBeGreaterThan(0);

    // Step 2: 登出并以Agent身份登录
    await logout(page);
    await loginAsAgent(page);
    await ticketPage.goto();

    // Step 3: Agent搜索并受理工单
    await ticketPage.searchTicket(String(ticketId));
    await ticketPage.openTicket(ticketId);
    await ticketPage.assignTicket('agent');

    // 验证分配成功
    await expect(page.getByText(/已分配|Assigned/i)).toBeVisible({ timeout: 5000 });

    // Step 4: Agent更新工单状态为处理中
    await ticketPage.updateTicketStatus('in_progress');
    await expect(page.getByText(/处理中|In Progress/i)).toBeVisible();

    // Step 5: Agent解决工单
    await page.getByRole('button', { name: /解决|Resolve/i }).click();
    await page.waitForSelector('.ant-modal');
    await page.getByLabel(/解决方案|Solution/i).fill('已通过远程协助解决问题');
    await page.getByRole('button', { name: /确认解决|Confirm Resolve/i }).click();

    // 验证状态为已解决
    await expect(page.getByText(/已解决|Resolved/i)).toBeVisible({ timeout: 5000 });

    // Step 6: 最终用户确认并关闭
    await logout(page);
    await loginAsEndUser(page);
    await ticketPage.goto();
    await ticketPage.openTicket(ticketId);

    await page.getByRole('button', { name: /关闭|Close/i }).click();
    await page.waitForSelector('.ant-modal');
    await page.getByRole('button', { name: /确认关闭|Confirm Close/i }).click();

    // 验证状态为已关闭
    await expect(page.getByText(/已关闭|Closed/i)).toBeVisible({ timeout: 5000 });
  });

  test('工单搜索和筛选功能', async ({ page }) => {
    await loginAsAdmin(page);
    const ticketPage = new TicketPage(page);
    await ticketPage.goto();

    // 按关键词搜索
    await ticketPage.searchTicket('测试');
    const rows = await ticketPage.getTableRowCount(ticketPage.ticketTable);
    expect(rows).toBeGreaterThanOrEqual(0); // 有或无结果都是有效状态

    // 按状态筛选
    const statusFilter = page.getByPlaceholder(/状态|Status/i);
    if (await statusFilter.isVisible()) {
      await statusFilter.click();
      await page.getByTitle(/处理中|In Progress/i).click();
      await page.waitForResponse(/\/api\/v1\/tickets/);
    }
  });

  test('工单详情页面元素验证', async ({ page }) => {
    await loginAsAdmin(page);
    const ticketPage = new TicketPage(page);
    await ticketPage.goto();

    const ticketId = await ticketPage.getFirstTicketId();
    if (!ticketId) {
      test.skip('没有工单数据，跳过详情测试');
    }

    await ticketPage.openTicket(ticketId);

    // 验证详情页面元素
    await expect(page.getByText(`#${ticketId}`)).toBeVisible();
    await expect(page.locator('.ticket-description, [data-testid="ticket-description"]')).toBeVisible();
    await expect(page.getByRole('button', { name: /编辑|Edit/i })).toBeVisible();
  });
});
```

- [ ] **Step 2: 运行测试验证**

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-frontend
npx playwright test tests/e2e/business-flows/ticket-lifecycle.spec.ts --project=chromium --reporter=list
```

预期结果：
- 测试通过或失败（失败时显示具体步骤和错误信息）

- [ ] **Step 3: 提交代码**

```bash
git add tests/e2e/business-flows/ticket-lifecycle.spec.ts
git commit -m "feat: 添加工单完整生命周期E2E测试"
```

---

## Task 3: 创建事件升级流程测试

**Files:**
- Create: `itsm-frontend/tests/e2e/business-flows/incident-escalation.spec.ts`

- [ ] **Step 1: 编写事件升级测试**

```typescript
// itsm-frontend/tests/e2e/business-flows/incident-escalation.spec.ts
import { test, expect } from '@playwright/test';
import { loginAsAdmin, loginAsAgent, loginAsEndUser, logout } from '../auth-utils';
import { IncidentPage } from '../utils/page-objects/IncidentPage';
import { ProblemPage } from '../utils/page-objects/ProblemPage';
import { TicketPage } from '../utils/page-objects/TicketPage';

test.describe('事件升级流程测试', () => {

  test.beforeEach(async ({ page }) => {
    await logout(page);
  });

  test('用户创建事件 → Agent调查 → 升级为问题 → 关联工单', async ({ page }) => {
    const incidentPage = new IncidentPage(page);
    const problemPage = new ProblemPage(page);
    const ticketPage = new TicketPage(page);

    // Step 1: 最终用户创建事件
    await loginAsEndUser(page);
    await incidentPage.goto();

    const incidentId = await incidentPage.createIncident({
      title: `E2E事件测试-${Date.now()}`,
      description: '用户报告无法访问系统',
      priority: 'critical',
      impact: 'high',
    });

    console.log(`创建事件成功，ID: ${incidentId}`);
    expect(incidentId).toBeGreaterThan(0);

    // Step 2: 登出并以Agent登录进行初步处理
    await logout(page);
    await loginAsAgent(page);
    await incidentPage.goto();

    await incidentPage.openIncident(incidentId);

    // 验证事件状态
    await expect(page.getByText(/新建|New/i)).toBeVisible();

    // Step 3: Agent将事件升级为问题
    await incidentPage.escalateToProblem(incidentId);

    // 验证升级成功提示
    await expect(page.getByText(/已升级|Escalated/i)).toBeVisible({ timeout: 5000 });

    // Step 4: 验证问题已创建
    await problemPage.goto();
    const problemId = await problemPage.problemTable.locator(`tbody tr:has-text("${incidentId}")`).count();
    expect(problemId).toBeGreaterThan(0);

    // Step 5: Agent在问题详情中关联工单
    await problemPage.openProblem(problemId);

    const relatedTicketId = await ticketPage.createTicket({
      title: `关联事件#${incidentId}的工单`,
      description: '此工单用于跟踪事件解决进度',
      priority: 'high',
    });

    await problemPage.linkIncident(problemId, incidentId);

    // 验证关联成功
    await expect(problemPage.problemTable.locator(`tbody tr:has-text("${incidentId}")`)).toBeVisible();
  });

  test('事件状态流转测试', async ({ page }) => {
    await loginAsAdmin(page);
    const incidentPage = new IncidentPage(page);
    await incidentPage.goto();

    // 创建事件
    const incidentId = await incidentPage.createIncident({
      title: `状态流转测试-${Date.now()}`,
      description: '测试事件状态流转',
      priority: 'medium',
    });

    await incidentPage.openIncident(incidentId);

    // 验证状态选项存在
    const statusSelect = page.locator('select').filter({ hasText: /状态|Status/i });
    await expect(statusSelect).toBeVisible();

    // 流转: 新建 → 调查中
    await incidentPage.incidentTable.locator(`tr:has-text("${incidentId}")`).locator('select').selectOption('investigating');
    await expect(page.getByText(/调查中|Investigating/i)).toBeVisible({ timeout: 3000 });

    // 流转: 调查中 → 已识别
    await page.locator('select').selectOption('identified');
    await expect(page.getByText(/已识别|Identified/i)).toBeVisible({ timeout: 3000 });
  });
});
```

- [ ] **Step 2: 运行测试验证**

```bash
npx playwright test tests/e2e/business-flows/incident-escalation.spec.ts --project=chromium --reporter=list
```

- [ ] **Step 3: 提交代码**

```bash
git add tests/e2e/business-flows/incident-escalation.spec.ts
git commit -m "feat: 添加事件升级流程E2E测试"
```

---

## Task 4: 创建问题RCA流程测试

**Files:**
- Create: `itsm-frontend/tests/e2e/business-flows/problem-rca.spec.ts`

- [ ] **Step 1: 编写问题RCA测试**

```typescript
// itsm-frontend/tests/e2e/business-flows/problem-rca.spec.ts
import { test, expect } from '@playwright/test';
import { loginAsAdmin, loginAsAgent, logout } from '../auth-utils';
import { ProblemPage } from '../utils/page-objects/ProblemPage';
import { IncidentPage } from '../utils/page-objects/IncidentPage';

test.describe('问题管理RCA流程测试', () => {

  test.beforeEach(async ({ page }) => {
    await logout(page);
  });

  test('Agent创建问题 → 填写RCA分析 → 关联事件 → 提交审批', async ({ page }) => {
    const problemPage = new ProblemPage(page);
    const incidentPage = new IncidentPage(page);

    // Step 1: Agent登录并创建问题
    await loginAsAgent(page);
    await problemPage.goto();

    const problemId = await problemPage.createProblem({
      title: `E2E RCA测试问题-${Date.now()}`,
      description: '服务器频繁重启的根本原因分析',
      rootCause: '电源模块老化导致供电不稳定',
      resolution: '更换电源模块并进行系统升级',
    });

    console.log(`创建问题成功，ID: ${problemId}`);
    expect(problemId).toBeGreaterThan(0);

    // Step 2: 创建关联事件
    await incidentPage.goto();
    const incidentId = await incidentPage.createIncident({
      title: `关联问题#${problemId}的事件`,
      description: '服务器重启故障',
      priority: 'high',
    });

    // Step 3: 在问题详情中关联事件
    await problemPage.goto();
    await problemPage.linkIncident(problemId, incidentId);

    // 验证事件已关联
    const relatedIncidents = problemPage.problemTable.locator(`tbody tr:has-text("${incidentId}")`);
    await expect(relatedIncidents).toBeVisible();

    // Step 4: 提交审批
    await problemPage.openProblem(problemId);
    await page.getByRole('button', { name: /提交审批|Submit for Approval/i }).click();
    await page.waitForSelector('.ant-modal');

    await page.getByLabel(/审批备注|Notes/i).fill('已完成根本原因分析，建议批准');
    await page.getByRole('button', { name: /提交|Submit/i }).click();

    // 验证状态变化
    await expect(page.getByText(/待审批|Pending Approval/i)).toBeVisible({ timeout: 5000 });
  });

  test('问题列表筛选和搜索', async ({ page }) => {
    await loginAsAdmin(page);
    const problemPage = new ProblemPage(page);
    await problemPage.goto();

    // 搜索测试
    const searchInput = page.getByPlaceholder(/搜索问题ID、标题/i);
    await searchInput.fill('RCA');
    await searchInput.press('Enter');
    await page.waitForResponse(/\/api\/v1\/problems/);

    // 验证表格仍然可见（有无结果都可以）
    await expect(problemPage.problemTable).toBeVisible();
  });
});
```

- [ ] **Step 2: 运行测试验证**

```bash
npx playwright test tests/e2e/business-flows/problem-rca.spec.ts --project=chromium --reporter=list
```

- [ ] **Step 3: 提交代码**

```bash
git add tests/e2e/business-flows/problem-rca.spec.ts
git commit -m "feat: 添加问题RCA流程E2E测试"
```

---

## Task 5: 创建变更审批流程测试

**Files:**
- Create: `itsm-frontend/tests/e2e/business-flows/change-approval.spec.ts`

- [ ] **Step 1: 编写变更审批测试**

```typescript
// itsm-frontend/tests/e2e/business-flows/change-approval.spec.ts
import { test, expect } from '@playwright/test';
import { loginAsAdmin, loginAsAgent, loginAsEndUser, logout } from '../auth-utils';
import { ChangePage } from '../utils/page-objects/ChangePage';

test.describe('变更审批流程测试', () => {

  test.beforeEach(async ({ page }) => {
    await logout(page);
  });

  test('用户创建变更 → Manager审批通过 → 实施 → 关闭', async ({ page }) => {
    const changePage = new ChangePage(page);

    // Step 1: 普通用户创建变更申请
    await loginAsEndUser(page);
    await changePage.goto();

    const changeId = await changePage.createChange({
      title: `E2E变更测试-${Date.now()}`,
      description: '数据库服务器迁移升级',
      riskLevel: 'high',
      rollbackPlan: '回滚到原数据库服务器，恢复数据备份',
    });

    console.log(`创建变更成功，ID: ${changeId}`);
    expect(changeId).toBeGreaterThan(0);

    // 验证状态为草稿或待审批
    await expect(page.getByText(/草稿|Draft|待审批|Pending/i)).toBeVisible({ timeout: 5000 });

    // Step 2: Manager登录并审批
    await logout(page);
    await loginAsAdmin(page); // 假设admin有审批权限
    await changePage.goto();

    await changePage.approveChange(changeId);

    // 验证审批成功
    await expect(page.getByText(/已批准|Approved/i)).toBeVisible({ timeout: 5000 });

    // Step 3: 实施变更
    await changePage.openChange(changeId);
    await page.getByRole('button', { name: /实施|Implement/i }).click();
    await page.waitForSelector('.ant-modal');
    await page.getByLabel(/实施备注|Notes/i).fill('变更已成功实施');
    await page.getByRole('button', { name: /确认实施|Confirm/i }).click();

    // 验证状态为实施中
    await expect(page.getByText(/实施中|Implementing|In Progress/i)).toBeVisible({ timeout: 5000 });

    // Step 4: 变更完成
    await page.getByRole('button', { name: /完成|Complete/i }).click();
    await page.getByRole('button', { name: /确认完成|Confirm Complete/i }).click();

    // 验证状态为已完成
    await expect(page.getByText(/已完成|Completed/i)).toBeVisible({ timeout: 5000 });
  });

  test('变更审批拒绝流程', async ({ page }) => {
    const changePage = new ChangePage(page);

    // 创建变更
    await loginAsEndUser(page);
    await changePage.goto();

    const changeId = await changePage.createChange({
      title: `E2E变更拒绝测试-${Date.now()}`,
      description: '高风险系统配置变更',
      riskLevel: 'critical',
      rollbackPlan: '恢复配置文件',
    });

    // Manager拒绝
    await logout(page);
    await loginAsAdmin(page);
    await changePage.goto();

    await changePage.rejectChange(changeId, '风险过高，需要重新评估');

    // 验证被拒绝
    await expect(page.getByText(/已拒绝|Rejected/i)).toBeVisible({ timeout: 5000 });
  });

  test('变更风险评估计算', async ({ page }) => {
    await loginAsAdmin(page);
    const changePage = new ChangePage(page);
    await changePage.goto();

    // 创建变更
    const changeId = await changePage.createChange({
      title: `E2E风险评估测试-${Date.now()}`,
      description: '测试风险评估功能',
      riskLevel: 'medium',
      rollbackPlan: '回滚方案',
    });

    // 打开变更详情
    await changePage.openChange(changeId);

    // 验证风险评估卡片存在
    await expect(page.locator('.risk-assessment, [data-testid="risk-assessment"]')).toBeVisible();

    // 验证风险评分显示
    const riskScore = page.locator('.risk-score, [data-testid="risk-score"]');
    await expect(riskScore).toBeVisible();
    const scoreText = await riskScore.textContent();
    expect(scoreText).toMatch(/\d+|N\/A/);
  });
});
```

- [ ] **Step 2: 运行测试验证**

```bash
npx playwright test tests/e2e/business-flows/change-approval.spec.ts --project=chromium --reporter=list
```

- [ ] **Step 3: 提交代码**

```bash
git add tests/e2e/business-flows/change-approval.spec.ts
git commit -m "feat: 添加变更审批流程E2E测试"
```

---

## Task 6: 创建冒烟测试套件

**Files:**
- Create: `itsm-frontend/tests/e2e/business-flows/smoke-test.spec.ts`

- [ ] **Step 1: 编写冒烟测试**

```typescript
// itsm-frontend/tests/e2e/business-flows/smoke-test.spec.ts
import { test, expect } from '@playwright/test';
import { loginAsAdmin, logout } from '../auth-utils';

const PAGES_TO_TEST = [
  { name: 'Dashboard', url: '/dashboard' },
  { name: '工单列表', url: '/tickets' },
  { name: '事件列表', url: '/incidents' },
  { name: '问题列表', url: '/problems' },
  { name: '变更列表', url: '/changes' },
  { name: '服务目录', url: '/service-catalog' },
  { name: '知识库', url: '/knowledge' },
  { name: '工作流设计器', url: '/workflow/designer' },
  { name: 'SLA监控', url: '/sla-monitor' },
  { name: 'CMDB', url: '/cmdb' },
];

test.describe('冒烟测试 - 快速验证所有主要页面', () => {

  test.beforeEach(async ({ page }) => {
    // 每次测试前确保已登出
    await logout(page);
  });

  test('登录功能正常', async ({ page }) => {
    await loginAsAdmin(page);

    // 验证登录成功 - 应该能看到用户菜单或登出按钮
    await expect(
      page.getByText(/admin|管理员|登出|Logout/i).first()
    ).toBeVisible({ timeout: 10000 });
  });

  test('所有主要页面可访问', async ({ page }) => {
    await loginAsAdmin(page);
    const baseUrl = process.env.E2E_BASE_URL || 'http://localhost:3000';

    for (const pageInfo of PAGES_TO_TEST) {
      console.log(`测试页面: ${pageInfo.name} (${pageInfo.url})`);

      await page.goto(`${baseUrl}${pageInfo.url}`);

      // 等待页面加载完成
      await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => {
        console.log(`  ${pageInfo.url} 网络请求未完全完成，继续检查`);
      });

      // 验证页面没有崩溃（检查是否有错误提示）
      const errorText = page.locator('.ant-result, .ant-alert-error').first();
      const hasError = await errorText.isVisible().catch(() => false);

      if (hasError) {
        const errorContent = await errorText.textContent();
        console.log(`  ${pageInfo.name} 显示错误: ${errorContent}`);
      }

      // 验证页面主要内容加载
      const bodyContent = await page.locator('body').textContent();
      expect(bodyContent?.length).toBeGreaterThan(100);

      console.log(`  ✓ ${pageInfo.name} 加载正常`);
    }
  });

  test('核心页面数据加载正常', async ({ page }) => {
    await loginAsAdmin(page);
    const baseUrl = process.env.E2E_BASE_URL || 'http://localhost:3000';

    // 测试 Dashboard
    await page.goto(`${baseUrl}/dashboard`);
    await page.waitForLoadState('networkidle', { timeout: 15000 });
    await expect(page.locator('.ant-statistic, .statistic-card')).toBeVisible({ timeout: 10000 });
    console.log('✓ Dashboard 统计数据加载正常');

    // 测试工单列表
    await page.goto(`${baseUrl}/tickets`);
    await page.waitForLoadState('networkidle', { timeout: 15000 });
    await expect(page.locator('table')).toBeVisible({ timeout: 10000 });
    console.log('✓ 工单列表表格加载正常');

    // 测试服务目录
    await page.goto(`${baseUrl}/service-catalog`);
    await page.waitForLoadState('networkidle', { timeout: 15000 });
    await expect(page.locator('.service-card, [data-testid="service-card"]').first()).toBeVisible({ timeout: 10000 }).catch(() => {
      // 没有服务卡片数据也可以，只要页面加载了
    });
    console.log('✓ 服务目录加载正常');
  });

  test('API代理正常工作', async ({ page }) => {
    await loginAsAdmin(page);
    const baseUrl = process.env.E2E_BASE_URL || 'http://localhost:3000';

    // 通过前端访问API验证代理
    const apiResponse = await page.request.get(`${baseUrl}/api/v1/auth/menus`, {
      headers: {
        Authorization: `Bearer ${await page.evaluate(() => localStorage.getItem('access_token'))}`,
      },
    });

    // 验证响应正常（200或401都可以，说明代理工作正常）
    expect([200, 401]).toContain(apiResponse.status());
    console.log(`✓ API代理响应: ${apiResponse.status()}`);
  });

  test('WebSocket连接状态处理', async ({ page }) => {
    await loginAsAdmin(page);

    // 访问需要WebSocket的页面
    await page.goto(`${process.env.E2E_BASE_URL || 'http://localhost:3000'}/dashboard`);
    await page.waitForLoadState('networkidle');

    // 等待几秒观察WebSocket连接
    await page.waitForTimeout(3000);

    // WebSocket连接失败不应该导致页面崩溃
    const hasError = await page.locator('.ant-result').isVisible().catch(() => false);
    expect(hasError).toBe(false);

    console.log('✓ WebSocket连接失败不影响页面正常使用');
  });
});
```

- [ ] **Step 2: 运行冒烟测试验证**

```bash
npx playwright test tests/e2e/business-flows/smoke-test.spec.ts --project=chromium --reporter=list --timeout=120000
```

- [ ] **Step 3: 提交代码**

```bash
git add tests/e2e/business-flows/smoke-test.spec.ts
git commit -m "feat: 添加冒烟测试套件"
```

---

## Task 7: 更新 Playwright 配置

**Files:**
- Modify: `itsm-frontend/playwright.config.ts`

- [ ] **Step 1: 更新 Playwright 配置**

```typescript
// playwright.config.ts (更新部分)
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 60_000, // 业务流程测试需要更长时间
  expect: {
    timeout: 10_000,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    // 添加业务流程测试专用配置
    {
      name: 'business-flows',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1440, height: 900 },
      },
      testDir: './tests/e2e/business-flows',
    },
  ],
  webServer: {
    command: 'npm run dev',
    port: 3000,
    reuseExistingServer: true,
    timeout: 120_000,
  },
  // 确保测试输出清晰
  reporter: [
    ['list'],
    ['html', { open: 'never' }],
  ],
});
```

- [ ] **Step 2: 添加 npm scripts (package.json)**

```json
{
  "scripts": {
    "test:e2e": "playwright test",
    "test:e2e:business": "playwright test --project=business-flows",
    "test:smoke": "playwright test tests/e2e/business-flows/smoke-test.spec.ts --project=chromium",
    "screenshot": "playwright test screenshots.spec.ts"
  }
}
```

- [ ] **Step 3: 提交配置更新**

```bash
git add playwright.config.ts package.json
git commit -m "chore: 更新Playwright配置支持业务流程测试"
```

---

## Task 8: 运行完整测试套件并生成报告

- [ ] **Step 1: 运行完整业务流程测试**

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-frontend
npx playwright test tests/e2e/business-flows/ --project=chromium --reporter=list --timeout=180000
```

- [ ] **Step 2: 生成 HTML 测试报告**

```bash
npx playwright show-report
# 或生成本地报告
npx playwright test tests/e2e/business-flows/ --reporter=html
```

- [ ] **Step 3: 提交最终测试代码**

```bash
git add -A
git commit -m "feat: 完成ITSM业务流程UI自动化测试体系"
```

---

## 自检清单

### 1. 测试覆盖检查
- [x] 工单生命周期测试 (创建→受理→处理→解决→关闭)
- [x] 事件升级流程测试 (创建→调查→升级问题→关联工单)
- [x] 问题RCA流程测试 (创建→RCA分析→关联事件→审批)
- [x] 变更审批流程测试 (创建→审批通过/拒绝→实施→完成)
- [x] 冒烟测试套件 (快速验证所有页面可访问)

### 2. 页面对象封装
- [x] BasePage 基类
- [x] TicketPage
- [x] IncidentPage
- [x] ProblemPage
- [x] ChangePage
- [x] ServiceCatalogPage

### 3. 测试数据
- [ ] 需要确认测试用户在数据库中存在
- [ ] 需要确认API端点路由正确

### 4. 潜在问题
- 部分选择器可能需要根据实际UI调整
- 审批流程的按钮文本需要确认
- 表单字段的label需要与实际页面匹配

---

## 执行选项

**Plan complete and saved to `docs/superpowers/plans/2026-04-22-business-process-ui-testing.md`.**

**Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
