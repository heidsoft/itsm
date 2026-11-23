import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import {
  mapServiceItems,
  mapKnowledgeItems,
  mapAiToolItems,
  ApprovalOverviewTable,
  APPROVAL_DOMAIN_LINKS,
  type DomainState,
} from '../ApprovalOverviewList';

jest.mock('next/link', () => ({
  __esModule: true,
  default: ({ href, children }: { href: string; children: React.ReactNode }) => (
    <a href={href}>{children}</a>
  ),
}));

const labels = {
  title: '待办事项',
  subtitle: '申请人 / 来源',
  time: '提交时间',
  status: '状态',
  actions: '操作',
  goHandle: '前往处理',
  empty: '暂无待审批事项',
  loadFailed: '加载待办失败，这不代表队列为空',
  retry: '重试',
};

const statusLabel = (status: string) => ({ pending: '待审批', under_review: '待审核' })[status] ?? status;

describe('待办映射纯函数', () => {
  it('目录审批：服务名作标题、申请人作来源、状态固定待审批', () => {
    const rows = mapServiceItems([{ id: 7, serviceName: '邮箱开通', requesterName: '张三', createdAt: '2026-11-20T10:00:00Z' }]);
    expect(rows[0]).toMatchObject({ title: '邮箱开通', subtitle: '张三', status: 'pending', key: 'service-7' });
  });

  it('知识审核：只保留 under_review，过滤掉已通过/草稿', () => {
    const rows = mapKnowledgeItems([
      { id: 1, title: '待审文章', reviewStatus: 'under_review', authorName: '李四' },
      { id: 2, title: '已过文章', reviewStatus: 'approved', authorName: '王五' },
      { id: 3, title: '草稿', reviewStatus: 'draft', authorName: '赵六' },
    ]);
    expect(rows).toHaveLength(1);
    expect(rows[0]).toMatchObject({ title: '待审文章', subtitle: '李四', status: 'under_review' });
  });

  it('工具调用审批：工具名作标题、风险等级进来源、保留真实审批态', () => {
    const rows = mapAiToolItems([
      { id: 9, toolName: 'create_ticket', approvalState: 'pending_agent', riskLevel: 'high', createdAt: '', arguments: '', status: '', needsApproval: true, conversationId: 1, userId: 1 },
    ]);
    expect(rows[0]).toMatchObject({ title: 'create_ticket', subtitle: 'risk: high', status: 'pending_agent' });
  });
});

describe('ApprovalOverviewTable', () => {
  const baseState: DomainState = {
    items: [{ key: 'service-1', title: '邮箱开通', subtitle: '张三', time: '3 天前', status: 'pending', statusColor: 'orange' }],
    loading: false,
    error: false,
  };

  it('加载失败：显示错误与重试，绝不渲染空态表格', () => {
    const onRetry = jest.fn();
    render(
      <ApprovalOverviewTable
        domain="service"
        state={{ items: [], loading: false, error: true }}
        onRetry={onRetry}
        statusLabel={statusLabel}
        labels={labels}
      />
    );
    expect(screen.getByRole('alert')).toHaveTextContent(labels.loadFailed);
    expect(screen.queryByText(labels.empty)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /重\s*试/ }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it('直达链接指向对应审批域原页', () => {
    render(
      <ApprovalOverviewTable domain="aiTool" state={baseState} onRetry={jest.fn()} statusLabel={statusLabel} labels={labels} />
    );
    const links = screen.getAllByRole('link');
    expect(links.length).toBeGreaterThan(0);
    links.forEach((link) => expect(link).toHaveAttribute('href', APPROVAL_DOMAIN_LINKS.aiTool));
  });

  it('三个域的直达路由互不串页', () => {
    expect(APPROVAL_DOMAIN_LINKS).toEqual({
      service: '/service-catalog/approvals',
      knowledge: '/knowledge/reviews',
      aiTool: '/ai/approval',
    });
  });
});
