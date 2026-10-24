/**
 * 工单详情操作栏按后端 RequirePermission 收口。
 *
 * 2026-10-18 隔离栈实测：报障人（end_user）打开自己提的工单，页面上照常给出
 * 指派 / 升级 / 抄送 / 删除，逐个点击都是 403（后端分别要求 ticket:assign、
 * ticket:escalate、ticket:assign、ticket:delete）。隐藏入口只负责呈现，
 * 授权仍以后端为准。
 *
 * 2026-10-20 prod 实测的另一半：抄送与审批曾挂在 workflow:write（目录语义是流程
 * 定义管理）上，不持该码的坐席/主管按钮全亮但必 403；归位到 ticket:approve /
 * ticket:assign 后，按钮可见性与后端授权共用同一套 ticket 动作码。
 */
import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import TicketDetail from '../TicketDetail';

const mockGetTicket = jest.fn();

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

// 每个用例改这份权限清单，其余 fixture 复用同一张工单。
const session = { user: { id: 7, username: 'viewer', name: '走查账号', permissions: [] as string[] } };

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

const OPERATOR_KEYS = [
  'ticketDetail.assign',
  'ticketDetail.escalate',
  'ticketDetail.cc',
  'common.delete',
];

async function renderWithPermissions(permissions: string[], actorId = 7) {
  session.user.permissions = permissions;
  session.user.id = actorId;
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
  render(
    <App>
      <TicketDetail />
    </App>
  );
  await waitFor(() => expect(mockGetTicket).toHaveBeenCalled());
  await screen.findByText('TKT-202609-000017');
}

describe('TicketDetail 操作栏 — 按权限码渲染', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('报障人（ticket:update，无派单/升级/流程写权限）看不到 operator 按钮', async () => {
    await renderWithPermissions([
      'ticket:read',
      'ticket:write',
      'ticket:create',
      'ticket:update',
      'workflow:read',
    ]);

    for (const key of OPERATOR_KEYS) {
      expect(screen.queryByText(key)).toBeNull();
    }
    // 自己的工单仍可编辑（PUT /tickets/:id 要 ticket:update）
    expect(screen.getByText('common.edit')).toBeInTheDocument();
  });

  it('只读角色没有任何操作入口，并显示只读说明', async () => {
    await renderWithPermissions(['ticket:read']);

    expect(screen.queryByText('common.edit')).toBeNull();
    expect(screen.getByText('ticketDetail.readonlyForViewer')).toBeInTheDocument();
  });

  it('持 ticket:admin 的租户管理员仍可见派单入口（与后端 resource:admin 判定一致）', async () => {
    await renderWithPermissions(['ticket:read', 'ticket:write', 'ticket:admin']);

    expect(screen.getByText('ticketDetail.assign')).toBeInTheDocument();
    expect(screen.getByText('ticketDetail.escalate')).toBeInTheDocument();
    expect(screen.getByText('common.delete')).toBeInTheDocument();
  });

  it('持 ticket:assign 才出现抄送入口', async () => {
    await renderWithPermissions(['ticket:read', 'ticket:assign']);

    expect(screen.getByText('ticketDetail.cc')).toBeInTheDocument();
  });

  it('workflow:write（流程定义管理）不再给出工单操作入口', async () => {
    await renderWithPermissions(['ticket:read', 'workflow:read', 'workflow:write']);

    for (const key of OPERATOR_KEYS) {
      expect(screen.queryByText(key)).toBeNull();
    }
  });

  // 坐席视角的回归锁：不持 workflow:* 也必须能审批与抄送，否则一线处理链路又是 403。
  it('一线坐席（ticket:approve/assign，非报障人）审批与抄送入口可用', async () => {
    await renderWithPermissions(
      ['ticket:read', 'ticket:assign', 'ticket:approve', 'ticket:resolve', 'ticket:close'],
      8,
    );

    const approve = screen.getByText('ticketDetail.approve').closest('button');
    expect(approve).not.toBeNull();
    expect(approve).not.toBeDisabled();
    expect(screen.getByText('ticketDetail.cc')).toBeInTheDocument();
    expect(screen.getByText('ticketDetail.assign')).toBeInTheDocument();
  });
});
