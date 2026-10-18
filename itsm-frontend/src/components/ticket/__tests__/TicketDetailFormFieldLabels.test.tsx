/**
 * 工单详情「动态表单字段」的标签解析回归。
 *
 * formFields 的 key 是租户在工单类型 custom_fields 里定义的**字段名**
 * （如 account_type / target_system），可展示的中文标签只存在于类型 schema。
 * 详情页此前用 label={key} 直接渲染键名，用户看到的是英文键而不是「账号类型」；
 * 且响应侧的 key 归一化会把它改写成 accountType，连类型 schema 都对不上。
 * 修复后：按 ticketTypeId 拉取类型定义做 name → label 映射，失败时回退原始键名。
 * 2026-10-18 浏览器实测发现（本地 Docker 生产栈）。
 */
import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import TicketDetail from '../TicketDetail';

const mockGetTicket = jest.fn();
const mockGetTicketType = jest.fn();

jest.mock('next/navigation', () => ({ useParams: () => ({ ticketId: '15' }) }));

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

// 类型 schema：字段名 → 标签的事实来源
jest.mock('@/lib/api/ticketTypeApi', () => ({
  TicketTypeApi: {
    get: (...args: unknown[]) => mockGetTicketType(...args),
  },
}));

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

const ticketWithFormFields = {
  id: 15,
  ticketNumber: 'TKT-202610-000015',
  title: '账号申请',
  description: '[账号申请] 账号类型: 新用户',
  status: 'new',
  priority: 'low',
  type: 'service_request',
  ticketTypeId: 5,
  ticketTypeCode: 'account_apply',
  ticketTypeName: '账号申请',
  formFields: { account_type: 'new', target_system: 'GitLab' },
  requesterId: 1,
  tenantId: 1,
  version: 1,
  createdAt: '2026-10-18T00:50:00Z',
  updatedAt: '2026-10-18T00:50:00Z',
};

const typeDefinition = {
  id: 5,
  code: 'account_apply',
  name: '账号申请',
  customFields: [
    { id: 'f1', name: 'account_type', label: '账号类型', type: 'select', required: true },
    { id: 'f2', name: 'target_system', label: '目标系统', type: 'text', required: true },
  ],
};

function renderDetail() {
  render(
    <App>
      <TicketDetail />
    </App>
  );
}

describe('TicketDetail 动态表单字段 — 标签解析回归', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockGetTicket.mockResolvedValue(ticketWithFormFields);
  });

  it('按工单类型 schema 显示中文字段标签，而不是 formFields 的原始键名', async () => {
    mockGetTicketType.mockResolvedValue(typeDefinition);
    renderDetail();

    await waitFor(() => expect(mockGetTicketType).toHaveBeenCalledWith(5));

    // 标签来自类型 schema
    expect(await screen.findByText('账号类型')).toBeInTheDocument();
    expect(screen.getByText('目标系统')).toBeInTheDocument();
    // 值照常渲染
    expect(screen.getByText('new')).toBeInTheDocument();
    expect(screen.getByText('GitLab')).toBeInTheDocument();
    // 原始键名不得再作为展示标签出现（此前 label={key} 的缺陷）
    expect(screen.queryByText('account_type')).toBeNull();
  });

  it('类型 schema 拉取失败时回退原始键名，详情仍然可读', async () => {
    mockGetTicketType.mockRejectedValue(new Error('ticket-types unavailable'));
    renderDetail();

    await waitFor(() => expect(mockGetTicketType).toHaveBeenCalled());
    expect(await screen.findByText('account_type')).toBeInTheDocument();
    expect(screen.getByText('target_system')).toBeInTheDocument();
  });
});
