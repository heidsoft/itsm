import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import '@testing-library/jest-dom';

/**
 * 审计日志（/audit-logs）中文化回归（D16）：
 * 操作/资源列必须显示中文，未收录的动作码按冒号前缀回退，
 * 不允许把 LOGIN_SUCCESS、ticket_type:1 这类内部码直接暴露给用户。
 */

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStoreHydration: () => {},
}));

jest.mock('@/lib/api/auditlog-api', () => ({
  listAuditLogs: jest.fn(),
}));

import AuditLogsPage from '../page';
import { listAuditLogs } from '@/lib/api/auditlog-api';

const mockListAuditLogs = listAuditLogs as jest.Mock;

describe('审计日志操作与资源中文化（D16）', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockListAuditLogs.mockResolvedValue({
      logs: [
        {
          id: 1,
          createdAt: '2026-10-01T08:00:00Z',
          tenantId: 1,
          userId: 1,
          userName: 'admin',
          requestId: 'r1',
          ip: '127.0.0.1',
          resource: 'auth',
          action: 'LOGIN_SUCCESS',
          path: '/api/v1/auth/login',
          method: 'POST',
          statusCode: 200,
          requestBody: '{}',
        },
        {
          id: 2,
          createdAt: '2026-10-01T08:01:00Z',
          tenantId: 1,
          userId: 1,
          userName: 'admin',
          requestId: 'r2',
          ip: '127.0.0.1',
          resource: 'ticket_type:1',
          action: 'update',
          path: '/api/v1/tickets/1',
          method: 'PATCH',
          statusCode: 200,
          requestBody: '{}',
        },
      ],
      total: 2,
      page: 1,
      pageSize: 20,
    });
  });

  it('操作列显示中文，资源列按前缀回退，不暴露内部码', async () => {
    render(
      <App>
        <AuditLogsPage />
      </App>
    );

    await waitFor(() => expect(mockListAuditLogs).toHaveBeenCalled());

    expect(await screen.findByText("Login Successful")).toBeInTheDocument();
    expect(screen.getByText("Authentication")).toBeInTheDocument();
    expect(screen.getByText("Ticket Types")).toBeInTheDocument();
    expect(screen.getByText("Edit")).toBeInTheDocument();

    const body = document.body.textContent || '';
    expect(body).not.toContain('LOGIN_SUCCESS');
    expect(body).not.toContain('ticket_type:1');
    // 操作码 `update` 不应原样出现（已映射为“编辑”）
    expect(body).not.toMatch(/\bupdate\b/);
  });
});
