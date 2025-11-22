/**
 * 工单筛选面板组件
 * 包含所有筛选条件
 */

import React from 'react';
import { Form, Select, DatePicker, Input, Button, Space, Card, Tag } from 'antd';
import { SearchOutlined, ClearOutlined, FilterOutlined } from '@ant-design/icons';
import type { TicketFilters } from '@/types/ticket';

const { RangePicker } = DatePicker;
const { Option } = Select;

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

  // 状态选项
  const statusOptions = [
    { label: '新建', value: 'new' },
    { label: '处理中', value: 'in_progress' },
    { label: '待审批', value: 'pending_approval' },
    { label: '已解决', value: 'resolved' },
    { label: '已关闭', value: 'closed' },
    { label: '已取消', value: 'cancelled' },
  ];

  // 优先级选项
  const priorityOptions = [
    { label: '低', value: 'low', color: 'default' },
    { label: '中', value: 'medium', color: 'blue' },
    { label: '高', value: 'high', color: 'orange' },
    { label: '紧急', value: 'urgent', color: 'red' },
    { label: '严重', value: 'critical', color: 'red' },
  ];

  // 类型选项
  const typeOptions = [
    { label: '事件', value: 'incident' },
    { label: '请求', value: 'request' },
    { label: '问题', value: 'problem' },
    { label: '变更', value: 'change' },
  ];

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

    if (filters.assignee_id) {
      tags.push(
        <Tag key="assignee" closable onClose={() => onChange({ assignee_id: undefined })}>
          指派人: {filters.assignee_id}
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
    return (
      <div className="mb-4">
        {renderActiveFilters()}
      </div>
    );
  }

  return (
    <Card 
      title={
        <Space>
          <FilterOutlined />
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
            <Input
              placeholder="搜索工单编号、标题"
              prefix={<SearchOutlined />}
              allowClear
            />
          </Form.Item>

          {/* 状态筛选 */}
          <Form.Item label="状态" name="status" data-testid="status-filter">
            <Select
              placeholder="选择状态"
              allowClear
              mode="multiple"
            >
              {statusOptions.map(option => (
                <Option key={option.value} value={option.value}>
                  {option.label}
                </Option>
              ))}
            </Select>
          </Form.Item>

          {/* 优先级筛选 */}
          <Form.Item label="优先级" name="priority" data-testid="priority-filter">
            <Select
              placeholder="选择优先级"
              allowClear
              mode="multiple"
            >
              {priorityOptions.map(option => (
                <Option key={option.value} value={option.value}>
                  <Tag color={option.color}>{option.label}</Tag>
                </Option>
              ))}
            </Select>
          </Form.Item>

          {/* 类型筛选 */}
          <Form.Item label="类型" name="type">
            <Select
              placeholder="选择类型"
              allowClear
            >
              {typeOptions.map(option => (
                <Option key={option.value} value={option.value}>
                  {option.label}
                </Option>
              ))}
            </Select>
          </Form.Item>

          {/* 指派人筛选 */}
          <Form.Item label="指派人" name="assignee_id">
            <Select
              placeholder="选择指派人"
              allowClear
              showSearch
              filterOption={(input, option) =>
                (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
              }
            >
              {/* 这里应该从API获取用户列表 */}
              <Option value={1} label="用户1">用户1</Option>
              <Option value={2} label="用户2">用户2</Option>
            </Select>
          </Form.Item>

          {/* 创建人筛选 */}
          <Form.Item label="创建人" name="requester_id">
            <Select
              placeholder="选择创建人"
              allowClear
              showSearch
            >
              <Option value={1}>用户1</Option>
              <Option value={2}>用户2</Option>
            </Select>
          </Form.Item>

          {/* 日期范围 */}
          <Form.Item label="创建时间" name="date_range">
            <RangePicker
              style={{ width: '100%' }}
              placeholder={['开始日期', '结束日期']}
            />
          </Form.Item>

          {/* 标签筛选 */}
          <Form.Item label="标签" name="tags">
            <Select
              mode="tags"
              placeholder="输入或选择标签"
              allowClear
            >
              <Option value="urgent">紧急</Option>
              <Option value="vip">VIP</Option>
              <Option value="bug">Bug</Option>
            </Select>
          </Form.Item>
        </div>

        {/* 操作按钮 */}
        <div className="flex justify-end space-x-2 mt-4">
          <Button
            icon={<ClearOutlined />}
            onClick={handleReset}
          >
            重置
          </Button>
          <Button
            type="primary"
            icon={<SearchOutlined />}
            onClick={onSearch}
            loading={loading}
          >
            搜索
          </Button>
        </div>
      </Form>
    </Card>
  );
};

export default TicketsFiltersPanel;

