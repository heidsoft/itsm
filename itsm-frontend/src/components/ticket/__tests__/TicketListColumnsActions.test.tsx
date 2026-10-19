/**
 * 工单列表行内操作按钮的权限收口。
 *
 * 后端 PUT /tickets/:id（编辑与关闭共用 ticket:update）之外的角色只有 ticket:read，
 * 此前列表照常给出编辑/关闭图标，点下去只会 403。列构造是纯函数，直接断言产物。
 */
import { render, screen } from '@testing-library/react';
import type { ColumnsType } from 'antd/es/table';
import type { Ticket } from '@/lib/api/types';
import { buildTicketListColumns } from '../TicketListColumns';

const ticket = (status: string) =>
  ({
    id: 1,
    ticketNumber: 'TKT-202610-000001',
    title: '打印机无法联网',
    status,
    priority: 'medium',
    type: 'service_request',
    source: 'portal',
    createdAt: '2026-10-18T00:00:00Z',
    updatedAt: '2026-10-18T00:00:00Z',
  }) as unknown as Ticket;

function renderActions(canUpdate: boolean, status: string) {
  const columns = buildTicketListColumns(
    { onOpen: jest.fn(), onEdit: jest.fn(), onClose: jest.fn(), canUpdate },
    { sortBy: 'created_at', sortOrder: 'desc' }
  ) as ColumnsType<Ticket>;
  const actionsColumn = columns.find(column => column.key === 'actions');
  if (!actionsColumn?.render) throw new Error('操作列缺少 render');
  const node = actionsColumn.render('', ticket(status), 0);
  return render(<>{node}</>);
}

describe('工单列表行内操作 — 权限收口', () => {
  it('无 ticket:update 时不提供编辑与关闭入口', () => {
    renderActions(false, 'resolved');
    expect(screen.getByLabelText('查看工单')).toBeInTheDocument();
    expect(screen.queryByLabelText('编辑工单')).toBeNull();
    expect(screen.queryByLabelText('关闭工单')).toBeNull();
  });

  it('有 ticket:update 且工单已解决时给出编辑与关闭', () => {
    renderActions(true, 'resolved');
    expect(screen.getByLabelText('编辑工单')).toBeInTheDocument();
    expect(screen.getByLabelText('关闭工单')).toBeInTheDocument();
  });

  it('终态工单只隐藏关闭，不隐藏编辑', () => {
    renderActions(true, 'closed');
    expect(screen.getByLabelText('编辑工单')).toBeInTheDocument();
    expect(screen.queryByLabelText('关闭工单')).toBeNull();
  });
});
