/**
 * 回归锁：工单详情深链 /tickets/1 → /tickets/2 复用组件实例时不得串页。
 *
 * 缺陷：ticketKey 变化时 per-id 状态（ticket/slaInfo/cis…）不重置，且 fetch
 * 无 generation 守卫——慢的旧 id 响应会覆盖新 id 数据，旧工单内容短暂显示在新 URL 下。
 */
import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import TicketDetail from '../TicketDetail';

const mockGetTicket = jest.fn();
let mockTicketId: string = '1';

jest.mock('next/navigation', () => ({
  useParams: () => ({ ticketId: mockTicketId }),
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
    updateTicket: jest.fn().mockResolvedValue(undefined),
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

const session = { user: { id: 7, username: 'viewer', name: '走查账号', permissions: ['ticket:read'] } };
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

const ticket = (id: number, number: string) => ({
  id, ticketNumber: number, title: `工单-${id}`, description: '描述',
  status: 'in_progress', priority: 'medium', type: 'incident', requesterId: 7, assigneeId: 8,
  tenantId: 1, version: 1, createdAt: '2026-09-11T00:00:00Z', updatedAt: '2026-09-11T00:00:00Z',
});

describe('TicketDetail 深链切 id 不串页', () => {
  beforeEach(() => jest.clearAllMocks());

  it('慢的旧 id 响应后到，不得覆盖新 id 的工单数据', async () => {
    let resolveId1: (v: unknown) => void = () => {};
    // id=1 故意慢：永不自动 resolve，由测试手动放行
    mockGetTicket.mockImplementation((key: number | string) => {
      if (key === 1) return new Promise(resolve => { resolveId1 = resolve; });
      return Promise.resolve(ticket(2, 'TKT-2'));
    });

    mockTicketId = '1';
    const { rerender } = render(<App><TicketDetail id="1" /></App>);

    // 切到 id=2（同一组件实例，仅 props 变化）
    mockTicketId = '2';
    rerender(<App><TicketDetail id="2" /></App>);

    // 新 id 的数据应当渲染
    await waitFor(() => expect(screen.getByText('TKT-2')).toBeInTheDocument());

    // 此时旧 id 的慢响应才到达——必须被 generation 守卫丢弃
    resolveId1(ticket(1, 'TKT-1'));
    await waitFor(() => expect(mockGetTicket).toHaveBeenCalledWith(1));

    expect(screen.queryByText('TKT-1')).toBeNull();
    expect(screen.getByText('TKT-2')).toBeInTheDocument();
  });
});
