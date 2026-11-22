import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import '@testing-library/jest-dom';



const mockListTasks = jest.fn();
const mockGet = jest.fn();

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn(), replace: jest.fn() }),
  usePathname: () => '/approvals',
  useSearchParams: () => new URLSearchParams(),
}));

jest.mock('@/lib/api/workflow-api', () => ({
  WorkflowApi: {
    listMyApprovalTasks: (...args: unknown[]) => mockListTasks(...args),
  },
}));

jest.mock('@/lib/api/http-client', () => ({
  httpClient: {
    get: (...args: unknown[]) => mockGet(...args),
  },
}));

import ApprovalsPage from '../page';

const renderPage = () =>
  render(
    <App>
      <ApprovalsPage />
    </App>
  );

const switchToLegacyTab = async () => {
  const tab = await screen.findByRole('tab', { name: /业务待办/ });
  fireEvent.click(tab);
};

it.each(['/api/v1/tickets', '/api/v1/changes', '/api/v1/service-requests'])('任一路 %s 失败不伪装空待办且可重试', async failedUrl => {
  mockListTasks.mockResolvedValue({ items: [] });
  mockGet.mockImplementation((url: string) => url === failedUrl ? Promise.reject(new Error('offline')) : Promise.resolve({ items: [] }));
  renderPage();
  await switchToLegacyTab();
  const retry = await screen.findByRole('button', { name: /重\s*试/ });
  expect(screen.queryByRole('table')).not.toBeInTheDocument();
  mockGet.mockResolvedValue({ items: [] });
  fireEvent.click(retry);
  await waitFor(() => expect(screen.queryByRole('button', { name: /重\s*试/ })).not.toBeInTheDocument());
});
