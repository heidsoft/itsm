import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App } from 'antd';
import { ApiRequestError, httpClient } from '@/lib/api/http-client';
import { CMDBApi } from '@/lib/api/cmdb-api';
import { TicketCategoryApi } from '@/lib/api/ticket-category-api';
import { TicketTypeApi } from '@/lib/api/ticketTypeApi';
import CreateTicketPage from '../page';

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn(), replace: jest.fn(), back: jest.fn() }),
}));

jest.mock('@/lib/api/ticketTypeApi', () => ({
  TicketTypeApi: { list: jest.fn() },
}));

jest.mock('@/lib/api/cmdb-api', () => ({
  CMDBApi: { getAllCIs: jest.fn() },
}));

jest.mock('@/lib/api/ticket-category-api', () => ({
  TicketCategoryApi: { getCategories: jest.fn() },
}));

jest.mock('@/lib/api/http-client', () => ({
  ...jest.requireActual('@/lib/api/http-client'),
  httpClient: { get: jest.fn(), post: jest.fn() },
}));

const mockedCategoryApi = TicketCategoryApi as jest.Mocked<typeof TicketCategoryApi>;
const mockedTypeApi = TicketTypeApi as jest.Mocked<typeof TicketTypeApi>;
const mockedHttp = httpClient as jest.Mocked<typeof httpClient>;
const mockedCmdb = CMDBApi as jest.Mocked<typeof CMDBApi>;

function renderPage() {
  return render(
    <App>
      <CreateTicketPage />
    </App>,
  );
}

describe('CreateTicketPage 工单分类加载', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedTypeApi.list.mockResolvedValue({ items: [] } as never);
    mockedCmdb.getAllCIs.mockResolvedValue([] as never);
    // 建单页还会拉用户与部门目录，与本次修复无关，按各自 DTO 形态给空集。
    mockedHttp.get.mockImplementation((url: string) =>
      Promise.resolve(url.includes('/users') ? { items: [] } : []),
    );
  });

  it('无权限时说明权限不足，而不是留下一个静默的空下拉', async () => {
    mockedCategoryApi.getCategories.mockRejectedValue(
      new ApiRequestError('权限不足', 403, 2003) as never,
    );

    renderPage();

    expect(await screen.findByText(/没有「查看工单分类」权限/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '重试加载分类' })).toBeEnabled();
    // 403 不是「后端没数据」，两种失败不得共用一句文案。
    expect(screen.queryByText(/工单分类加载失败/)).not.toBeInTheDocument();
  });

  it('服务端失败给出原因，重试成功后错误提示消失', async () => {
    mockedCategoryApi.getCategories
      .mockRejectedValueOnce(new ApiRequestError('HTTP error! status: 500', 500) as never)
      .mockResolvedValueOnce({
        items: [{ id: 1, name: '网络', isActive: true }],
      } as never);

    renderPage();

    expect(await screen.findByText(/工单分类加载失败：HTTP error! status: 500/)).toBeInTheDocument();

    await userEvent.click(screen.getByRole('button', { name: '重试加载分类' }));

    await waitFor(() => expect(mockedCategoryApi.getCategories).toHaveBeenCalledTimes(2));
    expect(
      await waitFor(() =>
        expect(screen.queryByText(/工单分类加载失败/)).not.toBeInTheDocument(),
      ),
    ).toBeUndefined();
  });

  it('加载成功时不显示任何失败提示', async () => {
    mockedCategoryApi.getCategories.mockResolvedValue({
      items: [{ id: 1, name: '网络', isActive: true }],
    } as never);

    renderPage();

    await waitFor(() => expect(mockedCategoryApi.getCategories).toHaveBeenCalledTimes(1));
    expect(screen.queryByText(/查看工单分类」权限/)).not.toBeInTheDocument();
    expect(screen.queryByText(/工单分类加载失败/)).not.toBeInTheDocument();
  });
});
