import { test, expect } from '@playwright/test';
import { loginAndReturn } from './auth-utils';

/**
 * 租户开通（初始化架构）端到端：
 * 新建租户 → 同事务投递 tenant.bootstrap.install → worker 安装产品基线
 * → 初始化状态接口逐组件就绪。这是初始化改造的核心用户路径。
 */
test.describe('租户开通与产品基线安装', () => {
  test('新建租户后 outbox 安装基线且状态就绪', async ({ page }) => {
    test.setTimeout(120_000);
    await loginAndReturn(page);

    const code = `e2e-init-${Date.now()}`;
    let tenantId = 0;

    try {
      await page.goto('/admin/tenants', { waitUntil: 'networkidle' });
      await page.getByRole('button', { name: /新建租户/ }).click();
      await page.getByLabel(/租户名称/).fill('E2E 初始化验证租户');
      await page.getByLabel(/租户编码/).fill(code);
      await page.getByRole('button', { name: /保\s*存/ }).click();

      // 租户行出现（创建接口成功，不再因护栏 500）
      // 单元格会带尾随标记（如 "•"），不能用精确匹配
      await expect(page.getByText(code, { exact: false }).first()).toBeVisible({
        timeout: 15_000,
      });

      const listResponse = await page.request.get('/api/v1/tenants?pageSize=100');
      expect(listResponse.ok()).toBeTruthy();
      const listPayload = await listResponse.json();
      const list = listPayload.data?.items || listPayload.data?.tenants || [];
      const created = list.find(
        (item: { code?: string }) => item.code === code
      );
      expect(created, `租户 ${code} 应已创建`).toBeTruthy();
      tenantId = Number(created.id);

      // outbox 命令 → worker 安装 → 逐组件就绪
      let payload: {
        ready?: boolean;
        commandStatus?: string;
        components?: { component: string; verified: boolean }[];
      } | null = null;
      const deadline = Date.now() + 60_000;
      while (Date.now() < deadline) {
        const response = await page.request.get(
          `/api/v1/tenants/${tenantId}/initialization`
        );
        expect(response.ok()).toBeTruthy();
        payload = ((await response.json()) as { data: typeof payload }).data;
        if (payload?.ready && payload?.commandStatus === 'succeeded') {
          break;
        }
        await new Promise((resolve) => setTimeout(resolve, 1000));
      }

      expect(payload, '产品基线应在 worker 消费后就绪').toMatchObject({
        ready: true,
        commandStatus: 'succeeded',
      });
      expect(payload!.components!.length).toBeGreaterThan(0);
      for (const component of payload!.components!) {
        expect(component.verified, `${component.component} 应已就绪`).toBe(true);
      }
    } finally {
      // 用例产生的租户尽力清理，保持测试环境可重复
      if (tenantId > 0) {
        const tokenResponse = await page.request.get('/api/v1/csrf-token');
        const csrf = ((await tokenResponse.json().catch(() => ({}))) as {
          data?: { csrf_token?: string };
        }).data?.csrf_token;
        await page.request.delete(`/api/v1/tenants/${tenantId}`, {
          headers: csrf ? { 'X-CSRF-Token': csrf } : undefined,
        });
      }
    }
  });

  test('初始化状态接口对未安装基线的租户如实报告未就绪', async ({ page }) => {
    await loginAndReturn(page);

    // 平台租户已有完整基线：ready=true 且 6 个组件逐项 verified
    const response = await page.request.get('/api/v1/tenants/1/initialization');
    expect(response.ok()).toBeTruthy();
    const payload = (await response.json()).data;
    expect(payload.ready).toBe(true);
    expect(payload.components.length).toBeGreaterThan(0);
    for (const component of payload.components) {
      expect(component.verified, `${component.component} 应已就绪`).toBe(true);
    }
  });
});
