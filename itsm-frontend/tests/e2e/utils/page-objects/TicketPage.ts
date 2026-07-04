// itsm-frontend/tests/e2e/utils/page-objects/TicketPage.ts
import type { Page, Locator } from '@playwright/test';
import { BasePage } from './BasePage';

export class TicketPage extends BasePage {
  readonly url = '/tickets';
  readonly createButton: Locator;
  readonly searchInput: Locator;
  readonly ticketTable: Locator;

  constructor(page: Page) {
    super(page);
    this.createButton = page.getByRole('button', { name: /新建工单|Create Ticket|新建/i });
    this.searchInput = page.getByPlaceholder(/搜索工单ID|搜索|Search/i);
    this.ticketTable = page.locator('table');
  }

  async goto(): Promise<void> {
    await this.page.goto(`${this.baseUrl}${this.url}`);
    await this.ticketTable.waitFor({ state: 'visible' }).catch(() => {});
  }

  async createTicket(data: {
    title: string;
    description: string;
    priority?: string;
    category?: string;
  }): Promise<number> {
    // 直接导航，不等待networkidle
    await this.page.goto(`${this.baseUrl}/tickets/create`);
    await this.page.waitForSelector('form, [data-testid="ticket-form"]', { timeout: 10000 }).catch(() => {});

    // 填写标题
    const titleInput = this.page.locator('[data-testid="ticket-title-input"], input[placeholder*="VPN"], #title').first();
    await titleInput.fill(data.title);

    // 填写描述
    const descInput = this.page.locator('[data-testid="ticket-description-input"], textarea[placeholder*="详细"]').first();
    await descInput.fill(data.description);

    // 提交表单
    const submitBtn = this.page.locator('[data-testid="ticket-submit-button"], button:has-text("创建工单")').first();
    await submitBtn.click();

    // 等待跳转
    await this.page.waitForURL(/\/tickets\/\d+/, { timeout: 15000 }).catch(() => {});

    const url = this.page.url();
    const match = url.match(/\/tickets\/(\d+)/);
    return match ? parseInt(match[1]) : 0;
  }

  async searchTicket(keyword: string): Promise<void> {
    await this.searchInput.clear();
    await this.searchInput.fill(keyword);
    await this.searchInput.press('Enter');
    await this.page.waitForResponse(/\/api\/v1\/tickets/, { timeout: 5000 }).catch(() => {});
  }

  async getFirstTicketId(): Promise<number | null> {
    const firstRow = this.ticketTable.locator('tbody tr').first();
    const idCell = firstRow.locator('td').first();
    const text = await idCell.textContent();
    return text ? parseInt(text.trim()) : null;
  }

  async openTicket(id: number): Promise<void> {
    // 点击包含工单ID的行
    const row = this.ticketTable.locator(`tbody tr:has-text("${id}")`).first();
    const link = row.locator('a').first();
    if (await link.isVisible()) {
      await link.click();
    } else {
      await row.click();
    }
    await this.page.waitForURL(new RegExp(`/tickets/${id}|/tickets/\\d+`), { timeout: 10000 }).catch(() => {});
  }

  async updateTicketStatus(status: string): Promise<void> {
    const statusSelect = this.page.locator('select[id*="status"], select[name*="status"]').first();
    if (await statusSelect.isVisible()) {
      await statusSelect.selectOption({ label: status });
    }
  }

  async assignTicket(assignee: string): Promise<void> {
    const assignButton = this.page.getByRole('button', { name: /分配|Assign/i });
    if (await assignButton.isVisible()) {
      await assignButton.click();
      await this.page.waitForSelector('.ant-select, .ant-modal', { timeout: 5000 }).catch(() => {});

      const assigneeSelect = this.page.locator('.ant-select-selector').first();
      if (await assigneeSelect.isVisible()) {
        await assigneeSelect.click();
        await this.page.getByTitle(assignee).click().catch(() =>
          this.page.getByText(assignee).click()
        );
      }

      const confirmButton = this.page.getByRole('button', { name: /确定|OK|Assign|确认/i });
      if (await confirmButton.isVisible()) {
        await confirmButton.click();
      }
    }
  }
}
