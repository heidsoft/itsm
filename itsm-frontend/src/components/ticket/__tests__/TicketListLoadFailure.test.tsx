import { fireEvent, render, screen } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import TicketList from '../TicketList';

const mockFetch = jest.fn();
const mockUpdate = jest.fn();
let mockError: string | null = 'boom';
let mockLoading = false;
jest.mock('@/lib/hooks/useTickets', () => ({
  useTickets: () => ({
    tickets: [],
    loading: mockLoading,
    error: mockError,
    pagination: { current: 1, pageSize: 10, total: 0 },
    filters: {},
    fetchTickets: mockFetch,
    updateFilters: mockUpdate,
    updatePagination: jest.fn(),
    deleteTicket: jest.fn(),
    batchDeleteTickets: jest.fn(),
  }),
}));
jest.mock('@/lib/hooks/use-permissions', () => ({
  usePermissions: () => ({ hasPermission: () => false }),
}));
jest.mock('../TicketListToolbar', () => ({ TicketListToolbar: () => null }));
jest.mock('../TicketListFilters', () => ({ TicketListFilters: () => null }));
jest.mock('../TicketBatchOperations', () => ({ __esModule: true, default: () => null }));

beforeEach(() => {
  mockError = 'boom';
  mockLoading = false;
});
const mount = () =>
  render(
    <ConfigProvider locale={zhCN}>
      <TicketList />
    </ConfigProvider>
  );

it('加载失败显示错误与重试，不能伪装为空列表', () => {
  mount();
  expect(screen.getByText('工单列表加载失败')).toBeInTheDocument();
  expect(screen.queryByText('暂无数据')).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: '重新加载' }));
  expect(mockFetch).toHaveBeenCalledTimes(1);
});
it('成功返回空列表保留原有空态', () => {
  mockError = null;
  mount();
  expect(screen.getAllByText('暂无数据').length).toBeGreaterThan(0);
  expect(screen.queryByText('工单列表加载失败')).not.toBeInTheDocument();
});
it('重试加载中仍保留表格加载指示', () => {
  mockLoading = true;
  const { container } = mount();
  expect(screen.queryByText('工单列表加载失败')).not.toBeInTheDocument();
  expect(container.querySelector('.ant-spin-spinning')).toBeInTheDocument();
});
