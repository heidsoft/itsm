import { fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';
import Page from '../page';
const mockGetTicketStats = jest.fn();
jest.mock('next/navigation', () => ({ useRouter: () => ({ push: jest.fn() }), useSearchParams: () => new URLSearchParams() }));
jest.mock('@/components/ticket/TicketList', () => ({ __esModule: true, default: () => null }));
jest.mock('@/components/ticket/TicketKanban', () => ({ __esModule: true, default: () => null }));
jest.mock('@/components/ticket/TicketAdvancedSearch', () => ({ __esModule: true, default: () => null }));
jest.mock('@/lib/hooks/use-permissions', () => ({ useOperationPermissions: () => ({ ticket: { canCreate: () => false } }) }));
jest.mock('@/lib/services/ticket-service', () => ({ ticketService: { getTicketStats: (...args: unknown[]) => mockGetTicketStats(...args) } }));
beforeEach(() => { jest.spyOn(console, 'error').mockImplementation(() => {}); });
it('统计失败显示破折号，重试成功恢复真实零值', async () => {
  mockGetTicketStats.mockRejectedValueOnce(new Error('offline')).mockResolvedValue({ total: 0, open: 0, overdue: 0 } as never);
  render(<App><Page /></App>);
  const retry = await screen.findByRole('button', { name: /重\s*试/ });
  expect(screen.getAllByText('—')).toHaveLength(4);
  fireEvent.click(retry);
  expect(await screen.findAllByText('0')).not.toHaveLength(0);
});
