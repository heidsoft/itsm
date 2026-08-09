'use client';

import type { Dayjs } from 'dayjs';
import { Button, Col, DatePicker, Row, Select, Tag } from 'antd';

import { TicketStatus, TicketPriority, TicketType } from '@/lib/services/ticket-service';

import { TICKET_STATUS_CONFIG, PRIORITY_CONFIG, TICKET_TYPE_CONFIG } from './TicketListColumns';

const { RangePicker } = DatePicker;

export interface TicketFilterValues {
  readonly status?: TicketStatus;
  readonly priority?: TicketPriority;
  readonly type?: TicketType;
  readonly createdRange?: readonly [Dayjs, Dayjs] | null;
}

interface TicketListFiltersProps {
  readonly values: TicketFilterValues;
  readonly onChange: (next: TicketFilterValues) => void;
  readonly onClear: () => void;
}

/**
 * Controlled filter panel for the ticket list. Renders four columns of
 * selects plus a clear button. The parent owns the actual filter state and
 * translates the values into the query shape used by `useTickets`.
 */
export function TicketListFilters({ values, onChange, onClear }: TicketListFiltersProps) {
  const handleStatusChange = (status: TicketStatus | undefined) => onChange({ ...values, status });

  const handlePriorityChange = (priority: TicketPriority | undefined) =>
    onChange({ ...values, priority });

  const handleTypeChange = (type: TicketType | undefined) => onChange({ ...values, type });

  const handleDateChange = (createdRange: readonly [Dayjs, Dayjs] | null) =>
    onChange({ ...values, createdRange });

  const statusOptions = (
    Object.entries(TICKET_STATUS_CONFIG) as [TicketStatus, { color: string; text: string }][]
  ).map(([value, config]) => ({
    value,
    label: <Tag color={config.color}>{config.text}</Tag>,
  }));

  const priorityOptions = (
    Object.entries(PRIORITY_CONFIG) as [TicketPriority, { color: string; text: string }][]
  ).map(([value, config]) => ({
    value,
    label: <Tag color={config.color}>{config.text}</Tag>,
  }));

  const typeOptions = (Object.entries(TICKET_TYPE_CONFIG) as [TicketType, string][]).map(
    ([value, text]) => ({ value, label: text })
  );

  return (
    <>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} md={6}>
          <Select<TicketStatus>
            placeholder='状态'
            value={values.status}
            onChange={handleStatusChange}
            allowClear
            style={{ width: '100%' }}
            options={statusOptions}
          />
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Select<TicketPriority>
            placeholder='优先级'
            value={values.priority}
            onChange={handlePriorityChange}
            allowClear
            style={{ width: '100%' }}
            options={priorityOptions}
          />
        </Col>
        <Col xs={24} sm={12} md={6}>
          <Select<TicketType>
            placeholder='类型'
            value={values.type}
            onChange={handleTypeChange}
            allowClear
            style={{ width: '100%' }}
            options={typeOptions}
          />
        </Col>
        <Col xs={24} sm={12} md={6}>
          <RangePicker
            placeholder={['开始日期', '结束日期']}
            value={values.createdRange ? (values.createdRange as [Dayjs, Dayjs]) : null}
            onChange={dates => handleDateChange(dates as readonly [Dayjs, Dayjs] | null)}
            style={{ width: '100%' }}
          />
        </Col>
      </Row>
      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col>
          <Button onClick={onClear}>清空过滤器</Button>
        </Col>
      </Row>
    </>
  );
}
