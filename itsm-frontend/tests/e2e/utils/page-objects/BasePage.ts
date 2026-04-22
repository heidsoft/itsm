// itsm-frontend/tests/e2e/utils/page-objects/BasePage.ts
import { Page, Locator } from '@playwright/test';

export abstract class BasePage {
  protected page: Page;
  protected baseUrl = process.env.PLAYWRIGHT_BASE_URL || 'http://localhost:3000';

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
