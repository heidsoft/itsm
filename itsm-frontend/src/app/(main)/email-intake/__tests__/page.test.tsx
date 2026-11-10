import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import '@testing-library/jest-dom';

/**
 * 邮件受理（/email-intake）主数据中文化回归（D15）：
 * 客户状态与合同状态必须显示中文（启用/已停用、生效中/已终止…），
 * 客户表列头必须是“别名”，不得出现英文 Alias 与原始状态码。
 */

jest.mock('@/lib/services/emailIntakeService', () => ({
  emailIntakeService: {
    conversations: jest.fn().mockResolvedValue({ items: [], total: 0, page: 1, pageSize: 20 }),
    customers: jest.fn().mockResolvedValue({
      items: [
        { id: 1, name: '示例客户', shortName: '示例', aliases: ['EXAMPLE'], status: 'active' },
        { id: 2, name: '停用客户', shortName: '停用', aliases: [], status: 'inactive' },
      ],
      total: 2,
    }),
    contracts: jest.fn().mockResolvedValue({
      items: [
        {
          id: 1,
          customerId: 1,
          contractNumber: 'SC-001',
          status: 'active',
          startAt: '2026-01-01T00:00:00Z',
          endAt: '2026-12-31T00:00:00Z',
        },
        {
          id: 2,
          customerId: 1,
          contractNumber: 'SC-002',
          status: 'terminated',
          startAt: '2025-01-01T00:00:00Z',
          endAt: '2025-12-31T00:00:00Z',
        },
      ],
      total: 2,
    }),
    sourceOrganizations: jest.fn().mockResolvedValue({ items: [], total: 0 }),
    externalContractReferences: jest.fn().mockResolvedValue({ items: [], total: 0 }),
    schedules: jest.fn().mockResolvedValue({ items: [], total: 0 }),
  },
}));

jest.mock('@/lib/api/user-api', () => ({
  UserApi: { getUsers: jest.fn().mockResolvedValue({ items: [], total: 0 }) },
}));

import EmailIntakePage from '../page';

describe('邮件受理主数据中文化（D15）', () => {
  it('客户与合同状态显示中文，别名列头为中文，不出现原始状态码', async () => {
    render(
      <App>
        <EmailIntakePage />
      </App>
    );

    await waitFor(() => expect(screen.getByText("Customers and Contracts")).toBeInTheDocument());

    // 切换到主数据页签（Ant Design Tabs 默认只渲染激活面板）
    fireEvent.click(screen.getByRole('tab', { name: "Customer and Contract Master Data" }));

    expect(await screen.findByText("Alias")).toBeInTheDocument();
    expect(screen.getAllByText("Enabled").length).toBeGreaterThan(0);
    expect(screen.getByText("Disabled")).toBeInTheDocument();
    expect(screen.getByText("In effect")).toBeInTheDocument();
    expect(screen.getByText("Terminated")).toBeInTheDocument();

    const body = document.body.textContent || '';
    expect(body).toContain('Alias');
    expect(body).not.toMatch(/\bactive\b/);
    expect(body).not.toMatch(/\b(inactive|terminated|expired|pending)\b/);
  });
});
