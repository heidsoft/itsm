/**
 * 工单列表「类型」列必须覆盖后端 type 词表的全部取值。
 *
 * 曾经的缺陷：TICKET_TYPE_CONFIG 只覆盖 ITSMMainType 的 4 个 ITIL 主类型，
 * 漏掉 `ticket`（建单页默认类型）与 `improvement`。列渲染是
 * `TICKET_TYPE_CONFIG[type] ?? type`，漏配即回落成原始枚举，
 * 于是表格里 incident 显示「事件」、同一列却出现英文 `ticket`。
 *
 * 这里按后端 dto.CreateTicketRequest 的 oneof 逐个断言，新增类型时测试会先红。
 */
import { render } from '@testing-library/react';
import type { ColumnsType } from 'antd/es/table';
import type { Ticket } from '@/lib/api/types';
import { buildTicketListColumns, TICKET_TYPE_CONFIG } from '../TicketListColumns';

/** 后端 dto.CreateTicketRequest.Type 的 oneof 词表。 */
const BACKEND_TYPE_VOCABULARY = [
  'incident',
  'service_request',
  'change',
  'ticket',
  'problem',
  'improvement',
] as const;

const ticketOfType = (type: string) =>
  ({
    id: 1,
    ticketNumber: 'TKT-202610-000001',
    title: '打印机无法联网',
    status: 'new',
    priority: 'medium',
    type,
    createdAt: '2026-10-18T00:00:00Z',
    updatedAt: '2026-10-18T00:00:00Z',
  }) as unknown as Ticket;

function typeColumn() {
  const columns = buildTicketListColumns(
    { onOpen: jest.fn(), onEdit: jest.fn(), onClose: jest.fn(), canUpdate: true },
    { sortBy: 'created_at', sortOrder: 'desc' }
  ) as ColumnsType<Ticket>;
  const column = columns.find(c => c.key === 'type');
  if (!column?.render) throw new Error('未找到类型列渲染器');
  return column.render as (value: string, record: Ticket) => React.ReactNode;
}

describe('工单列表类型列', () => {
  it('后端 type 词表每个取值都有中文显示名', () => {
    const missing = BACKEND_TYPE_VOCABULARY.filter(t => !TICKET_TYPE_CONFIG[t]);
    expect(missing).toEqual([]);
  });

  it('渲染结果不含原始英文枚举', () => {
    const renderType = typeColumn();
    for (const type of BACKEND_TYPE_VOCABULARY) {
      const { container, unmount } = render(<>{renderType(type, ticketOfType(type))}</>);
      const text = container.textContent ?? '';
      expect(text).not.toBe(type);
      expect(text).toMatch(/[\u4e00-\u9fa5]/);
      unmount();
    }
  });

  it('ticket 显示为「工单」而不是回落原文', () => {
    const renderType = typeColumn();
    const { container } = render(<>{renderType('ticket', ticketOfType('ticket'))}</>);
    expect(container.textContent).toBe('工单');
  });

  it('未知类型仍回落原文，不吞掉信息', () => {
    const renderType = typeColumn();
    const { container } = render(<>{renderType('brand_new_type', ticketOfType('brand_new_type'))}</>);
    expect(container.textContent).toBe('brand_new_type');
  });
});
