// itsm-frontend/tests/e2e/utils/page-objects/IncidentPage.ts
import type { Page, Locator } from '@playwright/test';
import { BasePage } from './BasePage';

export class IncidentPage extends BasePage {
  readonly url = '/incidents';
  readonly createButton: Locator;
  readonly incidentTable: Locator;
  readonly priorityFilter: Locator;

  constructor(page: Page) {
    super(page);
    this.createButton = page.getByRole('button', { name: /新建事件|Create Incident|新建/i });
    this.incidentTable = page.locator('table');
    this.priorityFilter = page.getByPlaceholder(/优先级|Priority/i);
  }

  async goto(): Promise<void> {
    await this.page.goto(`${this.baseUrl}${this.url}`);
    await this.incidentTable.waitFor({ state: 'visible' }).catch(() => {});
  }

  async createIncident(data: {
    title: string;
    description: string;
    priority?: string;
    impact?: string;
  }): Promise<number> {
    // 直接导航
    await this.page.goto(`${this.baseUrl}/incidents/create`);
    await this.page.waitForSelector('form', { timeout: 10000 }).catch(() => {});

    // 填写标题
    const titleInput = this.page.locator('[data-testid="incident-title-input"], input[placeholder*="事件"]').first();
    await titleInput.fill(data.title);

    // 填写描述
    const descInput = this.page.locator('[data-testid="incident-description-input"], textarea[placeholder*="详细"]').first();
    await descInput.fill(data.description);

    // 提交表单
    const submitBtn = this.page.locator('[data-testid="incident-submit-button"], button[type="submit"]').first();
    await submitBtn.click();

    // 等待跳转
    await this.page.waitForURL(/\/incidents/, { timeout: 15000 }).catch(() => {});

    const url = this.page.url();
    const match = url.match(/\/incidents\/(\d+)/);
    return match ? parseInt(match[1]) : 0;
  }

  async escalateToProblem(incidentId: number): Promise<void> {
    await this.goto();
    await this.openIncident(incidentId);

    const escalateButton = this.page.getByRole('button', { name: /升级为问题|Escalate|升级/i });
    if (await escalateButton.isVisible()) {
      await escalateButton.click();
      await this.page.waitForSelector('.ant-modal', { timeout: 5000 }).catch(() => {});
      const confirmButton = this.page.getByRole('button', { name: /确认|Confirm|确定/i });
      if (await confirmButton.isVisible()) {
        await confirmButton.click();
      }
    }
  }

  async openIncident(id: number): Promise<void> {
    const row = this.incidentTable.locator(`tbody tr:has-text("${id}")`).first();
    const link = row.locator('a').first();
    if (await link.isVisible()) {
      await link.click();
    } else {
      await row.click();
    }
    await this.page.waitForURL(new RegExp(`/incidents/${id}|/incidents/\\d+`), { timeout: 10000 }).catch(() => {});
  }
}
