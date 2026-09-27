/**
 * 工单筛选面板组件
 * 包含所有筛选条件
 */

import React from 'react';
import { Form, Select, DatePicker, Input, Button, Space, Card, Tag } from 'antd';
import { Search, Filter, X } from 'lucide-react';
import type { TicketFilters } from '@/types/ticket';
import {
  ITSMMainType,
  ITSMMainTypeConfig,
  TicketPriority,
  TicketPriorityConfig,
  TicketStatus,
  TicketStatusConfig,
} from '@/constants/taxonomy';

const { RangePicker } = DatePicker;

export interface TicketsFiltersPanelProps {
  filters: TicketFilters;
  onChange: (filters: Partial<TicketFilters>) => void;
  onReset: () => void;
  onSearch: () => void;
  loading?: boolean;
  collapsed?: boolean;
}

/**
 * 工单筛选面板
 */
export const TicketsFiltersPanel: React.FC<TicketsFiltersPanelProps> = ({
  filters,
  onChange,
  onReset,
  onSearch,
  loading = false,
  collapsed = false,
}) => {
  const [form] = Form.useForm();

  // 筛选项一律来自 @/constants/taxonomy，不在此手写字面量。
  // 旧列表包含 pending_approval 与 request，两者都不在后端工单状态机/type 白名单内，
  // 选中后后端筛选条件永远匹配不到任何工单。
  const statusOptions = Object.values(TicketStatus).map((status) => ({
    value: status,
    label: TicketStatusConfig[status].label,
  }));

  const priorityOptions = Object.values(TicketPriority).map((priority) => ({
    value: priority,
    label: TicketPriorityConfig[priority].label,
    color: TicketPriorityConfig[priority].color,
  }));

  const typeOptions = [
    ITSMMainType.INCIDENT,
    ITSMMainType.SERVICE_REQUEST,
    ITSMMainType.PROBLEM,
    ITSMMainType.CHANGE,
  ].map((type) => ({ value: type, label: ITSMMainTypeConfig[type].label }));

  // 处理表单值变化
  const handleValuesChange = (changedValues: Partial<TicketFilters>) => {
    onChange(changedValues);
  };

  // 重置筛选
  const handleReset = () => {
    form.resetFields();
    onReset();
  };

  // 渲染活跃的筛选标签
  const renderActiveFilters = () => {
    const tags: React.ReactNode[] = [];

    if (filters.status) {
      tags.push(
        <Tag key="status" closable onClose={() => onChange({ status: undefined })}>
          状态: {filters.status}
        </Tag>
      );
    }

    if (filters.priority) {
      tags.push(
        <Tag key="priority" closable onClose={() => onChange({ priority: undefined })}>
          优先级: {filters.priority}
        </Tag>
      );
    }

    if (filters.type) {
      tags.push(
        <Tag key="type" closable onClose={() => onChange({ type: undefined })}>
          类型: {filters.type}
        </Tag>
      );
    }

    if (filters.assigneeId && filters.assigneeId.length > 0) {
      tags.push(
        <Tag key="assignee" closable onClose={() => onChange({ assigneeId: undefined })}>
          指派人: {filters.assigneeId.join(', ')}
        </Tag>
      );
    }

    return tags.length > 0 ? (
      <div className="mb-4">
        <Space wrap>
          <span className="text-sm text-gray-600">已选筛选:</span>
          {tags}
          <Button type="link" size="small" onClick={handleReset}>
            清除所有
          </Button>
        </Space>
      </div>
    ) : null;
  };

  if (collapsed) {
    return <div className="mb-4">{renderActiveFilters()}</div>;
  }

  return (
    <Card
      title={
        <Space>
          <Filter />
          <span>筛选条件</span>
        </Space>
      }
      className="mb-4"
    >
      {renderActiveFilters()}

      <Form
        form={form}
        layout="vertical"
        initialValues={filters}
        onValuesChange={handleValuesChange}
      >
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {/* 搜索关键词 */}
          <Form.Item label="搜索" name="search">
            <Input placeholder="搜索工单编号、标题" prefix={<Search />} allowClear />
          </Form.Item>

          {/* 状态筛选 */}
          <Form.Item label="状态" name="status" data-testid="status-filter">
            <Select placeholder="选择状态" allowClear mode="multiple" options={statusOptions.map(option => ({
              value: option.value,
              label: option.label,
            }))} />
          </Form.Item>

          {/* 优先级筛选 */}
          <Form.Item label="优先级" name="priority" data-testid="priority-filter">
            <Select placeholder="选择优先级" allowClear mode="multiple" options={priorityOptions.map(option => ({
              value: option.value,
              label: <Tag color={option.color}>{option.label}</Tag>,
            }))} />
          </Form.Item>

          {/* 类型筛选 */}
          <Form.Item label="类型" name="type">
            <Select placeholder="选择类型" allowClear options={typeOptions.map(option => ({
              value: option.value,
              label: option.label,
            }))} />
          </Form.Item>

          {/* 指派人筛选 */}
          <Form.Item label="指派人" name="assignee_id">
            <Select
              placeholder="选择指派人"
              allowClear
              showSearch
              filterOption={(input, option) =>
                String(option?.label ?? '')
                  .toLowerCase()
                  .includes(String(input).toLowerCase())
              }
              options={[
                { value: 1, label: '用户1' },
                { value: 2, label: '用户2' },
              ]}
            />
          </Form.Item>

          {/* 创建人筛选 */}
          <Form.Item label="创建人" name="requester_id">
            <Select placeholder="选择创建人" allowClear showSearch options={[
              { value: 1, label: '用户1' },
              { value: 2, label: '用户2' },
            ]} />
          </Form.Item>

          {/* 日期范围 */}
          <Form.Item label="创建时间" name="date_range">
            <RangePicker style={{ width: '100%' }} placeholder={['开始日期', '结束日期']} />
          </Form.Item>

          {/* 标签筛选 */}
          <Form.Item label="标签" name="tags">
            <Select mode="tags" placeholder="输入或选择标签" allowClear options={[
              { value: 'urgent', label: '紧急' },
              { value: 'vip', label: 'VIP' },
              { value: 'bug', label: 'Bug' },
            ]} />
          </Form.Item>
        </div>

        {/* 操作按钮 */}
        <div className="flex justify-end space-x-2 mt-4">
          <Button icon={<X />} onClick={handleReset}>
            重置
          </Button>
          <Button type="primary" icon={<Search />} onClick={onSearch} loading={loading}>
            搜索
          </Button>
        </div>
      </Form>
    </Card>
  );
};

export default TicketsFiltersPanel;
