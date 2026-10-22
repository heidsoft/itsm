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

  it('持 knowledge:write 时草稿文章给出提交评审与编辑（不再给必然 409 的发布）', async () => {
    mockPermissions.push('knowledge:read', 'knowledge:write');
    await renderDetail();

    // 2026-10-22 发布门禁：未过评审的草稿只能提交评审，发布由审批通过触发。
    expect(await screen.findByText('提交评审')).toBeInTheDocument();
    expect(screen.queryByText('发布文章')).toBeNull();
    expect(screen.getByText('编辑文章')).toBeInTheDocument();
  });

  it('持 knowledge:admin 等同写权限（与后端 resource:admin 判定一致）', async () => {
    mockPermissions.push('knowledge:read', 'knowledge:admin');
    await renderDetail();

    expect(await screen.findByText('编辑文章')).toBeInTheDocument();
  });
});

/**
 * 发布门禁（2026-10-22）：后端 /publish 对 review_status != approved 返回
 * 409/4090，前端不能继续渲染一个必然失败的发布按钮——按钮形态必须与
 * 评审阶段一致，否则用户点下去只看到错误。
 */
describe('ArticleDetail 发布入口 — 按评审阶段渲染', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockPermissions.length = 0;
    mockPermissions.push('knowledge:read', 'knowledge:write');
  });

  it('已通过评审但未发布的文章给出发布入口', async () => {
    mockGetArticle.mockResolvedValue({ ...draftArticle, reviewStatus: 'approved' });
    await renderDetail();

    expect(await screen.findByText('发布文章')).toBeInTheDocument();
    expect(screen.queryByText('提交评审')).toBeNull();
  });

  it('评审中只给禁用的评审中提示，既不可发布也不可重复提交', async () => {
    mockGetArticle.mockResolvedValue({ ...draftArticle, reviewStatus: 'under_review' });
    await renderDetail();

    const label = await screen.findByText('评审中');
    expect(label.closest('button')).toBeDisabled();
    expect(screen.queryByText('发布文章')).toBeNull();
    expect(screen.queryByText('提交评审')).toBeNull();
  });

  it('已驳回的文章允许修改后重新提交评审', async () => {
    mockGetArticle.mockResolvedValue({ ...draftArticle, reviewStatus: 'rejected' });
    await renderDetail();

    expect(await screen.findByText('提交评审')).toBeInTheDocument();
    expect(screen.queryByText('发布文章')).toBeNull();
  });
});
