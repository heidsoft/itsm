import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import MarketplaceCatalogAdminPage from '../page';
import {
  listCatalogItems,
  publishCatalogItem,
  unpublishCatalogItem,
} from '@/lib/api/marketplace-catalog-api';
import type { CatalogItem } from '@/lib/api/marketplace-catalog-api';

/**
 * 商品目录管理页的回归。
 *
 * 这个页面存在的理由是「目录不再只有 seeder 能写」，所以断言围绕三件事：
 *   1. 状态必须可辨：草稿 / 已上架 / 已下架对应不同操作，把草稿显示成「已上架」
 *      会让运营以为商品已经在卖；
 *   2. 上架/下架按钮必须按当前状态出现——已上架的条目还给「上架」按钮，
 *      点下去就是一次无意义的 200；
 *   3. 后端 409 的处置指引（先发布版本 / 已有租户安装）必须原样透出，
 *      否则运营只会看到「操作失败」而不知道下一步。
 */

jest.mock('@/lib/api/marketplace-catalog-api', () => ({
  __esModule: true,
  listCatalogItems: jest.fn(),
  publishCatalogItem: jest.fn(),
  unpublishCatalogItem: jest.fn(),
  deleteCatalogItem: jest.fn(),
  getCatalogItem: jest.fn(),
  createCatalogItem: jest.fn(),
  updateCatalogItem: jest.fn(),
  createItemVersion: jest.fn(),
  publishItemVersion: jest.fn(),
  CATALOG_STATUS_META: {
    draft: { label: '草稿', color: 'default', hint: '尚未上架，租户看不到' },
    reviewing: { label: '待审', color: 'processing', hint: '审核中，尚未上架' },
    published: { label: '已上架', color: 'green', hint: '租户可浏览并安装' },
    rejected: { label: '已驳回', color: 'red', hint: '审核未通过，需修改后重新上架' },
    deprecated: { label: '已下架', color: 'orange', hint: '停止新安装；已安装租户不受影响' },
  },
}));

jest.mock('@/app/components/PageContainer', () => ({
  PageContainer: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

const mockedList = listCatalogItems as jest.MockedFunction<typeof listCatalogItems>;
const mockedPublish = publishCatalogItem as jest.MockedFunction<typeof publishCatalogItem>;
const mockedUnpublish = unpublishCatalogItem as jest.MockedFunction<typeof unpublishCatalogItem>;

const baseItem: CatalogItem = {
  id: 1,
  name: 'feishu-connector',
  type: 'connector',
  title: '飞书审批增强连接器',
  provider: 'ITSM Foundation',
  description: '把工单与飞书审批打通',
  tags: ['飞书'],
  rating: 0,
  installCount: 3,
  latestVersion: '1.2.0',
  status: 'published',
  isOfficial: true,
  isFree: true,
  price: 0,
  createdAt: '2026-10-01T00:00:00Z',
  updatedAt: '2026-10-20T00:00:00Z',
};

const items: CatalogItem[] = [
  baseItem,
  {
    ...baseItem,
    id: 2,
    name: 'excel-export-skill',
    type: 'skill',
    title: 'Excel 导出',
    latestVersion: '',
    status: 'draft',
    installCount: 0,
  },
  {
    ...baseItem,
    id: 3,
    name: 'legacy-connector',
    title: '旧版连接器',
    status: 'deprecated',
    installCount: 12,
  },
];

const rowOf = (title: string): HTMLElement => {
  const row = screen.getByText(title).closest('tr');
  if (!row) throw new Error(`找不到商品行：${title}`);
  return row as HTMLElement;
};

const actionsOf = (title: string): string[] =>
  Array.from(rowOf(title).querySelectorAll('button'))
    .map(button => button.textContent?.trim() ?? '')
    .filter(Boolean);

const renderPage = () =>
  render(
    <App>
      <MarketplaceCatalogAdminPage />
    </App>
  );

describe('商品目录管理', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedList.mockResolvedValue({ items, total: items.length, page: 1, pageSize: 20 });
    mockedPublish.mockResolvedValue({ item: { ...baseItem, status: 'published' }, versions: [] });
    mockedUnpublish.mockResolvedValue({ item: { ...baseItem, status: 'deprecated' }, versions: [] });
  });

  it('状态可辨：草稿 / 已上架 / 已下架分别显示', async () => {
    renderPage();

    expect(await screen.findByText('飞书审批增强连接器')).toBeInTheDocument();
    // 按行断言而不是全局文本：统计卡片里也有「已上架」字样，
    // 全局匹配会同时命中标题与标签，掩盖「某一行状态显示错了」这种真实缺陷。
    expect(rowOf('飞书审批增强连接器').textContent).toContain('已上架');
    expect(rowOf('Excel 导出').textContent).toContain('草稿');
    expect(rowOf('旧版连接器').textContent).toContain('已下架');
    // 未发布商品的线上版本必须显式说「未发布」，不能留空让人以为是加载失败。
    expect(rowOf('Excel 导出').textContent).toContain('未发布');
  });

  it('操作按状态出现：已上架给「下架」，草稿/已下架给「上架」', async () => {
    renderPage();

    await screen.findByText('飞书审批增强连接器');

    // 断言按钮文案而不是行文本：「已上架」里也含「上架」二字，用 textContent
    // 判断会得出相反结论（这正是本测试第一版踩到的坑）。
    expect(actionsOf('飞书审批增强连接器')).toEqual(['下架', '版本', '编辑']);
    expect(actionsOf('Excel 导出')).toEqual(['上架', '版本', '编辑', '删除']);
    expect(actionsOf('旧版连接器')).toEqual(['上架', '版本', '编辑', '删除']);
  });

  it('下架调用后端并给出「已安装租户不受影响」的说明', async () => {
    renderPage();

    await screen.findByText('飞书审批增强连接器');
    const publishedRow = screen.getByText('飞书审批增强连接器').closest('tr')!;
    const unpublishButton = Array.from(publishedRow.querySelectorAll('button')).find(button =>
      button.textContent?.includes('下架')
    );
    expect(unpublishButton).toBeDefined();
    fireEvent.click(unpublishButton!);

    await waitFor(() => expect(mockedUnpublish).toHaveBeenCalledWith(1));
  });

  it('后端拒绝时原样透出处置指引（先发布版本），而不是笼统「操作失败」', async () => {
    mockedPublish.mockRejectedValueOnce(new Error('该目录项还没有已发布版本，请先新增并发布一个版本再上架'));
    renderPage();

    await screen.findByText('Excel 导出');
    const draftRow = screen.getByText('Excel 导出').closest('tr')!;
    const publishButton = Array.from(draftRow.querySelectorAll('button')).find(button =>
      button.textContent?.includes('上架')
    );
    fireEvent.click(publishButton!);

    expect(await screen.findByText(/请先新增并发布一个版本再上架/)).toBeInTheDocument();
  });

  it('目录为空时给出可执行空态', async () => {
    mockedList.mockResolvedValueOnce({ items: [], total: 0, page: 1, pageSize: 20 });
    renderPage();

    expect(await screen.findByText(/目录为空/)).toBeInTheDocument();
  });
});
