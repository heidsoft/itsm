/**
 * 工单详情页「接单 / 解决 / 关闭」由后端 state-v2 的 availableActions 驱动。
 *
 * P0b 回归锁：修复前详情页零引用 availableActions，动作按钮只按权限码收口，
 * 工单状态推进后流程实例仍悬挂；接口不可用时也不能凭权限码猜状态机。
 */
import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { App } from 'antd';
import TicketDetail from '../TicketDetail';

const mockGetTicket = jest.fn();
const mockTryGetWorkflowState = jest.fn();
const mockAcceptTicket = jest.fn();
const mockResolveTicketWorkflow = jest.fn();
const mockCloseTicketWorkflow = jest.fn();

jest.mock('next/navigation', () => ({ useParams: () => ({ ticketId: '17' }) }));

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
    updateTicket: jest.fn().mockResolvedValue(undefined),
    updateTicketStatus: jest.fn().mockResolvedValue(undefined),
    acceptTicket: (...args: unknown[]) => mockAcceptTicket(...args),
    resolveTicketWorkflow: (...args: unknown[]) => mockResolveTicketWorkflow(...args),
    closeTicketWorkflow: (...args: unknown[]) => mockCloseTicketWorkflow(...args),
  },
}));

jest.mock('@/lib/api/ticket-workflow-state-api', () => ({
  TicketWorkflowStateApi: {
    getStateV2: jest.fn().mockResolvedValue(null),
    tryGetStateV2: jest.fn().mockResolvedValue(null),
    tryGetWorkflowState: (...args: unknown[]) => mockTryGetWorkflowState(...args),
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

const session = { user: { id: 8, username: 'operator', name: '处理人', permissions: [] as string[] } };

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStore: () => session,
}));

jest.mock('@/lib/hooks/useErrorHandler', () => ({
  useErrorHandler: () => ({ handleError: jest.fn() }),
}));

jest.mock('@/lib/i18n/useI18n', () => ({
  useI18n: () => ({ t: (key: string) => key, locale: 'zh-CN' }),
}));

jest.mock('@/components/common/SafeContent', () => ({
  SafeTextBlock: ({ text }: { text: string }) => <span>{text}</span>,
}));

jest.mock('@/components/business/AISuggestionPanel', () => ({
  AISuggestionPanel: () => null,
}));

jest.mock('@/components/business/WorkflowProgressCard', () => ({
  WorkflowProgressCard: () => null,
}));

jest.mock('@/components/business/detail-tabs', () => ({
  CommentPanel: () => null,
  AttachmentPanel: () => null,
  HistoryTimeline: () => null,
  ApprovalWorkflowPanel: () => null,
  RelationPanel: () => null,
  ticketCommentAdapter: {},
  ticketAttachmentAdapter: {},
  fetchAuditLogHistory: jest.fn().mockResolvedValue([]),
}));

jest.mock('@/components/ticket-relations/RelationPanel', () => ({
  RelationPanel: () => null,
}));

const ALL_PERMISSIONS = [
  'ticket:read', 'ticket:assign', 'ticket:update', 'ticket:resolve', 'ticket:close',
];

async function renderDetail(actions: string[], permissions = ALL_PERMISSIONS) {
  session.user.permissions = permissions;
  mockGetTicket.mockResolvedValue({
    id: 17,
    ticketNumber: 'TKT-202609-000017',
    title: '打印机无法联网',
    description: '报障',
    status: 'in_progress',
    priority: 'medium',
    type: 'incident',
    requesterId: 7,
    assigneeId: 8,
    tenantId: 1,
    version: 1,
    createdAt: '2026-09-11T22:50:50Z',
    updatedAt: '2026-09-11T22:50:50Z',
  });
  mockTryGetWorkflowState.mockResolvedValue({ availableActions: actions });
  mockAcceptTicket.mockResolvedValue({ message: 'ok' });
  mockResolveTicketWorkflow.mockResolvedValue({ message: 'ok' });
  mockCloseTicketWorkflow.mockResolvedValue({ message: 'ok' });
  render(
    <App>
      <TicketDetail />
    </App>
  );
  await screen.findByText('TKT-202609-000017');
  await waitFor(() => expect(mockTryGetWorkflowState).toHaveBeenCalledWith(17));
}

describe('TicketDetail 流程动作门控', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('只渲染后端 availableActions 与权限码同时命中的动作', async () => {
    await renderDetail(['resolve']);
    expect(screen.getByRole('button', { name: /解\s*决/ })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /接\s*单/ })).toBeNull();
    expect(screen.queryByRole('button', { name: /关\s*闭/ })).toBeNull();
  });

  it('state-v2 不可用时按 fail-closed 隐藏流程动作，不按权限码猜状态机', async () => {
    mockGetTicket.mockResolvedValue({
      id: 17, ticketNumber: 'TKT-202609-000017', title: '打印机无法联网',
      description: '报障', status: 'in_progress', priority: 'medium', type: 'incident',
      requesterId: 7, assigneeId: 8, tenantId: 1, version: 1,
      createdAt: '2026-09-11T22:50:50Z', updatedAt: '2026-09-11T22:50:50Z',
    });
    session.user.permissions = ALL_PERMISSIONS;
    mockTryGetWorkflowState.mockResolvedValue(null);
    render(<App><TicketDetail /></App>);
    await screen.findByText('TKT-202609-000017');
    await waitFor(() => expect(mockTryGetWorkflowState).toHaveBeenCalledWith(17));
    expect(screen.queryByRole('button', { name: /接\s*单/ })).toBeNull();
    expect(screen.queryByRole('button', { name: /解\s*决/ })).toBeNull();
  });

  it('无 ticket:update/resolve 时不给出解决按钮（与后端 RequirePermission 对齐）', async () => {
    await renderDetail(['resolve'], ['ticket:read', 'ticket:assign']);
    expect(screen.queryByRole('button', { name: /解\s*决/ })).toBeNull();
  });

  it('点接单调用 /tickets/workflow/accept 并重新拉取状态机', async () => {
    await renderDetail(['accept']);
    fireEvent.click(screen.getByRole('button', { name: /接\s*单/ }));
    await waitFor(() => expect(mockAcceptTicket).toHaveBeenCalledWith(17));
    await waitFor(() => expect(mockTryGetWorkflowState).toHaveBeenCalledTimes(2));
  });

  it('解决动作提交走 workflow 事务端点并带上解决方案', async () => {
    await renderDetail(['resolve']);
    fireEvent.click(screen.getByRole('button', { name: /解\s*决/ }));
    const dialog = await screen.findByRole('dialog');
    const textarea = dialog.querySelector('textarea');
    expect(textarea).not.toBeNull();
    fireEvent.change(textarea!, { target: { value: '更换网线后恢复' } });
    fireEvent.click(screen.getByRole('button', { name: /确\s*认/ }));
    await waitFor(() =>
      expect(mockResolveTicketWorkflow).toHaveBeenCalledWith(17, {
        resolution: '更换网线后恢复',
        workNotes: '更换网线后恢复',
      })
    );
  });
});
