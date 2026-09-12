/**
 * 回归：分配工单弹窗的处理人搜索。
 *
 * 修复前 filterOption 写作 (option?.label as unknown as string)?.toLowerCase()，
 * 而 options 的 label 是带部门 Tag 的 React 节点，对象上没有 toLowerCase，
 * 用户在搜索框一输入就抛 TypeError 并连带卸载整个弹窗——派单是工单域的核心
 * 动作，等于该功能实际不可用。修复后 option 额外携带 searchText 字符串供匹配。
 */
import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { App } from 'antd';
import TicketDetail from '../TicketDetail';

const mockAssignTicket = jest.fn();
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
    assignTicket: (...args: unknown[]) => mockAssignTicket(...args),
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

jest.mock('@/lib/hooks/useUserListQuery', () => ({
  useUserListQuery: () => ({
    isLoading: false,
    data: {
      users: [
        { id: 1, username: 'admin', name: '系统管理员', department: 'IT部门' },
        { id: 2, username: 'zhangsan', name: '张三', department: '运维部' },
      ],
    },
  }),
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

const ticket = {
  id: 17,
  ticketNumber: 'TKT-202609-000018',
  title: '审计测试工单',
  description: '用于验证分配弹窗搜索',
  status: 'new',
  priority: 'medium',
  type: 'incident',
  requesterId: 1,
  assigneeId: 1,
  tenantId: 1,
  version: 1,
  createdAt: '2026-09-11T22:50:50Z',
  updatedAt: '2026-09-11T22:50:50Z',
};

async function openAssignModal() {
  mockGetTicket.mockResolvedValue(ticket);
  render(
    <App>
      <TicketDetail />
    </App>
  );
  await waitFor(() => expect(mockGetTicket).toHaveBeenCalled());
  const assignButton = await screen.findByText('ticketDetail.assign');
  fireEvent.click(assignButton);
  return await screen.findByText('ticketDetail.assignTitle');
}

describe('TicketDetail 分配工单弹窗 — 处理人搜索回归', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('在处理人搜索框输入不会抛错，弹窗保持打开', async () => {
    await openAssignModal();

    const searchInput = document.querySelector('.ant-modal .ant-select input');
    expect(searchInput).not.toBeNull();

    // 修复前：label 是 React 节点，.toLowerCase 为 undefined，此处抛
    // TypeError: (intermediate value)?.toLowerCase is not a function
    expect(() => fireEvent.change(searchInput as Element, { target: { value: 'zhang' } })).not.toThrow();

    await waitFor(() =>
      expect(document.querySelector('.ant-modal-wrap')).not.toBeNull()
    );
    expect(screen.getByText('ticketDetail.assignTitle')).toBeInTheDocument();
  });

  it('按用户名过滤出匹配的处理人', async () => {
    await openAssignModal();

    const searchInput = document.querySelector('.ant-modal .ant-select input') as Element;
    fireEvent.change(searchInput, { target: { value: 'zhangsan' } });

    const dropdown = await waitFor(() => {
      const dd = [...document.querySelectorAll('.ant-select-dropdown')].find(
        d => !d.classList.contains('ant-select-dropdown-hidden')
      );
      expect(dd).toBeTruthy();
      return dd as HTMLElement;
    });

    const optionTexts = [...dropdown.querySelectorAll('.ant-select-item-option')].map(o =>
      o.textContent?.replace(/\s+/g, '') ?? ''
    );
    expect(optionTexts).toEqual(['张三(zhangsan)运维部']);
  });

  it('按部门名过滤，部门 Tag 文本也参与匹配', async () => {
    await openAssignModal();

    const searchInput = document.querySelector('.ant-modal .ant-select input') as Element;
    fireEvent.change(searchInput, { target: { value: 'IT部门' } });

    const dropdown = await waitFor(() => {
      const dd = [...document.querySelectorAll('.ant-select-dropdown')].find(
        d => !d.classList.contains('ant-select-dropdown-hidden')
      );
      expect(dd).toBeTruthy();
      return dd as HTMLElement;
    });

    const optionTexts = [...dropdown.querySelectorAll('.ant-select-item-option')].map(o =>
      o.textContent?.replace(/\s+/g, '') ?? ''
    );
    expect(optionTexts).toEqual(['系统管理员(admin)IT部门']);
  });

  it('无匹配时展示空状态而不是崩溃', async () => {
    await openAssignModal();

    const searchInput = document.querySelector('.ant-modal .ant-select input') as Element;
    fireEvent.change(searchInput, { target: { value: '不存在的人' } });

    await waitFor(() => {
      const dd = [...document.querySelectorAll('.ant-select-dropdown')].find(
        d => !d.classList.contains('ant-select-dropdown-hidden')
      );
      expect(dd).toBeTruthy();
      expect(dd!.querySelectorAll('.ant-select-item-option')).toHaveLength(0);
    });
    expect(screen.getByText('ticketDetail.assignTitle')).toBeInTheDocument();
  });
});
