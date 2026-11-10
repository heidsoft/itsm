'use client';

import React, { useState, useCallback, useMemo } from 'react';
import {
  Card,
  Form,
  Input,
  Select,
  Button,
  Space,
  Row,
  Col,
  Divider,
  Tag,
  Popover,
  Collapse,
  Typography,
} from 'antd';
import { Search, Filter, RotateCcw } from 'lucide-react';
import dayjs, { type Dayjs } from 'dayjs';
import type { RangePickerProps } from 'antd/es/date-picker';
import { AppDateRangePicker } from '@/components/ui/AppDatePicker';
import { useAuthStore } from '@/lib/store/auth-store';

const { Panel } = Collapse;
const { TextArea } = Input;
const { Text } = Typography;

export interface AdvancedSearchFilters {
  // 基础信息
  keyword?: string;
  ticketNumber?: string;
  title?: string;
  description?: string;

  // 状态和分类
  status?: string[];
  priority?: string[];
  type?: string[];
  category?: string[];

  // 人员
  reporterId?: number;
  assigneeId?: number;
  createdBy?: number;

  // 时间范围
  createdAfter?: string;
  createdBefore?: string;
  updatedAfter?: string;
  updatedBefore?: string;
  dueAfter?: string;
  dueBefore?: string;
  resolvedAfter?: string;
  resolvedBefore?: string;

  // 配置项
  configurationItemId?: number;

  // 来源和渠道
  source?: string[];
  channel?: string[];

  // SLA相关
  slaBreach?: boolean;
  slaWarning?: boolean;

  // 自定义字段
  tags?: string[];
  metadata?: Record<string, unknown>;
}

export interface TicketAdvancedSearchProps {
  onSearch: (filters: AdvancedSearchFilters) => void;
  onReset: () => void;
  loading?: boolean;
  initialValues?: Partial<AdvancedSearchFilters>;
}

// 预定义的筛选选项
const TICKET_STATUS_OPTIONS = [
  { label: 'Create', value: 'new', color: 'blue' },
  { label: 'Pending', value: 'open', color: 'blue' },
  { label: 'In Progress', value: 'in_progress', color: 'orange' },
  { label: 'Waiting', value: 'pending', color: 'yellow' },
  { label: 'Resolved', value: 'resolved', color: 'green' },
  { label: 'Closed', value: 'closed', color: 'default' },
  { label: 'Cancelled', value: 'cancelled', color: 'red' },
];

const TICKET_PRIORITY_OPTIONS = [
  { label: 'Low', value: 'low', color: 'green' },
  { label: 'Medium', value: 'medium', color: 'orange' },
  { label: 'High', value: 'high', color: 'red' },
  { label: 'Urgent', value: 'urgent', color: 'purple' },
  { label: 'Critical', value: 'critical', color: 'red' },
];

const TICKET_TYPE_OPTIONS = [
  { label: 'Incident', value: 'incident' },
  { label: 'Request', value: 'request' },
  { label: 'Problem', value: 'problem' },
  { label: 'Change', value: 'change' },
  { label: 'Task', value: 'task' },
];

const TICKET_SOURCE_OPTIONS = [
  { label: 'Email', value: 'email' },
  { label: 'Phone', value: 'phone' },
  { label: 'Portal', value: 'portal' },
  { label: 'API', value: 'api' },
  { label: 'Monitoring', value: 'monitoring' },
  { label: 'Manual', value: 'manual' },
];

// 预设搜索模板 — assigneeId 使用 CURRENT_USER_ID 占位符，在 applyTemplate 时由认证上下文替换
const CURRENT_USER_ID = Symbol('currentUserId');

const SEARCH_TEMPLATES = [
  {
    name: "My to-do work order",
    description: 'Unfinished work order assigned to me',
    filters: {
      status: ['new', 'open', 'in_progress'],
      assigneeId: CURRENT_USER_ID as unknown as number,
    },
  },
  {
    name: "High priority ticket",
    description: 'All high priority and emergency tickets',
    filters: {
      priority: ['high', 'urgent', 'critical'],
    },
  },
  {
    name: "SLA is about to time out",
    description: 'Work order with SLA about to expire',
    filters: {
      slaWarning: true,
    },
  },
  {
    name: "Create a work order this week",
    description: 'All tickets created within this week',
    filters: {
      createdAfter: dayjs().startOf('week').format('YYYY-MM-DD'),
      createdBefore: dayjs().endOf('week').format('YYYY-MM-DD'),
    },
  },
  {
    name: "Resolved but not closed",
    description: 'Resolved but not closed ticket',
    filters: {
      status: ['resolved'],
    },
  },
];

