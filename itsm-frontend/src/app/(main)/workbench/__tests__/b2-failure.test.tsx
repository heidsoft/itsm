import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import '@testing-library/jest-dom';



jest.mock('@/lib/user-preferences', () => ({
  userPreferences: {
    get: () => ({ language: 'zh-CN' }),
    subscribe: () => () => {},
  },
}));

const mockQueryWorkbench = jest.fn();

jest.mock('@/lib/api/workbench-api', () => ({
  queryWorkbench: (...args: unknown[]) => mockQueryWorkbench(...args),
}));

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStore: (selector: (s: { user: { id: number } }) => unknown) =>
    selector({ user: { id: 1 } }),
}));

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn() }),
}));

import WorkbenchPage from '../page';

it('错误态重试重新请求列表', async () => {
  mockQueryWorkbench.mockRejectedValueOnce(new Error('offline')).mockResolvedValue({ items: [], total: 0 });
  render(<App><WorkbenchPage /></App>);
  fireEvent.click(await screen.findByRole('button', { name: /重\s*试/ }));
  await waitFor(() => expect(mockQueryWorkbench).toHaveBeenCalledTimes(2));
});
