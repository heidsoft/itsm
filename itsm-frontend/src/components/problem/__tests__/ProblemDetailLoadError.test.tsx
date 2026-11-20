/**
 * 回归锁：问题详情加载失败（5xx/断网）不能伪装成「未找到」，且必须有重试入口。
 * 同时锁住首帧：loading 初值必须为 true，否则 data=null 的首帧先闪现 notFound。
 */
import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import ProblemDetail from '../ProblemDetail';

const mockGetProblem = jest.fn();
let mockShouldFail = false;

jest.mock('next/navigation', () => ({
  useParams: () => ({ id: '9' }),
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('@/lib/api/', () => ({
  ProblemApi: {
    getProblem: (...args: unknown[]) => {
      if (mockShouldFail) return Promise.reject(new Error('500'));
      return mockGetProblem(...args);
    },
    updateProblem: jest.fn(),
  },
  KEDBApi: {},
}));

jest.mock('@/lib/api/kedb-api', () => ({ KEDBApi: {} }));
jest.mock('@/components/problem/ProblemInvestigationTab', () => ({ __esModule: true, default: () => null }));
jest.mock('@/components/problem/BasicInfoCard', () => ({ __esModule: true, default: () => null }));

const okProblem = {
  id: 9, problemNumber: 'PRB-9', title: '根因待查', status: 'open', priority: 'high',
  createdAt: '2026-09-11T00:00:00Z', updatedAt: '2026-09-11T00:00:00Z',
};

describe('ProblemDetail 加载错误态', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockShouldFail = false;
    mockGetProblem.mockResolvedValue(okProblem);
  });

  it('首帧不闪现「未找到」：loading 初值为 true', () => {
    // getProblem 永不 resolve → 组件应停在骨架屏，而不是 notFound
    mockGetProblem.mockReturnValue(new Promise(() => {}));
    render(<ProblemDetail id="9" />);
    expect(screen.queryByText('未找到问题')).toBeNull();
  });

  it('加载失败显示「加载失败」并提供「重试」，重试成功则渲染详情', async () => {
    mockShouldFail = true;
    render(<ProblemDetail id="9" />);

    await waitFor(() => expect(screen.getByText('加载失败')).toBeInTheDocument());
    expect(screen.queryByText('未找到问题')).toBeNull();

    // 切成功后再点重试
    mockShouldFail = false;
    // Antd 对两个汉字的按钮自动加空格（「重 试」），用正则匹配
    const retry = screen.getByRole('button', { name: /重\s*试/ });
    await waitFor(() => expect(retry).toBeEnabled());
    retry.click();

    // 重试后离开错误态：错误文案消失，详情骨架/内容出现（BasicInfoCard 已 mock）
    await waitFor(() => expect(screen.queryByText('加载失败')).toBeNull());
    expect(screen.queryByText('未找到问题')).toBeNull();
  });
});
