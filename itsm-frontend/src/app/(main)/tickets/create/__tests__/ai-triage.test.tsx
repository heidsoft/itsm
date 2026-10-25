/**
 * 建单页「AI 辅助分类」回归。
 *
 * 2026-10-25 本地 Docker 生产栈浏览器实测发现两个叠加缺陷：
 *
 * 1. handleAITriage 用 form.validateFields() 取标题/描述，这会连带校验当前
 *    工单类型的全部必填动态字段（账号类型/目标系统/用户名/权限说明）。只要
 *    其中任一为空就抛异常，被 catch 吞掉后统一弹出「AI 分类服务暂时不可用」——
 *    把「表单没填完」误报成「服务故障」。实测该接口本身返回 200 且给出
 *    network/0.98/high 的正常建议，是前端自己把请求挡住了。
 *
 * 2. AI 返回的 category 是英文 code（network），而下拉选项的 value 是分类
 *    名称（网络故障）。缺 code→name 映射时匹配永远失败，分类不会回填，但
 *    成功提示照旧弹「AI 分类建议已应用」——提示与事实相反。
 *
 * 这两个用例在修复前失败、修复后通过，锁住上述语义。
 */
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App } from 'antd';
import { httpClient } from '@/lib/api/http-client';
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

// 账号申请：两个必填动态字段，用来复现「表单没填完就点 AI」的真实路径。
const accountApplyType = {
  id: 5,
  code: 'account_apply',
  name: '账号申请',
  description: '申请系统账号、VPN账号、堡垒机账号等',
  customFields: [
    {
      id: 'f1',
      name: 'account_type',
      label: '账号类型',
      type: 'select',
      required: true,
      options: [
        { label: '新用户', value: 'new' },
        { label: '权限变更', value: 'change' },
      ],
    },
    {
      id: 'f2',
      name: 'target_system',
      label: '目标系统',
      type: 'text',
      required: true,
    },
  ],
};

function renderPage() {
  return render(
    <App>
      <CreateTicketPage />
    </App>,
  );
}

describe('CreateTicketPage AI 辅助分类', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedTypeApi.list.mockResolvedValue({ items: [accountApplyType] } as never);
    mockedCmdb.getAllCIs.mockResolvedValue([] as never);
    mockedHttp.get.mockImplementation((url: string) =>
      Promise.resolve(url.includes('/users') ? { items: [] } : []),
    );
    // 分类主数据：名称是中文，code 是英文 slug —— 与后端 TicketCategory 一致
    mockedCategoryApi.getCategories.mockResolvedValue({
      items: [
        { id: 3, name: '网络故障', code: 'network', isActive: true },
        { id: 1, name: '硬件故障', code: 'hardware', isActive: true },
      ],
    } as never);
  });

  it('必填动态字段为空时仍应发出请求，而不是谎报服务不可用', async () => {
    mockedHttp.post.mockResolvedValue({
      suggestions: { category: 'network', priority: 'high', confidence: 0.98 },
    } as never);

    renderPage();

    // 选中账号申请，出现必填动态字段，但**不填**
    await userEvent.click(await screen.findByText('账号申请'));
    expect(await screen.findByLabelText('目标系统')).toBeInTheDocument();

    await userEvent.type(screen.getByLabelText('标题'), 'VPN 无法连接内网');

    await userEvent.click(screen.getByRole('button', { name: /获取 AI 建议/ }));

    // 关键断言：请求真的发出去了
    await waitFor(() =>
      expect(mockedHttp.post).toHaveBeenCalledWith(
        '/api/v1/ai/triage',
        expect.objectContaining({ title: 'VPN 无法连接内网' }),
      ),
    );

    // 关键断言：不得把「表单没填完」说成「服务故障」
    expect(screen.queryByText(/AI 分类服务暂时不可用/)).not.toBeInTheDocument();
  });

  it('AI 返回英文分类 code 时应映射成中文分类名并如实提示已应用', async () => {
    mockedHttp.post.mockResolvedValue({
      suggestions: { category: 'network', priority: 'high', confidence: 0.98 },
    } as never);

    renderPage();

    await userEvent.click(await screen.findByText('账号申请'));
    await userEvent.type(await screen.findByLabelText('标题'), 'VPN 无法连接内网');
    await userEvent.click(screen.getByRole('button', { name: /获取 AI 建议/ }));

    // 提示必须说清回填了什么，且分类用中文名而不是 network
    expect(await screen.findByText(/AI 建议已应用：分类「网络故障」/)).toBeInTheDocument();
    // 原始 slug 不应出现在建议标签里
    expect(screen.queryByText(/分类: network/)).not.toBeInTheDocument();
  });

  it('AI 分类不在主数据中时不得谎报成功', async () => {
    mockedHttp.post.mockResolvedValue({
      suggestions: { category: 'unknown_slug', priority: 'medium', confidence: 0.4 },
    } as never);

    renderPage();

    await userEvent.click(await screen.findByText('账号申请'));
    await userEvent.type(await screen.findByLabelText('标题'), '打印机坏了');
    await userEvent.click(screen.getByRole('button', { name: /获取 AI 建议/ }));

    // 只有优先级被应用，不能整体报成功
    expect(await screen.findByText(/AI 建议分类「unknown_slug」不在当前分类主数据中/)).toBeInTheDocument();
    expect(screen.queryByText(/AI 分类建议已应用/)).not.toBeInTheDocument();
  });
});
