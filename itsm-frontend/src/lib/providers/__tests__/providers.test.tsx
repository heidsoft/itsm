/**
 * Tests for src/lib/providers
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

jest.mock('@ant-design/nextjs-registry', () => ({
  AntdRegistry: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock('antd', () => ({
  ConfigProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  App: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock('antd/locale/zh_CN', () => ({}));

jest.mock('@/lib/design-system/theme', () => ({
  useTheme: () => ({ isDark: false }),
  getAntdTheme: () => ({}),
}));

jest.mock('@tanstack/react-query', () => ({
  QueryClient: jest.fn(() => ({})),
  QueryClientProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

jest.mock('@tanstack/react-query-devtools', () => ({
  ReactQueryDevtools: () => null,
}));

import React from 'react';
import { render } from '@testing-library/react';
import { AntdProvider } from '../AntdProvider';
import { QueryProvider } from '../QueryProvider';

describe('AntdProvider', () => {
  it('renders children', () => {
    const { getByText } = render(
      <AntdProvider>
        <div>Hello</div>
      </AntdProvider>
    );
    expect(getByText('Hello')).toBeDefined();
  });
});

describe('QueryProvider', () => {
  it('renders children', () => {
    const { getByText } = render(
      <QueryProvider>
        <div>World</div>
      </QueryProvider>
    );
    expect(getByText('World')).toBeDefined();
  });
});
