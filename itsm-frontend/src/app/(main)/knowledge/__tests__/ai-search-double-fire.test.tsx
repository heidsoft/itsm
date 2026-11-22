/**
 * 知识库 AI 搜索双发收口（计划 F-P0-2）。
 *
 * Input.Search 同时绑 onSearch 与 onPressEnter：antd 内部 onPressEnter 先调
 * 自定义 onPressEnter 再调 onSearch → 一次回车双发 /api/v1/ai/rag/search。
 * 修复：删掉 onPressEnter 双绑 + in-flight ref 防重入
 * （setAiSearchLoading 异步生效，连续触发间闭包仍持旧 false，拦不住二次进入）。
 */
import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@/lib/test-utils';
import KnowledgePage from '../page';
import { httpClient } from '@/lib/api/http-client';
import { KnowledgeBaseApi } from '@/lib/api/knowledge-base-api';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { post: jest.fn() },
}));

jest.mock('@/lib/api/knowledge-base-api', () => ({
  KnowledgeBaseApi: {
    getStats: jest.fn(),
    getArticles: jest.fn(),
  },
}));

jest.mock('@/lib/i18n/useI18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}));

jest.mock('@/lib/hooks/use-permissions', () => ({
  useOperationPermissions: () => ({ knowledge: { canCreate: () => false } }),
}));

jest.mock('@/components/knowledge/ArticleList', () => ({
  __esModule: true,
  default: () => null,
}));

const mockPost = httpClient.post as jest.Mock;
const mockGetStats = KnowledgeBaseApi.getStats as jest.Mock;
const mockGetArticles = KnowledgeBaseApi.getArticles as jest.Mock;

async function renderPageAndGetInput(): Promise<HTMLElement> {
  render(<KnowledgePage />);
  return await screen.findByPlaceholderText('knowledgeBase.aiSearchPlaceholder');
}

// rc-input 用 keyLockRef 防按住回车重复触发：keydown 置锁、keyup 复位。
// 真实键盘必有 keyup，测试须成对发，否则第一次回车后锁死。
const pressEnter = (input: HTMLElement) => {
  fireEvent.keyDown(input, { key: 'Enter', keyCode: 13 });
  fireEvent.keyUp(input, { key: 'Enter', keyCode: 13 });
};

beforeEach(() => {
  jest.clearAllMocks();
  mockPost.mockReset();
  mockGetStats.mockResolvedValue({
    total: 1,
    published: 1,
    draft: 0,
    views: 0,
    rating: 0,
    categories: [],
  });
  mockGetArticles.mockResolvedValue({ items: [] });
  mockPost.mockResolvedValue({ results: [] });
});

describe('知识库 AI 搜索 — 双发与防重入', () => {
  it('回车触发一次 AI 搜索（不再因 onSearch + onPressEnter 双绑发两次）', async () => {
    const input = await renderPageAndGetInput();

    fireEvent.change(input, { target: { value: '什么是SLA' } });
    pressEnter(input);

    await waitFor(() => expect(mockPost).toHaveBeenCalledTimes(1));
    expect(mockPost).toHaveBeenCalledWith('/api/v1/ai/rag/search', {
      query: '什么是SLA',
      limit: 10,
      type: 'kb',
    });
  });

  it('请求在途时重复触发不再发第二次；完成后可再搜', async () => {
    const input = await renderPageAndGetInput();
    // 自定义 enterButton 是 antd Button：antd 不给它传 loading、onClick 直连 onSearch，
    // 无任何 loading 守卫——这正是 in-flight ref 要收口的通道。
    const searchButton = screen.getByRole('button', { name: 'AI 搜索' });

    let release!: (value: unknown) => void;
    mockPost.mockImplementationOnce(
      () =>
        new Promise(resolve => {
          release = resolve;
        })
    );

    fireEvent.change(input, { target: { value: 'SLA' } });
    fireEvent.click(searchButton);
    await waitFor(() => expect(mockPost).toHaveBeenCalledTimes(1));

    // 在途期间重复点击：被 in-flight ref 拦下
    fireEvent.click(searchButton);
    expect(mockPost).toHaveBeenCalledTimes(1);

    // 放行首个请求，等 finally 复位 ref
    await act(async () => {
      release({ results: [] });
    });

    // 完成后恢复可搜
    fireEvent.click(searchButton);
    await waitFor(() => expect(mockPost).toHaveBeenCalledTimes(2));
  });

  it('空查询不发请求，且回车只提示一次（双绑会提示两次）', async () => {
    const input = await renderPageAndGetInput();

    pressEnter(input);
    expect(mockPost).not.toHaveBeenCalled();
    // 空查询在置 in-flight ref 之前就 return，ref 拦不住双绑的第二发——
    // 提示次数是能区分「单绑 vs onSearch+onPressEnter 双绑」的判别式。
    const notices = document.querySelectorAll('.ant-message-notice');
    expect(notices).toHaveLength(1);
  });
});
