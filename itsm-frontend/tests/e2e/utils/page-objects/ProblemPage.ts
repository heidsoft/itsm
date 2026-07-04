// itsm-frontend/tests/e2e/utils/page-objects/ProblemPage.ts
import type { Page, Locator } from '@playwright/test';
import { BasePage } from './BasePage';

export class ProblemPage extends BasePage {
  readonly url = '/problems';
  readonly createButton: Locator;
  readonly problemTable: Locator;

  constructor(page: Page) {
    super(page);
    this.createButton = page.getByRole('button', { name: /新建问题|Create Problem|新建/i });
    this.problemTable = this.page.locator('table');
  }

  async goto(): Promise<void> {
    await this.page.goto(`${this.baseUrl}${this.url}`);
    await this.problemTable.waitFor({ state: 'visible' }).catch(() => {});
  }

  async createProblem(data: {
    title: string;
    description: string;
    rootCause?: string;
    resolution?: string;
  }): Promise<number> {
    // 直接导航到创建页面
    await this.page.goto(`${this.baseUrl}/problems/create`);
    await this.page.waitForLoadState('networkidle');

    // 填写标题
    const titleInput = this.page.locator('input[id*="title"], input[name*="title"], input[placeholder*="标题"]').first();
    await titleInput.fill(data.title);

    // 填写描述
    const descInput = this.page.locator('textarea[id*="description"], textarea[name*="description"], textarea[placeholder*="描述"]').first();
    await descInput.fill(data.description);

    // 填写根本原因
    if (data.rootCause) {
      const rcaTab = this.page.getByRole('tab', { name: /RCA|Root Cause|根本原因/i });
      if (await rcaTab.isVisible()) {
        await rcaTab.click();
      }

      const rootCauseInput = this.page.locator('textarea[id*="rootCause"], textarea[name*="rootCause"], textarea[placeholder*="根本原因"]').first();
      if (await rootCauseInput.isVisible()) {
        await rootCauseInput.fill(data.rootCause);
      }
    }

    // 填写解决方案
    if (data.resolution) {
      const resolutionInput = this.page.locator('textarea[id*="resolution"], textarea[name*="resolution"], textarea[placeholder*="解决方案"]').first();
      if (await resolutionInput.isVisible()) {
        await resolutionInput.fill(data.resolution);
      }
    }

    // 提交表单
    await this.page.getByRole('button', { name: /提交|Submit|创建|Create/i }).first().click();

    // 等待跳转
    await this.page.waitForURL(/\/problems\/\d+|\/problems/, { timeout: 15000 }).catch(() => {});

    const url = this.page.url();
    const match = url.match(/\/problems\/(\d+)/);
    return match ? parseInt(match[1]) : 0;
  }

  async linkIncident(problemId: number, incidentId: number): Promise<void> {
    await this.goto();
    await this.openProblem(problemId);

    const linkButton = this.page.getByRole('button', { name: /关联事件|Link Incident|关联/i });
    if (await linkButton.isVisible()) {
      await linkButton.click();
      await this.page.waitForSelector('.ant-modal, .ant-drawer', { timeout: 5000 }).catch(() => {});

      const incidentInput = this.page.getByPlaceholder(/输入事件ID|Enter Incident ID|事件ID/i);
      if (await incidentInput.isVisible()) {
        await incidentInput.fill(String(incidentId));
      }

      const addButton = this.page.getByRole('button', { name: /添加|Add|确定/i });
      if (await addButton.isVisible()) {
        await addButton.click();
      }
    }
  }

  async openProblem(id: number): Promise<void> {
    const row = this.problemTable.locator(`tbody tr:has-text("${id}")`).first();
    const link = row.locator('a').first();
    if (await link.isVisible()) {
      await link.click();
    } else {
      await row.click();
    }
    await this.page.waitForURL(new RegExp(`/problems/${id}|/problems/\\d+`), { timeout: 10000 }).catch(() => {});
  }
}
