// itsm-frontend/tests/e2e/utils/page-objects/ServiceCatalogPage.ts
import type { Page, Locator } from '@playwright/test';
import { BasePage } from './BasePage';

export class ServiceCatalogPage extends BasePage {
  readonly url = '/service-catalog';
  readonly serviceList: Locator;
  readonly requestButton: Locator;

  constructor(page: Page) {
    super(page);
    this.serviceList = this.page.locator('.service-card, [data-testid="service-card"], .catalog-item');
    this.requestButton = this.page.getByRole('button', { name: /申请|Request|请求/i });
  }

  async goto(): Promise<void> {
    await this.page.goto(`${this.baseUrl}${this.url}`);
    await this.serviceList.first().waitFor({ state: 'visible', timeout: 10000 }).catch(() => {
      // 页面可能没有服务卡片数据
    });
  }

  async requestService(serviceName: string, justification: string): Promise<number> {
    // 点击服务卡片
    await this.page.getByText(serviceName).first().click();
    await this.page.waitForSelector('.ant-drawer, .ant-modal', { timeout: 5000 }).catch(() => {});

    // 点击申请按钮
    const requestBtn = this.requestButton.first();
    if (await requestBtn.isVisible()) {
      await requestBtn.click();
    }

    // 填写申请表单
    const justificationInput = this.page.locator('textarea[id*="justification"], textarea[placeholder*="申请理由"], textarea[placeholder*="理由"]').first();
    if (await justificationInput.isVisible()) {
      await justificationInput.fill(justification);
    }

    // 提交
    await this.page.getByRole('button', { name: /提交|Submit/i }).first().click();

    // 等待跳转到工单页面
    await this.page.waitForURL(/\/tickets\/\d+|\/service-requests\/\d+/, { timeout: 15000 }).catch(() => {});

    const url = this.page.url();
    const match = url.match(/\/(tickets|service-requests)\/(\d+)/);
    return match ? parseInt(match[2]) : 0;
  }

  async browseCategories(): Promise<string[]> {
    const categories = this.page.locator('.category-tab, [data-testid="category-tab"], .ant-tabs-tab');
    const count = await categories.count();
    const names: string[] = [];

    for (let i = 0; i < count; i++) {
      const name = await categories.nth(i).textContent();
      if (name) names.push(name.trim());
    }

    return names;
  }
}
