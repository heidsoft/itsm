import React from 'react';
import { fireEvent, render, screen, waitFor } from '@/lib/test-utils';
import { TicketModalContainer } from '../TicketModalContainer';
import { ticketService } from '@/lib/services/ticket-service';
import { TicketType, TicketPriority } from '@/lib/services/ticket-service';

jest.mock('@/lib/services/ticket-service', () => {
  const actual = jest.requireActual('@/lib/services/ticket-service');
  return {
    ...actual,
    ticketService: {
      createTicket: jest.fn(),
      updateTicket: jest.fn(),
    },
  };
});

jest.mock('../TicketModal', () => ({
  TicketModal: ({ onSubmit, loading }: { onSubmit: (values: unknown) => Promise<void>; loading: boolean }) => (
    <button
      type="button"
      disabled={loading}
      onClick={() =>
        void onSubmit({
          title: '网络故障',
          description: '办公网络无法连接',
          type: TicketType.INCIDENT,
          category: 'Network',
          priority: TicketPriority.HIGH,
        }).catch(() => undefined)
      }
    >
      保存
    </button>
  ),
}));

describe('TicketModalContainer', () => {
  beforeEach(() => jest.clearAllMocks());

  it('owns persistence and creates a ticket exactly once', async () => {
    jest.mocked(ticketService.createTicket).mockResolvedValue({ id: 1 } as never);
    const onSuccess = jest.fn();
    render(
      <TicketModalContainer
        visible
        editingTicket={null}
        onCancel={jest.fn()}
        onSuccess={onSuccess}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: '保存' }));

    await waitFor(() => expect(ticketService.createTicket).toHaveBeenCalledTimes(1));
    expect(ticketService.updateTicket).not.toHaveBeenCalled();
    expect(onSuccess).toHaveBeenCalledTimes(1);
  });

  it('does not report success when persistence fails', async () => {
    jest.mocked(ticketService.createTicket).mockRejectedValue(new Error('保存失败'));
    const onSuccess = jest.fn();
    render(
      <TicketModalContainer
        visible
        editingTicket={null}
        onCancel={jest.fn()}
        onSuccess={onSuccess}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: '保存' }));

    await waitFor(() => expect(ticketService.createTicket).toHaveBeenCalledTimes(1));
    expect(onSuccess).not.toHaveBeenCalled();
  });
});
