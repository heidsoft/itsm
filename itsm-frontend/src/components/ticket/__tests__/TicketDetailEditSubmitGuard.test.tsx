/**
 * 回归锁：工单详情「保存修改」必须防重复提交。
 *
 * 缺陷：updateTicket 带 version 乐观锁，双击会让第二次请求 409「已被修改」，
 * 第一次其实已保存成功，用户被误导为保存失败。原保存按钮无 loading/disabled，
 * 也无 in-flight ref 守卫。
 */
import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import TicketDetail from '../TicketDetail';

const mockGetTicket = jest.fn();
const mockUpdateTicket = jest.fn();

jest.mock('next/navigation', () => ({
  useParams: () => ({ ticketId: '17' }),
  useRouter: () => ({ push: jest.fn(), replace: jest.fn(), back: jest.fn() }),
}));

jest.mock('next/link', () => {
  const Link = ({ children }: { children: React.ReactNode }) => <a>{children}</a>;
  return { __esModule: true, default: Link };
});

jest.mock('@/lib/api/ticket-api', () => ({
  TicketApi: {
    getTicket: (...args: unknown[]) => mockGetTicket(...args),
    getTicketSLA: jest.fn().mockResolvedValue(null),
    getTicketConfigurationItems: jest.fn().mockResolvedValue([]),
    getTicketHistory: jest.fn().mockResolvedValue([]),
    assignTicket: jest.fn().mockResolvedValue(undefined),
    ccTicket: jest.fn().mockResolvedValue(undefined),
    deleteTicket: jest.fn().mockResolvedValue(undefined),
    updateTicket: (...args: unknown[]) => mockUpdateTicket(...args),
    updateTicketStatus: jest.fn().mockResolvedValue(undefined),
  },
}));

jest.mock('@/lib/api/ticket-approval-api', () => ({
  TicketApprovalApi: {
    getApprovalChain: jest.fn().mockResolvedValue([]),
    getApprovalRecords: jest.fn().mockResolvedValue({ items: [], total: 0 }),
    approve: jest.fn().mockResolvedValue(undefined),
    reject: jest.fn().mockResolvedValue(undefined),
  },
}));

jest.mock('@/lib/hooks/useUserListQuery', () => ({
  useUserListQuery: () => ({ isLoading: false, data: { items: [] } }),
}));

const session = { user: { id: 7, username: 'viewer', name: '走查账号', permissions: ['ticket:update'] } };
jest.mock('@/lib/store/auth-store', () => ({ useAuthStore: () => session }));
jest.mock('@/lib/hooks/useErrorHandler', () => ({ useErrorHandler: () => ({ handleError: jest.fn() }) }));
jest.mock('@/lib/i18n/useI18n', () => ({ useI18n: () => ({ t: (key: string) => key, locale: 'zh-CN' }) }));
jest.mock('@/components/common/SafeContent', () => ({ SafeTextBlock: ({ text }: { text: string }) => <span>{text}</span> }));
jest.mock('@/components/business/AISuggestionPanel', () => ({ AISuggestionPanel: () => null }));
jest.mock('@/components/business/WorkflowProgressCard', () => ({ WorkflowProgressCard: () => null }));
jest.mock('@/components/business/detail-tabs', () => ({
  CommentPanel: () => null, AttachmentPanel: () => null, HistoryTimeline: () => null,
  ApprovalWorkflowPanel: () => null, RelationPanel: () => null,
  ticketCommentAdapter: {}, ticketAttachmentAdapter: {}, fetchAuditLogHistory: jest.fn().mockResolvedValue([]),
}));
jest.mock('@/components/ticket-relations/RelationPanel', () => ({ RelationPanel: () => null }));

describe('TicketDetail 编辑保存防重复提交', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetTicket.mockResolvedValue({
      id: 17, ticketNumber: 'TKT-17', title: '打印机无法联网', description: '报障描述足够长',
      status: 'in_progress', priority: 'medium', type: 'incident', requesterId: 7, assigneeId: 8,
      tenantId: 1, version: 3, createdAt: '2026-09-11T22:50:50Z', updatedAt: '2026-09-11T22:50:50Z',
    });
  });

  it('保存请求在途时按钮进入 loading/disabled，双击只发一次 updateTicket', async () => {
    let resolveUpdate: () => void = () => {};
    mockUpdateTicket.mockImplementation(
      () => new Promise<void>(resolve => { resolveUpdate = resolve; })
    );

    render(<App><TicketDetail /></App>);
    await screen.findByText('TKT-17');

    fireEvent.click(screen.getByText('common.edit'));
    const saveBtn = await screen.findByRole('button', { name: /ticketDetail.saveChanges/ });

    // 第一次点击让请求进入在途
    fireEvent.click(saveBtn);
    await waitFor(() => expect(mockUpdateTicket).toHaveBeenCalledTimes(1));

    // 在途期间按钮必须不可再点
    expect(saveBtn).toBeDisabled();

    // 再点两次，仍只能是一次请求
    fireEvent.click(saveBtn);
    fireEvent.click(saveBtn);
    expect(mockUpdateTicket).toHaveBeenCalledTimes(1);

    // 放行请求：成功保存后弹窗关闭，但请求数始终为 1（乐观锁版本未被第二次点击消费）
    resolveUpdate();
    await waitFor(() => expect(mockUpdateTicket).toHaveBeenCalledTimes(1));
  });
});
