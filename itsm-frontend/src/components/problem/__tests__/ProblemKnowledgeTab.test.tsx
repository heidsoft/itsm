/**
 * Tests for ProblemKnowledgeTab（2026-10-22 知识沉淀闭环）
 *
 * 覆盖：
 *   - 列表渲染：draft 必须标注「草稿」（沉淀产物恒为草稿，不能渲染成已发布）；
 *   - 空态：Empty 明示「尚未沉淀」，不用 null 蒙混；
 *   - 沉淀成功：message.success + 列表刷新（含新沉淀的草稿）；
 *   - 沉淀 422：质量门理由原样呈现（问题无根因且无解决方案）；
 *   - 反查失败：Alert 明示读取失败，不把「读不到」渲染成「没有沉淀」。
 */

import React from 'react';
import { render, screen, waitFor, fireEvent, within } from '@testing-library/react';
import { ProblemKnowledgeTab } from '../ProblemKnowledgeTab';
import { ProblemApi } from '@/lib/api/problem-api';
import { ApiRequestError } from '@/lib/api/http-client';

const mockPush = jest.fn();
jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: mockPush }),
}));

const mockMessageSuccess = jest.fn();
const mockMessageError = jest.fn();
jest.mock('antd', () => {
  const actual = jest.requireActual('antd');
  return {
    ...actual,
    message: {
      success: (...args: unknown[]) => mockMessageSuccess(...args),
      error: (...args: unknown[]) => mockMessageError(...args),
    },
  };
});

jest.mock('@/lib/api/problem-api', () => ({
  ProblemApi: {
    listProblemKnowledge: jest.fn(),
    promoteProblemKnowledge: jest.fn(),
  },
}));

const mockList = ProblemApi.listProblemKnowledge as jest.Mock;
const mockPromote = ProblemApi.promoteProblemKnowledge as jest.Mock;

const draftArticle = {
  id: 7,
  title: '连接池耗尽导致服务雪崩',
  content: '## 根因分析\n\n连接池未设上限',
  category: '故障排查',
  status: 'draft',
  tags: ['问题沉淀', 'PRB-2026-0042'],
  sourceProblemId: 1,
  createdAt: '2026-10-22T09:00:00Z',
  updatedAt: '2026-10-22T09:00:00Z',
};

describe('ProblemKnowledgeTab', () => {
  beforeEach(() => {
    // 只重置本文件的两个 API mock：clearAllMocks 不清 mockResolvedValueOnce 队列
    // （失败用例的残留队列值会污染后续用例）；而 resetAllMocks 会连带重置
    // jest.setup.js 的全局 matchMedia mock，导致 antd responsiveObserver 崩溃。
    jest.clearAllMocks();
    mockList.mockReset();
    mockPromote.mockReset();
  });

  it('renders promoted articles with draft status and source echo', async () => {
    mockList.mockResolvedValueOnce({ items: [draftArticle], total: 1 });

    render(<ProblemKnowledgeTab problemId={1} />);

    await waitFor(() => {
      expect(screen.getByText('连接池耗尽导致服务雪崩')).toBeInTheDocument();
    });
    // 沉淀产物恒为草稿：必须标注「草稿」，绝不能渲染成已发布
    expect(screen.getByText('草稿')).toBeInTheDocument();
    expect(screen.queryByText('已发布')).not.toBeInTheDocument();
    expect(mockList).toHaveBeenCalledWith(1);
  });

  it('shows empty state when nothing was promoted', async () => {
    mockList.mockResolvedValueOnce({ items: [], total: 0 });

    render(<ProblemKnowledgeTab problemId={1} />);

    await waitFor(() => {
      expect(screen.getByText('尚未沉淀知识')).toBeInTheDocument();
    });
  });

  it('promotes with empty overrides, announces draft, and refreshes the list', async () => {
    mockList
      .mockResolvedValueOnce({ items: [], total: 0 })
      .mockResolvedValueOnce({ items: [draftArticle], total: 1 });
    mockPromote.mockResolvedValueOnce(draftArticle);

    render(<ProblemKnowledgeTab problemId={1} />);
    await waitFor(() => {
      expect(screen.getByText('尚未沉淀知识')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('promote-knowledge-button'));
    // Modal 有开门动画，等 dialog 出现再操作；antd 给两字按钮文本插空格，
    // 先圈定 dialog 再命中 OK，避免与触发按钮「沉淀为知识」的前缀撞车（同 sla-definitions 先例）
    const modal = await screen.findByRole('dialog');
    fireEvent.click(within(modal).getByRole('button', { name: /沉\s*淀/ }));

    await waitFor(() => {
      expect(mockPromote).toHaveBeenCalledWith(1, { tags: [] });
    });
    await waitFor(() => {
      expect(mockMessageSuccess).toHaveBeenCalledWith(
        expect.stringContaining('已沉淀为知识草稿')
      );
    });
    // 沉淀成功后刷新列表（第二次调用）
    expect(mockList).toHaveBeenCalledTimes(2);
    await waitFor(() => {
      expect(screen.getByText('连接池耗尽导致服务雪崩')).toBeInTheDocument();
    });
  });

  it('surfaces the 422 quality gate reason verbatim', async () => {
    mockList.mockResolvedValueOnce({ items: [], total: 0 });
    mockPromote.mockRejectedValueOnce(
      new ApiRequestError('问题尚无根因或解决方案，沉淀为知识没有可复用价值', 422, 4220)
    );

    render(<ProblemKnowledgeTab problemId={1} />);
    await waitFor(() => {
      expect(screen.getByText('尚未沉淀知识')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('promote-knowledge-button'));
    const modal = await screen.findByRole('dialog');
    fireEvent.click(within(modal).getByRole('button', { name: /沉\s*淀/ }));

    await waitFor(() => {
      expect(mockMessageError).toHaveBeenCalledWith(expect.stringContaining('根因或解决方案'));
    });
    expect(mockPromote).toHaveBeenCalledTimes(1);
  });

  it('alerts on reverse-lookup failure instead of rendering an empty list', async () => {
    mockList.mockRejectedValueOnce(new ApiRequestError('查询沉淀知识失败', 500));

    render(<ProblemKnowledgeTab problemId={1} />);

    await waitFor(() => {
      expect(screen.getByText('沉淀知识读取失败')).toBeInTheDocument();
    });
    expect(screen.queryByText('尚未沉淀知识')).not.toBeInTheDocument();
  });
});