// 提取各面板内容为常量（在组件外部使用会有类型问题，需要内联）
const TicketAdvancedSearch: React.FC<TicketAdvancedSearchProps> = ({
  onSearch,
  onReset,
  loading = false,
  initialValues = {},
}) => {
  const [form] = Form.useForm();
  const [activeTemplate, setActiveTemplate] = useState<string>('');
  const currentUserId = useAuthStore(state => state.user?.id);

  // 快速日期范围（需要在面板定义之前）
  const quickDateRanges: RangePickerProps['ranges'] = {
    Today: [dayjs().startOf('day'), dayjs().endOf('day')],
    Yesterday: [dayjs().subtract(1, 'day').startOf('day'), dayjs().subtract(1, 'day').endOf('day')],
    'This week': [dayjs().startOf('week'), dayjs().endOf('week')],
    'Last week': [dayjs().subtract(1, 'week').startOf('week'), dayjs().subtract(1, 'week').endOf('week')],
    'This month': [dayjs().startOf('month'), dayjs().endOf('month')],
    'Last month': [
      dayjs().subtract(1, 'month').startOf('month'),
      dayjs().subtract(1, 'month').endOf('month'),
    ],
    'Last 7 days': [dayjs().subtract(7, 'day'), dayjs()],
    'Last 30 days': [dayjs().subtract(30, 'day'), dayjs()],
  };

  // 面板内容
  const basicInfoPanel = (
    <Row gutter={[16, 0]}>
      <Col span={6}>
        <Form.Item label="Keyword search" name="keyword">
          <Input placeholder="Title, description, ticket number..." prefix={<Search />} />
        </Form.Item>
      </Col>
      <Col span={6}>
        <Form.Item label="Work order number" name="ticket_number">
          <Input placeholder="Precise work order number" />
        </Form.Item>
      </Col>
      <Col span={6}>
        <Form.Item label="Ticket Title" name="title">
          <Input placeholder="Ticket Title" />
        </Form.Item>
      </Col>
      <Col span={6}>
        <Form.Item label="Ticket Description" name="description">
          <TextArea rows={1} placeholder="Ticket Description" />
        </Form.Item>
      </Col>
    </Row>
  );

  const statusPanel = (
    <Row gutter={[16, 0]}>
      <Col span={6}>
        <Form.Item label="Status" name="status">
          <Select mode="multiple" placeholder="Select status" allowClear options={TICKET_STATUS_OPTIONS.map(option => ({ value: option.value, label: <Tag color={option.color}>{option.label}</Tag> }))} />
        </Form.Item>
      </Col>
      <Col span={6}>
        <Form.Item label="Priority" name="priority">
          <Select mode="multiple" placeholder="Select priority" allowClear options={TICKET_PRIORITY_OPTIONS.map(option => ({ value: option.value, label: <Tag color={option.color}>{option.label}</Tag> }))} />
        </Form.Item>
      </Col>
      <Col span={6}>
        <Form.Item label="Ticket Types" name="type">
          <Select mode="multiple" placeholder="Select" allowClear options={TICKET_TYPE_OPTIONS.map(option => ({ value: option.value, label: option.label }))} />
        </Form.Item>
      </Col>
      <Col span={6}>
        <Form.Item label="Source" name="source">
          <Select mode="multiple" placeholder="Select Source" allowClear options={TICKET_SOURCE_OPTIONS.map(option => ({ value: option.value, label: option.label }))} />
        </Form.Item>
      </Col>
    </Row>
  );

  const timePanel = (
    <Row gutter={[16, 0]}>
      <Col span={12}>
        <Form.Item label="Created" name="created_range">
          <AppDateRangePicker
            ranges={quickDateRanges}
            onChange={(dates: [Dayjs | null, Dayjs | null] | null) => {
              if (dates && dates[0] && dates[1]) {
                form.setFieldsValue({
                  createdAfter: dates[0].format('YYYY-MM-DD'),
                  createdBefore: dates[1].format('YYYY-MM-DD'),
                });
              } else {
                form.setFieldsValue({ createdAfter: undefined, createdBefore: undefined });
              }
            }}
          />
        </Form.Item>
      </Col>
      <Col span={12}>
        <Form.Item label="Updated At" name="updated_range">
          <AppDateRangePicker
            ranges={quickDateRanges}
            onChange={(dates: [Dayjs | null, Dayjs | null] | null) => {
              if (dates && dates[0] && dates[1]) {
                form.setFieldsValue({
                  updatedAfter: dates[0].format('YYYY-MM-DD'),
                  updatedBefore: dates[1].format('YYYY-MM-DD'),
                });
              } else {
                form.setFieldsValue({ updatedAfter: undefined, updatedBefore: undefined });
              }
            }}
          />
        </Form.Item>
      </Col>
      <Col span={12}>
        <Form.Item label="Deadline" name="due_range">
          <AppDateRangePicker
            ranges={quickDateRanges}
            onChange={(dates: [Dayjs | null, Dayjs | null] | null) => {
              if (dates && dates[0] && dates[1]) {
                form.setFieldsValue({
                  dueAfter: dates[0].format('YYYY-MM-DD'),
                  dueBefore: dates[1].format('YYYY-MM-DD'),
                });
              } else {
                form.setFieldsValue({ dueAfter: undefined, dueBefore: undefined });
              }
            }}
          />
        </Form.Item>
      </Col>
      <Col span={12}>
        <Form.Item label="Resolved At" name="resolved_range">
          <AppDateRangePicker
            ranges={quickDateRanges}
            onChange={(dates: [Dayjs | null, Dayjs | null] | null) => {
              if (dates && dates[0] && dates[1]) {
                form.setFieldsValue({
                  resolvedAfter: dates[0].format('YYYY-MM-DD'),
                  resolvedBefore: dates[1].format('YYYY-MM-DD'),
                });
              } else {
                form.setFieldsValue({ resolvedAfter: undefined, resolvedBefore: undefined });
              }
            }}
          />
        </Form.Item>
      </Col>
    </Row>
  );

  const slaPanel = (
    <Row gutter={[16, 0]}>
      <Col span={6}>
        <Form.Item label="SLAStatus" name="sla_status">
          <Select placeholder="Select SLA status" allowClear options={[{ value: "breach", label: "Timed out" }, { value: "warning", label: "Timeout imminent" }, { value: "normal", label: "OK" }]} />
        </Form.Item>
      </Col>
    </Row>
  );

  // 应用搜索模板 — 将 CURRENT_USER_ID 占位符替换为实际用户 ID
  const applyTemplate = useCallback(
    (template: (typeof SEARCH_TEMPLATES)[0]) => {
      const resolvedFilters: Record<string, unknown> = { ...template.filters };
      if (resolvedFilters.assigneeId === (CURRENT_USER_ID as unknown)) {
        if (currentUserId != null) {
          resolvedFilters.assigneeId = currentUserId;
        } else {
          delete resolvedFilters.assigneeId;
        }
      }
      form.setFieldsValue(resolvedFilters);
      setActiveTemplate(template.name);
    },
    [form, currentUserId]
  );

  // 执行搜索
  const handleSearch = useCallback(() => {
    const values = form.getFieldsValue() as Record<string, unknown>;

    // 处理日期范围
    const processedValues: AdvancedSearchFilters = {};

    Object.keys(values).forEach(key => {
      const value = values[key];
      if (
        value !== undefined &&
        value !== null &&
        value !== '' &&
        (Array.isArray(value) ? value.length > 0 : true)
      ) {
        Object.assign(processedValues, { [key]: value });
      }
    });

    onSearch(processedValues);
  }, [form, onSearch]);

  // 重置搜索条件
  const handleReset = useCallback(() => {
    form.resetFields();
    setActiveTemplate('');
    onReset();
  }, [form, onReset]);

  return (
    <Card
      title={
        <div className="flex items-center">
          <Filter className="mr-2" />
          <span>Advanced Search</span>
        </div>
      }
      size="small"
    >
      {/* 预设模板 */}
      <div className="mb-4">
        <Text strong className="mb-2 block">

          Quick search template:
        </Text>
        <Space wrap>
          {SEARCH_TEMPLATES.map(template => (
            <Popover key={template.name} content={template.description} placement="bottom">
              <Button
                size="small"
                type={activeTemplate === template.name ? 'primary' : 'default'}
                onClick={() => applyTemplate(template)}
              >
                {template.name}
              </Button>
            </Popover>
          ))}
        </Space>
      </div>

      <Divider />

      <Form form={form} layout="vertical" initialValues={initialValues} onFinish={handleSearch}>
        <Collapse
          defaultActiveKey={['basic']}
          ghost
          items={[
            { key: 'basic', label: 'Basic Information', children: basicInfoPanel },
            { key: 'status', label: 'Status and Classification', children: statusPanel },
            { key: 'time', label: 'Time Range', children: timePanel },
            { key: 'sla', label: 'SLA Monitor', children: slaPanel },
          ]}
        />

        {/* 操作按钮 */}
        <div className="flex justify-between items-center mt-6">
          <Space>
            <Button type="primary" icon={<Search />} htmlType="submit" loading={loading}>

              Search
            </Button>
            <Button icon={<RotateCcw />} onClick={handleReset}>

              Reset
            </Button>
          </Space>
        </div>
      </Form>
    </Card>
  );
};

export default TicketAdvancedSearch;
