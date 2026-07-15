// itsm-frontend/tests/e2e/utils/page-objects/ChangePage.ts
import type { Page, Locator } from '@playwright/test';
import { BasePage } from './BasePage';

export class ChangePage extends BasePage {
  readonly url = '/changes';
  readonly createButton: Locator;
  readonly changeTable: Locator;

  constructor(page: Page) {
    super(page);
    this.createButton = page.getByRole('button', { name: /新建变更|Create Change|新建/i });
    this.changeTable = this.page.locator('table');
  }

  async goto(): Promise<void> {
    await this.page.goto(`${this.baseUrl}${this.url}`, { waitUntil: 'domcontentloaded' });
    await this.changeTable.waitFor({ state: 'visible' }).catch(() => {});
  }

  async createChange(data: {
    title: string;
    description: string;
    riskLevel?: string;
    rollbackPlan?: string;
  }): Promise<number> {
    // 直接导航到创建页面
    await this.page.goto(`${this.baseUrl}/changes/new`, { waitUntil: 'domcontentloaded' });

    // 填写标题
    const titleInput = this.page.locator('input[id*="title"], input[name*="title"], input[placeholder*="标题"]').first();
    await titleInput.fill(data.title);

    // 填写描述
    const descInput = this.page.locator('textarea[id*="description"], textarea[name*="description"], textarea[placeholder*="描述"]').first();
    await descInput.fill(data.description);

    // 选择风险级别
    if (data.riskLevel) {
      const riskSelect = this.page.locator('select[id*="risk"], select[name*="risk"]').first();
      if (await riskSelect.isVisible()) {
        await riskSelect.selectOption({ label: data.riskLevel });
      }
    }

    // 填写回滚计划
    if (data.rollbackPlan) {
      const rollbackInput = this.page.locator('textarea[id*="rollback"], textarea[name*="rollback"], textarea[placeholder*="回滚"]').first();
      if (await rollbackInput.isVisible()) {
        await rollbackInput.fill(data.rollbackPlan);
      }
    }

    // 提交表单
    await this.page.getByRole('button', { name: /提交|Submit|创建|Create/i }).first().click();

    // 等待跳转
    await this.page.waitForURL(/\/changes\/\d+|\/changes/, { timeout: 15000 }).catch(() => {});

    const url = this.page.url();
    const match = url.match(/\/changes\/(\d+)/);
    return match ? parseInt(match[1]) : 0;
  }

  async approveChange(changeId: number): Promise<void> {
    await this.goto();
    await this.openChange(changeId);

    const approveButton = this.page.getByRole('button', { name: /审批|Approve|批准/i });
    if (await approveButton.isVisible()) {
      await approveButton.click();
      await this.page.waitForSelector('.ant-modal', { timeout: 5000 }).catch(() => {});

      const commentInput = this.page.locator('textarea[id*="comment"], textarea[placeholder*="意见"]').first();
      if (await commentInput.isVisible()) {
        await commentInput.fill('Approved - looks good');
      }

      const confirmButton = this.page.getByRole('button', { name: /通过|Approve|Confirm|确认/i });
      if (await confirmButton.isVisible()) {
        await confirmButton.click();
      }
    }
  }

  async rejectChange(changeId: number, reason: string): Promise<void> {
    await this.goto();
    await this.openChange(changeId);

    const rejectButton = this.page.getByRole('button', { name: /拒绝|Reject/i });
    if (await rejectButton.isVisible()) {
      await rejectButton.click();
      await this.page.waitForSelector('.ant-modal', { timeout: 5000 }).catch(() => {});

      const reasonInput = this.page.locator('textarea[id*="reason"], textarea[placeholder*="原因"]').first();
      if (await reasonInput.isVisible()) {
        await reasonInput.fill(reason);
      }

      const confirmButton = this.page.getByRole('button', { name: /确认拒绝|Confirm Reject|确认/i });
      if (await confirmButton.isVisible()) {
        await confirmButton.click();
      }
    }
  }

  async openChange(id: number): Promise<void> {
    const row = this.changeTable.locator(`tbody tr:has-text("${id}")`).first();
    const link = row.locator('a').first();
    if (await link.isVisible()) {
      await link.click();
    } else {
      await row.click();
    }
    await this.page.waitForURL(new RegExp(`/changes/${id}|/changes/\\d+`), { timeout: 10000 }).catch(() => {});
  }
}
