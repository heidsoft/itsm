/**
 * 知识文章详情的写操作入口按 knowledge:write 收口。
 *
 * 后端 create/update/publish/unpublish 全部挂在 knowledge:write 上，而最终用户
 * 只有 knowledge:read；2026-10-18 隔离栈实测草稿文章详情页照常给出
 * 「发布文章 / 编辑文章」，点击只得到 403。
 */
import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import ArticleDetail from '../ArticleDetail';

const mockGetArticle = jest.fn();
const mockPermissions: string[] = [];

jest.mock('next/navigation', () => ({
  useParams: () => ({ id: '42' }),
  useRouter: () => ({ push: jest.fn(), replace: jest.fn(), back: jest.fn() }),
}));

// markdown 渲染链路是 ESM，测试只关心按钮是否存在。
jest.mock('react-markdown', () => ({ __esModule: true, default: () => null }));
jest.mock('remark-gfm', () => ({ __esModule: true, default: () => null }));
jest.mock('rehype-sanitize', () => ({ __esModule: true, default: () => null }));

jest.mock('../ArticleVersionControl', () => ({
  __esModule: true,
  default: () => null,
}));

jest.mock('@/lib/api/knowledge-base-api', () => ({
  KnowledgeBaseApi: {
    getArticle: (...args: unknown[]) => mockGetArticle(...args),
    publishArticle: jest.fn(),
    unpublishArticle: jest.fn(),
  },
  __esModule: true,
}));

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStore: () => ({
    user: { id: 9, username: 'reader', name: '阅读者', permissions: mockPermissions },
  }),
}));

const draftArticle = {
  id: 42,
  title: '如何重置密码',
  content: '步骤一',
  status: 'draft',
  author: '',
  views: 0,
  createdAt: '2026-10-18T00:00:00Z',
  updatedAt: '2026-10-18T00:00:00Z',
};

async function renderDetail() {
  render(<ArticleDetail />);
  await waitFor(() => expect(mockGetArticle).toHaveBeenCalledWith('42'));
}

describe('ArticleDetail 写操作入口 — 按权限码渲染', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockPermissions.length = 0;
    mockGetArticle.mockResolvedValue(draftArticle);
  });

  it('只有 knowledge:read 时不给出发布与编辑入口', async () => {
    mockPermissions.push('knowledge:read');
    await renderDetail();

    expect(await screen.findByText('如何重置密码')).toBeInTheDocument();
    expect(screen.queryByText('发布文章')).toBeNull();
    expect(screen.queryByText('编辑文章')).toBeNull();
  });

  it('持 knowledge:write 时草稿文章给出发布与编辑', async () => {
    mockPermissions.push('knowledge:read', 'knowledge:write');
    await renderDetail();

    expect(await screen.findByText('发布文章')).toBeInTheDocument();
    expect(screen.getByText('编辑文章')).toBeInTheDocument();
  });

  it('持 knowledge:admin 等同写权限（与后端 resource:admin 判定一致）', async () => {
    mockPermissions.push('knowledge:read', 'knowledge:admin');
    await renderDetail();

    expect(await screen.findByText('编辑文章')).toBeInTheDocument();
  });
});
