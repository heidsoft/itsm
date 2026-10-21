/**
 * 工单详情「处理人/申请人」的显示回归。
 *
 * TicketResponse 历史上挂着从未赋值的 requester/assignee 对象，前端要么显示空、
 * 要么把已派工单显示成「未分配」，只能靠详情页再拉一次全量通讯录去猜姓名。
 * 现在姓名由后端按租户投影（requesterName/assigneeName），详情页直接显示；
 * 投影不到时如实回退用户 ID，未派工才显示「未分配」。
 */
import React from 'react';
import { render, screen } from '@testing-library/react';
import { App } from 'antd';
import TicketDetail from '../TicketDetail';

const mockGetTicket = jest.fn();

jest.mock('next/navigation', () => ({ useParams: () => ({ ticketId: '21' }) }));

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
    approve: jest.fn().mockResolvedValue(undefined),
    reject: jest.fn().mockResolvedValue(undefined),
  },
}));

// 通讯录按弹窗启用；本用例只校验详情投影，mock 成空目录即可。
jest.mock('@/lib/hooks/useUserListQuery', () => ({
  useUserListQuery: () => ({ isLoading: false, data: { users: [] } }),
}));

jest.mock('@/lib/store/auth-store', () => ({
  useAuthStore: () => ({ user: { id: 1, username: 'admin', name: '系统管理员' } }),
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

const baseTicket = {
  id: 21,
  ticketNumber: 'TKT-202610-000021',
  title: 'VPN 无法连接',
  description: '早上开始 VPN 一直掉线',
  status: 'in_progress',
  priority: 'high',
  type: 'incident',
  requesterId: 3,
  tenantId: 1,
  version: 2,
  createdAt: '2026-10-19T00:50:00Z',
  updatedAt: '2026-10-19T02:10:00Z',
};

function renderDetail() {
  render(
    <App>
      <TicketDetail />
    </App>
  );
}

describe('TicketDetail 处理人/申请人显示名 — 服务端投影回归', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('显示后端投影的处理人与申请人姓名', async () => {
    mockGetTicket.mockResolvedValue({
      ...baseTicket,
      assigneeId: 7,
      assigneeName: '张三',
      requesterName: '李四',
    });
    renderDetail();

    expect(await screen.findByText('张三')).toBeInTheDocument();
    expect(screen.getByText('李四')).toBeInTheDocument();
    expect(screen.queryByText('用户 #7')).toBeNull();
  });

  it('投影不到姓名时回退用户 ID，而不是谎称未分配', async () => {
    mockGetTicket.mockResolvedValue({ ...baseTicket, assigneeId: 7 });
    renderDetail();

    expect(await screen.findByText('用户 #7')).toBeInTheDocument();
    expect(screen.queryByText('ticketDetail.unassigned')).toBeNull();
  });

  it('只有确实未派工时才显示未分配', async () => {
    mockGetTicket.mockResolvedValue(baseTicket);
    renderDetail();

    expect(await screen.findByText('ticketDetail.unassigned')).toBeInTheDocument();
  });
});
